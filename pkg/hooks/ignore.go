package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ignore adds the provided resultIds to the ignore list stored in the ".checkmarx_ignore.txt" file
func Ignore(resultIds []string) error {
	ignoreFilePath := filepath.Join(".", ".checkmarx_ignore.txt")
	existingIDs := make(map[string]struct{})

	// If the file exists, read its content once into a map
	if _, err := os.Stat(ignoreFilePath); err == nil {
		data, err := os.ReadFile(ignoreFilePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", ignoreFilePath, err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				existingIDs[trimmed] = struct{}{}
			}
		}
	}

	// Collect only new IDs that are not already present
	var newIDs []string
	for _, id := range resultIds {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := existingIDs[id]; !exists {
			newIDs = append(newIDs, id)
		}
	}

	// If there are no new IDs, nothing to do.
	if len(newIDs) == 0 {
		fmt.Println("No new resultIds to add")
		return nil
	}

	// Open file for appending (creates it if it doesn't exist)
	file, err := os.OpenFile(ignoreFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", ignoreFilePath, err)
	}
	defer file.Close()

	// Write all new IDs, each on a new line
	for _, id := range newIDs {
		if _, err = file.WriteString(id + "\n"); err != nil {
			return fmt.Errorf("failed to write to %s: %w", ignoreFilePath, err)
		}
	}

	fmt.Printf("Added %d new IDs to %s\n", len(newIDs), ignoreFilePath)
	return nil
}

// getIgnoredResultIds reads the ".checkmarx_ignore.txt" file located in the current directory and
// returns a slice of ignored result IDs. Each line in the file is expected to contain a single result ID.
func getIgnoredResultIds() ([]string, error) {
	ignoreFilePath := filepath.Join(".", ".checkmarx_ignore.txt")
	data, err := os.ReadFile(ignoreFilePath)
	if err != nil {
		// If the file doesn't exist, return an empty slice without error
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var resultIds []string
	for _, line := range strings.Split(string(data), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			resultIds = append(resultIds, trimmed)
		}
	}
	return resultIds, nil
}
