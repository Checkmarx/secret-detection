package report

import (
	"fmt"
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/plugins"
	"github.com/gitleaks/go-gitdiff/gitdiff"
	"strconv"
	"strings"
)

const (
	SecretMaxCharacters    = 4
	ObfuscatedSecretString = "***"
	BeginPrivateKeyString  = "-----BEGIN"
	PrivateKeySeparator    = "-----"
)

type FileInfo struct {
	File        *gitdiff.File
	ContentType plugins.DiffType
}

func PrintReport(report *reporting.Report) {
	var sb strings.Builder
	sb.Grow(512 * len(report.Results)) // avoid multiple reallocations

	// TODO write something before Total Secrets Found?
	sb.WriteString("\nTotal Secrets Found: ")
	sb.WriteString(strconv.Itoa(report.TotalSecretsFound))
	sb.WriteString("\n\n")
	resultIndex := 1

	for _, secrets := range report.Results {
		for _, secret := range secrets {
			sourceInfo := strings.Split(secret.Source, ":")
			if len(sourceInfo) < 3 {
				// TODO handle error? skip? check this
				continue
			}
			contentType := sourceInfo[0]
			commitID := sourceInfo[1]
			fileName := strings.Join(sourceInfo[2:], ":") // to handle cases where the file name has ":"

			secretObfuscated := obfuscateSecret(secret.Value)

			sb.WriteString("Result #")
			sb.WriteString(strconv.Itoa(resultIndex))
			sb.WriteString(":\n\tCommit ID: ")
			sb.WriteString(commitID)
			sb.WriteString("\n\tFile Path: ")
			sb.WriteString(fileName)
			sb.WriteString("\n\tResult ID: ")
			sb.WriteString(secret.ID)
			sb.WriteString("\n\tContent Type: ")
			sb.WriteString(contentType)
			sb.WriteString("\n\tLocation: Line ")
			sb.WriteString(strconv.Itoa(secret.StartLine))
			sb.WriteString("\n\tRule ID: ")
			sb.WriteString(secret.RuleID)
			sb.WriteString("\n\tSecret Value: ")
			sb.WriteString(secretObfuscated)
			sb.WriteString("\n\n")
			resultIndex++
		}
	}

	sb.WriteString("A pre-receive hook set server side prevented you from push secrets.")
	sb.WriteString("Options for proceeding with the push:\n\n")
	sb.WriteString("  - Remediate detected secrets using the following workflow:\n")
	sb.WriteString("      1. Rewrite the git history to remove detected secrets from files and store them securely. Options:\n")
	sb.WriteString("         - Use environmental variables\n")
	sb.WriteString("         - Use a secret management service\n")
	sb.WriteString("         - Use a configuration management tool\n")
	sb.WriteString("         - Encrypt files containing secrets (least secure method)\n")
	sb.WriteString("      2. Push fixed code.\n\n")

	sb.WriteString("  - Ignore detected secrets:\n")
	sb.WriteString("      TODO\n")

	fmt.Print(sb.String())
}

func obfuscateSecret(snippet string) string {
	truncatedSecret := snippet

	// If the snippet is a private key get the secret part of it
	if strings.HasPrefix(truncatedSecret, BeginPrivateKeyString) {
		// Find the string between the second and third separator
		truncatedSecret = strings.Split(truncatedSecret, PrivateKeySeparator)[2]
		// Remove the first new line character if it exists
		truncatedSecret = strings.TrimPrefix(truncatedSecret, "\n")
		truncatedSecret = strings.TrimPrefix(truncatedSecret, "\\n")
	}

	if len(truncatedSecret) > SecretMaxCharacters {
		truncatedSecret = truncatedSecret[:SecretMaxCharacters] + ObfuscatedSecretString
	}
	return truncatedSecret
}
