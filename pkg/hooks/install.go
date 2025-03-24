package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
)

// Install sets up pre-commit hooks, either locally or globally.
func Install(global bool) error {
	if global {
		return installGlobal()
	}
	return installLocal()
}

// installLocal sets up pre-commit hooks in the current Git repository.
func installLocal() error {
	fmt.Println("Installing local pre-commit hooks...")

	if !isGitRepo() {
		return fmt.Errorf("current directory is not a Git repository")
	}

	configFilePath := filepath.Join(".", ".pre-commit-config.yaml")

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err := config.WritePreloadedConfig(configFilePath); err != nil {
			return fmt.Errorf("failed to write .pre-commit-config.yaml: %v", err)
		}
	} else {
		if err := updateConfigFile(configFilePath); err != nil {
			return fmt.Errorf("failed to update .pre-commit-config.yaml: %v", err)
		}
	}

	cmd := exec.Command("pre-commit", "install")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install local pre-commit hooks: %v\n%s", err, output)
	}

	fmt.Println(string(output))
	return nil
}

// installGlobal sets up global pre-commit hooks using a Git template directory.
func installGlobal() error {
	fmt.Println("Installing global pre-commit hook...")

	// Retrieve the global hooks path from Git configuration.
	cmd := exec.Command("git", "config", "--global", "core.hooksPath")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get global hooks path: %v", err)
	}

	// Trim any extraneous whitespace from the output.
	globalHooksPath := strings.TrimSpace(string(output))

	// If core.hooksPath is not set, default to ~/.git/hooks.
	if globalHooksPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %v", err)
		}
		globalHooksPath = filepath.Join(homeDir, ".git", "hooks")
	}

	// Ensure the global hooks directory exists.
	if err := os.MkdirAll(globalHooksPath, 0755); err != nil {
		return fmt.Errorf("failed to create global hooks directory: %v", err)
	}

	// Path to the global pre-commit hook script.
	preCommitHookPath := filepath.Join(globalHooksPath, "pre-commit")

	// Write the pre-commit hook script.
	hookScript := `#!/bin/sh
cx hooks pre-commit secrets-scan
`
	if err := os.WriteFile(preCommitHookPath, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("failed to write pre-commit hook script: %v", err)
	}

	fmt.Println("Global pre-commit hook installed successfully.")
	return nil
}

// isGitRepo checks if the current directory is a Git repository.
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// updateConfigFile updates a .pre-commit-config.yaml file with the required hook if not present.
func updateConfigFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read .pre-commit-config.yaml: %v", err)
	}

	var preCommitConfig config.PreCommitConfig
	if err := yaml.Unmarshal(data, &preCommitConfig); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	foundLocalRepo := false
	for i := range preCommitConfig.Repos {
		if preCommitConfig.Repos[i].Repo == "local" {
			foundLocalRepo = true
			for _, hook := range preCommitConfig.Repos[i].Hooks {
				if hook.ID == "cx-secret-detection" {
					return nil // already present
				}
			}
			preCommitConfig.Repos[i].Hooks = append(preCommitConfig.Repos[i].Hooks, config.PreloadedConfig.Repos[0].Hooks[0])
			break
		}
	}

	if !foundLocalRepo {
		preCommitConfig.Repos = append(preCommitConfig.Repos, config.PreloadedConfig.Repos...)
	}

	updatedData, err := yaml.Marshal(preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated .pre-commit-config.yaml: %v", err)
	}

	return nil
}
