package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
)

// Install installs local or global pre-commit hooks using the pre-commit framework.
func Install(global bool) error {
	if global {
		return installGlobal()
	}
	return installLocal()
}

// installLocal installs pre-commit hooks in the current Git repository.
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

// installGlobal installs global pre-commit hooks (without touching core.hooksPath).
func installGlobal() error {
	fmt.Println("Installing global pre-commit hooks...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %v", err)
	}

	configFilePath := filepath.Join(homeDir, ".pre-commit-config.yaml")

	// Ensure the global configuration file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err := config.WritePreloadedConfig(configFilePath); err != nil {
			return fmt.Errorf("failed to write global .pre-commit-config.yaml: %v", err)
		}
	} else {
		if err := updateConfigFile(configFilePath); err != nil {
			return fmt.Errorf("failed to update global .pre-commit-config.yaml: %v", err)
		}
	}

	// Unset core.hooksPath to avoid conflicts
	cmd := exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to unset core.hooksPath: %v\n%s", err, output)
	}

	// Install the pre-commit hooks using the global configuration
	cmd = exec.Command("pre-commit", "install", "--config", configFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install global pre-commit hooks: %v\n%s", err, output)
	}

	fmt.Println(string(output))
	fmt.Printf("Global pre-commit hooks installed using config at: %s\n", configFilePath)
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
