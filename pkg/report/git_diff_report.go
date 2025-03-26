package report

import (
	"bufio"
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/parser"
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/lib/secrets"
	"github.com/fatih/color"
	"sort"
	"strings"
	"unicode"
)

// PrintGitDiffReport formats and prints the report to the console.
func PrintGitDiffReport(report *reporting.Report, fileDiffs map[string][]parser.Hunk) {
	const contextBefore = 2
	const contextAfter = 2
	const maxDisplayedResults = 100

	// Group secrets by file.
	secretsByFile := groupSecretsByFile(report.Results)

	// Sort file names.
	var files []string
	for file := range secretsByFile {
		files = append(files, file)
	}
	sort.Strings(files)

	totalFiles := len(secretsByFile)
	totalSecrets := report.TotalSecretsFound

	color.New(color.FgWhite).Printf("Commit scanned for secrets:\n\n")
	color.New(color.FgWhite).Printf("Detected ")
	color.New(color.FgRed).Printf("%d %s ", totalSecrets, pluralize(totalSecrets, "secret", "secrets"))
	color.New(color.FgWhite).Printf("in ")
	color.New(color.FgRed).Printf("%d %s\n\n", totalFiles, pluralize(totalFiles, "file", "files"))
	if totalSecrets > maxDisplayedResults {
		color.New(color.FgWhite).Printf("Presenting first 100 results\n\n")
	}

	printedSecrets := 0
	fileIndex := 1

resultsLoop:
	for _, file := range files {
		secretsInFile := secretsByFile[file]
		sortSecrets(secretsInFile)
		numSecrets := len(secretsInFile)

		printFileHeader(fileIndex, file, numSecrets)

		hunks := fileDiffs[file]
		secretGroups, err := groupSecretsByHunk(hunks, secretsInFile)
		if err != nil {
			color.New(color.FgRed).Printf("Error grouping secrets by diff hunk for file %s: %v\n", file, err)
			fileIndex++
			continue
		}

		// Process each hunk group.
		for _, hunkIndex := range sortedKeys(secretGroups) {
			secretsInHunk := secretGroups[hunkIndex]
			hunk := hunks[hunkIndex]

			// Compute cumulative offset once for this hunk group.
			_, cumulative := findHunkIndex(hunks, secretsInHunk[0].StartLine)
			for secretIdx, secret := range secretsInHunk {
				localSecretLine := secret.StartLine - cumulative
				globalSecretLine := getSecretGlobalStartLine(secret.StartLine, hunks, hunkIndex)

				color.New(color.FgWhite).Println("")
				color.New(color.FgWhite).Printf("\tSecret detected: ")
				color.New(color.FgHiYellow).Printf("%s\n", secret.RuleID)
				color.New(color.FgWhite).Printf("\tResult ID: ")
				color.New(color.FgHiYellow).Printf("%s\n", secret.ID)
				color.New(color.FgWhite).Printf("\tRisk Score: ")
				color.New(color.FgHiYellow).Printf("%.1f\n", secret.CvssScore)
				color.New(color.FgWhite).Printf("\tLocation: ")
				color.New(color.FgHiYellow).Printf("Line %d\n", globalSecretLine)

				secretLinesCount := countSecretLines(secret.Value)
				startIndex := localSecretLine - contextBefore
				endIndex := localSecretLine + secretLinesCount + contextAfter

				contextContent := ProcessContent(hunk.Content, secretsInFile, secretIdx, hunk.StartLine)
				secretContext := extractLineRange(contextContent, startIndex, endIndex)
				fmt.Print(secretContext)

				printedSecrets++
				// If we've already printed 100 secrets, break out of all loops.
				if printedSecrets >= maxDisplayedResults {
					fmt.Println()
					break resultsLoop
				}
			}
		}
		fileIndex++
		color.New(color.FgWhite).Println("")
	}

	printOptions()
}

// groupSecretsByFile groups secrets by their source file.
func groupSecretsByFile(results map[string][]*secrets.Secret) map[string][]*secrets.Secret {
	groups := make(map[string][]*secrets.Secret)
	for _, secretsList := range results {
		for _, secret := range secretsList {
			groups[secret.Source] = append(groups[secret.Source], secret)
		}
	}
	return groups
}

// printFileHeader prints the header for each file's report.
func printFileHeader(fileIndex int, file string, numSecrets int) {
	color.New(color.FgWhite).Printf("#%d File: ", fileIndex)
	color.New(color.FgHiYellow).Printf("%s\n", file)
	color.New(color.FgRed).Printf("%d ", numSecrets)
	color.New(color.FgWhite).Printf("%s detected in file\n", pluralize(numSecrets, "Secret", "Secrets"))
}

// sortedKeys returns sorted keys of a map.
func sortedKeys(m map[int][]*secrets.Secret) []int {
	var keys []int
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// printOptions prints available options for the commit.
func printOptions() {
	color.New(color.FgWhite).Printf("Options for proceeding with the commit:\n\n")
	color.New(color.FgWhite).Printf("  - Remediate detected secrets using the following workflow (")
	color.New(color.FgGreen).Printf("recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("      1. Remove detected secrets from files and store them securely. Options:\n")
	color.New(color.FgWhite).Printf("         - Use environmental variables\n")
	color.New(color.FgWhite).Printf("         - Use a secret management service\n")
	color.New(color.FgWhite).Printf("         - Use a configuration management tool\n")
	color.New(color.FgWhite).Printf("         - Encrypt files containing secrets (least secure method)\n")
	color.New(color.FgWhite).Printf("      2. Commit fixed code.\n\n")

	color.New(color.FgWhite).Printf("  - Ignore detected secrets (")
	color.New(color.FgYellow).Printf("not recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("      Use one of the following commands:\n")
	color.New(color.FgHiBlue).Print("          cx hooks pre-commit secrets-ignore --all\n")
	color.New(color.FgHiBlue).Print("          cx hooks pre-commit secrets-ignore --resultIds=id1,id2\n\n")

	color.New(color.FgWhite).Printf("  - Bypass the pre-commit secret detection scanner (")
	color.New(color.FgRed).Printf("not recommended")
	color.New(color.FgWhite).Printf("):\n")
	color.New(color.FgWhite).Printf("      Use one of the following commands based on your OS:\n\n")
	color.New(color.FgWhite).Printf("        Bash/Zsh:\n")
	color.New(color.FgHiBlue).Printf("          SKIP=cx-secret-detection git commit -m \"<your message>\"\n\n")
	color.New(color.FgWhite).Printf("        Windows CMD:\n")
	color.New(color.FgHiBlue).Printf("          set SKIP=cx-secret-detection && git commit -m \"<your message>\"\n\n")
	color.New(color.FgWhite).Printf("        PowerShell:\n")
	color.New(color.FgHiBlue).Printf("          $env:SKIP=\"cx-secret-detection\"\n")
	color.New(color.FgHiBlue).Printf("          git commit -m \"<your message>\"\n")
}

func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func getSecretGlobalStartLine(secretLine int, hunks []parser.Hunk, hunkIndex int) int {
	cumulative := 0
	for i, hunk := range hunks {
		if i == hunkIndex {
			return hunk.StartLine - cumulative + secretLine
		}
		cumulative += hunk.Size
	}
	return secretLine
}

func findHunkIndex(hunks []parser.Hunk, globalLine int) (int, int) {
	cumulative := 0
	for i, hunk := range hunks {
		if globalLine < cumulative+hunk.Size {
			return i, cumulative
		}
		cumulative += hunk.Size
	}
	return 0, cumulative
}

func groupSecretsByHunk(hunks []parser.Hunk, secretsList []*secrets.Secret) (map[int][]*secrets.Secret, error) {
	groups := make(map[int][]*secrets.Secret)
	for _, sec := range secretsList {
		index, _ := findHunkIndex(hunks, sec.StartLine)
		groups[index] = append(groups[index], sec)
	}
	return groups, nil
}

func extractLineRange(content string, from, to int) string {
	var builder strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0
	for scanner.Scan() {
		if lineNum >= from && lineNum < to {
			builder.WriteString(scanner.Text())
			builder.WriteByte('\n')
		}
		lineNum++
		if lineNum >= to {
			break
		}
	}
	return builder.String()
}

func ObfuscateSecrets(content string, secretsList []*secrets.Secret, highlightIdx int) string {
	var builder strings.Builder
	start := 0
	for i, sec := range secretsList {
		idx := strings.Index(content[start:], sec.Value)
		if idx == -1 {
			builder.WriteString(content[start:])
			return builder.String()
		}
		idx += start
		builder.WriteString(content[start:idx])
		obf := getObfuscatedSecret(sec.Value)
		if i == highlightIdx {
			if strings.Contains(obf, "\n") {
				lines := strings.Split(obf, "\n")
				for j, line := range lines {
					lines[j] = color.RedString(line)
				}
				obf = strings.Join(lines, "\n")
			} else {
				obf = color.RedString(obf)
			}
		}
		builder.WriteString(obf)
		start = idx + len(sec.Value)
	}
	builder.WriteString(content[start:])
	return builder.String()
}

func hasRed(line string) bool {
	return strings.Contains(line, "\x1b[31m") || strings.Contains(line, "\033[31m")
}

func AddLineNumbers(content string, startLine int) string {
	lines := strings.Split(content, "\n")
	var builder strings.Builder
	for i, line := range lines {
		lineNum := i + startLine
		var numStr string
		if hasRed(line) {
			numStr = color.New(color.FgHiYellow).Sprintf("\t%12d |", lineNum)
		} else {
			numStr = fmt.Sprintf("\t%12d |", lineNum)
		}
		builder.WriteString(numStr + " " + line)
		if i < len(lines)-1 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func ProcessContent(content string, secretsList []*secrets.Secret, highlightIdx, startLine int) string {
	highlighted := ObfuscateSecrets(content, secretsList, highlightIdx)
	return AddLineNumbers(highlighted, startLine)
}

func getObfuscatedSecret(secret string) string {
	if len(secret) == 0 {
		return secret
	}
	var builder strings.Builder
	visibleCount := 0
	for _, r := range secret {
		if unicode.IsSpace(r) {
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

func countSecretLines(secret string) int {
	lines := strings.Split(secret, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return len(lines) - 1
	}
	return len(lines)
}

func sortSecrets(secrets []*secrets.Secret) {
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].StartLine != secrets[j].StartLine {
			return secrets[i].StartLine < secrets[j].StartLine
		}
		return secrets[i].StartColumn < secrets[j].StartColumn
	})
}
