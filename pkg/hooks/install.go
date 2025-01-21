package hooks

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Checkmarx/secret-detection/pkg/config"
	"gopkg.in/yaml.v2"
)

// Install sets up pre-commit hooks
func Install() error {
	fmt.Println("Installing pre-commit hooks...")

	// Check if the current directory is a Git repository
	if !isGitRepo() {
		return fmt.Errorf("current directory is not a Git repository")
	}

	// Define the path to the .pre-commit-config.yaml file
	configFilePath := filepath.Join(".", ".pre-commit-config.yaml")

	// Check if the .pre-commit-config.yaml file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// File does not exist, create it with the pre-loaded configuration
		err := config.WritePreloadedConfig(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to write .pre-commit-config.yaml: %v", err)
		}
	} else {
		// File exists, update it with the new configuration
		err := updateConfigFile(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to update .pre-commit-config.yaml: %v", err)
		}
	}

	// Run the pre-commit install command
	cmd := exec.Command("pre-commit", "install")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install pre-commit hooks: %v\n%s", err, output)
	}

	fmt.Println(string(output))
	return nil
}

// isGitRepo checks if the current directory is a Git repository
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// updateConfigFile updates the existing .pre-commit-config.yaml file with the new configuration
func updateConfigFile(filePath string) error {
	// Read the existing .pre-commit-config.yaml file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read .pre-commit-config.yaml: %v", err)
	}

	// Unmarshal the YAML data into a PreCommitConfig object
	var preCommitConfig config.PreCommitConfig
	err = yaml.Unmarshal(data, &preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Add the new configuration to the existing configuration
	preCommitConfig.Repos = append(preCommitConfig.Repos, config.PreloadedConfig.Repos...)

	// Marshal the updated PreCommitConfig object back to YAML
	updatedData, err := yaml.Marshal(preCommitConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %v", err)
	}

	// Write the updated YAML data back to the .pre-commit-config.yaml file
	err = ioutil.WriteFile(filePath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write .pre-commit-config.yaml: %v", err)
	}

	return nil
}
