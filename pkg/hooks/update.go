package hooks

import (
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
)

// Update refreshes the cx-secret-detection hook in the .pre-commit-config.yaml file.
func Update(global bool) error {
	if global {
		return updateGlobal()
	}
	return updateLocal()
}

// updateLocal updates the cx-secret-detection hook in the local .pre-commit-config.yaml.
func updateLocal() error {
	fmt.Println("Updating local cx-secret-detection hook...")

	configFilePath := ".pre-commit-config.yaml"

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return fmt.Errorf("no .pre-commit-config.yaml found in the current directory")
	}

	if err := updateHookInConfig(configFilePath); err != nil {
		return err
	}

	// Reinstall the pre-commit hooks to apply changes.
	cmd := exec.Command("pre-commit", "install")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reinstall local pre-commit hooks: %v\n%s", err, output)
	}

	fmt.Println("Local cx-secret-detection hook updated successfully.")
	return nil
}

// updateGlobal updates the global pre-commit hook.
func updateGlobal() error {
	fmt.Println("Updating global pre-commit hook...")

	// Uninstall the existing global pre-commit hook.
	if err := uninstallGlobal(); err != nil {
		return fmt.Errorf("failed to uninstall existing global pre-commit hook: %v", err)
	}

	// Install the new global pre-commit hook.
	if err := installGlobal(); err != nil {
		return fmt.Errorf("failed to install new global pre-commit hook: %v", err)
	}

	fmt.Println("Global pre-commit hook updated successfully.")
	return nil
}

// updateHookInConfig updates the cx-secret-detection hook in the specified pre-commit config file.
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
