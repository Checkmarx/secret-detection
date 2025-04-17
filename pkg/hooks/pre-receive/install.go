package pre_receive

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Install sets up pre-receive hooks, either locally or globally.
func Install(global bool) error {
	if global {
		return installGlobal()
	}
	return installLocal()
}

// installLocal sets up pre-receive hooks in the current Git repository.
func installLocal() error {
	fmt.Println("Installing local pre-receive hooks...")

	if !isGitRepo() {
		return fmt.Errorf("current directory is not a Git repository")
	}

	hooksDir := filepath.Join(".git", "hooks")
	preReceiveHookPath := filepath.Join(hooksDir, "pre-receive")

	// Ensure the hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %v", err)
	}

	// Write the pre-receive hook script
	hookScript := `#!/bin/sh
cx hooks pre-receive secrets-scan
`
	if err := os.WriteFile(preReceiveHookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("failed to write pre-receive hook script: %v", err)
	}

	fmt.Println("Local pre-receive hook installed successfully.")
	return nil
}

// installGlobal sets up global pre-receive hooks using a Git template directory.
func installGlobal() error {
	fmt.Println("Installing global pre-receive hook...")

	var globalHooksPath string

	// Retrieve the global hooks path from Git configuration.
	cmd := exec.Command("git", "config", "--global", "core.hooksPath")
	output, err := cmd.Output()
	if err != nil {
		// If the hooks path is not set, default to ~/.git/hooks.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %v", err)
		}
		globalHooksPath = filepath.Join(homeDir, ".git", "hooks")

		// Optionally, set this as the global hooks path in Git configuration.
		cmd := exec.Command("git", "config", "--global", "core.hooksPath", globalHooksPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set global hooks path: %v", err)
		}
	} else {
		// Trim any extraneous whitespace from the output.
		globalHooksPath = strings.TrimSpace(string(output))
	}

	// Ensure the global hooks directory exists.
	if err := os.MkdirAll(globalHooksPath, 0755); err != nil {
		return fmt.Errorf("failed to create global hooks directory: %v", err)
	}

	// Path to the global pre-receive hook script.
	preReceiveHookPath := filepath.Join(globalHooksPath, "pre-receive")

	// Write the pre-receive hook script.
	hookScript := `#!/bin/sh
cx hooks pre-receive secrets-scan
`
	if err := os.WriteFile(preReceiveHookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("failed to write pre-receive hook script: %v", err)
	}

	fmt.Println("Global pre-receive hook installed successfully.")
	return nil
}

// isGitRepo checks if the current directory is a Git repository.
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}
