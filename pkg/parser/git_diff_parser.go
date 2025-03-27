package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	hunkRegex       = regexp.MustCompile(`@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)
	diffHeaderRegex = regexp.MustCompile(`^diff --git\s+"?a/(.+?)"?\s+"?b/(.+?)"?$`)
	maxFileDiffSize = 10 * 1024 * 1024 // 10 MB per file
)

// Hunk represents a diff hunk for secret scanning.
type Hunk struct {
	StartLine int
	Content   string
	Size      int
}

// DiffParser encapsulates state for parsing a git diff stream.
type DiffParser struct {
	currentFile      string
	currentHunks     []Hunk
	hunkContent      strings.Builder
	hunkStart        int
	hunkSize         int
	inHunk           bool
	currentFileBytes int
	skipFile         bool
	FileDiffs        map[string][]Hunk
}

// NewDiffParser creates a new DiffParser.
func NewDiffParser() *DiffParser {
	return &DiffParser{
		FileDiffs: make(map[string][]Hunk),
	}
}

// flushHunk flushes the current hunk if one is in progress.
func (dp *DiffParser) flushHunk() {
	if dp.inHunk {
		dp.currentHunks = append(dp.currentHunks, Hunk{
			StartLine: dp.hunkStart,
			Content:   dp.hunkContent.String(),
			Size:      dp.hunkSize,
		})
		dp.hunkContent.Reset()
		dp.inHunk = false
	}
}

// flushFile flushes the current file and resets file-specific state.
func (dp *DiffParser) flushFile() {
	dp.flushHunk()
	if dp.currentFile != "" && !dp.skipFile {
		dp.FileDiffs[dp.currentFile] = dp.currentHunks
	}
	dp.currentFile = ""
	dp.currentHunks = nil
	dp.currentFileBytes = 0
	dp.skipFile = false
}

// ParseDiffStream processes the git diff stream and builds the file-to-hunks mapping.
func (dp *DiffParser) ParseDiffStream(r io.Reader) error {
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		// Remove the newline character.
		line = strings.TrimRight(line, "\n")
		if err != nil {
			if err == io.EOF {
				// Process any remaining data if needed.
				if len(line) == 0 {
					break
				}
			} else {
				return err
			}
		}

		// Check for a new file header.
		if matches := diffHeaderRegex.FindStringSubmatch(line); matches != nil {
			dp.flushFile()
			// Use second capture group as file name.
			dp.currentFile = matches[2]
			continue
		}

		// Check for a hunk header.
		if strings.HasPrefix(line, "@@ ") {
			dp.flushHunk()
			dp.inHunk = true
			start, size, err := parseHunkHeader(line)
			if err != nil {
				return err
			}
			dp.hunkStart = start
			dp.hunkSize = size
			continue
		}

		// Process addition lines (only if inside a hunk).
		if dp.inHunk && strings.HasPrefix(line, "+") {
			if dp.skipFile {
				continue
			}
			addLine := line[1:]
			dp.currentFileBytes += len(addLine)
			if dp.currentFileBytes > maxFileDiffSize {
				dp.skipFile = true
				dp.currentHunks = nil
				dp.hunkContent.Reset()
				dp.inHunk = false
				continue
			}
			dp.hunkContent.WriteString(addLine)
			dp.hunkContent.WriteByte('\n')
		}

		if err == io.EOF {
			break
		}
	}
	dp.flushFile()
	return nil
}

// parseHunkHeader extracts the start line and size from a hunk header.
func parseHunkHeader(header string) (int, int, error) {
	matches := hunkRegex.FindStringSubmatch(header)
	if len(matches) < 2 {
		return 0, 0, fmt.Errorf("failed to parse hunk header: %q", header)
	}
	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start line: %w", err)
	}
	size := 1
	if len(matches) >= 3 && matches[2] != "" {
		size, err = strconv.Atoi(matches[2])
		if err != nil {
			return start, 0, fmt.Errorf("invalid hunk size: %w", err)
		}
	}
	return start, size, nil
}
