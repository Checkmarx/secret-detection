package report

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/checkmarx/2ms/v3/lib/reporting"
	"github.com/checkmarx/2ms/v3/lib/secrets"
	"github.com/checkmarx/2ms/v3/plugins"
	"github.com/gitleaks/go-gitdiff/gitdiff"
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

// Serialization schema for JSON output

type ReportOutput struct {
	TotalSecretsFound int             `json:"total_secrets_found"`
	Commits           []CommitSummary `json:"commits"`
}

type CommitSummary struct {
	CommitID string        `json:"commit_id"`
	Author   string        `json:"author"`
	Date     time.Time     `json:"date"`
	Files    []FileSummary `json:"files"`
}

type FileSummary struct {
	FileName string        `json:"file_name"`
	Secrets  []SecretEntry `json:"secrets"`
}

type SecretEntry struct {
	ID          string `json:"id"`
	Value       string `json:"value"`
	RuleID      string `json:"rule_id"`
	StartLine   int    `json:"start_line"`
	ContentType string `json:"content_type"`
}

func PreReceiveReportTextFromJSON(jsonData []byte) (string, error) {
	var data ReportOutput
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", err
	}
	return buildReportString(&data), nil
}

func PreReceiveReport(
	report *reporting.Report,
	commitInfo map[string]CommitInfo,
) (string, []byte, error) {
	jsonBlob, err := generateReportJSON(report, commitInfo)
	if err != nil {
		return "", nil, err
	}
	text, err := PreReceiveReportTextFromJSON(jsonBlob)
	if err != nil {
		return "", nil, err
	}
	return text, jsonBlob, nil
}

func generateReportJSON(
	report *reporting.Report,
	commitInfo map[string]CommitInfo,
) ([]byte, error) {
	// Group results by commit
	secretsByCommit := groupReportResultsByCommitID(report)

	// Sort commit IDs by date desc
	commitIDs := make([]string, 0, len(secretsByCommit))
	for cid := range secretsByCommit {
		commitIDs = append(commitIDs, cid)
	}
	sort.Slice(commitIDs, func(i, j int) bool {
		return commitInfo[commitIDs[i]].Date.After(commitInfo[commitIDs[j]].Date)
	})

	reportOutput := ReportOutput{
		TotalSecretsFound: report.TotalSecretsFound,
		Commits:           make([]CommitSummary, 0, len(commitIDs)),
	}

	for _, cid := range commitIDs {
		ci := commitInfo[cid]
		byFile := groupSecretsByFileName(secretsByCommit[cid])

		// Sort file names for deterministic ordering
		fileNames := make([]string, 0, len(byFile))
		for filename := range byFile {
			fileNames = append(fileNames, filename)
		}
		sort.Strings(fileNames)

		files := make([]FileSummary, 0, len(fileNames))
		for _, filename := range fileNames {
			secretsList := byFile[filename]
			// Sort secrets within a file
			sort.Slice(secretsList, func(i, j int) bool {
				si, sj := secretsList[i].secret, secretsList[j].secret
				if si.StartLine != sj.StartLine {
					return si.StartLine < sj.StartLine
				}
				return si.ID < sj.ID
			})
			entries := make([]SecretEntry, len(secretsList))
			for i, s := range secretsList {
				entries[i] = SecretEntry{
					ID:          s.secret.ID,
					Value:       s.secret.Value,
					RuleID:      s.secret.RuleID,
					StartLine:   s.secret.StartLine,
					ContentType: s.source.contentType,
				}
			}
			files = append(files, FileSummary{FileName: filename, Secrets: entries})
		}

		reportOutput.Commits = append(reportOutput.Commits, CommitSummary{
			CommitID: cid,
			Author:   ci.Author,
			Date:     ci.Date,
			Files:    files,
		})
	}

	return json.MarshalIndent(reportOutput, "", "  ")
}

func buildReportString(data *ReportOutput) string {
	var sb strings.Builder
	// Preallocate based on secrets count
	sb.Grow(512 * data.TotalSecretsFound)

	sb.WriteString("\n----- Cx Secret Scanner Report -----\n")
	sb.WriteString("\nDetected ")
	sb.WriteString(strconv.Itoa(data.TotalSecretsFound))
	sb.WriteString(pluralize(data.TotalSecretsFound, " secret", " secrets"))
	sb.WriteString(" across ")
	sb.WriteString(strconv.Itoa(len(data.Commits)))
	sb.WriteString(pluralize(len(data.Commits), " commit", " commits"))

	if data.TotalSecretsFound > maxDisplayedResults {
		sb.WriteString("\n\nPresenting first ")
		sb.WriteString(strconv.Itoa(maxDisplayedResults))
		sb.WriteString(" results")
	}

	sb.WriteString("\n\n")

	printed := 0

	// Label to break out when maxDisplayedResults reached
outer:
	for idx, commit := range data.Commits {
		numSecrets := countSecrets(commit)
		sb.WriteString(fmt.Sprintf("Commit #%d (%s): %d%s in %d%s\n",
			idx+1,
			commit.CommitID,
			numSecrets,
			pluralize(numSecrets, " secret", " secrets"),
			len(commit.Files),
			pluralize(len(commit.Files), " file", " files"),
		))
		sb.WriteString("Author: " + commit.Author + "\n")
		sb.WriteString("Date: " + commit.Date.Format(commitDateLayout) + "\n\n")

		for _, file := range commit.Files {
			sb.WriteString(fmt.Sprintf("    File: %s (%d%s)\n",
				file.FileName,
				len(file.Secrets),
				pluralize(len(file.Secrets), " secret", " secrets"),
			))

			for _, secret := range file.Secrets {
				if printed >= maxDisplayedResults {
					break outer
				}

				sb.WriteString("        Result ID       : " + secret.ID + "\n")
				sb.WriteString("        Secret Detected : " + secret.Value + "\n")
				sb.WriteString("        Rule ID         : " + secret.RuleID + "\n")
				sb.WriteString("        Location        : Line " + strconv.Itoa(secret.StartLine) + "\n")
				sb.WriteString("        Content Type    : " + secret.ContentType + "\n\n")
				printed++
			}
		}
	}

	// Footer always appended
	sb.WriteString(
		`A pre-receive hook set server side prevented you from push secrets.
To proceed, choose one of the following workflows:

  - Sanitize and Push:
      1. Rewrite your local Git history to remove all exposed secrets.
      2. Store secrets securely using one of these methods:
         - Use environmental variables
         - Use a secret management service
         - Use a configuration management tool
         - Encrypt files containing secrets (the least secure method)
      3. Push code.

  - Ignore detected secrets:
      1. Contact your system administrator to update the server-side secret scanner
          configuration to ignore the detected secret.
      2. Once the new ignore rules are in place, retry pushing your code.

  - Bypass the secret scanner:
      1. Run ` + "`git push -o skip-secret-scanner`" + `
      2. If that does not work, ask your system administrator to update the server-side
          configuration to allow skipping the secret scanner.

You can set up pre-commit secret scanning to avoid rewriting git history in the future:
 - https://docs.checkmarx.com/en/34965-364702-pre-commit-secret-scanning.html

`)
	return sb.String()
}

func countSecrets(c CommitSummary) int {
	total := 0
	for _, f := range c.Files {
		total += len(f.Secrets)
	}
	return total
}

func obfuscateSecret(snippet string) string {
	truncatedSecret := snippet

	if strings.HasPrefix(truncatedSecret, beginPrivateKeyString) {
		truncatedSecret = strings.Split(truncatedSecret, privateKeySeparator)[2]
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
				continue
			}

			contentType := parts[0]
			commitID := parts[1]
			fileName := parts[2]

			resultCopy := *result
			resultCopy.Value = obfuscateSecret(result.Value)

			secretInfo := &SecretInfo{secret: &resultCopy}
			secretInfo.source = SourceInfo{contentType: contentType, fileName: fileName}

			secretsByCommitID[commitID] = append(secretsByCommitID[commitID], secretInfo)
		}
	}
	return secretsByCommitID
}

func groupSecretsByFileName(secrets []*SecretInfo) map[string][]*SecretInfo {
	secretsByFile := make(map[string][]*SecretInfo)
	for _, secret := range secrets {
		secretsByFile[secret.source.fileName] = append(secretsByFile[secret.source.fileName], secret)
	}
	return secretsByFile
}
