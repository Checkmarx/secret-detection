package hooks

import (
	"fmt"
	. "github.com/Checkmarx/secret-detection/pkg/parser"
	. "github.com/Checkmarx/secret-detection/pkg/report"
	"github.com/checkmarx/2ms/lib/reporting"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"strings"
)

// Scan is the entry point for the secret scanning hook.
func Scan() error {
	color.NoColor = false

	report, fileDiffs, err := runSecretScan()
	if err != nil {
		return fmt.Errorf("failed to run scan: %w", err)
	}

	if report.TotalSecretsFound > 0 {
		PrintGitDiffReport(report, fileDiffs)
		os.Exit(1)
	}
	return nil
}

// runSecretScan executes the secret scan workflow.
func runSecretScan() (*reporting.Report, map[string][]Hunk, error) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	// Execute the diff parsing workflow.
	fileDiffs, err := runDiffParsing()
	if err != nil {
		return nil, nil, err
	}

	ignoredIDs, err := getIgnoredResultIds()
	if err != nil {
		return nil, nil, err
	}

	// Create the secrets scanner.
	scanner := twoms.NewScanner()
	itemsCh := make(chan twoms.ScanItem)
	reportCh := make(chan *reporting.Report)
	errScanCh := make(chan error, 1)

	// Start the scanning in a separate goroutine.
	go func() {
		report, err := scanner.ScanDynamic(itemsCh, twoms.ScanConfig{IgnoreResultIds: ignoredIDs})
		if err != nil {
			errScanCh <- err
			return
		}
		reportCh <- report
	}()

	// For each file, send its secret diff content for scanning.
	for file, hunks := range fileDiffs {
		sendDiffContentForScanning(file, hunks, itemsCh)
	}
	close(itemsCh)

	// Wait for the scanner to finish.
	select {
	case rep := <-reportCh:
		return rep, fileDiffs, nil
	case err := <-errScanCh:
		return nil, nil, err
	}
}

// runDiffParsing executes the git diff command and returns the parsed file diffs.
func runDiffParsing() (map[string][]Hunk, error) {
	cmd := exec.Command("git", "diff", "--unified=0", "--staged")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start git diff: %w", err)
	}
	parser := NewDiffParser()
	if err := parser.ParseDiffStream(pipe); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git diff command failed: %w", err)
	}
	return parser.FileDiffs, nil
}

// sendDiffContentForScanning sends the concatenated hunk content of a file to the scan channel.
func sendDiffContentForScanning(file string, hunks []Hunk, items chan<- twoms.ScanItem) {
	var builder strings.Builder
	for _, hunk := range hunks {
		builder.WriteString(hunk.Content)
	}
	content := builder.String()
	items <- twoms.ScanItem{
		Content: &content,
		ID:      fmt.Sprintf("pre-commit-%s", file),
		Source:  file,
	}
}
