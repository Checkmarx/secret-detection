package hooks

import (
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Uninstall removes the cx-secret-detection hook from the .pre-commit-config.yaml file
func Uninstall() error {
	fmt.Println("Uninstalling cx-secret-detection hook...")

	// Read the .pre-commit-config.yaml file
	data, err := ioutil.ReadFile(".pre-commit-config.yaml")
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
	err = ioutil.WriteFile(".pre-commit-config.yaml", updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write .pre-commit-config.yaml: %v", err)
	}

	fmt.Println("cx-secret-detection hook uninstalled successfully.")
	return nil
}
