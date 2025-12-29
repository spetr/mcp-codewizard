// Package index provides file watching for automatic re-indexing.
package index

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/spetr/mcp-codewizard/internal/config"
	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// Watcher watches for file changes and triggers re-indexing.
type Watcher struct {
	config     *config.Config
	store      provider.VectorStore
	embedding  provider.EmbeddingProvider
	chunker    provider.ChunkingStrategy
	projectDir string
	configHash string

	watcher    *fsnotify.Watcher
	onProgress func(types.IndexProgress)

	// Debouncing
	pendingMu    sync.Mutex
	pendingFiles map[string]time.Time
	debounceTime time.Duration
}

// WatcherConfig contains watcher configuration.
type WatcherConfig struct {
	ProjectDir   string
	Config       *config.Config
	Store        provider.VectorStore
	Embedding    provider.EmbeddingProvider
	Chunker      provider.ChunkingStrategy
	OnProgress   func(types.IndexProgress)
	DebounceTime time.Duration // Default: 500ms
}

// NewWatcher creates a new file watcher.
func NewWatcher(cfg WatcherConfig) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	debounceTime := cfg.DebounceTime
	if debounceTime == 0 {
		debounceTime = 500 * time.Millisecond
	}

	return &Watcher{
		config:       cfg.Config,
		store:        cfg.Store,
		embedding:    cfg.Embedding,
		chunker:      cfg.Chunker,
		projectDir:   cfg.ProjectDir,
		configHash:   cfg.Config.Hash(),
		watcher:      watcher,
		onProgress:   cfg.OnProgress,
		pendingFiles: make(map[string]time.Time),
		debounceTime: debounceTime,
	}, nil
}

// Watch starts watching for file changes.
// It blocks until the context is cancelled.
func (w *Watcher) Watch(ctx context.Context) error {
	// Add directories to watch
	if err := w.addWatchDirs(); err != nil {
		return err
	}

	slog.Info("watching for file changes", "dir", w.projectDir)

	// Start debounce processor
	go w.processDebounced(ctx)

	// Event loop
	for {
		select {
		case <-ctx.Done():
			slog.Info("stopping watcher")
			return w.watcher.Close()

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(ctx, event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn("watcher error", "error", err)
		}
	}
}

// addWatchDirs recursively adds directories to watch.
func (w *Watcher) addWatchDirs() error {
	return filepath.WalkDir(w.projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			// Skip excluded directories
			relPath, _ := filepath.Rel(w.projectDir, path)
			for _, pattern := range w.config.Index.Exclude {
				if matchGlob(pattern, relPath+"/") {
					return filepath.SkipDir
				}
			}

			// Skip hidden directories (except .mcp-codewizard)
			if strings.HasPrefix(d.Name(), ".") && d.Name() != ".mcp-codewizard" {
				return filepath.SkipDir
			}

			if err := w.watcher.Add(path); err != nil {
				slog.Warn("failed to watch directory", "path", path, "error", err)
			}
		}
		return nil
	})
}

// handleEvent processes a file system event.
func (w *Watcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	// Skip if not a relevant operation
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) {
		return
	}

	path := event.Name

	// Check if path matches include patterns
	relPath, err := filepath.Rel(w.projectDir, path)
	if err != nil {
		return
	}

	// Check if file should be indexed
	included := false
	for _, pattern := range w.config.Index.Include {
		if matchGlob(pattern, relPath) {
			included = true
			break
		}
	}

	if !included {
		return
	}

	// Check exclude patterns
	for _, pattern := range w.config.Index.Exclude {
		if matchGlob(pattern, relPath) {
			return
		}
	}

	// Add to pending with debounce
	w.pendingMu.Lock()
	w.pendingFiles[path] = time.Now()
	w.pendingMu.Unlock()

	slog.Debug("file changed", "path", relPath, "op", event.Op.String())
}

// processDebounced processes pending files after debounce period.
func (w *Watcher) processDebounced(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processPendingFiles(ctx)
		}
	}
}

// processPendingFiles processes files that have been stable for debounce period.
func (w *Watcher) processPendingFiles(ctx context.Context) {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	now := time.Now()
	var toProcess []string

	for path, changedAt := range w.pendingFiles {
		if now.Sub(changedAt) >= w.debounceTime {
			toProcess = append(toProcess, path)
			delete(w.pendingFiles, path)
		}
	}

	if len(toProcess) == 0 {
		return
	}

	// Process files
	w.pendingMu.Unlock()
	w.reindexFiles(ctx, toProcess)
	w.pendingMu.Lock()
}

// reindexFiles re-indexes the specified files.
func (w *Watcher) reindexFiles(ctx context.Context, paths []string) {
	slog.Info("re-indexing changed files", "count", len(paths))

	for _, path := range paths {
		if ctx.Err() != nil {
			return
		}

		// Check if file was deleted
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			// File was deleted, remove from index
			if err := w.store.DeleteChunksByFile(path); err != nil {
				slog.Warn("failed to delete chunks", "file", path, "error", err)
			}
			if err := w.store.DeleteFileCache(path); err != nil {
				slog.Warn("failed to delete cache", "file", path, "error", err)
			}
			slog.Info("removed deleted file from index", "file", path)
			continue
		}

		if err != nil {
			slog.Warn("failed to stat file", "file", path, "error", err)
			continue
		}

		// Skip directories
		if info.IsDir() {
			continue
		}

		// Read and index file
		if err := w.indexFile(ctx, path); err != nil {
			slog.Warn("failed to index file", "file", path, "error", err)
		}
	}
}

// indexFile indexes a single file.
func (w *Watcher) indexFile(ctx context.Context, path string) error {
	// Check file size
	maxSize := parseSize(w.config.Limits.MaxFileSize)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > maxSize {
		return nil // Skip large files
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	file := &types.SourceFile{
		Path:     path,
		Content:  content,
		Language: detectLanguage(path),
	}
	file.Hash = file.ComputeHash()

	// Check if already up to date
	cachedHash, err := w.store.GetFileHash(path)
	if err == nil && cachedHash == file.Hash {
		return nil // File hasn't changed
	}

	// Delete old chunks
	if err := w.store.DeleteChunksByFile(path); err != nil {
		slog.Warn("failed to delete old chunks", "file", path, "error", err)
	}

	// Chunk file
	chunks, err := w.chunker.Chunk(file)
	if err != nil {
		return err
	}

	if len(chunks) == 0 {
		// Update cache even if no chunks
		w.store.SetFileHash(path, file.Hash, w.configHash)
		return nil
	}

	// Extract symbols and references
	var symbols []*types.Symbol
	var refs []*types.Reference

	if w.config.Analysis.ExtractSymbols {
		symbols, _ = w.chunker.ExtractSymbols(file)
	}
	if w.config.Analysis.ExtractReferences {
		refs, _ = w.chunker.ExtractReferences(file)
	}

	// Generate embeddings
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := w.embedding.Embed(ctx, texts)
	if err != nil {
		return err
	}

	// Create chunks with embeddings
	chunksWithEmbeddings := make([]*types.ChunkWithEmbedding, len(chunks))
	for i, chunk := range chunks {
		chunksWithEmbeddings[i] = &types.ChunkWithEmbedding{
			Chunk:     chunk,
			Embedding: embeddings[i],
		}
	}

	// Store
	if err := w.store.StoreChunks(chunksWithEmbeddings); err != nil {
		return err
	}

	if len(symbols) > 0 {
		w.store.StoreSymbols(symbols)
	}
	if len(refs) > 0 {
		w.store.StoreReferences(refs)
	}

	// Update cache
	w.store.SetFileHash(path, file.Hash, w.configHash)

	relPath, _ := filepath.Rel(w.projectDir, path)
	slog.Info("indexed file", "file", relPath, "chunks", len(chunks))

	return nil
}

// detectLanguage detects the programming language from file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "c" // or cpp
	default:
		return "unknown"
	}
}

// Close closes the watcher.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}
