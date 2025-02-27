package hooks

import (
	"fmt"
	"github.com/checkmarx/2ms/lib/reporting"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/fatih/color"
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
	ignoredResultIds, err := getIgnoredResultIds()
	if err != nil {
		return err
	}

	scanner := twoms.NewScanner()

	report, err := scanner.Scan(fileChanges, twoms.ScanConfig{IgnoreResultIds: ignoredResultIds})
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Error:", err)
	}

	// TODO use report instead of print
	printReport(report)

	return nil
}

func printReport(report reporting.Report) {
	for sha, results := range report.Results {
		for _, result := range results {
			color.New(color.FgRed).Printf("Secret type: %s\n", result.RuleDescription)
			color.New(color.FgYellow).Printf("Secret severity: %.1f\n", result.CvssScore)
			color.New(color.FgCyan).Printf("Secret SHA: %s\n", sha)
			color.New(color.FgGreen).Printf("File path: %s\n", result.Source)
			color.New(color.FgMagenta).Printf("Line the secret was added to: %d\n", result.StartLine)
			color.New(color.FgWhite).Printf("Code Diff where the secret is added:\n%s\n", result.LineContent)
			fmt.Println()
		}
	}
}

func parseGitDiff(diff string) []twoms.ScanItem {
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
			source := matches[2]
			currentFile = &twoms.ScanItem{
				ID:     fmt.Sprintf("pre-commit-%s", source),
				Source: source,
			}
			builder.Reset()
			isProcessingContent = false
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
