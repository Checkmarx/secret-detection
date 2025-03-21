package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Checkmarx/secret-detection/pkg/config"
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
	fmt.Println("Uninstalling global pre-commit hooks...")

	// Determine the user's home directory.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %v", err)
	}

	// Define the Git template directory and hooks subdirectory.
	templateDir := filepath.Join(homeDir, ".git-templates")
	hooksDir := filepath.Join(templateDir, "hooks")

	// Path to the global pre-commit hook script.
	preCommitHookPath := filepath.Join(hooksDir, "pre-commit")

	// Remove the pre-commit hook script if it exists.
	if _, err := os.Stat(preCommitHookPath); err == nil {
		if err := os.Remove(preCommitHookPath); err != nil {
			return fmt.Errorf("failed to remove pre-commit hook script: %v", err)
		}
		fmt.Println("Removed pre-commit hook script.")
	} else if os.IsNotExist(err) {
		fmt.Println("No pre-commit hook script found.")
	} else {
		return fmt.Errorf("failed to check pre-commit hook script: %v", err)
	}

	// Unset the Git template directory configuration.
	cmd := exec.Command("git", "config", "--global", "--unset", "init.templateDir")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unset git init.templateDir: %v\n%s", err, output)
	}

	fmt.Println("Unset git init.templateDir configuration.")

	fmt.Println("Global pre-commit hooks uninstalled successfully.")
	return nil
}

// removeHookFromConfig removes the cx-secret-detection hook from the given pre-commit config file
func removeHookFromConfig(configFilePath string) error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", configFilePath, err)
	}

	var preCommitConfig config.PreCommitConfig
	if err := yaml.Unmarshal(data, &preCommitConfig); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	updatedRepos := make([]config.Repo, 0, len(preCommitConfig.Repos))
	for _, repo := range preCommitConfig.Repos {
		if repo.Repo != "local" {
			updatedRepos = append(updatedRepos, repo)
			continue
		}

		// Filter out cx-secret-detection from local hooks
		updatedHooks := make([]config.Hook, 0, len(repo.Hooks))
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
