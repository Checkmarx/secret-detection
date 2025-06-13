package pre_receive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Environment variable keys for identifying the pusher
const (
	envGitHubUserLogin   = "GITHUB_USER_LOGIN" // GitHub Enterprise Server
	envGitLabUsername    = "GL_USERNAME"       // GitLab CE/EE
	envBitbucketUserName = "BB_USER_NAME"      // Bitbucket Server/DC
)

type skipLogEntry struct {
	User string         `json:"user"`
	Refs []skipRefEntry `json:"refs"`
}

type skipRefEntry struct {
	OldObject string `json:"old_object"`
	NewObject string `json:"new_object"`
	RefName   string `json:"ref_name"`
}

// logSkip writes a JSON skip log named skip_<timestamp>.json
func logSkip(folderPath string, refs []string) error {
	if folderPath == "" {
		return nil
	}

	// Determine pusher username
	user := os.Getenv(envGitHubUserLogin)
	if user == "" {
		user = os.Getenv(envGitLabUsername)
	}
	if user == "" {
		user = os.Getenv(envBitbucketUserName)
	}
	if user == "" {
		user = "<unknown>"
	}

	// Parse each ref line into structured fields
	parsed := make([]skipRefEntry, 0, len(refs))
	for _, r := range refs {
		parts := strings.Fields(r)
		if len(parts) < 3 {
			continue
		}
		parsed = append(parsed, skipRefEntry{
			OldObject: parts[0],
			NewObject: parts[1],
			RefName:   parts[2],
		})
	}

	// Build JSON payload
	entry := skipLogEntry{
		User: user,
		Refs: parsed,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skip log JSON: %w", err)
	}

	ts := time.Now().UTC().Format("2006-01-02_15-04-05.000000000")
	fileName := fmt.Sprintf("skip_%s.json", ts)
	path := filepath.Join(folderPath, fileName)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write skip log JSON to %q: %w", path, err)
	}
	return nil
}

// validateLogsFolderPath checks if the given non-empty folderPath exists and is a directory, returning an error otherwise.
func validateLogsFolderPath(folderPath string) error {
	if folderPath == "" {
		return nil
	}

	info, err := os.Stat(folderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log folder %q does not exist", folderPath)
		}
		return fmt.Errorf("unable to stat folder %q: %w", folderPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q exists but is not a directory", folderPath)
	}
	return nil
}

// logJSONReport writes the given JSON report to a file named by creation time.
func logJSONReport(folderPath string, jsonReport []byte) error {
	if folderPath == "" {
		return nil
	}

	now := time.Now().UTC()
	timestamp := now.Format("2006-01-02_15-04-05.000000000")
	fileName := fmt.Sprintf("report_%s.json", timestamp)
	fullPath := filepath.Join(folderPath, fileName)

	if err := os.WriteFile(fullPath, jsonReport, 0o644); err != nil {
		return fmt.Errorf("failed to write JSON report to %q: %w", fullPath, err)
	}
	return nil
}
