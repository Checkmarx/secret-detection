package hooks

import (
	"fmt"
	"os"

	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
)

// Update updates the cx-secret-detection hook in the .pre-commit-config.yaml file
func Update() error {
	fmt.Println("Updating cx-secret-detection hook...")

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

	// Update the cx-secret-detection hook in the repos
	for i, repo := range preCommitConfig.Repos {
		for j, hook := range repo.Hooks {
			if hook.ID == "cx-secret-detection" {
				preCommitConfig.Repos[i].Hooks[j] = config.PreloadedConfig.Repos[0].Hooks[0]
			}
		}
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

	fmt.Println("cx-secret-detection hook updated successfully.")
	return nil
}
