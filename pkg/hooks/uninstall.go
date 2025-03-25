package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Uninstall removes pre-commit hooks, either locally or globally.
func Uninstall(global bool) error {
	if global {
		return uninstallGlobal()
	}
	return uninstallLocal()
}

// uninstallLocal removes pre-commit hooks from the current Git repository.
func uninstallLocal() error {
	fmt.Println("Uninstalling local pre-commit hooks...")

	if !isGitRepo() {
		return fmt.Errorf("current directory is not a Git repository")
	}

	cmd := exec.Command("pre-commit", "uninstall")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to uninstall local pre-commit hooks: %v\n%s", err, output)
	}

	fmt.Println(string(output))
	return nil
}

// uninstallGlobal removes the global pre-commit hook.
func uninstallGlobal() error {
	fmt.Println("Uninstalling global pre-commit hook...")

	// Retrieve the global hooks path from Git configuration
	cmd := exec.Command("git", "config", "--global", "core.hooksPath")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get global hooks path: %v", err)
	}

	globalHooksPath := filepath.Clean(strings.TrimSpace(string(output)))

	// If core.hooksPath is not set, default to ~/.git/hooks
	if globalHooksPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %v", err)
		}
		globalHooksPath = filepath.Join(homeDir, ".git", "hooks")
	}

	// Path to the global pre-commit hook script
	preCommitHookPath := filepath.Join(globalHooksPath, "pre-commit")

	// Remove the pre-commit hook script if it exists
	if _, err := os.Stat(preCommitHookPath); err == nil {
		if err := os.Remove(preCommitHookPath); err != nil {
			return fmt.Errorf("failed to remove global pre-commit hook: %v", err)
		}
		fmt.Println("Global pre-commit hook removed successfully.")
	} else {
		fmt.Println("No global pre-commit hook found.")
	}

	// Unset the global core.hooksPath configuration if it was set
	if globalHooksPath != filepath.Join(os.Getenv("HOME"), ".git", "hooks") {
		cmd = exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to unset global hooks path: %v", err)
		}
		fmt.Println("Unset global core.hooksPath configuration.")
	}

	return nil
}
