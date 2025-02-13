package hooks

import (
	"fmt"
	runner "github.com/checkmarx/2ms/pkg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Scan runs the 2ms binary against the git diff on the pre-commit event
func Scan() error {
	fmt.Println("Running 2ms scan on git diff...")

	//Get basepath
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)

	// Get the git diff
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get git diff: %v\n%s", err, output)
	}
	// Write the git diff to a temp file
	diffFiles := string(output)
	if diffFiles == "" {
		fmt.Println("No changes to scan.")
		return nil
	}

	tmpFile, err := os.CreateTemp(basePath, "git-diff-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}

	if _, err := tmpFile.Write(output); err != nil {
		return fmt.Errorf("failed to write git diff to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	if err != nil {
		return fmt.Errorf("failed to get absolute path for 2ms binary: %v", err)
	}

	fsRunner := runner.NewFileSystemRunner()
	err = fsRunner.Run(tmpFile.Name(), "myproject", []string{".git"})
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Error:", err)
	}

	return nil
}
