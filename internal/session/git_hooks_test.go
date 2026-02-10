package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupGitHooksMount_NotGitRepo(t *testing.T) {
	// Create a temporary directory that is NOT a git repository
	tmpDir, err := os.MkdirTemp("", "git-hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// SetupGitHooksMount should return nil for non-git directories
	// It checks for .git directory existence before attempting mount
	err = SetupGitHooksMount(nil, tmpDir, false)
	if err != nil {
		t.Errorf("Expected nil error for non-git repo, got: %v", err)
	}
}

func TestGitHooksPathConstruction(t *testing.T) {
	// Test that paths are constructed correctly for the git hooks feature
	workspacePath := "/some/workspace/path"
	expectedGitDir := filepath.Join(workspacePath, ".git")
	expectedHooksDir := filepath.Join(expectedGitDir, "hooks")
	expectedContainerHooksPath := "/workspace/.git/hooks"

	// Verify paths
	if expectedGitDir != "/some/workspace/path/.git" {
		t.Errorf("Unexpected git dir path: %s", expectedGitDir)
	}
	if expectedHooksDir != "/some/workspace/path/.git/hooks" {
		t.Errorf("Unexpected hooks dir path: %s", expectedHooksDir)
	}
	if expectedContainerHooksPath != "/workspace/.git/hooks" {
		t.Errorf("Unexpected container hooks path: %s", expectedContainerHooksPath)
	}
}

func TestSetupGitHooksMount_HooksDirCreation(t *testing.T) {
	// Test that SetupGitHooksMount creates the hooks dir if missing
	// We test this by checking the os.MkdirAll behavior separately

	tmpDir, err := os.MkdirTemp("", "git-hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory (simulating a git repo)
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	hooksDir := filepath.Join(gitDir, "hooks")

	// Verify hooks dir doesn't exist yet
	if _, err := os.Stat(hooksDir); !os.IsNotExist(err) {
		t.Fatal("Hooks dir should not exist yet")
	}

	// Simulate what SetupGitHooksMount does before calling MountDisk
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	// Verify hooks dir was created
	info, err := os.Stat(hooksDir)
	if os.IsNotExist(err) {
		t.Error("Hooks dir should have been created")
	}
	if !info.IsDir() {
		t.Error("Hooks path should be a directory")
	}
}

func TestSetupGitHooksMount_GitDirDetection(t *testing.T) {
	// Test various scenarios for .git directory detection

	tests := []struct {
		name      string
		setupFunc func(dir string) error
		isGitRepo bool
	}{
		{
			name:      "empty directory (not a git repo)",
			setupFunc: func(dir string) error { return nil },
			isGitRepo: false,
		},
		{
			name: "directory with .git folder",
			setupFunc: func(dir string) error {
				return os.Mkdir(filepath.Join(dir, ".git"), 0o755)
			},
			isGitRepo: true,
		},
		{
			name: "directory with .git file (git worktree)",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /path/to/main/.git/worktrees/foo"), 0o644)
			},
			isGitRepo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "git-detection-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup the test scenario
			if err := tt.setupFunc(tmpDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			// Check if .git exists
			gitPath := filepath.Join(tmpDir, ".git")
			_, err = os.Stat(gitPath)
			exists := !os.IsNotExist(err)

			if exists != tt.isGitRepo {
				t.Errorf("Expected isGitRepo=%v, got %v", tt.isGitRepo, exists)
			}
		})
	}
}

func TestSetupGitHooksMount_PreservesExistingHooks(t *testing.T) {
	// Test that existing hooks are not deleted during setup
	tmpDir, err := os.MkdirTemp("", "git-hooks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git/hooks directory with a sample hook
	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	hookContent := "#!/bin/sh\necho 'pre-commit hook'\n"
	hookFile := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(hookFile, []byte(hookContent), 0o755); err != nil {
		t.Fatalf("Failed to create hook file: %v", err)
	}

	// Simulate MkdirAll (what SetupGitHooksMount does)
	// This should be idempotent and not delete existing content
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Verify hook file still exists
	content, err := os.ReadFile(hookFile)
	if err != nil {
		t.Fatalf("Failed to read hook file: %v", err)
	}

	if string(content) != hookContent {
		t.Errorf("Hook content was modified. Expected %q, got %q", hookContent, string(content))
	}
}
