package hooks

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/Checkmarx/secret-detection/pkg/config"
)

// Install sets up pre-commit hooks
func Install() error {
	fmt.Println("Installing pre-commit hooks...")

	// Check if the current directory is a Git repository
	if !isGitRepo() {
		return fmt.Errorf("current directory is not a Git repository")
	}

	// Write the pre-loaded .pre-commit-config.yaml file to the root of the repository
	err := config.WritePreloadedConfig(filepath.Join(".", ".pre-commit-config.yaml"))
	if err != nil {
		return fmt.Errorf("failed to write .pre-commit-config.yaml: %v", err)
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
