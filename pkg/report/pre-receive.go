package report

import (
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/lib/secrets"
	"github.com/checkmarx/2ms/plugins"
	"github.com/gitleaks/go-gitdiff/gitdiff"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	secretMaxCharacters    = 4
	obfuscatedSecretString = "***"
	beginPrivateKeyString  = "-----BEGIN"
	privateKeySeparator    = "-----"
	maxDisplayedResults    = 100
	commitDateLayout       = "Mon Jan 2 15:04:05 2006 -0700 UTC"
)

type CommitInfo struct {
	Author string
	Date   time.Time
}

type FileInfo struct {
	File        *gitdiff.File
	ContentType plugins.DiffType
}

type SecretInfo struct {
	secret *secrets.Secret
	source SourceInfo
}

type SourceInfo struct {
	contentType string
	fileName    string
}

func PreReceiveReport(report *reporting.Report, commitInfo map[string]CommitInfo) string {
	var sb strings.Builder
	sb.Grow(512 * len(report.Results)) // avoid multiple reallocations

	secretsByCommitID := groupReportResultsByCommitID(report)

	sb.WriteString("\n----- Cx Secret Scanner Report -----\n")
	sb.WriteString("\nDetected ")
	sb.WriteString(strconv.Itoa(report.TotalSecretsFound))
	sb.WriteString(pluralize(report.TotalSecretsFound, " secret", " secrets"))
	sb.WriteString(" across ")
	sb.WriteString(strconv.Itoa(len(secretsByCommitID)))
	sb.WriteString(pluralize(len(secretsByCommitID), " commit", " commits"))

	if report.TotalSecretsFound > maxDisplayedResults {
		sb.WriteString("\n\nPresenting first ")
		sb.WriteString(strconv.Itoa(maxDisplayedResults))
		sb.WriteString(" results")
	}

	sb.WriteString("\n\n")

	commitIDs := make([]string, 0, len(secretsByCommitID))
	for commitID := range secretsByCommitID {
		commitIDs = append(commitIDs, commitID)
	}

	sort.Slice(commitIDs, func(i, j int) bool {
		return commitInfo[commitIDs[i]].Date.After(commitInfo[commitIDs[j]].Date)
	})

	printed := 0
	commitIndex := 1

PrintLoop:
	for _, commitID := range commitIDs {
		secretsInfo := secretsByCommitID[commitID]
		secretsByFileName := groupSecretsByFileName(secretsInfo)

		numberOfSecretsInCommit := len(secretsInfo)
		numberOfFiles := len(secretsByFileName)
		author := commitInfo[commitID].Author
		date := commitInfo[commitID].Date

		sb.WriteString("Commit #")
		sb.WriteString(strconv.Itoa(commitIndex))
		sb.WriteString(" (")
		sb.WriteString(commitID)
		sb.WriteString("): ")
		sb.WriteString(strconv.Itoa(numberOfSecretsInCommit))
		sb.WriteString(pluralize(numberOfSecretsInCommit, " secret", " secrets"))
		sb.WriteString(" in ")
		sb.WriteString(strconv.Itoa(numberOfFiles))
		sb.WriteString(pluralize(numberOfFiles, " file", " files"))
		sb.WriteString("\n")
		sb.WriteString("Author: ")
		sb.WriteString(author)
		sb.WriteString("\n")
		sb.WriteString("Date: ")
		sb.WriteString(date.Format(commitDateLayout))
		sb.WriteString("\n\n")

		fileNames := make([]string, 0, len(secretsByFileName))
		for fileName := range secretsByFileName {
			fileNames = append(fileNames, fileName)
		}
		sort.Strings(fileNames)

		for _, fileName := range fileNames {
			secretsInFile := secretsByFileName[fileName]
			sort.Slice(secretsInFile, func(i, j int) bool {
				if secretsInFile[i].secret.StartLine != secretsInFile[j].secret.StartLine {
					return secretsInFile[i].secret.StartLine < secretsInFile[j].secret.StartLine
				}
				return secretsInFile[i].secret.ID < secretsInFile[j].secret.ID
			})

			numberOfSecretsInFile := len(secretsInFile)

			sb.WriteString("    File: ")
			sb.WriteString(fileName)
			sb.WriteString(" (")
			sb.WriteString(strconv.Itoa(numberOfSecretsInFile))
			sb.WriteString(pluralize(numberOfSecretsInFile, " secret", " secrets"))
			sb.WriteString(")\n")

			for _, secret := range secretsInFile {
				if printed >= maxDisplayedResults {
					break PrintLoop
				}

				sb.WriteString("        Result ID       : ")
				sb.WriteString(secret.secret.ID)
				sb.WriteString("\n        Secret Detected : ")
				sb.WriteString(secret.secret.Value)
				sb.WriteString("\n        Rule ID         : ")
				sb.WriteString(secret.secret.RuleID)
				sb.WriteString("\n        Location        : Line ")
				sb.WriteString(strconv.Itoa(secret.secret.StartLine))
				sb.WriteString("\n        Content Type    : ")
				sb.WriteString(secret.source.contentType)
				sb.WriteString("\n\n")

				printed++
			}
		}

		commitIndex++
	}

	sb.WriteString("A pre-receive hook set server side prevented you from push secrets.\n")
	sb.WriteString("To proceed, choose one of the following workflows:\n\n")
	sb.WriteString("  - Sanitize and Push:\n")
	sb.WriteString("      1. Rewrite your local Git history to remove all exposed secrets.\n")
	sb.WriteString("      2. Store secrets securely using one of these methods:\n")
	sb.WriteString("         - Use environmental variables\n")
	sb.WriteString("         - Use a secret management service\n")
	sb.WriteString("         - Use a configuration management tool\n")
	sb.WriteString("         - Encrypt files containing secrets (the least secure method)\n")
	sb.WriteString("      3. Push code.\n\n")

	sb.WriteString("  - Ignore detected secrets:\n")
	sb.WriteString("      1. Contact your system administrator to update the server-side secret scanner\n")
	sb.WriteString("          configuration to ignore the detected secret.\n")
	sb.WriteString("      2. Once the new ignore rules are in place, retry pushing your code.\n\n")

	sb.WriteString("  - Bypass the secret scanner:\n")
	sb.WriteString("      1. Run `git push -o skip-secret-scanner`\n")
	sb.WriteString("      2. If that does not work, ask your system administrator to update the server-side\n")
	sb.WriteString("          configuration to allow skipping the secret scanner.\n\n")

	sb.WriteString("You can set up pre-commit secret scanning to avoid rewriting git history in the future:\n")
	sb.WriteString(" - https://docs.checkmarx.com/en/34965-364702-pre-commit-secret-scanning.html\n\n")

	return sb.String()
}

func obfuscateSecret(snippet string) string {
	truncatedSecret := snippet

	// If the snippet is a private key get the secret part of it
	if strings.HasPrefix(truncatedSecret, beginPrivateKeyString) {
		// Find the string between the second and third separator
		truncatedSecret = strings.Split(truncatedSecret, privateKeySeparator)[2]
		// Remove the first new line character if it exists
		truncatedSecret = strings.TrimPrefix(truncatedSecret, "\n")
		truncatedSecret = strings.TrimPrefix(truncatedSecret, "\\n")
	}

	if len(truncatedSecret) > secretMaxCharacters {
		truncatedSecret = truncatedSecret[:secretMaxCharacters] + obfuscatedSecretString
	}
	return truncatedSecret
}

func groupReportResultsByCommitID(report *reporting.Report) map[string][]*SecretInfo {
	secretsByCommitID := make(map[string][]*SecretInfo)

	for _, results := range report.Results {
		for _, result := range results {
			parts := strings.SplitN(result.Source, ":", 3)
			if len(parts) != 3 {
				// TODO handle error? skip? check this
				continue
			}
			// Added:commitID:fileName
			contentType := parts[0]
			commitID := parts[1]
			fileName := parts[2]

			resultCopy := *result
			resultCopy.Value = obfuscateSecret(result.Value)

			secretInfo := &SecretInfo{}
			secretInfo.secret = &resultCopy
			secretInfo.source = SourceInfo{
				contentType: contentType,
				fileName:    fileName,
			}

			secretsByCommitID[commitID] = append(secretsByCommitID[commitID], secretInfo)
		}
	}

	return secretsByCommitID
}

func groupSecretsByFileName(secrets []*SecretInfo) map[string][]*SecretInfo {
	secretsByFile := make(map[string][]*SecretInfo)
	for _, secret := range secrets {
		fileName := secret.source.fileName
		secretsByFile[fileName] = append(secretsByFile[fileName], secret)
	}
	return secretsByFile
}
