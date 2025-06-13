package integration

import (
	"encoding/json"
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/report"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const (
	ignoreResultIdConfig = "testdata/configs/ignore-result-id.yaml"
	ignoreRuleIdConfig   = "testdata/configs/ignore-rule-id.yaml"
	pathExclusionConfig  = "testdata/configs/path-exclusion.yaml"
	misconfiguredConfig  = "testdata/configs/misconfigured.yaml"
	logsFolderConfig     = "testdata/configs/logs_folder_path.yaml"
	allowSkipTrueConfig  = "testdata/configs/allow_skip_true.yaml"
	allowSkipFalseConfig = "testdata/configs/allow_skip_false.yaml"
	emptyConfig          = "testdata/configs/empty.yaml"
)

func TestPreReceiveScan(t *testing.T) {
	t.Run("commit files without secrets and push", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t, "")
		defer cleanup()

		// Create files without secrets in the client repo
		file1 := filepath.Join(workDir, "no-secrets.txt")
		err := os.WriteFile(file1, []byte("dummy content 1"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "no-secrets2.txt")
		err = os.WriteFile(file2, []byte("dummy content 2"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "no-secrets.txt", "no-secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "no secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should not fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "No secrets detected by Cx Secret Scanner")
	})
	t.Run("commit files with secrets and push", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t, "")
		defer cleanup()

		// Create files with secrets in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err := os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file2, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Detected 2 secrets across 1 commit")
	})
	t.Run("commit files with secrets and push with skip option (allow_skip=true)", func(t *testing.T) {
		rel := allowSkipTrueConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files with secrets in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file2, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "-o", "skip-secret-scanner")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should not fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Cx Secret Scanner bypassed")
	})
	t.Run("commit files with secrets and push with skip option (allow_skip=false)", func(t *testing.T) {
		rel := allowSkipFalseConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files with secrets in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file2, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "-o", "skip-secret-scanner")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.NotContains(t, outputString, "Cx Secret Scanner bypassed")
	})
	t.Run("commit files with secrets and push with skip option (allow_skip not set)", func(t *testing.T) {
		rel := emptyConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files with secrets in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file2, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "-o", "skip-secret-scanner")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.NotContains(t, outputString, "Cx Secret Scanner bypassed")
	})
	t.Run("first commit no secrets and second commit with secrets", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t, "")
		defer cleanup()

		// Create files without secrets in the client repo
		file1 := filepath.Join(workDir, "no-secrets.txt")
		err := os.WriteFile(file1, []byte("dummy content 1"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "no-secrets2.txt")
		err = os.WriteFile(file2, []byte("dummy content 2"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "no-secrets.txt", "no-secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "no secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Create files with secrets in the client repo
		file3 := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file3, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDA"), 0644)
		assert.NoError(t, err)

		file4 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file4, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd = exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err = cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit = exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Detected 2 secrets across 1 commit")
	})
	t.Run("create tag with secret and push", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t, "")
		defer cleanup()

		// Create file with secret in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err := os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd := exec.Command("git", "add", "secrets.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// tag the commit
		cmdTag := exec.Command("git", "tag", "v1.0-secrets")
		cmdTag.Dir = workDir
		output, err = cmdTag.CombinedOutput()
		assert.NoError(t, err, "should not fail to tag: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "origin", "refs/tags/v1.0-secrets")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Detected 1 secret across 1 commit")
	})
	t.Run("create multiple branches with secrets and push", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t, "")
		defer cleanup()

		// checkout to branch1
		cmdCheckout := exec.Command("git", "checkout", "-b", "branch1")
		cmdCheckout.Dir = workDir
		output, err := cmdCheckout.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Create file with secret in the client repo
		file1 := filepath.Join(workDir, "secrets1.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDA"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd := exec.Command("git", "add", "secrets1.txt")
		cmdAdd.Dir = workDir
		output, err = cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets1")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// checkout to branch2
		cmdCheckout = exec.Command("git", "checkout", "-b", "branch2")
		cmdCheckout.Dir = workDir
		output, err = cmdCheckout.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Create file with secret in the client repo
		file1 = filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDB"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd = exec.Command("git", "add", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err = cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit = exec.Command("git", "commit", "-m", "secrets2")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// checkout to branch3
		cmdCheckout = exec.Command("git", "checkout", "-b", "branch3")
		cmdCheckout.Dir = workDir
		output, err = cmdCheckout.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Create file with secret in the client repo
		file1 = filepath.Join(workDir, "secrets3.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDC"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd = exec.Command("git", "add", "secrets3.txt")
		cmdAdd.Dir = workDir
		output, err = cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit = exec.Command("git", "commit", "-m", "secrets3")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "origin", "branch1", "branch2", "branch3")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Detected 3 secrets across 3 commits")
	})
	t.Run("push secrets ignored by result id", func(t *testing.T) {
		rel := ignoreResultIdConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files with secrets in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file2, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should not fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "No secrets detected by Cx Secret Scanner")
	})
	t.Run("push secrets ignored by rule id", func(t *testing.T) {
		rel := ignoreRuleIdConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files with secrets in the client repo
		file1 := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file1, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "secrets2.txt")
		err = os.WriteFile(file2, []byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJtb2NrU3ViMiIsIm5hbWUiOiJtb2NrTmFtZTIifQ.dummysignature"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "secrets.txt", "secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should not fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "No secrets detected by Cx Secret Scanner")
	})
	t.Run("push secrets ignored by path exclusion", func(t *testing.T) {
		rel := pathExclusionConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files with secrets in excluded paths
		paths := []string{
			"secret.txt",                // excluded by exact name
			"internal/tests/secret.txt", // excluded by glob internal/tests/*
			"docs/secret.md",            // excluded by *.md
		}
		for _, p := range paths {
			full := filepath.Join(workDir, p)
			// ensure parent dirs exist
			err = os.MkdirAll(filepath.Dir(full), 0755)
			assert.NoError(t, err, "failed to create directory for %s", p)

			err = os.WriteFile(full, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
			assert.NoError(t, err, "failed to write %s", p)
		}

		// Stage files
		cmdAdd := exec.Command("git", "add", ".")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should not fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "No secrets detected by Cx Secret Scanner")
	})
	t.Run("commit files without secrets and push with misconfigured config", func(t *testing.T) {
		rel := misconfiguredConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create files without secrets in the client repo
		file1 := filepath.Join(workDir, "no-secrets.txt")
		err = os.WriteFile(file1, []byte("dummy content 1"), 0644)
		assert.NoError(t, err)

		file2 := filepath.Join(workDir, "no-secrets2.txt")
		err = os.WriteFile(file2, []byte("dummy content 2"), 0644)
		assert.NoError(t, err)

		// Stage files
		cmdAdd := exec.Command("git", "add", "no-secrets.txt", "no-secrets2.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "no secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, fmt.Sprintf("configuration file at %s is misconfigured", configPath))
	})
	t.Run("should create a report when pushing secrets with a valid logs folder path", func(t *testing.T) {
		rel := logsFolderConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// ensure logs folder exists on the server side
		serverDir := filepath.Join(filepath.Dir(workDir), "server")
		logsDir := filepath.Join(serverDir, "logsFolder")
		err = os.MkdirAll(logsDir, 0755)
		assert.NoError(t, err, "should create logs folder on server")

		// Create file with secret in the client repo
		file := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd := exec.Command("git", "add", "secrets.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "[remote rejected]")
		assert.Contains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Detected 1 secret across 1 commit")

		// verify the report was written
		entries, err := os.ReadDir(logsDir)
		assert.NoError(t, err, "should be able to read logs folder")
		assert.Len(t, entries, 1, "exactly one log file should have been created")

		fname := entries[0].Name()
		// timestamp format: 2006-01-02_15-04-05.000000000
		assert.Regexp(t, `^report_\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}\.\d{9}\.json$`, fname,
			"file name should include UTC timestamp")

		contents, err := os.ReadFile(filepath.Join(logsDir, fname))
		assert.NoError(t, err, "should read the report file")

		var out report.ReportOutput
		assert.NoError(t, json.Unmarshal(contents, &out), "report must be valid JSON")

		// Top‐level checks
		assert.Equal(t, 1, out.TotalSecretsFound, "there should be exactly 1 secret")
		assert.Len(t, out.Commits, 1, "there should be exactly 1 commit")

		// Inspect the single commit
		commit := out.Commits[0]
		assert.NotEmpty(t, commit.CommitID, "commit_id must not be empty")
		assert.NotEmpty(t, commit.Author, "author must not be empty")
		assert.False(t, commit.Date.IsZero(), "date must be set")

		// Exactly one file in that commit
		assert.Len(t, commit.Files, 1, "commit should have exactly 1 file")
		fileSummary := commit.Files[0]
		assert.NotEmpty(t, fileSummary.FileName, "file_name must not be empty")

		// Exactly one secret in that file
		assert.Len(t, fileSummary.Secrets, 1, "file should have exactly 1 secret")
		secret := fileSummary.Secrets[0]
		assert.NotEmpty(t, secret.ID, "secret.id must not be empty")
		assert.NotEmpty(t, secret.Value, "secret.value must not be empty")
		assert.NotEmpty(t, secret.RuleID, "secret.rule_id must not be empty")
		assert.NotZero(t, secret.StartLine, "secret.start_line must be non-zero")
		assert.NotEmpty(t, secret.ContentType, "secret.content_type must not be empty")
	})
	t.Run("should create a report when skipping secret scanner with a valid logs folder path", func(t *testing.T) {
		rel := logsFolderConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// ensure logs folder exists on the server side
		serverDir := filepath.Join(filepath.Dir(workDir), "server")
		logsDir := filepath.Join(serverDir, "logsFolder")
		err = os.MkdirAll(logsDir, 0755)
		assert.NoError(t, err, "should create logs folder on server")

		// Create file with secret in the client repo
		file := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd := exec.Command("git", "add", "secrets.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "-o", "skip-secret-scanner")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.NotContains(t, outputString, "Detected 1 secret across 1 commit")

		// verify the report was written
		entries, err := os.ReadDir(logsDir)
		assert.NoError(t, err, "should be able to read logs folder")
		assert.Len(t, entries, 1, "exactly one log file should have been created")

		fname := entries[0].Name()
		// timestamp format: 2006-01-02_15-04-05.000000000
		assert.Regexp(t, `^skip_\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}\.\d{9}\.json$`, fname,
			"file name should include UTC timestamp")

		contents, err := os.ReadFile(filepath.Join(logsDir, fname))
		assert.NoError(t, err, "should read the report file")

		// Unmarshal to verify the schema
		var root map[string]interface{}
		assert.NoError(t, json.Unmarshal(contents, &root), "should be valid JSON")

		// Top‐level keys
		user, ok := root["user"].(string)
		assert.True(t, ok, `"user" must be a string`)
		assert.NotEmpty(t, user, `"user" should not be empty`)

		refs, ok := root["refs"].([]interface{})
		assert.True(t, ok, `"refs" must be an array`)

		assert.Len(t, refs, 1, `"refs" array length`)

		entry, ok := refs[0].(map[string]interface{})
		assert.True(t, ok, `refs[0] should be an object`)

		// old_object
		oo, ok := entry["old_object"].(string)
		assert.True(t, ok, `refs[0].old_object must be a string`)
		assert.NotEmpty(t, oo, "old object should not be empty")

		// new_object
		no, ok := entry["new_object"].(string)
		assert.True(t, ok, `refs[0].new_object must be a string`)
		assert.NotEmpty(t, no, "new object should not be empty")

		// ref_name
		rn, ok := entry["ref_name"].(string)
		assert.True(t, ok, `refs[0].ref_name must be a string`)
		assert.NotEmpty(t, rn, "ref name should not be empty")

		// no extra keys lived in the JSON:
		assert.Len(t, root, 2, `only "user" and "refs" should appear`)
	})
	t.Run("should fail when logs folder is set but does not exist", func(t *testing.T) {
		rel := logsFolderConfig
		configPath, err := filepath.Abs(rel)
		assert.NoError(t, err, "should not fail to get configPath")
		workDir, cleanup := setupPreReceiveTmpDir(t, configPath)
		defer cleanup()

		// Create file with secret in the client repo
		file := filepath.Join(workDir, "secrets.txt")
		err = os.WriteFile(file, []byte("ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"), 0644)
		assert.NoError(t, err)

		// Stage file
		cmdAdd := exec.Command("git", "add", "secrets.txt")
		cmdAdd.Dir = workDir
		output, err := cmdAdd.CombinedOutput()
		assert.NoError(t, err, "failed to stage files: %s", string(output))

		// Commit changes
		cmdCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdCommit.Dir = workDir
		output, err = cmdCommit.CombinedOutput()
		assert.NoError(t, err, "should not fail to commit: %s", string(output))

		// Push changes
		cmdPush := exec.Command("git", "push", "-o", "skip-secret-scanner")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.Error(t, err, "should fail to push: %s", outputString)
		assert.Contains(t, outputString, "log folder \"logsFolder\" does not exist")
	})
}

// setupPreReceiveTmpDir creates a temporary directory, initializes a bare Git server
// repository with push-options advertised, clones it as a client repository,
// and returns the client working directory.
func setupPreReceiveTmpDir(t *testing.T, configPath string) (workDir string, cleanup func()) {
	// Save original working directory
	origWD, err := os.Getwd()
	assert.NoError(t, err)

	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Init bare repo in a 'server' subdir
	serverDir := filepath.Join(tmpDir, "server")
	err = exec.Command("git", "init", "--bare", serverDir).Run()
	assert.NoError(t, err)

	// cd into the bare repo, enable push-options, then cd back
	err = os.Chdir(serverDir)
	assert.NoError(t, err)
	err = exec.Command("git", "config", "receive.advertisePushOptions", "true").Run()
	assert.NoError(t, err)
	// return to original dir
	err = os.Chdir(origWD)
	assert.NoError(t, err)

	// Write pre-receive hook
	hooksDir := filepath.Join(serverDir, "hooks")
	preReceivePath := filepath.Join(hooksDir, "pre-receive")
	cmd := "exec cx hooks pre-receive secrets-scan"
	if configPath != "" {
		cmd = fmt.Sprintf(`%s --config "%s"`, cmd, configPath)
	}
	hookScript := fmt.Sprintf(`#!/bin/sh
%s "$@"
`, cmd)
	err = os.WriteFile(preReceivePath, []byte(hookScript), 0755)
	assert.NoError(t, err)

	// Clone the bare repository into 'client'
	clientDir := filepath.Join(tmpDir, "client")
	err = exec.Command("git", "clone", serverDir, clientDir).Run()
	assert.NoError(t, err)

	// cd into the client working directory
	workDir = clientDir
	err = os.Chdir(workDir)
	assert.NoError(t, err)

	// Configure dummy user details
	if out, err := exec.Command("git", "config", "user.email", "dummy@example.com").CombinedOutput(); err != nil {
		t.Fatalf("failed to set user.email: %v – %s", err, out)
	}
	if out, err := exec.Command("git", "config", "user.name", "dummy Name").CombinedOutput(); err != nil {
		t.Fatalf("failed to set user.name: %v – %s", err, out)
	}

	// cleanup restores original working directory
	cleanup = func() {
		_ = os.Chdir(origWD)
	}

	return workDir, cleanup
}
