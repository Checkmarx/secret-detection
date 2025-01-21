package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Install sets up pre-commit hooks
func Install() error {
	fmt.Println("Installing pre-commit hooks...")

	// Check if the current directory is a Git repository
	if !isGitRepo() {
		return fmt.Errorf("current directory is not a Git repository")
	}

	// Copy the .pre-commit-config.yaml file to the root of the repository
	err := copyPreCommitConfig()
	if err != nil {
		return fmt.Errorf("failed to copy .pre-commit-config.yaml: %v", err)
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

// copyPreCommitConfig copies the .pre-commit-config.yaml file to the root of the repository
func copyPreCommitConfig() error {
	src := "path/to/.pre-commit-config.yaml" // Update this path to the actual location of your config file
	dst := ".pre-commit-config.yaml"

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
