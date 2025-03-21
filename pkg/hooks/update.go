package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
)

// Update updates the cx-secret-detection hook in the .pre-commit-config.yaml file
func Update(global bool) error {
	if global {
		return updateGlobal()
	}
	return updateLocal()
}

// updateLocal updates the cx-secret-detection hook in the local .pre-commit-config.yaml
func updateLocal() error {
	fmt.Println("Updating local cx-secret-detection hook...")

	configFilePath := ".pre-commit-config.yaml"

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("no .pre-commit-config.yaml found in the current directory")
	}

	if err := updateHookInConfig(configFilePath); err != nil {
		return err
	}

	fmt.Println("Local cx-secret-detection hook updated successfully.")
	return nil
}

// updateGlobal updates the cx-secret-detection hook in the global .pre-commit-config.yaml
func updateGlobal() error {
	fmt.Println("Updating global cx-secret-detection hook...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %v", err)
	}

	configFilePath := filepath.Join(homeDir, ".pre-commit-config.yaml")

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("no global .pre-commit-config.yaml found")
	}

	if err := updateHookInConfig(configFilePath); err != nil {
		return err
	}

	fmt.Println("Global cx-secret-detection hook updated successfully.")
	return nil
}

// updateHookInConfig updates the cx-secret-detection hook in the given pre-commit config file
func updateHookInConfig(configFilePath string) error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", configFilePath, err)
	}

	var preCommitConfig config.PreCommitConfig
	if err := yaml.Unmarshal(data, &preCommitConfig); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	updated := false
	for i := range preCommitConfig.Repos {
		if preCommitConfig.Repos[i].Repo == "local" {
			for j := range preCommitConfig.Repos[i].Hooks {
				if preCommitConfig.Repos[i].Hooks[j].ID == "cx-secret-detection" {
					preCommitConfig.Repos[i].Hooks[j] = config.PreloadedConfig.Repos[0].Hooks[0]
					updated = true
					break
				}
			}
		}
	}

	if !updated {
		return fmt.Errorf("cx-secret-detection hook not found in %s", configFilePath)
	}

	updatedData, err := yaml.Marshal(preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	if err := os.WriteFile(configFilePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %v", configFilePath, err)
	}

	return nil
}
