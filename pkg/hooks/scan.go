package hooks

import (
	"fmt"
	"github.com/checkmarx/2ms/lib/reporting"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/fatih/color"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type LineContext struct {
	lineNumber int
	context    *string
}

var diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
var hunkLineNumber = regexp.MustCompile(`^@@\s*-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s*@@`)

// Scan runs the 2ms binary against the git diff on the pre-commit event
func Scan() error {
	fmt.Println("Running 2ms scan on git diff...")

	report, fileLineContextMap, err := scanAndGenerateReport()
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Error:", err)
	}

	printReport(report, fileLineContextMap)
	return nil
}

func scanAndGenerateReport() (*reporting.Report, map[string][]LineContext, error) {
	// Get the git diff
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get git diff: %v\n%s", err, output)
	}

	diffFiles := string(output)
	fileChanges, fileLineContextMap, err := parseGitDiff(diffFiles)
	if err != nil {
		return nil, nil, err
	}

	ignoredResultIds, err := getIgnoredResultIds()
	if err != nil {
		return nil, nil, err
	}

	scanner := twoms.NewScanner()
	report, err := scanner.Scan(fileChanges, twoms.ScanConfig{IgnoreResultIds: ignoredResultIds})
	return report, fileLineContextMap, err
}

func printReport(report *reporting.Report, fileLineContextMap map[string][]LineContext) {
	for sha, results := range report.Results {
		for _, result := range results {
			color.New(color.FgRed).Printf("Secret type: %s\n", result.RuleDescription)
			color.New(color.FgYellow).Printf("Secret severity: %.1f\n", result.CvssScore)
			color.New(color.FgCyan).Printf("Secret SHA: %s\n", sha)
			color.New(color.FgGreen).Printf("File path: %s\n", result.Source)

			// Retrieve the LineContext for the secret (using result.StartLine as index)
			lineContext := fileLineContextMap[result.Source][result.StartLine]
			color.New(color.FgMagenta).Printf("Line the secret was added to: %d\n", lineContext.lineNumber)
			color.New(color.FgWhite).Printf("Code Diff where the secret is added:\n")

			// Print each line of the diff with color based on its first character.
			printDiffWithColors(*lineContext.context)
			fmt.Println()
		}
	}
}

func printDiffWithColors(diff string) {
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			fmt.Println()
			continue
		}
		switch line[0] {
		case '-':
			color.New(color.FgRed).Println(line)
		case '+':
			color.New(color.FgGreen).Println(line)
		case ' ':
			fallthrough
		default:
			color.New(color.FgWhite).Println(line)
		}
	}
}

func parseGitDiff(diff string) ([]twoms.ScanItem, map[string][]LineContext, error) {
	var changes []twoms.ScanItem
	var currentFile *twoms.ScanItem
	var builder strings.Builder
	// will store the mapping: file name -> list of LineContext entries
	fileLineContextMap := make(map[string][]LineContext)

	// Variables for tracking the current hunk state.
	var isProcessingContent bool
	var currentLineNumberAddition int
	var currentLineNumberDeletion int
	var currentAdditionLineNumbers []int
	// currentHunkContext accumulates the hunk context (content of additions and context lines)
	currentHunkContext := ""

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		// Check for a diff file header.
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Flush the previous file if any.
			if currentFile != nil {
				// Flush any pending hunk context for the last hunk.
				if currentHunkContext != "" && len(currentAdditionLineNumbers) > 0 {
					contextCopy := currentHunkContext
					for _, ln := range currentAdditionLineNumbers {
						fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
							lineNumber: ln,
							context:    &contextCopy,
						})
					}
				}
				content := builder.String()
				currentFile.Content = &content
				changes = append(changes, *currentFile)
			}
			// Start a new file.
			source := matches[2]
			currentFile = &twoms.ScanItem{
				ID:     fmt.Sprintf("pre-commit-%s", source),
				Source: source,
			}
			builder.Reset()
			// Reset hunk-related variables.
			isProcessingContent = false
			currentHunkContext = ""
			currentAdditionLineNumbers = nil
			continue
		}

		// Process only if a file is active.
		if currentFile == nil {
			continue
		}

		// Check if this line is a hunk header.
		if matches := hunkLineNumber.FindStringSubmatch(line); matches != nil {
			// Before starting a new hunk, flush the accumulated hunk context
			// to all addition lines recorded so far.
			if currentHunkContext != "" && len(currentAdditionLineNumbers) > 0 {
				contextCopy := currentHunkContext
				for _, ln := range currentAdditionLineNumbers {
					fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
						lineNumber: ln,
						context:    &contextCopy,
					})
				}
			}
			// Reset hunk-specific accumulators.
			currentHunkContext = ""
			currentAdditionLineNumbers = nil

			newStartDeletion, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, nil, fmt.Errorf("unexpected number format in git diff hunk addition: %w", err)
			}
			newStartAddition, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, nil, fmt.Errorf("unexpected number format in git diff hunk addition: %w", err)
			}

			currentLineNumberDeletion = newStartDeletion
			currentLineNumberAddition = newStartAddition

			isProcessingContent = true
			continue
		}

		// Skip processing if we haven't started a hunk.
		if !isProcessingContent {
			continue
		}

		// Process lines that belong to a hunk.
		if strings.HasPrefix(line, "+") {
			// Addition line: record its content (without the '+' marker).
			addedContent := line[1:]
			builder.WriteString(addedContent + "\n")
			// Record this line number as one with an addition.
			currentAdditionLineNumbers = append(currentAdditionLineNumbers, currentLineNumberAddition)
			// Also add this line to the hunk context.
			currentHunkContext += fmt.Sprintf("+%12d| %s\n", currentLineNumberAddition, addedContent)
			currentLineNumberAddition++
		} else if strings.HasPrefix(line, "-") {
			removedContent := line[1:]
			currentHunkContext += fmt.Sprintf("-%12d| %s\n", currentLineNumberDeletion, removedContent)
			currentLineNumberDeletion++
		} else if strings.HasPrefix(line, " ") {
			content := line[1:]
			currentHunkContext += fmt.Sprintf(" %12d| %s\n", currentLineNumberAddition, content)
			currentLineNumberAddition++
			currentLineNumberDeletion++
		}
	}

	if currentFile != nil {
		// After processing all lines, flush any remaining hunk context.
		if currentHunkContext != "" && len(currentAdditionLineNumbers) > 0 {
			contextCopy := currentHunkContext
			for _, ln := range currentAdditionLineNumbers {
				fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
					lineNumber: ln,
					context:    &contextCopy,
				})
			}
		}
		content := builder.String()
		currentFile.Content = &content
		changes = append(changes, *currentFile)
	}

	return changes, fileLineContextMap, nil
}
