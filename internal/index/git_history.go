// Package index implements parallel file indexing with progress reporting.
package index

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/spetr/mcp-codewizard/internal/analysis"
	"github.com/spetr/mcp-codewizard/pkg/provider"
	"github.com/spetr/mcp-codewizard/pkg/types"
)

// GitHistoryIndexerConfig contains configuration for git history indexing.
type GitHistoryIndexerConfig struct {
	ProjectDir       string
	Store            provider.VectorStore // Base store, will check for VectorStoreWithHistory
	Embedding        provider.EmbeddingProvider
	OnProgress       func(GitHistoryProgress)
	MaxCommits       int  // Maximum number of commits to index (0 = no limit)
	IncludeOldDiffs  bool // Whether to embed diffs for old commits
	EmbedDiffs       bool // Whether to generate embeddings for diffs
	EmbedMessages    bool // Whether to generate embeddings for commit messages
}

// GitHistoryProgress reports git history indexing progress.
type GitHistoryProgress struct {
	Phase            string // "scanning", "analyzing", "embedding", "storing"
	TotalCommits     int
	ProcessedCommits int
	TotalChanges     int
	ProcessedChanges int
	CurrentCommit    string
	Error            error
}

// GitHistoryIndexer handles git history indexing.
type GitHistoryIndexer struct {
	projectDir       string
	store            provider.VectorStore
	historyStore     provider.GitHistoryStore
	embedding        provider.EmbeddingProvider
	analyzer         *analysis.GitHistoryAnalyzer
	onProgress       func(GitHistoryProgress)
	maxCommits       int
	shouldEmbedDiffs bool
	shouldEmbedMsgs  bool
}

// NewGitHistoryIndexer creates a new git history indexer.
func NewGitHistoryIndexer(cfg GitHistoryIndexerConfig) (*GitHistoryIndexer, error) {
	// Check if store supports git history
	historyStore, ok := cfg.Store.(provider.GitHistoryStore)
	if !ok {
		return nil, fmt.Errorf("store does not support git history")
	}

	// Check if project is a git repository
	if !isGitRepo(cfg.ProjectDir) {
		return nil, fmt.Errorf("project is not a git repository: %s", cfg.ProjectDir)
	}

	analyzer := analysis.NewGitHistoryAnalyzer(cfg.ProjectDir, nil) // nil uses default config

	return &GitHistoryIndexer{
		projectDir:       cfg.ProjectDir,
		store:            cfg.Store,
		historyStore:     historyStore,
		embedding:        cfg.Embedding,
		analyzer:         analyzer,
		onProgress:       cfg.OnProgress,
		maxCommits:       cfg.MaxCommits,
		shouldEmbedDiffs: cfg.EmbedDiffs,
		shouldEmbedMsgs:  cfg.EmbedMessages,
	}, nil
}

// Index indexes git history.
// It supports incremental indexing - only new commits since last index are processed.
func (idx *GitHistoryIndexer) Index(ctx context.Context, force bool) error {
	startTime := time.Now()

	// Phase 1: Get commits to index
	idx.updateProgress("scanning", 0, 0, 0, 0, "")

	var commits []*types.Commit
	var err error

	if force {
		// Force: get all commits
		commits, err = idx.analyzer.GetCommits("", idx.maxCommits) // "" = no since filter
	} else {
		// Incremental: get commits since last indexed
		lastHash, _ := idx.historyStore.GetLastIndexedCommit()
		if lastHash == "" {
			commits, err = idx.analyzer.GetCommits("", idx.maxCommits)
		} else {
			commits, err = idx.analyzer.GetCommitsSinceHash(lastHash)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if len(commits) == 0 {
		slog.Info("no new commits to index")
		return nil
	}

	// Apply max commits limit if set
	if idx.maxCommits > 0 && len(commits) > idx.maxCommits {
		commits = commits[:idx.maxCommits]
	}

	slog.Info("found commits to index", "count", len(commits))
	idx.updateProgress("scanning", len(commits), 0, 0, 0, "")

	// Phase 2: Analyze commits and get changes
	idx.updateProgress("analyzing", len(commits), 0, 0, 0, "")

	var allChanges []*types.Change
	var chunkHistoryEntries []*types.ChunkHistoryEntry

	for i, commit := range commits {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		idx.updateProgress("analyzing", len(commits), i, 0, 0, commit.ShortHash)

		// Get changes for this commit
		changes, err := idx.analyzer.GetChangesForCommit(commit.Hash)
		if err != nil {
			slog.Warn("failed to get changes for commit", "commit", commit.ShortHash, "error", err)
			continue
		}

		// Identify affected functions for each change
		for _, change := range changes {
			change.AffectedFunctions = idx.analyzer.IdentifyAffectedFunctions(change)
		}

		// Create chunk history entries
		for _, change := range changes {
			for _, chunkID := range change.AffectedChunkIDs {
				entry := &types.ChunkHistoryEntry{
					ChunkID:     chunkID,
					CommitHash:  commit.Hash,
					ChangeType:  string(change.ChangeType),
					DiffSummary: truncateString(change.DiffContent, 500),
					Date:        commit.Date,
					Author:      commit.Author,
				}
				chunkHistoryEntries = append(chunkHistoryEntries, entry)
			}
		}

		allChanges = append(allChanges, changes...)
	}

	slog.Info("analyzed commits",
		"commits", len(commits),
		"changes", len(allChanges),
		"chunk_history_entries", len(chunkHistoryEntries),
	)

	// Phase 3: Generate embeddings (if enabled)
	if idx.shouldEmbedMsgs || idx.shouldEmbedDiffs {
		idx.updateProgress("embedding", len(commits), len(commits), len(allChanges), 0, "")

		// Embed commit messages
		if idx.shouldEmbedMsgs {
			if err := idx.embedCommitMessages(ctx, commits); err != nil {
				slog.Warn("failed to embed commit messages", "error", err)
			}
		}

		// Embed diffs (only for recent/hot commits if tiered)
		if idx.shouldEmbedDiffs {
			changesToEmbed := filterChangesForEmbedding(allChanges, commits)
			if err := idx.embedChangeDiffs(ctx, changesToEmbed); err != nil {
				slog.Warn("failed to embed diffs", "error", err)
			}
		}
	}

	// Phase 4: Store everything
	idx.updateProgress("storing", len(commits), len(commits), len(allChanges), len(allChanges), "")

	// Store commits
	if err := idx.historyStore.StoreCommits(commits); err != nil {
		return fmt.Errorf("failed to store commits: %w", err)
	}

	// Store changes
	if err := idx.historyStore.StoreChanges(allChanges); err != nil {
		return fmt.Errorf("failed to store changes: %w", err)
	}

	// Store chunk history
	if len(chunkHistoryEntries) > 0 {
		if err := idx.historyStore.StoreChunkHistory(chunkHistoryEntries); err != nil {
			slog.Warn("failed to store chunk history", "error", err)
		}
	}

	// Update last indexed commit
	if len(commits) > 0 {
		// commits are sorted newest first, so first one is the latest
		if err := idx.historyStore.SetLastIndexedCommit(commits[0].Hash); err != nil {
			slog.Warn("failed to set last indexed commit", "error", err)
		}
	}

	duration := time.Since(startTime)
	slog.Info("git history indexing complete",
		"commits", len(commits),
		"changes", len(allChanges),
		"chunk_history_entries", len(chunkHistoryEntries),
		"duration", duration.Round(time.Millisecond),
	)

	return nil
}

// embedCommitMessages generates embeddings for commit messages.
func (idx *GitHistoryIndexer) embedCommitMessages(ctx context.Context, commits []*types.Commit) error {
	if len(commits) == 0 {
		return nil
	}

	batchSize := idx.embedding.MaxBatchSize()

	for i := 0; i < len(commits); i += batchSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		end := i + batchSize
		if end > len(commits) {
			end = len(commits)
		}

		batch := commits[i:end]
		texts := make([]string, len(batch))
		for j, commit := range batch {
			texts[j] = commit.Message
		}

		embeddings, err := idx.embedding.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("embedding failed for commit batch %d: %w", i/batchSize, err)
		}

		for j, commit := range batch {
			commit.MessageEmbedding = embeddings[j]
		}
	}

	return nil
}

// embedChangeDiffs generates embeddings for changes.
func (idx *GitHistoryIndexer) embedChangeDiffs(ctx context.Context, changes []*types.Change) error {
	if len(changes) == 0 {
		return nil
	}

	batchSize := idx.embedding.MaxBatchSize()

	for i := 0; i < len(changes); i += batchSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		end := i + batchSize
		if end > len(changes) {
			end = len(changes)
		}

		batch := changes[i:end]
		texts := make([]string, len(batch))
		for j, change := range batch {
			// Use diff content for embedding
			texts[j] = change.DiffContent
		}

		embeddings, err := idx.embedding.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("embedding failed for diff batch %d: %w", i/batchSize, err)
		}

		for j, change := range batch {
			change.DiffEmbedding = embeddings[j]
		}
	}

	return nil
}

// updateProgress updates the progress state.
func (idx *GitHistoryIndexer) updateProgress(phase string, totalCommits, processedCommits, totalChanges, processedChanges int, currentCommit string) {
	if idx.onProgress == nil {
		return
	}

	progress := GitHistoryProgress{
		Phase:            phase,
		TotalCommits:     totalCommits,
		ProcessedCommits: processedCommits,
		TotalChanges:     totalChanges,
		ProcessedChanges: processedChanges,
		CurrentCommit:    currentCommit,
	}

	idx.onProgress(progress)
}

// isGitRepo checks if a directory is a git repository.
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

// filterChangesForEmbedding filters changes based on tiered storage rules.
// Only embeds diffs for recent commits (hot tier).
func filterChangesForEmbedding(changes []*types.Change, commits []*types.Commit) []*types.Change {
	// Build a map of commit hashes to their dates
	commitDates := make(map[string]time.Time)
	for _, c := range commits {
		commitDates[c.Hash] = c.Date
	}

	// Only embed diffs from the last 30 days (hot tier)
	hotThreshold := time.Now().AddDate(0, 0, -30)

	var filtered []*types.Change
	for _, change := range changes {
		if date, ok := commitDates[change.CommitHash]; ok {
			if date.After(hotThreshold) && change.DiffContent != "" {
				filtered = append(filtered, change)
			}
		}
	}

	return filtered
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GetStatus returns the current git history indexing status.
func (idx *GitHistoryIndexer) GetStatus(ctx context.Context) (*types.GitHistoryStats, error) {
	return idx.historyStore.GetGitHistoryStats()
}

// IndexFromCommit indexes git history starting from a specific commit.
func (idx *GitHistoryIndexer) IndexFromCommit(ctx context.Context, fromHash string) error {
	// Set the last indexed commit to the one before fromHash
	// This will make the next incremental index start from fromHash
	cmd := exec.Command("git", "rev-parse", fromHash+"^")
	cmd.Dir = idx.projectDir
	output, err := cmd.Output()
	if err != nil {
		// Probably no parent (initial commit), just clear
		if err := idx.historyStore.SetLastIndexedCommit(""); err != nil {
			return err
		}
	} else {
		parentHash := strings.TrimSpace(string(output))
		if err := idx.historyStore.SetLastIndexedCommit(parentHash); err != nil {
			return err
		}
	}

	return idx.Index(ctx, false)
}
