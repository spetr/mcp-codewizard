package memory

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// GitIntegration provides git-related functionality for memory.
type GitIntegration struct {
	projectDir string
	store      *Store
}

// NewGitIntegration creates a new git integration.
func NewGitIntegration(store *Store) *GitIntegration {
	return &GitIntegration{
		projectDir: store.projectDir,
		store:      store,
	}
}

// IsGitRepo checks if the project is a git repository.
func (g *GitIntegration) IsGitRepo() bool {
	gitDir := filepath.Join(g.projectDir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetCurrentBranch returns the current git branch.
func (g *GitIntegration) GetCurrentBranch() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetCurrentCommit returns the current commit hash.
func (g *GitIntegration) GetCurrentCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))[:7]
}

// GetCommitMessage returns the current commit message.
func (g *GitIntegration) GetCommitMessage() string {
	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetRemoteURL returns the remote origin URL.
func (g *GitIntegration) GetRemoteURL() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// IsDirty returns true if there are uncommitted changes.
func (g *GitIntegration) IsDirty() bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// GetDirtyFiles returns list of modified files.
func (g *GitIntegration) GetDirtyFiles() []string {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files
}

// GetAheadBehind returns how many commits ahead/behind the remote.
func (g *GitIntegration) GetAheadBehind() (ahead, behind int) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		ahead, _ = strconv.Atoi(parts[0])
		behind, _ = strconv.Atoi(parts[1])
	}
	return
}

// IsMergeInProgress returns true if a merge is in progress.
func (g *GitIntegration) IsMergeInProgress() bool {
	mergePath := filepath.Join(g.projectDir, ".git", "MERGE_HEAD")
	_, err := os.Stat(mergePath)
	return err == nil
}

// IsRebaseInProgress returns true if a rebase is in progress.
func (g *GitIntegration) IsRebaseInProgress() bool {
	rebasePaths := []string{
		filepath.Join(g.projectDir, ".git", "rebase-merge"),
		filepath.Join(g.projectDir, ".git", "rebase-apply"),
	}
	for _, path := range rebasePaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// GetConflictFiles returns list of files with merge conflicts.
func (g *GitIntegration) GetConflictFiles() []string {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		files = append(files, scanner.Text())
	}
	return files
}

// GetGitState returns the complete git state.
func (g *GitIntegration) GetGitState() GitState {
	ahead, behind := g.GetAheadBehind()

	return GitState{
		Branch:           g.GetCurrentBranch(),
		Commit:           g.GetCurrentCommit(),
		CommitMessage:    g.GetCommitMessage(),
		IsDirty:          g.IsDirty(),
		DirtyFiles:       g.GetDirtyFiles(),
		Remote:           g.GetRemoteURL(),
		Ahead:            ahead,
		Behind:           behind,
		MergeInProgress:  g.IsMergeInProgress(),
		RebaseInProgress: g.IsRebaseInProgress(),
		ConflictFiles:    g.GetConflictFiles(),
	}
}

// GetProjectID returns a unique project ID based on git.
func (g *GitIntegration) GetProjectID() string {
	// Use the hash of the first commit as project ID
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		// Fallback to directory name
		return filepath.Base(g.projectDir)
	}

	commit := strings.TrimSpace(string(output))
	if len(commit) > 8 {
		return commit[:8]
	}
	return commit
}

// InstallHooks installs git hooks for memory sync.
func (g *GitIntegration) InstallHooks() error {
	if !g.IsGitRepo() {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := filepath.Join(g.projectDir, ".git", "hooks")

	// Post-merge hook
	postMergeHook := `#!/bin/bash
# mcp-codewizard post-merge hook
# Syncs memory after git merge

if command -v mcp-codewizard &> /dev/null; then
    echo "🔄 mcp-codewizard: Syncing memory after merge..."
    mcp-codewizard memory sync --on-merge 2>/dev/null || true
fi
`
	if err := g.installHook(hooksDir, "post-merge", postMergeHook); err != nil {
		return err
	}

	// Post-checkout hook
	postCheckoutHook := `#!/bin/bash
# mcp-codewizard post-checkout hook
# Updates context on branch switch

OLD_REF=$1
NEW_REF=$2
BRANCH_SWITCH=$3

if [ "$BRANCH_SWITCH" = "1" ] && command -v mcp-codewizard &> /dev/null; then
    OLD_BRANCH=$(git rev-parse --abbrev-ref HEAD@{1} 2>/dev/null || echo "unknown")
    NEW_BRANCH=$(git rev-parse --abbrev-ref HEAD)

    if [ "$OLD_BRANCH" != "$NEW_BRANCH" ]; then
        echo "🔀 mcp-codewizard: Branch switch detected ($OLD_BRANCH -> $NEW_BRANCH)"
        mcp-codewizard memory branch-switch --from "$OLD_BRANCH" --to "$NEW_BRANCH" 2>/dev/null || true
    fi
fi
`
	if err := g.installHook(hooksDir, "post-checkout", postCheckoutHook); err != nil {
		return err
	}

	// Pre-commit hook (optional, for memory validation)
	preCommitHook := `#!/bin/bash
# mcp-codewizard pre-commit hook
# Validates memory files before commit

if command -v mcp-codewizard &> /dev/null; then
    # Check if any memory files are staged
    if git diff --cached --name-only | grep -q ".mcp-codewizard/memory/"; then
        mcp-codewizard memory validate 2>/dev/null || true
    fi
fi
`
	if err := g.installHook(hooksDir, "pre-commit", preCommitHook); err != nil {
		return err
	}

	return nil
}

// installHook installs a single git hook.
func (g *GitIntegration) installHook(hooksDir, name, content string) error {
	hookPath := filepath.Join(hooksDir, name)

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil {
		// Read existing hook
		existing, err := os.ReadFile(hookPath)
		if err != nil {
			return err
		}

		// Check if our hook is already there
		if strings.Contains(string(existing), "mcp-codewizard") {
			return nil // Already installed
		}

		// Append to existing hook
		content = string(existing) + "\n" + content
	}

	// Write hook
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to install %s hook: %w", name, err)
	}

	return nil
}

// UninstallHooks removes git hooks.
func (g *GitIntegration) UninstallHooks() error {
	if !g.IsGitRepo() {
		return nil
	}

	hooksDir := filepath.Join(g.projectDir, ".git", "hooks")
	hooks := []string{"post-merge", "post-checkout", "pre-commit"}

	for _, hook := range hooks {
		hookPath := filepath.Join(hooksDir, hook)

		// Read existing hook
		existing, err := os.ReadFile(hookPath)
		if err != nil {
			continue
		}

		// Remove our sections
		lines := strings.Split(string(existing), "\n")
		var newLines []string
		inOurSection := false

		for _, line := range lines {
			if strings.Contains(line, "mcp-codewizard") {
				inOurSection = true
				continue
			}
			if inOurSection && strings.HasPrefix(line, "#") && !strings.Contains(line, "mcp-codewizard") {
				inOurSection = false
			}
			if !inOurSection {
				newLines = append(newLines, line)
			}
		}

		newContent := strings.TrimSpace(strings.Join(newLines, "\n"))
		if newContent == "" || newContent == "#!/bin/bash" {
			// Remove empty hook
			os.Remove(hookPath)
		} else {
			os.WriteFile(hookPath, []byte(newContent+"\n"), 0755)
		}
	}

	return nil
}

// GetChangedFilesSinceCommit returns files changed since a commit.
func (g *GitIntegration) GetChangedFilesSinceCommit(commit string) []string {
	cmd := exec.Command("git", "diff", "--name-only", commit, "HEAD")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		files = append(files, scanner.Text())
	}
	return files
}

// StageMemoryFiles stages all memory files for commit.
func (g *GitIntegration) StageMemoryFiles() error {
	memoryDir := filepath.Join(g.projectDir, ".mcp-codewizard", "memory")

	cmd := exec.Command("git", "add", memoryDir)
	cmd.Dir = g.projectDir
	return cmd.Run()
}

// GetMemoryFileStatus returns git status of memory files.
func (g *GitIntegration) GetMemoryFileStatus() map[string]string {
	cmd := exec.Command("git", "status", "--porcelain", ".mcp-codewizard/memory/")
	cmd.Dir = g.projectDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 3 {
			status := strings.TrimSpace(line[:2])
			file := strings.TrimSpace(line[3:])
			result[file] = status
		}
	}
	return result
}
