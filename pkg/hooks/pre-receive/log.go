package pre_receive

import (
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

// logReport writes the given report to a file named by creation time.
func logReport(folderPath, scanReport string) error {
	if folderPath == "" {
		return nil
	}

	now := time.Now().UTC()
	timestamp := now.Format("2006-01-02_15-04-05.000000000")
	fileName := fmt.Sprintf("report_%s.log", timestamp)

	logFilePath := filepath.Join(folderPath, fileName)

	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", logFilePath, err)
	}
	defer f.Close()

	if _, err = f.WriteString(scanReport); err != nil {
		return fmt.Errorf("failed to write to log file %q: %w", logFilePath, err)
	}

	return nil
}

// logSkip writes a one-off skip log named skip_<timestamp>.log,
// including the pusher's username (if available) and exactly which refs were pushed.
func logSkip(folderPath string, refs []string) error {
	if folderPath == "" {
		return nil
	}

	// Determine pusher username from known environment variables
	user := os.Getenv(envGitHubUserLogin)
	if user == "" {
		user = os.Getenv(envGitLabUsername)
	}
	if user == "" {
		user = os.Getenv(envBitbucketUserName)
	}

	// Prepare user information line
	userInfo := fmt.Sprintf("User: %s\n", user)
	if user == "" {
		userInfo = "User: <unknown> (could not retrieve pusher username)\n"
	}

	// Build the full log content
	var b strings.Builder
	b.WriteString(userInfo)
	b.WriteString("Push skipped by secret scanner for refs:\n")
	b.WriteString("Format: <old object> <new object> <ref name>\n")
	for _, r := range refs {
		b.WriteString(r)
		b.WriteString("\n")
	}

	// Timestamp and file path
	ts := time.Now().UTC().Format("2006-01-02_15-04-05.000000000")
	filePath := filepath.Join(folderPath, fmt.Sprintf("skip_%s.log", ts))

	// Create or truncate the log file
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open skip log file %q: %w", filePath, err)
	}
	defer f.Close()

	// Write the complete content
	if _, err = f.WriteString(b.String()); err != nil {
		return fmt.Errorf("writing skip log: %w", err)
	}

	return nil
}
