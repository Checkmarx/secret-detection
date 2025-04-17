package pre_receive

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Uninstall removes pre-receive hooks, either locally or globally.
func Uninstall(global bool) error {
	if global {
		return uninstallGlobal()
	}
	return uninstallLocal()
}

// uninstallLocal removes pre-receive hooks from the current Git repository.
func uninstallLocal() error {
	fmt.Println("Uninstalling local pre-receive hook...")

	hooksDir := filepath.Join(".git", "hooks")
	preReceiveHookPath := filepath.Join(hooksDir, "pre-receive")

	// Remove the pre-receive hook script if it exists
	if _, err := os.Stat(preReceiveHookPath); err == nil {
		if err := os.Remove(preReceiveHookPath); err != nil {
			return fmt.Errorf("failed to remove local pre-receive hook: %v", err)
		}
		fmt.Println("Local pre-receive hook removed successfully.")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking pre-receive hook: %v", err)
	} else {
		fmt.Println("No local pre-receive hook found.")
	}

	return nil
}

// uninstallGlobal removes the global pre-receive hook and unsets the global configuration.
func uninstallGlobal() error {
	fmt.Println("Uninstalling global pre-receive hook...")

	// Retrieve the global hooks path from Git configuration
	cmd := exec.Command("git", "config", "--global", "core.hooksPath")
	output, err := cmd.Output()
	var globalHooksPath string
	if err != nil {
		// If error indicates that key is not set, treat it as not set.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			globalHooksPath = ""
		} else {
			return fmt.Errorf("failed to get global hooks path: %v", err)
		}
	} else {
		globalHooksPath = filepath.Clean(strings.TrimSpace(string(output)))
	}

	// If core.hooksPath is not set, default to ~/.git/hooks
	if globalHooksPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %v", err)
		}
		globalHooksPath = filepath.Join(homeDir, ".git", "hooks")
	}

	// Path to the global pre-receive hook script
	preReceiveHookPath := filepath.Join(globalHooksPath, "pre-receive")

	// Remove the pre-receive hook script if it exists
	if _, err := os.Stat(preReceiveHookPath); err == nil {
		if err := os.Remove(preReceiveHookPath); err != nil {
			return fmt.Errorf("failed to remove global pre-receive hook: %v", err)
		}
		fmt.Println("Global pre-receive hook removed successfully.")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking pre-receive hook: %v", err)
	} else {
		fmt.Println("No global pre-receive hook found.")
	}

	// Unset the global core.hooksPath configuration
	cmd = exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unset global hooks path: %v", err)
	}

	return nil
}
