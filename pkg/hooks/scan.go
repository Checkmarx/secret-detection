package hooks

import (
	"bufio"
	"fmt"
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/lib/secrets"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/fatih/color"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type LineContext struct {
	hunkStartLine *int
	index         int
	context       *string
}

var diffHeaderRegex = regexp.MustCompile(`^diff --git\s+"?a/(.+?)"?\s+"?b/(.+?)"?$`)
var hunkLineNumber = regexp.MustCompile(`^@@\s*-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s*@@`)

// builderPool lets us reuse strings.Builders to reduce allocations.
var builderPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

// Scan runs the 2ms binary against the git diff on the pre-commit event
func Scan() error {
	color.NoColor = false

	report, fileLineContextMap, err := scanAndGenerateReport()
	if err != nil {
		return fmt.Errorf("failed to run scan: %w", err)
	}

	if report.TotalSecretsFound > 0 {
		printReport(report, fileLineContextMap)
		os.Exit(1)
	}
	return nil
}

func scanAndGenerateReport() (*reporting.Report, map[string][]LineContext, error) {
	// Get the git diff.
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get git diff: %v\n%s", err, output)
	}
	diffFiles := string(output)

	// Create a channel for dynamically receiving ScanItems.
	itemsCh := make(chan twoms.ScanItem)
	var fileLineContextMap map[string][]LineContext
	var parseErr error

	// Launch the diff parser as a goroutine.
	go func() {
		// parseGitDiff will send each ScanItem on itemsCh and close it when done.
		fileLineContextMap, parseErr = parseGitDiff(diffFiles, itemsCh)
	}()

	// Get any ignored result IDs.
	ignoredResultIds, err := getIgnoredResultIds()
	if err != nil {
		return nil, nil, err
	}

	// Create a scanner and process items dynamically.
	scanner := twoms.NewScanner()
	report, err := scanner.ScanDynamic(itemsCh, twoms.ScanConfig{IgnoreResultIds: ignoredResultIds})
	// If the parser encountered an error, return it.
	if parseErr != nil {
		return nil, nil, parseErr
	}

	return report, fileLineContextMap, err
}

func parseGitDiff(diff string, out chan<- twoms.ScanItem) (map[string][]LineContext, error) {
	var currentFile *twoms.ScanItem

	// Obtain builders from the pool.
	builder := builderPool.Get().(*strings.Builder)
	builder.Reset()

	fileLineContextMap := make(map[string][]LineContext)

	var isProcessingContent bool
	var currentHunkStartLine *int
	var currentHunkIndex int
	var currentAddedIndices []int

	hunkContextBuilder := builderPool.Get().(*strings.Builder)
	hunkContextBuilder.Reset()

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for a diff file header.
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			// Flush previous file if any.
			if currentFile != nil {
				if hunkContextBuilder.Len() > 0 && len(currentAddedIndices) > 0 {
					contextCopy := hunkContextBuilder.String()
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
				// Send completed ScanItem.
				out <- *currentFile
			}
			// Start a new file.
			source := matches[2]
			currentFile = &twoms.ScanItem{
				ID:     fmt.Sprintf("pre-commit-%s", source),
				Source: source,
			}
			builder.Reset()
			hunkContextBuilder.Reset()
			currentAddedIndices = nil
			isProcessingContent = false
			continue
		}

		// If no file is active, skip.
		if currentFile == nil {
			continue
		}

		// Check for a hunk header.
		if matches := hunkLineNumber.FindStringSubmatch(line); matches != nil {
			if hunkContextBuilder.Len() > 0 && len(currentAddedIndices) > 0 {
				contextCopy := hunkContextBuilder.String()
				for _, idx := range currentAddedIndices {
					fileLineContextMap[currentFile.Source] = append(fileLineContextMap[currentFile.Source], LineContext{
						hunkStartLine: currentHunkStartLine,
						index:         idx,
						context:       &contextCopy,
					})
				}
			}
			hunkContextBuilder.Reset()
			currentAddedIndices = nil
			currentHunkIndex = 0

			newStartAddition, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, fmt.Errorf("unexpected number format in git diff hunk addition: %w", err)
			}
			temp := new(int)
			*temp = newStartAddition
			currentHunkStartLine = temp
			isProcessingContent = true
			continue
		}

		// Process hunk lines.
		if !isProcessingContent {
			continue
		}
		if strings.HasPrefix(line, "+") {
			addedContent := line[1:]
			builder.WriteString(addedContent + "\n")
			currentAddedIndices = append(currentAddedIndices, currentHunkIndex)
			hunkContextBuilder.WriteString(addedContent + "\n")
			currentHunkIndex++
		} else if strings.HasPrefix(line, " ") {
			content := line[1:]
			hunkContextBuilder.WriteString(content + "\n")
			currentHunkIndex++
		}
	}

	// Check for scanning error.
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Flush remaining hunk context for the last file.
	if currentFile != nil {
		if hunkContextBuilder.Len() > 0 && len(currentAddedIndices) > 0 {
			contextCopy := hunkContextBuilder.String()
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
		out <- *currentFile
	}

	// Return builders to the pool.
	builderPool.Put(builder)
	builderPool.Put(hunkContextBuilder)

	// Close the output channel to signal completion.
	close(out)
	return fileLineContextMap, nil
}

// printReport prints the report with files sorted by source (asc)
// and secrets within each file sorted by their start line.
func printReport(report *reporting.Report, fileLineContextMap map[string][]LineContext) {
	// Group secrets per file (source).
	secretsPerFile := make(map[string][]*secrets.Secret)
	for _, results := range report.Results {
		for _, result := range results {
			secretsPerFile[result.Source] = append(secretsPerFile[result.Source], result)
		}
	}

	// Get the list of files and sort them alphabetically (ascending).
	var fileKeys []string
	for file := range secretsPerFile {
		fileKeys = append(fileKeys, file)
	}
	sort.Strings(fileKeys)

	totalFiles := len(secretsPerFile)
	totalSecrets := report.TotalSecretsFound

	color.New(color.FgWhite).Printf("Commit scanned for secrets:\n\n")
	color.New(color.FgWhite).Printf("Detected ")
	color.New(color.FgRed).Printf("%d %s ", totalSecrets, pluralize(totalSecrets, "secret", "secrets"))
	color.New(color.FgWhite).Printf("in ")
	color.New(color.FgRed).Printf("%d %s\n\n", totalFiles, pluralize(totalFiles, "file", "files"))

	fileIndex := 1
	// Iterate over sorted file keys.
	for _, file := range fileKeys {
		secretsInFile := secretsPerFile[file]
		// Sort the secrets by their start line.
		sort.Slice(secretsInFile, func(i, j int) bool {
			return secretsInFile[i].StartLine < secretsInFile[j].StartLine
		})
		numberOfSecrets := len(secretsInFile)

		color.New(color.FgWhite).Printf("#%d File: ", fileIndex)
		color.New(color.FgHiYellow).Printf("%s\n", file)
		color.New(color.FgRed).Printf("%d ", numberOfSecrets)
		color.New(color.FgWhite).Printf("%s detected in file\n\n", pluralize(numberOfSecrets, "Secret", "Secrets"))

		repeatedSecretOccurrences := make(map[string]int)
		for _, secret := range secretsInFile {
			// Calculate the secret start line using the file line context.
			secretLineContext := fileLineContextMap[secret.Source][secret.StartLine]
			secretStartLine := *secretLineContext.hunkStartLine + secretLineContext.index

			color.New(color.FgWhite).Printf("\tSecret detected: ")
			color.New(color.FgHiYellow).Printf("%s\n", secret.RuleID)
			color.New(color.FgWhite).Printf("\tResult ID: ")
			color.New(color.FgHiYellow).Printf("%s\n", secret.ID)
			color.New(color.FgWhite).Printf("\tRisk Score: ")
			color.New(color.FgHiYellow).Printf("%.1f\n", secret.CvssScore)
			color.New(color.FgWhite).Printf("\tLocation: ")
			color.New(color.FgHiYellow).Printf("Line %d\n", secretStartLine)

			key := fmt.Sprintf("%s:%d", secret.Value, secretStartLine)

			// Handle cases where the same secret appears multiple times on the same line.
			repeatedIndexPerLine, exists := repeatedSecretOccurrences[key]
			if !exists {
				repeatedIndexPerLine = 0
			}
			contextBeforeSecretStartLine := getLinesInRange(*secretLineContext.context, 0, secretLineContext.index)
			repeatedSecretsBeforeLine := strings.Count(contextBeforeSecretStartLine, secret.Value)
			secretHighlightIndex := repeatedIndexPerLine + repeatedSecretsBeforeLine

			printSecretLinesContext(secret, secretsInFile, secretHighlightIndex, secretLineContext)

			// Update the occurrence count for this secret (value and line combination).
			repeatedSecretOccurrences[key] = repeatedIndexPerLine + 1
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
	color.New(color.FgWhite).Printf("      Use one of the following commands:\n")
	color.New(color.FgHiBlue).Print("          cx hooks pre-commit secrets-ignore --all\n")
	color.New(color.FgHiBlue).Print("          cx hooks pre-commit secrets-ignore --resultId=id1,id2\n\n")

	// 3) Bypass
	color.New(color.FgWhite).Printf("  - Bypass the pre-commit secret detection scanner (")
	color.New(color.FgRed).Printf("not recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("      Use one of the following commands based on your OS:\n\n")
	color.New(color.FgWhite).Printf("        Bash/Zsh:\n")
	color.New(color.FgHiBlue).Printf("          SKIP=cx-secret-detection git commit -m \"<your message>\"\n\n")
	color.New(color.FgWhite).Printf("        Windows CMD:\n")
	color.New(color.FgHiBlue).Printf("          set SKIP=cx-secret-detection && git commit -m \"<your message>\"\n\n")
	color.New(color.FgWhite).Printf("        PowerShell:\n")
	color.New(color.FgHiBlue).Printf("          $env:SKIP=\"cx-secret-detection\"; git commit -m \"<your message>\"\n")
}

func highlightSecret(secretToHighlight *secrets.Secret, secretsToObfuscate []*secrets.Secret, repeatedSecretIndex int, text string) string {
	red := color.New(color.FgRed)

	// Process each secret in the list.
	for _, s := range secretsToObfuscate {
		obf := getObfuscatedSecret(s.Value)
		if s.Value != secretToHighlight.Value {
			text = strings.ReplaceAll(text, s.Value, obf)
		} else {
			// For the secret to highlight, only the occurrence with index repeatedSecretIndex gets red;
			// the others get white.
			var result strings.Builder
			start := 0
			occurrenceCount := 0
			for {
				idx := strings.Index(text[start:], s.Value)
				if idx == -1 {
					// Append the remainder of the text.
					result.WriteString(text[start:])
					break
				}
				idx += start // absolute index

				// Append text before the found occurrence.
				result.WriteString(text[start:idx])
				// Decide which color to use for this occurrence.
				if occurrenceCount == repeatedSecretIndex {
					// For multi-line secrets, split by newline and wrap each line in red.
					lines := strings.Split(obf, "\n")
					for i, l := range lines {
						// Wrap each non-empty line with the red color.
						if l != "" {
							lines[i] = red.Sprint(l)
						}
					}
					result.WriteString(strings.Join(lines, "\n"))
				} else {
					result.WriteString(obf)
				}
				occurrenceCount++
				// Move start index past the occurrence.
				start = idx + len(s.Value)
			}
			text = result.String()
		}
	}
	return text
}

// hasRed returns true if the given line contains the ANSI escape sequence for red.
func hasRed(line string) bool {
	// The ANSI escape sequence for red is usually "\x1b[31m" or "\033[31m".
	return strings.Contains(line, "\x1b[31m") || strings.Contains(line, "\033[31m")
}

func printSecretLinesContext(secretToHighlight *secrets.Secret, secretsToObfuscate []*secrets.Secret, repeatedSecretIndex int, secretLinesContext LineContext) {
	secretSizeInLines := countSecretLines(secretToHighlight.Value)
	upperLimit := 2
	lowerLimit := 2
	contextCopy := *secretLinesContext.context
	text := highlightSecret(secretToHighlight, secretsToObfuscate, repeatedSecretIndex, contextCopy)
	text = getLinesInRange(text, secretLinesContext.index-upperLimit, secretLinesContext.index+secretSizeInLines+lowerLimit)
	lines := strings.Split(text, "\n")

	startLineNumber := getStartLine(*secretLinesContext.hunkStartLine, secretLinesContext.index-upperLimit)
	for i, line := range lines {
		// Compute the actual line number based on the hunk start line.
		lineNumber := startLineNumber + i
		var numberStr string
		if hasRed(line) {
			numberStr = color.New(color.FgHiYellow).Sprint(lineNumber)
		} else {
			numberStr = color.New(color.FgWhite).Sprint(lineNumber)
		}
		// Reserve 12 spaces for the line number (right aligned).
		numberStr = fmt.Sprintf("%12s", numberStr)
		// Print the line number (colored) followed by the line content.
		fmt.Printf("\t\t%s | %s\n", numberStr, line)
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

func getStartLine(hunkStartLine, index int) int {
	if index < 0 {
		index = 0
	}
	return hunkStartLine + index
}

// getLinesInRange returns the lines from index 'start' (inclusive)
// to 'end' (exclusive) from the given multi-line text.
func getLinesInRange(text string, start, end int) string {
	// Split the text into individual lines.
	lines := strings.Split(text, "\n")

	// Adjust start and end to valid bounds.
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	// Return an empty string if start is greater than or equal to end.
	if start >= end {
		return ""
	}

	// Join and return only the lines from start to end.
	return strings.Join(lines[start:end], "\n")
}

// countSecretLines returns the number of lines in the given secret.
func countSecretLines(secret string) int {
	lines := strings.Split(secret, "\n")

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
