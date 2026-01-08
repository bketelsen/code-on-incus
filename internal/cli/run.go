package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mensfeld/claude-on-incus/internal/container"
	"github.com/mensfeld/claude-on-incus/internal/session"
	"github.com/spf13/cobra"
)

var (
	capture bool
	timeout int
	format  string
)

var runCmd = &cobra.Command{
	Use:   "run COMMAND",
	Short: "Run a command in an ephemeral container",
	Long: `Execute a command in an ephemeral Incus container.

The container is automatically cleaned up after the command completes.

Examples:
  coi run "echo hello"
  coi run "npm test" --capture
  coi run "pytest" --slot 2
  coi run --workspace ~/project "make build"
`,
	Args: cobra.MinimumNArgs(1),
	RunE: runCommand,
}

func init() {
	runCmd.Flags().BoolVar(&capture, "capture", false, "Capture output instead of streaming")
	runCmd.Flags().IntVar(&timeout, "timeout", 120, "Command timeout in seconds")
	runCmd.Flags().StringVar(&format, "format", "pretty", "Output format (pretty|json)")
}

func runCommand(cmd *cobra.Command, args []string) error {
	// Get absolute workspace path
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return fmt.Errorf("invalid workspace path: %w", err)
	}

	// Check if Incus is available
	if !container.Available() {
		return fmt.Errorf("incus is not available - please install Incus and ensure you're in the incus-admin group")
	}

	// Allocate slot if not specified
	slotNum := slot
	if slotNum == 0 {
		slotNum, err = session.AllocateSlot(absWorkspace, 10)
		if err != nil {
			return fmt.Errorf("failed to allocate slot: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Auto-allocated slot %d\n", slotNum)
	}

	// Generate container name
	containerName := session.ContainerName(absWorkspace, slotNum)

	// Determine image (use custom if specified, otherwise default)
	img := imageName
	if img == "" {
		img = "coi-sandbox"
		if privileged {
			img = "coi-privileged"
		}
	}

	// Check if image exists
	exists, err := container.ImageExists(img)
	if err != nil {
		return fmt.Errorf("failed to check image: %w", err)
	}
	if !exists {
		return fmt.Errorf("image '%s' not found - run 'coi build %s' first", img, img)
	}

	fmt.Fprintf(os.Stderr, "Launching container %s from image %s...\n", containerName, img)

	// Create manager
	mgr := container.NewManager(containerName)

	// Check if persistent container already exists
	containerExists, err := mgr.Exists()
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	}

	if containerExists && persistent {
		// Restart existing persistent container
		fmt.Fprintf(os.Stderr, "Restarting existing persistent container...\n")
		if err := mgr.Start(); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
	} else if containerExists {
		// Ephemeral container with same name exists - delete and recreate
		fmt.Fprintf(os.Stderr, "Removing existing container...\n")
		if err := mgr.Delete(true); err != nil {
			return fmt.Errorf("failed to delete existing container: %w", err)
		}
		// Launch new container
		ephemeral := !persistent
		if err := mgr.Launch(img, ephemeral); err != nil {
			return fmt.Errorf("failed to launch container: %w", err)
		}
	} else {
		// Launch new container
		ephemeral := !persistent
		if err := mgr.Launch(img, ephemeral); err != nil {
			return fmt.Errorf("failed to launch container: %w", err)
		}
	}

	// Cleanup container on exit (only if ephemeral)
	defer func() {
		if !persistent {
			fmt.Fprintf(os.Stderr, "Cleaning up container %s...\n", containerName)
			_ = mgr.Delete(true) // Best effort cleanup
		} else {
			fmt.Fprintf(os.Stderr, "Stopping persistent container %s...\n", containerName)
			_ = mgr.Stop(false) // Best effort stop
		}
	}()

	// Wait for container to be ready
	fmt.Fprintf(os.Stderr, "Waiting for container to be ready...\n")
	if err := waitForContainer(mgr, 30); err != nil {
		return err
	}

	// Mount workspace (skip if restarting existing persistent container)
	wasRestarted := containerExists && persistent
	if !wasRestarted {
		fmt.Fprintf(os.Stderr, "Mounting workspace %s...\n", absWorkspace)
		if err := mgr.MountDisk("workspace", absWorkspace, "/workspace", true); err != nil {
			return fmt.Errorf("failed to mount workspace: %w", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Reusing existing workspace mount...\n")
	}

	// Build command
	command := args[0]

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range envVars {
		// Simple KEY=VALUE parsing
		// TODO: More robust parsing
		env["USER_ENV"] = e
	}

	// Execute command
	fmt.Fprintf(os.Stderr, "Executing: %s\n", command)

	user := container.ClaudeUID
	opts := container.ExecCommandOptions{
		User:    &user,
		Cwd:     "/workspace",
		Env:     env,
		Capture: capture,
	}

	output, err := mgr.ExecCommand(command, opts)
	if err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	if capture {
		fmt.Println(output)
	}

	fmt.Fprintf(os.Stderr, "Command completed successfully\n")
	return nil
}

// waitForContainer waits for container to be ready
func waitForContainer(mgr *container.Manager, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		running, err := mgr.Running()
		if err != nil {
			return err
		}
		if running {
			// Additional check: try to execute a simple command
			_, err := mgr.ExecCommand("echo ready", container.ExecCommandOptions{Capture: true})
			if err == nil {
				return nil
			}
		}
		// Wait before retry
		if i < maxRetries-1 {
			fmt.Fprintf(os.Stderr, ".")
		}
	}
	return fmt.Errorf("container failed to become ready")
}
