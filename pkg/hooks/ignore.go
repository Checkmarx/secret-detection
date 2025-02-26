package hooks

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ignore adds the provided resultId to the ignore list stored in the ".checkmarx_ignore.txt" file
func Ignore(resultId string) error {
	ignoreFilePath := filepath.Join(".", ".checkmarx_ignore.txt")

	if _, err := os.Stat(ignoreFilePath); os.IsNotExist(err) {
		err = os.WriteFile(ignoreFilePath, []byte(resultId+"\n"), 0644)
		if err != nil {
			return fmt.Errorf("failed to create .checkmarx_ignore.txt: %w", err)
		}
	} else {
		err = appendIgnoredResultId(resultId)
		if err != nil {
			return err
		}
	}

	return nil
}

func readIgnoredResultIds() ([]string, error) {
	ignoreFilePath := filepath.Join(".", ".checkmarx_ignore.txt")
	if _, err := os.Stat(ignoreFilePath); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(ignoreFilePath)
	if err != nil {
		return nil, err
	}

	var shas []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Trim leading spaces and tab characters
		sha := strings.TrimLeft(line, " \t")
		if sha != "" {
			shas = append(shas, sha)
		}
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}

	err = file.Close()
	if err != nil {
		return nil, err
	}

	return shas, nil
}

func appendIgnoredResultId(resultId string) error {
	ignoreFilePath := filepath.Join(".", ".checkmarx_ignore.txt")
	file, err := os.Open(ignoreFilePath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	exists := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == resultId {
			exists = true
			break
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	if !exists {
		file, err = os.OpenFile(ignoreFilePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		if _, err = file.WriteString(resultId + "\n"); err != nil {
			return err
		}

		err = file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
