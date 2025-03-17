package hooks

import (
	"fmt"
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/lib/secrets"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/fatih/color"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type LineContext struct {
	hunkStartLine int
	index         int
	context       *string
}

var diffHeaderRegex = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
var hunkLineNumber = regexp.MustCompile(`^@@\s*-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s*@@`)

// Scan runs the 2ms binary against the git diff on the pre-commit event
func Scan() error {
	fmt.Println("Running 2ms scan on git diff...")

	report, fileLineContextMap, err := scanAndGenerateReport()
	if err != nil {
		return fmt.Errorf("failed to run scan: %w", err)
	}

	if report.TotalSecretsFound > 0 {
		color.NoColor = false
		printReport(report, fileLineContextMap)
		os.Exit(1)
	}
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

func parseGitDiff(diff string) ([]twoms.ScanItem, map[string][]LineContext, error) {
	var changes []twoms.ScanItem
	var currentFile *twoms.ScanItem
	var builder strings.Builder
	// Mapping: file name -> slice of LineContext entries.
	fileLineContextMap := make(map[string][]LineContext)

	// Variables for tracking the current hunk.
	var isProcessingContent bool
	// currentHunkStartLine is set when a hunk header is processed.
	var currentHunkStartLine int
	// currentHunkIndex is the index within the current hunk (resets on each new hunk).
	var currentHunkIndex int
	// currentAddedIndices holds the relative indices (within the hunk) for added lines.
	var currentAddedIndices []int
	// currentHunkContext accumulates the hunk's context (all addition and context lines).
	currentHunkContext := ""

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		// Check for a diff file header.
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Flush the previous file if any.
			if currentFile != nil {
				// Flush any pending hunk context for the last hunk.
				if currentHunkContext != "" && len(currentAddedIndices) > 0 {
					contextCopy := currentHunkContext
					for _, idx := range currentAddedIndices {
						fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
							hunkStartLine: currentHunkStartLine,
							index:         idx,
							context:       &contextCopy,
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
			currentAddedIndices = nil
			continue
		}

		// Process only if a file is active.
		if currentFile == nil {
			continue
		}

		// Check if this line is a hunk header.
		if matches := hunkLineNumber.FindStringSubmatch(line); matches != nil {
			// Before starting a new hunk, flush the accumulated hunk context
			// for all addition lines recorded so far.
			if currentHunkContext != "" && len(currentAddedIndices) > 0 {
				contextCopy := currentHunkContext
				for _, idx := range currentAddedIndices {
					fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
						hunkStartLine: currentHunkStartLine,
						index:         idx,
						context:       &contextCopy,
					})
				}
			}
			// Reset hunk-specific accumulators.
			currentHunkContext = ""
			currentAddedIndices = nil
			currentHunkIndex = 0

			// Parse the new hunk's starting line from the hunk header.
			newStartAddition, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, nil, fmt.Errorf("unexpected number format in git diff hunk addition: %w", err)
			}
			currentHunkStartLine = newStartAddition
			isProcessingContent = true
			continue
		}

		// Skip processing if we haven't started a hunk.
		if !isProcessingContent {
			continue
		}

		// Process lines within the hunk.
		if strings.HasPrefix(line, "+") {
			// Addition line: record its content without the '+' marker.
			addedContent := line[1:]
			builder.WriteString(addedContent + "\n")
			// Record the current hunk index for this addition.
			currentAddedIndices = append(currentAddedIndices, currentHunkIndex)
			// Append the line to the current hunk context.
			currentHunkContext += fmt.Sprintf("%s\n", addedContent)
			currentHunkIndex++
		} else if strings.HasPrefix(line, " ") {
			// Context line: record its content (without the leading space).
			content := line[1:]
			currentHunkContext += fmt.Sprintf("%s\n", content)
			currentHunkIndex++
		}
	}

	// Flush any remaining hunk context after processing all lines.
	if currentFile != nil {
		if currentHunkContext != "" && len(currentAddedIndices) > 0 {
			contextCopy := currentHunkContext
			for _, idx := range currentAddedIndices {
				fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
					hunkStartLine: currentHunkStartLine,
					index:         idx,
					context:       &contextCopy,
				})
			}
		}
		content := builder.String()
		currentFile.Content = &content
		changes = append(changes, *currentFile)
	}

	return changes, fileLineContextMap, nil
}

func printReport(report *reporting.Report, fileLineContextMap map[string][]LineContext) {
	secretsPerFile := make(map[string][]*secrets.Secret)
	for _, results := range report.Results {
		for _, result := range results {
			secretsPerFile[result.Source] = append(secretsPerFile[result.Source], result)
		}
	}

	totalFiles := len(secretsPerFile)
	totalSecrets := report.TotalSecretsFound

	color.New(color.FgWhite).Printf("\nCommit scanned for secrets:\n\n")
	color.New(color.FgWhite).Printf("Detected ")
	color.New(color.FgRed).Printf("%d %s ", totalSecrets, pluralize(totalSecrets, "secret", "secrets"))
	color.New(color.FgWhite).Printf("in ")
	color.New(color.FgRed).Printf("%d %s\n\n", totalFiles, pluralize(totalFiles, "file", "files"))

	fileIndex := 1
	for file, secrets := range secretsPerFile {
		numberOfSecrets := len(secrets)

		color.New(color.FgWhite).Printf("#%d File: ", fileIndex)
		color.New(color.FgHiYellow).Printf("%s\n", file)
		color.New(color.FgRed).Printf("%d ", numberOfSecrets)
		color.New(color.FgWhite).Printf("%s detected in file\n\n", pluralize(numberOfSecrets, "Secret", "Secrets"))

		for _, secret := range secrets {
			secretLineContext := fileLineContextMap[secret.Source][secret.StartLine]
			secretStartLine := secretLineContext.hunkStartLine + secretLineContext.index

			color.New(color.FgWhite).Printf("\tSecret detected: ")
			color.New(color.FgHiYellow).Printf("%s\n", secret.RuleID)
			color.New(color.FgWhite).Printf("\tSHA: ")
			color.New(color.FgHiYellow).Printf("%s\n", secret.ID)
			color.New(color.FgWhite).Printf("\tRisk Score: ")
			color.New(color.FgHiYellow).Printf("%.1f\n", secret.CvssScore)
			color.New(color.FgWhite).Printf("\tLocation: ")
			color.New(color.FgHiYellow).Printf("Line %d\n", secretStartLine)

			printSecretLinesContext(secret, secrets, secretLineContext)
		}
		fileIndex++
	}

	// Print section header.
	color.New(color.FgWhite).Printf("\nOptions for proceeding with the commit:\n\n")

	// 1) Remediate
	color.New(color.FgWhite).Printf("  - Remediate detected secrets using the following workflow (")
	color.New(color.FgGreen).Printf("recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("      1. Remove detected secrets from files and store them securely. Options:\n")
	color.New(color.FgWhite).Printf("         - Use environmental variables\n")
	color.New(color.FgWhite).Printf("         - Use a secret management service\n")
	color.New(color.FgWhite).Printf("         - Use a configuration management tool\n")
	color.New(color.FgWhite).Printf("         - Encrypt files containing secrets (least secure method)\n")
	color.New(color.FgWhite).Printf("      2. Commit fixed code.\n\n")

	// 2) Ignore
	color.New(color.FgWhite).Printf("  - Ignore detected secrets (")
	color.New(color.FgYellow).Printf("not recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("  Run command: ")
	color.New(color.FgHiBlue).Print("cx pre-commit ignore --all\n\n")

	// 3) Bypass
	color.New(color.FgWhite).Printf("  - Bypass the pre-commit secret detection scanner (")
	color.New(color.FgRed).Printf("not recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("  Use command: ")
	color.New(color.FgHiBlue).Print("SKIP=cx-secret-detection git commit -m \"<your message>\"\n\n")
}

func highlightSecret(secretToHighlight *secrets.Secret, secretsToObfuscate []*secrets.Secret, text string) string {
	red := color.New(color.FgRed)

	for _, s := range secretsToObfuscate {
		obf := getObfuscatedSecret(s.Value)
		if s.Value != secretToHighlight.Value {
			text = strings.ReplaceAll(text, s.Value, obf)
		} else {
			// For the secret to highlight, wrap every occurrence in red.
			var replacement string
			if strings.Contains(obf, "\n") {
				// For multi-line secrets, split by newline and wrap each line in red.
				lines := strings.Split(obf, "\n")
				for i, line := range lines {
					if line != "" {
						lines[i] = red.Sprint(line)
					}
				}
				replacement = strings.Join(lines, "\n")
			} else {
				replacement = red.Sprint(obf)
			}
			text = strings.ReplaceAll(text, s.Value, replacement)
		}
	}
	return text
}

func RemoveANSI(input string) string {
	// This regex matches ANSI escape sequences.
	re := regexp.MustCompile("\x1b\\[[0-9;]*m")
	return re.ReplaceAllString(input, "")
}

// hasRed returns true if the given line contains the ANSI escape sequence for red.
func hasRed(line string) bool {
	// The ANSI escape sequence we are using for red is "\x1b[31m"
	return strings.Contains(line, "\x1b[31m")
}

func printSecretLinesContext(secretToHighlight *secrets.Secret, secretsToObfuscate []*secrets.Secret, secretLinesContext LineContext) {
	contextCopy := *secretLinesContext.context
	text := highlightSecret(secretToHighlight, secretsToObfuscate, contextCopy)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lineContent := line
		// Compute the actual line number based on the hunk start line.
		lineNumber := secretLinesContext.hunkStartLine + i
		var numberStr string
		if hasRed(line) {
			// If the current line is not within the highlighted secret block,
			// remove ANSI escape sequences.
			if i < secretLinesContext.index || i >= secretLinesContext.index+getSecretLines(secretToHighlight) {
				lineContent = RemoveANSI(lineContent)
				numberStr = color.New(color.FgWhite).Sprint(lineNumber)
			} else {
				numberStr = color.New(color.FgHiYellow).Sprint(lineNumber)
			}
		} else {
			numberStr = color.New(color.FgWhite).Sprint(lineNumber)
		}
		// Reserve 12 spaces for the line number (right aligned).
		numberStr = fmt.Sprintf("%12s", numberStr)
		// Print the line number (colored) followed by the line content.
		fmt.Printf("\t\t%s | %s\n", numberStr, lineContent)
	}

	color.New(color.FgWhite).Println("")
}

// getObfuscatedSecret returns an obfuscated version of secret.
// It leaves the first 4 non-whitespace characters intact and replaces
// every subsequent non-whitespace character with "*". Whitespace characters are preserved.
func getObfuscatedSecret(secret string) string {
	if len(secret) == 0 {
		return secret
	}

	var builder strings.Builder
	var visibleCount int
	for _, r := range secret {
		if unicode.IsSpace(r) {
			// Preserve whitespace as-is.
			builder.WriteRune(r)
			continue
		}
		if visibleCount < 4 {
			builder.WriteRune(r)
			visibleCount++
		} else {
			builder.WriteRune('*')
		}
	}
	return builder.String()
}

func getSecretLines(secret *secrets.Secret) int {
	if secret == nil || secret.Value == "" {
		return 0
	}
	lines := strings.Split(secret.Value, "\n")
	// If the secret ends with a newline, the last element is empty;
	// so we subtract one from the count.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return len(lines) - 1
	}
	return len(lines)
}

// pluralize returns singular if count equals 1, otherwise returns plural.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
