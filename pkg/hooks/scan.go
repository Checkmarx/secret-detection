package hooks

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Scan runs the 2ms binary against the git diff on the pre-commit event
func Scan() error {
	fmt.Println("Running 2ms scan on git diff...")

	// Get the git diff
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get git diff: %v\n%s", err, output)
	}

	// Run the 2ms binary against the git diff
	diffFiles := string(output)
	if diffFiles == "" {
		fmt.Println("No changes to scan.")
		return nil
	}

	// Construct the path to the 2ms binary
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	absPath := filepath.Join(basePath, "2ms", "2ms")
	if err != nil {
		return fmt.Errorf("failed to get absolute path for 2ms binary: %v", err)
	}

	cmd = exec.Command(absPath, "scan", "--files", diffFiles)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("2ms scan failed: %v\n%s", err, output)
	}

	fmt.Println(string(output))
	return nil
}
