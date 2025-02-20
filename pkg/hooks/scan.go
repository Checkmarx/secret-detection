package hooks

import (
	"fmt"
	twoms "github.com/checkmarx/2ms/pkg"
	"os/exec"
	"regexp"
	"strings"
)

var diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)

// Scan runs the 2ms binary against the git diff on the pre-commit event
func Scan() error {
	fmt.Println("Running 2ms scan on git diff...")

	// Get the git diff
	cmd := exec.Command("git", "diff", "--cached", "--unified=0")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to get git diff: %v\n%s", err, output)
	}

	diffFiles := string(output)
	if diffFiles == "" {
		fmt.Println("No changes to scan.")
		return nil
	}

	fileChanges := parseGitDiff(diffFiles)

	scanner := twoms.NewScanner()

	report, err := scanner.Scan(fileChanges)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Error:", err)
	}

	// TODO use report instead of print
	fmt.Println(report)

	return nil
}

func parseGitDiff(diff string) []twoms.ScanItem { // TODO update ID? it is "" for now
	var changes []twoms.ScanItem
	var currentFile *twoms.ScanItem
	var builder strings.Builder
	var isProcessingContent bool

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			if currentFile != nil {
				content := builder.String()
				currentFile.Content = &content
				changes = append(changes, *currentFile)
			}
			currentFile = &twoms.ScanItem{
				Source: matches[2],
			}
			builder.Reset()
		} else if currentFile != nil {
			if !isProcessingContent && !strings.HasPrefix(line, "@@") {
				continue
			}
			isProcessingContent = true
			if strings.HasPrefix(line, "+") {
				builder.WriteString(line[1:] + "\n")
			}
		}
	}

	if currentFile != nil {
		content := builder.String()
		currentFile.Content = &content
		changes = append(changes, *currentFile)
	}

	return changes
}
