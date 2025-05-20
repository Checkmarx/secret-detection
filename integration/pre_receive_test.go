package integration

import (
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPreReceiveScan(t *testing.T) {
	t.Run("commit files without secrets and push", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t)
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
		workDir, cleanup := setupPreReceiveTmpDir(t)
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
	t.Run("commit files with secrets and push with skip option", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t)
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
		cmdPush := exec.Command("git", "push", "-o", "skip-secret-scanner")
		cmdPush.Dir = workDir
		output, err = cmdPush.CombinedOutput()
		outputString := string(output)
		assert.NoError(t, err, "should not fail to push: %s", outputString)
		assert.NotContains(t, outputString, "[remote rejected]")
		assert.NotContains(t, outputString, "(pre-receive hook declined)")
		assert.Contains(t, outputString, "Cx Secret Scanner bypassed")
	})
	t.Run("first commit no secrets and second commit with secrets", func(t *testing.T) {
		workDir, cleanup := setupPreReceiveTmpDir(t)
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
		workDir, cleanup := setupPreReceiveTmpDir(t)
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
		workDir, cleanup := setupPreReceiveTmpDir(t)
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
}

// setupPreReceiveTmpDir creates a temporary directory, initializes a bare Git server
// repository with push-options advertised, clones it as a client repository,
// and returns the client working directory.
func setupPreReceiveTmpDir(t *testing.T) (workDir string, cleanup func()) {
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
	hookContent := []byte(`#!/bin/sh
exec cx hooks pre-receive secrets-scan "$@"
`)
	err = os.WriteFile(preReceivePath, hookContent, 0755)
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
