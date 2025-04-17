package pre_receive

import (
	"bufio"
	"fmt"
	report2 "github.com/Checkmarx/secret-detection/pkg/report"
	"github.com/checkmarx/2ms/lib/reporting"
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/checkmarx/2ms/plugins"
	"github.com/gitleaks/go-gitdiff/gitdiff"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	zeroRev = "0000000000000000000000000000000000000000"
)

func Scan() error {
	report, fileDiffs, err := runSecretScan()
	if err != nil {
		return fmt.Errorf("failed to run scan: %w", err)
	}

	if report.TotalSecretsFound > 0 {
		UpdateResultsStartAndEndLine(report, fileDiffs)
		report2.PrintReport(report) // TODO update package name to not be "report2"
		os.Exit(1)
	}
	return nil
}

func runSecretScan() (*reporting.Report, map[string]*report2.FileInfo, error) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	// TODO: ignoredIDs, err := getIgnoredResultIds()

	procs := runtime.GOMAXPROCS(0) // TODO update to use it in 2ms, just for testing right now

	// Create the scanner.
	scanner := twoms.NewScanner()
	itemsCh := make(chan twoms.ScanItem, procs)
	reportCh := make(chan *reporting.Report)
	errScanCh := make(chan error, 1)

	go func() {
		report, err := scanner.ScanDynamic(itemsCh, twoms.ScanConfig{ /* TODO: IgnoreResultIds: ignoredIDs */ })
		if err != nil {
			errScanCh <- err
			return
		}
		reportCh <- report
	}()

	fileDiffs, err := runDiffParsing(itemsCh)
	if err != nil {
		return nil, nil, err
	}
	close(itemsCh)

	// Wait for the scanner to finish.
	select {
	case rep := <-reportCh:
		return rep, fileDiffs, nil
	case err = <-errScanCh:
		return nil, nil, err
	}
}

func runDiffParsing(itemsChan chan twoms.ScanItem) (map[string]*report2.FileInfo, error) {
	fileDiffs := make(map[string]*report2.FileInfo)
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		// Expect exactly three fields: oldRev, newRev, and refName.
		parts := strings.Fields(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid input line: %s", line)
		}
		oldRev, newRev, refName := parts[0], parts[1], parts[2]

		var diffCmd *exec.Cmd
		switch {
		case oldRev == zeroRev && newRev != zeroRev:
			// New ref — show the patch for the root commit.
			diffCmd = exec.Command("git", "log", "-p", "--root", newRev)
		case newRev == zeroRev:
			// Ref deletion.
			continue
		default:
			// Normal update: show commit logs with patches between the old and new revisions.
			diffCmd = exec.Command("git", "log", "-p", fmt.Sprintf("%s..%s", oldRev, newRev))
		}

		// Get the stdout pipe to parse the log output.
		pipe, err := diffCmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe for ref %s: %w", refName, err)
		}
		if err = diffCmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start log command for ref %s: %w", refName, err)
		}

		diffs, err := gitdiff.Parse(pipe)
		if err != nil {
			return nil, fmt.Errorf("failed to parse diff for ref %s: %w", refName, err)
		}
		for file := range diffs {
			processFileDiff(file, itemsChan, fileDiffs)
		}
		if err = diffCmd.Wait(); err != nil {
			return nil, fmt.Errorf("log command failed for ref %s: %w", refName, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed reading input: %w", err)
	}

	return fileDiffs, nil
}

func processFileDiff(file *gitdiff.File, itemsChan chan twoms.ScanItem, fileDiffs map[string]*report2.FileInfo) {
	if file.PatchHeader == nil {
		// When parsing the PatchHeader, the token size limit may be exceeded, resulting in a nil value.
		// This scenario is unlikely but may cause the scan to never complete.
		file.PatchHeader = &gitdiff.PatchHeader{}
	}

	log.Debug().Msgf("file: %s; Commit: %s", file.NewName, file.PatchHeader.Title)

	// Skip binary files.
	if file.IsBinary {
		return
	}

	// Extract the changes (added and removed) from the text fragments.
	addedChanges, removedChanges := extractChanges(file.TextFragments)

	var fileName string
	if file.IsDelete {
		fileName = file.OldName
	} else {
		fileName = file.NewName
	}
	id := fmt.Sprintf("hooks-%s", fileName)

	if addedChanges != "" {
		source := fmt.Sprintf("Added:%s:%s", file.PatchHeader.SHA, fileName)
		itemsChan <- twoms.ScanItem{
			Content: &addedChanges,
			ID:      id,
			Source:  source,
		}
		fileDiffs[source] = &report2.FileInfo{File: file, ContentType: plugins.AddedContent}
	}

	if removedChanges != "" {
		source := fmt.Sprintf("Deleted:%s:%s", file.PatchHeader.SHA, fileName)
		itemsChan <- twoms.ScanItem{
			Content: &removedChanges,
			ID:      id,
			Source:  source,
		}
		fileDiffs[source] = &report2.FileInfo{File: file, ContentType: plugins.RemovedContent}
	}
}

func extractChanges(fragments []*gitdiff.TextFragment) (added string, removed string) {
	var addedBuilder, removedBuilder strings.Builder

	for _, tf := range fragments {
		if tf == nil {
			continue
		}
		for i := range tf.Lines {
			switch tf.Lines[i].Op {
			case gitdiff.OpAdd:
				addedBuilder.WriteString(tf.Lines[i].Line)
			case gitdiff.OpDelete:
				removedBuilder.WriteString(tf.Lines[i].Line)
			default:
			}
			// Clean up the line content to free memory.
			tf.Lines[i].Line = ""
		}
	}

	return addedBuilder.String(), removedBuilder.String()
}

func UpdateResultsStartAndEndLine(report *reporting.Report, fileDiffs map[string]*report2.FileInfo) {
	for id, secrets := range report.Results {
		for secretIndex, secret := range secrets {
			fileDiff := fileDiffs[secret.Source]
			newStartLine, newEndLine := plugins.GetGitStartAndEndLine(&plugins.GitInfo{
				Hunks:       fileDiff.File.TextFragments,
				ContentType: fileDiff.ContentType,
			}, secret.StartLine, secret.EndLine)
			report.Results[id][secretIndex].StartLine = newStartLine
			report.Results[id][secretIndex].EndLine = newEndLine
		}
	}
}
