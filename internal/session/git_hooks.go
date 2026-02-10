package session

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mensfeld/code-on-incus/internal/container"
)

// SetupGitHooksMount mounts .git/hooks as read-only for security.
// This prevents containers from modifying git hooks which could be used
// to inject malicious code that executes on the host during git operations.
// Returns nil if no .git directory exists (not a git repo).
func SetupGitHooksMount(mgr *container.Manager, workspacePath string, useShift bool) error {
	gitDir := filepath.Join(workspacePath, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")

	// Check if this is a git repository
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil // Not a git repo, nothing to protect
	}

	// Ensure hooks directory exists (create if missing to prevent container from creating it)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .git/hooks directory: %w", err)
	}

	// Mount hooks directory as read-only (overlays the workspace mount)
	containerHooksPath := "/workspace/.git/hooks"
	return mgr.MountDisk("git-hooks", hooksDir, containerHooksPath, useShift, true)
}
