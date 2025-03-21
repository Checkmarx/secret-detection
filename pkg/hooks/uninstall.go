package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Uninstall removes the cx-secret-detection hook from the .pre-commit-config.yaml file
func Uninstall(global bool) error {
	if global {
		return uninstallGlobal()
	}
	return uninstallLocal()
}

// uninstallLocal removes the cx-secret-detection hook from the local .pre-commit-config.yaml
func uninstallLocal() error {
	fmt.Println("Uninstalling local cx-secret-detection hook...")

	configFilePath := ".pre-commit-config.yaml"

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("no .pre-commit-config.yaml found in the current directory")
	}

	if err := removeHookFromConfig(configFilePath); err != nil {
		return err
	}

	cmd := exec.Command("pre-commit", "uninstall")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to uninstall local pre-commit hooks: %v\n%s", err, output)
	}

	fmt.Println(string(output))
	fmt.Println("Local cx-secret-detection hook uninstalled successfully.")
	return nil
}

// uninstallGlobal removes the cx-secret-detection hook from the global .pre-commit-config.yaml
func uninstallGlobal() error {
	fmt.Println("Uninstalling global cx-secret-detection hook...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %v", err)
	}

	configFilePath := filepath.Join(homeDir, ".pre-commit-config.yaml")

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("no global .pre-commit-config.yaml found")
	}

	if err := removeHookFromConfig(configFilePath); err != nil {
		return err
	}

	// Optionally, unset the init.templateDir if it was set during global installation
	cmd := exec.Command("git", "config", "--global", "--unset", "init.templateDir")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to unset init.templateDir: %v\n%s", err, output)
	}

	fmt.Println("Global cx-secret-detection hook uninstalled successfully.")
	return nil
}

// removeHookFromConfig removes the cx-secret-detection hook from the given pre-commit config file
func removeHookFromConfig(configFilePath string) error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", configFilePath, err)
	}

	var preCommitConfig struct {
		Repos []struct {
			Repo  string `yaml:"repo"`
			Hooks []struct {
				ID string `yaml:"id"`
			} `yaml:"hooks"`
		} `yaml:"repos"`
	}

	if err := yaml.Unmarshal(data, &preCommitConfig); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	updatedRepos := preCommitConfig.Repos[:0]
	for _, repo := range preCommitConfig.Repos {
		updatedHooks := repo.Hooks[:0]
		for _, hook := range repo.Hooks {
			if hook.ID != "cx-secret-detection" {
				updatedHooks = append(updatedHooks, hook)
			}
		}
		if len(updatedHooks) > 0 {
			repo.Hooks = updatedHooks
			updatedRepos = append(updatedRepos, repo)
		}
	}
	preCommitConfig.Repos = updatedRepos

	updatedData, err := yaml.Marshal(preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(configFilePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %v", configFilePath, err)
	}

	return nil
}
