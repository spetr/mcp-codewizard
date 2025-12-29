// Package index implements parallel file indexing with progress reporting.
package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spetr/mcp-codewizard/builtin/chunking/simple"
	"github.com/spetr/mcp-codewizard/internal/config"
	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// Indexer handles parallel file indexing.
type Indexer struct {
	config     *config.Config
	store      provider.VectorStore
	embedding  provider.EmbeddingProvider
	chunker    provider.ChunkingStrategy
	projectDir string
	configHash string

	// Progress tracking
	progressMu sync.Mutex
	progress   types.IndexProgress
	onProgress func(types.IndexProgress)
}

// Config contains indexer configuration.
type Config struct {
	ProjectDir string
	Config     *config.Config
	Store      provider.VectorStore
	Embedding  provider.EmbeddingProvider
	Chunker    provider.ChunkingStrategy
	OnProgress func(types.IndexProgress)
}

// New creates a new indexer.
func New(cfg Config) *Indexer {
	return &Indexer{
		config:     cfg.Config,
		store:      cfg.Store,
		embedding:  cfg.Embedding,
		chunker:    cfg.Chunker,
		projectDir: cfg.ProjectDir,
		configHash: cfg.Config.Hash(),
		onProgress: cfg.OnProgress,
	}
}

// Index indexes the project directory.
// It supports checkpoint/resume - if interrupted, run again to continue from where it left off.
func (idx *Indexer) Index(ctx context.Context, force bool) error {
	startTime := time.Now()

	// Phase 1: Scan files
	idx.updateProgress("scanning", 0, 0, 0, 0, "")

	files, err := idx.scanFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	slog.Info("scanned files", "total", len(files))
	idx.updateProgress("scanning", len(files), 0, 0, 0, "")

	// Filter for changed files if not forcing
	var filesToProcess []*types.SourceFile
	if force {
		filesToProcess = files
	} else {
		filesToProcess, err = idx.filterChangedFiles(ctx, files)
		if err != nil {
			return fmt.Errorf("failed to filter files: %w", err)
		}
	}

	slog.Info("files to process", "count", len(filesToProcess), "total", len(files))

	if len(filesToProcess) == 0 {
		slog.Info("no files need indexing")
		return nil
	}

	// Phase 2: Parallel chunking
	idx.updateProgress("chunking", len(filesToProcess), 0, 0, 0, "")

	allChunks, allSymbols, allRefs, err := idx.chunkFilesParallel(ctx, filesToProcess)
	if err != nil {
		return fmt.Errorf("chunking failed: %w", err)
	}

	slog.Info("chunking complete",
		"files", len(filesToProcess),
		"chunks", len(allChunks),
		"symbols", len(allSymbols),
		"refs", len(allRefs),
	)

	if len(allChunks) == 0 {
		slog.Info("no chunks to process")
		// Still mark files as processed
		for _, file := range filesToProcess {
			if err := idx.store.SetFileHash(file.Path, file.Hash, idx.configHash); err != nil {
				slog.Warn("failed to cache file hash", "file", file.Path, "error", err)
			}
		}
		return nil
	}

	// Phase 3: Generate embeddings in batches
	idx.updateProgress("embedding", len(filesToProcess), len(filesToProcess), len(allChunks), 0, "")

	chunksWithEmbeddings, err := idx.embedChunks(ctx, allChunks)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("embedding failed: %w", err)
	}

	// Phase 4: Store all data
	idx.updateProgress("storing", len(filesToProcess), len(filesToProcess), len(allChunks), len(allChunks), "")

	if err := idx.store.StoreChunks(chunksWithEmbeddings); err != nil {
		return fmt.Errorf("failed to store chunks: %w", err)
	}

	if len(allSymbols) > 0 {
		if err := idx.store.StoreSymbols(allSymbols); err != nil {
			slog.Warn("failed to store symbols", "error", err)
		}
	}

	if len(allRefs) > 0 {
		if err := idx.store.StoreReferences(allRefs); err != nil {
			slog.Warn("failed to store references", "error", err)
		}
	}

	// Mark all files as processed
	for _, file := range filesToProcess {
		if err := idx.store.SetFileHash(file.Path, file.Hash, idx.configHash); err != nil {
			slog.Warn("failed to cache file hash", "file", file.Path, "error", err)
		}
	}

	totalChunks := len(allChunks)
	totalSymbols := len(allSymbols)
	totalRefs := len(allRefs)
	processedFiles := len(filesToProcess)

	// Update metadata
	meta := &types.IndexMetadata{
		SchemaVersion:       1,
		CreatedAt:           time.Now(),
		LastUpdated:         time.Now(),
		ToolVersion:         "0.1.0",
		ConfigHash:          idx.configHash,
		EmbeddingProvider:   idx.embedding.Name(),
		EmbeddingModel:      idx.config.Embedding.Model,
		EmbeddingDimensions: idx.embedding.Dimensions(),
		ChunkingStrategy:    idx.chunker.Name(),
		RerankerModel:       idx.config.Reranker.Model,
	}

	if err := idx.store.SetMetadata(meta); err != nil {
		return fmt.Errorf("failed to store metadata: %w", err)
	}

	duration := time.Since(startTime)
	slog.Info("indexing complete",
		"files", processedFiles,
		"chunks", totalChunks,
		"symbols", totalSymbols,
		"refs", totalRefs,
		"duration", duration.Round(time.Millisecond),
	)

	return nil
}

// scanFiles scans the project for files to index.
func (idx *Indexer) scanFiles(ctx context.Context) ([]*types.SourceFile, error) {
	var files []*types.SourceFile

	// Try to use git ls-files first
	if idx.config.Index.UseGitIgnore {
		gitFiles, err := idx.scanWithGit(ctx)
		if err == nil && len(gitFiles) > 0 {
			return gitFiles, nil
		}
		slog.Debug("git scan failed, falling back to filesystem", "error", err)
	}

	// Fall back to filesystem walk
	slog.Debug("starting filesystem walk", "dir", idx.projectDir, "include_patterns", idx.config.Index.Include)
	err := filepath.WalkDir(idx.projectDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Skip directories in exclude list
		if d.IsDir() {
			relPath, _ := filepath.Rel(idx.projectDir, path)
			for _, pattern := range idx.config.Index.Exclude {
				if matchGlob(pattern, relPath+"/") {
					slog.Debug("excluding directory", "path", relPath, "pattern", pattern)
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check include patterns
		relPath, _ := filepath.Rel(idx.projectDir, path)
		included := false
		for _, pattern := range idx.config.Index.Include {
			if matchGlob(pattern, relPath) {
				included = true
				slog.Debug("file matched include", "path", relPath, "pattern", pattern)
				break
			}
		}

		if !included {
			slog.Debug("file not included", "path", relPath)
			return nil
		}

		// Check exclude patterns
		for _, pattern := range idx.config.Index.Exclude {
			if matchGlob(pattern, relPath) {
				return nil
			}
		}

		// Read file
		file, err := idx.readFile(path)
		if err != nil {
			slog.Warn("failed to read file", "path", path, "error", err)
			return nil
		}

		files = append(files, file)

		if len(files) >= idx.config.Limits.MaxFiles {
			return fmt.Errorf("max files limit reached: %d", idx.config.Limits.MaxFiles)
		}

		return nil
	})

	return files, err
}

// scanWithGit uses git ls-files to get tracked files.
func (idx *Indexer) scanWithGit(ctx context.Context) ([]*types.SourceFile, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = idx.projectDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []*types.SourceFile
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check include patterns
		included := false
		for _, pattern := range idx.config.Index.Include {
			if matchGlob(pattern, line) {
				included = true
				break
			}
		}

		if !included {
			continue
		}

		// Check exclude patterns
		excluded := false
		for _, pattern := range idx.config.Index.Exclude {
			if matchGlob(pattern, line) {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		fullPath := filepath.Join(idx.projectDir, line)
		file, err := idx.readFile(fullPath)
		if err != nil {
			slog.Warn("failed to read file", "path", line, "error", err)
			continue
		}

		files = append(files, file)

		if len(files) >= idx.config.Limits.MaxFiles {
			break
		}
	}

	return files, nil
}

// readFile reads a file and creates a SourceFile.
func (idx *Indexer) readFile(path string) (*types.SourceFile, error) {
	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	maxSize := parseSize(idx.config.Limits.MaxFileSize)
	if info.Size() > maxSize {
		return nil, fmt.Errorf("file too large: %d > %d", info.Size(), maxSize)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	file := &types.SourceFile{
		Path:     path,
		Content:  content,
		Language: simple.DetectLanguage(path),
	}
	file.Hash = file.ComputeHash()

	return file, nil
}

// filterChangedFiles filters out files that haven't changed.
func (idx *Indexer) filterChangedFiles(ctx context.Context, files []*types.SourceFile) ([]*types.SourceFile, error) {
	var changed []*types.SourceFile

	for _, file := range files {
		cachedHash, err := idx.store.GetFileHash(file.Path)
		if err != nil {
			slog.Warn("failed to get cached hash", "file", file.Path, "error", err)
			changed = append(changed, file)
			continue
		}

		if cachedHash != file.Hash {
			// File has changed, delete old chunks first
			if err := idx.store.DeleteChunksByFile(file.Path); err != nil {
				slog.Warn("failed to delete old chunks", "file", file.Path, "error", err)
			}
			changed = append(changed, file)
		}
	}

	return changed, nil
}

// chunkFilesParallel chunks files in parallel.
func (idx *Indexer) chunkFilesParallel(ctx context.Context, files []*types.SourceFile) ([]*types.Chunk, []*types.Symbol, []*types.Reference, error) {
	workers := idx.config.Limits.Workers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	type result struct {
		chunks  []*types.Chunk
		symbols []*types.Symbol
		refs    []*types.Reference
		err     error
	}

	// Channel for work
	fileCh := make(chan *types.SourceFile, len(files))
	resultCh := make(chan result, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileCh {
				if ctx.Err() != nil {
					resultCh <- result{err: ctx.Err()}
					return
				}

				idx.updateProgress("chunking", 0, 0, 0, 0, file.Path)

				chunks, err := idx.chunker.Chunk(file)
				if err != nil {
					// Log warning but continue with empty chunks
					slog.Warn("chunking failed", "file", file.Path, "error", err)
					chunks = nil
				}

				symbols, _ := idx.chunker.ExtractSymbols(file)
				refs, _ := idx.chunker.ExtractReferences(file)

				resultCh <- result{
					chunks:  chunks,
					symbols: symbols,
					refs:    refs,
				}
			}
		}()
	}

	// Send work
	for _, file := range files {
		fileCh <- file
	}
	close(fileCh)

	// Wait for workers
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var allChunks []*types.Chunk
	var allSymbols []*types.Symbol
	var allRefs []*types.Reference

	for res := range resultCh {
		if res.err != nil {
			return nil, nil, nil, res.err
		}
		allChunks = append(allChunks, res.chunks...)
		allSymbols = append(allSymbols, res.symbols...)
		allRefs = append(allRefs, res.refs...)
	}

	return allChunks, allSymbols, allRefs, nil
}

// embedChunks generates embeddings for chunks in batches.
func (idx *Indexer) embedChunks(ctx context.Context, chunks []*types.Chunk) ([]*types.ChunkWithEmbedding, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	batchSize := idx.embedding.MaxBatchSize()
	results := make([]*types.ChunkWithEmbedding, len(chunks))

	for i := 0; i < len(chunks); i += batchSize {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		texts := make([]string, len(batch))
		for j, chunk := range batch {
			texts[j] = chunk.Content
		}

		embeddings, err := idx.embedding.Embed(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("embedding failed for batch %d: %w", i/batchSize, err)
		}

		for j, chunk := range batch {
			results[i+j] = &types.ChunkWithEmbedding{
				Chunk:     chunk,
				Embedding: embeddings[j],
			}
		}

		idx.updateProgress("embedding", 0, 0, len(chunks), i+len(batch), "")
	}

	return results, nil
}

// updateProgress updates the progress state.
func (idx *Indexer) updateProgress(phase string, totalFiles, processedFiles, totalChunks, processedChunks int, currentFile string) {
	idx.progressMu.Lock()
	defer idx.progressMu.Unlock()

	if phase != "" {
		idx.progress.Phase = phase
	}
	if totalFiles > 0 {
		idx.progress.TotalFiles = totalFiles
	}
	if processedFiles > 0 {
		idx.progress.ProcessedFiles = processedFiles
	}
	if totalChunks > 0 {
		idx.progress.TotalChunks = totalChunks
	}
	if processedChunks > 0 {
		idx.progress.ProcessedChunks = processedChunks
	}
	if currentFile != "" {
		idx.progress.CurrentFile = currentFile
	}

	if idx.onProgress != nil {
		idx.onProgress(idx.progress)
	}
}

// matchGlob matches a path against a glob pattern.
func matchGlob(pattern, path string) bool {
	// Handle ** for recursive matching
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			// Check prefix and suffix
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")

			// Check prefix
			if prefix != "" && !strings.HasPrefix(path, prefix) {
				return false
			}

			// For suffix, we need to handle globs like "*.go"
			if suffix == "" {
				return true
			}

			// If suffix contains *, use filepath.Match on the basename or remaining path
			if strings.Contains(suffix, "*") {
				// Try matching suffix against basename
				base := filepath.Base(path)
				matched, _ := filepath.Match(suffix, base)
				if matched {
					return true
				}
				// Try matching against each path component
				remaining := path
				if prefix != "" {
					remaining = strings.TrimPrefix(path, prefix)
					remaining = strings.TrimPrefix(remaining, "/")
				}
				matched, _ = filepath.Match(suffix, remaining)
				return matched
			}

			// Simple suffix match
			return strings.HasSuffix(path, suffix) || strings.Contains(path, suffix)
		}
	}

	// Standard glob match
	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	// Try matching against basename
	base := filepath.Base(path)
	matched, _ = filepath.Match(pattern, base)
	return matched
}

// parseSize parses a size string like "1MB" to bytes.
func parseSize(s string) int64 {
	s = strings.ToUpper(strings.TrimSpace(s))

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}

	var value int64
	_, _ = fmt.Sscanf(s, "%d", &value)

	return value * multiplier
}

// ComputeConfigHash computes a hash of configuration that affects indexing.
func ComputeConfigHash(cfg *config.Config) string {
	data := fmt.Sprintf("%s:%s:%s:%d",
		cfg.Embedding.Provider,
		cfg.Embedding.Model,
		cfg.Chunking.Strategy,
		cfg.Chunking.MaxChunkSize,
	)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// IndexWithGitHistory indexes the codebase and optionally includes git history.
// This is a convenience method that combines code indexing with git history indexing.
func (idx *Indexer) IndexWithGitHistory(ctx context.Context, force bool, includeGitHistory bool) error {
	// First, index the code
	if err := idx.Index(ctx, force); err != nil {
		return err
	}

	// If git history is enabled, index it
	if includeGitHistory {
		gitIndexer, err := NewGitHistoryIndexer(GitHistoryIndexerConfig{
			ProjectDir:    idx.projectDir,
			Store:         idx.store,
			Embedding:     idx.embedding,
			MaxCommits:    0, // No limit
			EmbedDiffs:    true,
			EmbedMessages: true,
			OnProgress: func(p GitHistoryProgress) {
				slog.Debug("git history progress",
					"phase", p.Phase,
					"commits", p.ProcessedCommits,
					"total", p.TotalCommits,
				)
			},
		})
		if err != nil {
			// Git history indexing is optional, just log warning
			slog.Warn("git history indexing not available", "error", err)
			return nil
		}

		if err := gitIndexer.Index(ctx, force); err != nil {
			slog.Warn("git history indexing failed", "error", err)
		}
	}

	return nil
}

// IsGitRepo returns true if the project directory is a git repository.
func (idx *Indexer) IsGitRepo() bool {
	return isGitRepo(idx.projectDir)
}
