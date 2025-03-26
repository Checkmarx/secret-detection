package hooks

import (
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
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
	fmt.Println("Uninstalling cx-secret-detection hook...")

	// Read the .pre-commit-config.yaml file
	data, err := os.ReadFile(".pre-commit-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to read .pre-commit-config.yaml: %v", err)
	}

	// Unmarshal the YAML data into a PreCommitConfig object
	var preCommitConfig config.PreCommitConfig
	err = yaml.Unmarshal(data, &preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Remove the cx-secret-detection hook from the repos
	for i, repo := range preCommitConfig.Repos {
		var updatedHooks []config.Hook
		for _, hook := range repo.Hooks {
			if hook.ID != "cx-secret-detection" {
				updatedHooks = append(updatedHooks, hook)
			}
		}
		preCommitConfig.Repos[i].Hooks = updatedHooks
	}

	// Marshal the updated PreCommitConfig object back to YAML
	updatedData, err := yaml.Marshal(preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	// Write the updated YAML data back to the .pre-commit-config.yaml file
	err = os.WriteFile(".pre-commit-config.yaml", updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write .pre-commit-config.yaml: %v", err)
	}

	fmt.Println("cx-secret-detection hook uninstalled successfully.")
	return nil
}

// uninstallGlobal removes the global pre-commit hook and unsets the global configuration.
func uninstallGlobal() error {
	fmt.Println("Uninstalling global pre-commit hook...")

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

	// Path to the global pre-commit hook script
	preCommitHookPath := filepath.Join(globalHooksPath, "pre-commit")

	// Remove the pre-commit hook script if it exists
	if _, err := os.Stat(preCommitHookPath); err == nil {
		if err := os.Remove(preCommitHookPath); err != nil {
			return fmt.Errorf("failed to remove global pre-commit hook: %v", err)
		}
		fmt.Println("Global pre-commit hook removed successfully.")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking pre-commit hook: %v", err)
	} else {
		fmt.Println("No global pre-commit hook found.")
	}

	// Unset the global core.hooksPath configuration
	cmd = exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unset global hooks path: %v", err)
	}

	return nil
}
