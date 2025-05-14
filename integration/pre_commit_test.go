package integration

import (
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/config"
	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreCommitInstall(t *testing.T) {
	t.Run("Not a Git repository", func(t *testing.T) {
		_, cleanup := setupTmpDir(t)
		defer cleanup()

		// Run the pre-commit install and expect failure because it's not a git repo.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook")
		output, err := cmdPreCommitInstall.CombinedOutput()
		assert.Error(t, err, "should fail because it is not a git repo")
		assert.Contains(t, string(output), "current directory is not a Git repository")
	})
	t.Run("Git repository with no config file", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		_, err := cmdGitInit.CombinedOutput()
		assert.NoError(t, err, "git init should not fail")

		// Run the pre-commit install and expect success.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook")
		cmdPreCommitInstall.Dir = tmpDir
		output, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		// Verify the configuration file was created.
		configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")
		_, err = os.Stat(configPath)
		assert.NoError(t, err, "pre-commit config file should exist")

		// Compare expected YAML configuration.
		expectedYAML, err := yaml.Marshal(config.PreloadedConfig)
		assert.NoError(t, err)

		actualYAML, err := os.ReadFile(configPath)
		assert.NoError(t, err)

		var expectedData, actualData interface{}
		err = yaml.Unmarshal(expectedYAML, &expectedData)
		assert.NoError(t, err)
		err = yaml.Unmarshal(actualYAML, &actualData)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)

		// Verify expected output messages for pre-commit install.
		assert.Contains(t, string(output), "Installing local pre-commit hooks...")
		assert.Contains(t, string(output), "pre-commit installed at .git")
	})
	t.Run("Git repository with config file", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")

		dummyConfig := config.PreCommitConfig{
			Repos: []config.Repo{
				{
					Repo: "local",
					Hooks: []config.Hook{
						{
							ID:                      "other-hook",
							Name:                    "Other Hook",
							Entry:                   "other",
							Description:             "A different hook",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"other", "run"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "3.2.0",
						},
					},
				},
			},
		}
		dummyYAML, err := yaml.Marshal(dummyConfig)
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(configPath, dummyYAML, 0644))

		expectedConfig := config.PreCommitConfig{
			Repos: []config.Repo{
				{
					Repo: "local",
					Hooks: []config.Hook{
						{
							ID:                      "other-hook",
							Name:                    "Other Hook",
							Entry:                   "other",
							Description:             "A different hook",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"other", "run"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "3.2.0",
						},
						{
							ID:                      "cx-secret-detection",
							Name:                    "Cx Secret Detection",
							Entry:                   "cx",
							Description:             "Run Cx CLI secret detection",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"hooks", "pre-commit", "secrets-scan"},
							Language:                "system",
							PassFilenames:           false,
							MinimumPreCommitVersion: "3.2.0",
						},
					},
				},
			},
		}

		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook")
		cmdPreCommitInstall.Dir = tmpDir
		output, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		// Compare expected YAML configuration.
		expectedYAML, err := yaml.Marshal(expectedConfig)
		assert.NoError(t, err)

		actualYAML, err := os.ReadFile(configPath)
		assert.NoError(t, err)

		var expectedData, actualData interface{}
		err = yaml.Unmarshal(expectedYAML, &expectedData)
		assert.NoError(t, err)
		err = yaml.Unmarshal(actualYAML, &actualData)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)

		// Verify expected output messages for pre-commit install.
		assert.Contains(t, string(output), "Installing local pre-commit hooks...")
		assert.Contains(t, string(output), "pre-commit installed at .git")
	})
	t.Run("Install global", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Run the pre-commit hook installation with --global flag.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook", "--global")
		cmdPreCommitInstall.Dir = tmpDir
		output, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		assert.Contains(t, string(output), "Installing global pre-commit hook...")
		assert.Contains(t, string(output), "Global pre-commit hook installed successfully")

		// Retrieve the global hooks path from Git config.
		cmdGlobalHooks := exec.Command("git", "config", "--global", "core.hooksPath")
		hooksPathBytes, err := cmdGlobalHooks.Output()
		if err != nil {
			t.Fatalf("failed to retrieve global hooks path: %s", err)
		}
		hooksPath := strings.TrimSpace(string(hooksPathBytes))
		if hooksPath == "" {
			t.Fatal("global hooks path is not set")
		}

		defer func() {
			cmdUnset := exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
			if output, err := cmdUnset.CombinedOutput(); err != nil {
				t.Logf("failed to unset global hooks path: %s: %s", err, string(output))
			}
		}()

		// Build the full path to the pre-commit hook file.
		hookFilePath := filepath.Join(hooksPath, "pre-commit")
		if _, err := os.Stat(hookFilePath); os.IsNotExist(err) {
			t.Fatalf("pre-commit hook file does not exist at %s", hookFilePath)
		}

		// Read and compare the file content with the expected content
		content, err := os.ReadFile(hookFilePath)
		if err != nil {
			t.Fatalf("failed to read pre-commit hook file: %s", err)
		}
		expectedContent := "#!/bin/sh\ncx hooks pre-commit secrets-scan\n"
		assert.Equal(t, expectedContent, string(content), "pre-commit hook file content mismatch")
	})
}

func TestPreCommitUninstall(t *testing.T) {
	t.Run("Remove cx-secret-detection hook from config", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")

		dummyConfig := config.PreCommitConfig{
			Repos: []config.Repo{
				{
					Repo: "local",
					Hooks: []config.Hook{
						{
							ID:                      "other-hook",
							Name:                    "Other Hook",
							Entry:                   "other",
							Description:             "A different hook",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"other", "run"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "3.2.0",
						},
						{
							ID:                      "cx-secret-detection",
							Name:                    "Cx Secret Detection",
							Entry:                   "cx",
							Description:             "Run Cx CLI secret detection",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"pre-commit", "scan"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "3.2.0",
						},
					},
				},
			},
		}
		dummyYAML, err := yaml.Marshal(dummyConfig)
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(configPath, dummyYAML, 0644))

		expectedConfig := config.PreCommitConfig{
			Repos: []config.Repo{
				{
					Repo: "local",
					Hooks: []config.Hook{
						{
							ID:                      "other-hook",
							Name:                    "Other Hook",
							Entry:                   "other",
							Description:             "A different hook",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"other", "run"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "3.2.0",
						},
					},
				},
			},
		}

		cmdPreCommitUninstall := exec.Command("cx", "hooks", "pre-commit", "secrets-uninstall-git-hook")
		cmdPreCommitUninstall.Dir = tmpDir
		output, err := cmdPreCommitUninstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit uninstall should not fail in a git repo")

		// Compare expected YAML configuration.
		expectedYAML, err := yaml.Marshal(expectedConfig)
		assert.NoError(t, err)

		actualYAML, err := os.ReadFile(configPath)
		assert.NoError(t, err)

		var expectedData, actualData interface{}
		err = yaml.Unmarshal(expectedYAML, &expectedData)
		assert.NoError(t, err)
		err = yaml.Unmarshal(actualYAML, &actualData)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)

		assert.Contains(t, string(output), "cx-secret-detection hook uninstalled successfully.")
	})
	t.Run("Uninstall global", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// First, install the global pre-commit hook to ensure it exists.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook", "--global")
		cmdPreCommitInstall.Dir = tmpDir
		installOutput, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")
		assert.Contains(t, string(installOutput), "Installing global pre-commit hook...")
		assert.Contains(t, string(installOutput), "Global pre-commit hook installed successfully")

		// Retrieve the global hooks path from Git config.
		cmdGlobalHooks := exec.Command("git", "config", "--global", "core.hooksPath")
		hooksPathBytes, err := cmdGlobalHooks.Output()
		if err != nil {
			t.Fatalf("failed to retrieve global hooks path: %s", err)
		}
		hooksPath := strings.TrimSpace(string(hooksPathBytes))
		if hooksPath == "" {
			t.Fatal("global hooks path is not set")
		}

		// Verify that the pre-commit hook file exists.
		hookFilePath := filepath.Join(hooksPath, "pre-commit")
		if _, err := os.Stat(hookFilePath); os.IsNotExist(err) {
			t.Fatalf("pre-commit hook file does not exist at %s", hookFilePath)
		}

		// Run the uninstall command.
		cmdPreCommitUninstall := exec.Command("cx", "hooks", "pre-commit", "secrets-uninstall-git-hook", "--global")
		cmdPreCommitUninstall.Dir = tmpDir
		uninstallOutput, err := cmdPreCommitUninstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit uninstall should not fail")
		assert.Contains(t, string(uninstallOutput), "Uninstalling global pre-commit hook...")
		assert.Contains(t, string(uninstallOutput), "Global pre-commit hook removed successfully")

		// Confirm that the pre-commit hook file has been removed.
		if _, err := os.Stat(hookFilePath); !os.IsNotExist(err) {
			t.Fatalf("pre-commit hook file still exists at %s after uninstall", hookFilePath)
		}

		// Check that the global Git configuration for hooksPath has been unset.
		cmdGlobalHooks = exec.Command("git", "config", "--global", "core.hooksPath")
		hooksPathBytes, err = cmdGlobalHooks.Output()
		if err == nil {
			hooksPathAfterUninstall := strings.TrimSpace(string(hooksPathBytes))
			if hooksPathAfterUninstall != "" {
				t.Fatalf("global hooks path config still set to %s after uninstall", hooksPathAfterUninstall)
			}
		}
	})
}

func TestPreCommitUpdate(t *testing.T) {
	t.Run("Update cx-secret-detection hook from config", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		configPath := filepath.Join(tmpDir, ".pre-commit-config.yaml")

		dummyConfig := config.PreCommitConfig{
			Repos: []config.Repo{
				{
					Repo: "local",
					Hooks: []config.Hook{
						{
							ID:                      "cx-secret-detection",
							Name:                    "Cx Secret Detection",
							Entry:                   "cx",
							Description:             "Run Cx CLI secret detection",
							Stages:                  []string{"pre-commit"},
							Args:                    []string{"pre-commit", "dummy"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "0.0.0",
						},
					},
				},
			},
		}
		dummyYAML, err := yaml.Marshal(dummyConfig)
		assert.NoError(t, err)
		assert.NoError(t, os.WriteFile(configPath, dummyYAML, 0644))

		expectedConfig := config.PreloadedConfig

		cmdPreCommitUpdate := exec.Command("cx", "hooks", "pre-commit", "secrets-update-git-hook")
		cmdPreCommitUpdate.Dir = tmpDir
		output, err := cmdPreCommitUpdate.CombinedOutput()
		assert.NoError(t, err, "pre-commit update should not fail in a git repo")

		// Compare expected YAML configuration.
		expectedYAML, err := yaml.Marshal(expectedConfig)
		assert.NoError(t, err)

		actualYAML, err := os.ReadFile(configPath)
		assert.NoError(t, err)

		var expectedData, actualData interface{}
		err = yaml.Unmarshal(expectedYAML, &expectedData)
		assert.NoError(t, err)
		err = yaml.Unmarshal(actualYAML, &actualData)
		assert.NoError(t, err)
		assert.Equal(t, expectedData, actualData)

		assert.Contains(t, string(output), "Updating local cx-secret-detection hook...")
		assert.Contains(t, string(output), "cx-secret-detection hook updated successfully")
	})
	t.Run("Update global", func(t *testing.T) {
		// Set up the test binary environment.
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Run the install command to set up the global hooks configuration.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook", "--global")
		cmdPreCommitInstall.Dir = tmpDir
		installOutput, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")
		assert.Contains(t, string(installOutput), "Installing global pre-commit hook...")
		assert.Contains(t, string(installOutput), "Global pre-commit hook installed successfully")

		// Retrieve the global hooks path from Git config.
		cmdGlobalHooks := exec.Command("git", "config", "--global", "core.hooksPath")
		hooksPathBytes, err := cmdGlobalHooks.Output()
		if err != nil {
			t.Fatalf("failed to retrieve global hooks path: %s", err)
		}
		hooksPath := strings.TrimSpace(string(hooksPathBytes))
		if hooksPath == "" {
			t.Fatal("global hooks path is not set")
		}
		hookFilePath := filepath.Join(hooksPath, "pre-commit")

		// Create a pre-commit file with dummy content.
		dummyContent := "dummy content"
		if err := os.WriteFile(hookFilePath, []byte(dummyContent), 0755); err != nil {
			t.Fatalf("failed to write dummy pre-commit file: %v", err)
		}

		// Verify the dummy content is present.
		content, err := os.ReadFile(hookFilePath)
		if err != nil {
			t.Fatalf("failed to read pre-commit hook file: %v", err)
		}
		assert.Equal(t, dummyContent, string(content), "dummy pre-commit content mismatch before update")

		// Run the update command.
		cmdPreCommitUpdate := exec.Command("cx", "hooks", "pre-commit", "secrets-update-git-hook", "--global")
		cmdPreCommitUpdate.Dir = tmpDir
		updateOutput, err := cmdPreCommitUpdate.CombinedOutput()
		assert.NoError(t, err, "pre-commit update should not fail")
		assert.Contains(t, string(updateOutput), "Updating global pre-commit hook...")
		assert.Contains(t, string(updateOutput), "Global pre-commit hook updated successfully")

		// Expected content after the update (same as install global expected content).
		expectedContent := "#!/bin/sh\ncx hooks pre-commit secrets-scan\n"
		updatedContent, err := os.ReadFile(hookFilePath)
		if err != nil {
			t.Fatalf("failed to read pre-commit hook file after update: %v", err)
		}
		assert.Equal(t, expectedContent, string(updatedContent), "pre-commit hook content mismatch after update")

		cmdPreCommitUninstall := exec.Command("cx", "hooks", "pre-commit", "secrets-uninstall-git-hook", "--global")
		cmdPreCommitUninstall.Dir = tmpDir
		_, err = cmdPreCommitUninstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit uninstall should not fail")
	})
}

func TestPreCommitIgnore(t *testing.T) {
	t.Run("Ignore resultIds", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		mockSha1, mockSha2, mockSha3 := "sha1", "sha2", "sha3"
		cmdPreCommitIgnore := exec.Command("cx", "hooks", "pre-commit", "secrets-ignore", "--resultIds", fmt.Sprintf("%s,%s,%s", mockSha1, mockSha2, mockSha3))
		cmdPreCommitIgnore.Dir = tmpDir
		output, err := cmdPreCommitIgnore.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Added 3 new IDs to .checkmarx_ignore")

		ignorePath := filepath.Join(tmpDir, ".checkmarx_ignore")

		// Verify the ignore file was created.
		_, err = os.Stat(ignorePath)
		assert.NoError(t, err, "pre-commit ignore file should exist")

		// Read from file and verify expected shas
		data, err := os.ReadFile(ignorePath)
		assert.NoError(t, err)
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		expectedShas := []string{mockSha1, mockSha2, mockSha3}
		assert.Equal(t, expectedShas, lines, "ignore file does not contain the expected 3 shas")

		mockSha4, mockSha5 := "sha4", "sha5"
		cmdPreCommitIgnore = exec.Command("cx", "hooks", "pre-commit", "secrets-ignore", "--resultIds", fmt.Sprintf("%s,%s", mockSha4, mockSha5))
		cmdPreCommitIgnore.Dir = tmpDir
		output, err = cmdPreCommitIgnore.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Added 2 new IDs to .checkmarx_ignore")

		// Read from file and verify expected shas
		data, err = os.ReadFile(ignorePath)
		assert.NoError(t, err)
		lines = strings.Split(strings.TrimSpace(string(data)), "\n")
		expectedShas = []string{mockSha1, mockSha2, mockSha3, mockSha4, mockSha5}
		assert.Equal(t, expectedShas, lines, "ignore file does not contain all expected shas")
	})
	t.Run("Ignore all results", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Create two files with secrets
		secret1 := "ghp_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		secret2 := "ghp_BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
		secret3 := "ghp_CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"
		secret4 := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJtb2NrU3ViMSIsIm5hbWUiOiJtb2NrTmFtZTEifQ.dummysignature1"
		secret5 := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJtb2NrU3ViMiIsIm5hbWUiOiJtb2NrTmFtZTIifQ.dummysignature2"
		file1Path := filepath.Join(tmpDir, "github-pat.txt")
		file2Path := filepath.Join(tmpDir, "jwt.txt")
		err := os.WriteFile(file1Path, []byte(fmt.Sprintf("%s\n%s\n%s", secret1, secret2, secret3)), 0644)
		assert.NoError(t, err)
		err = os.WriteFile(file2Path, []byte(fmt.Sprintf("%s\n%s\n%s", secret4, secret5, secret5)), 0644)
		assert.NoError(t, err)

		// Stage the new files.
		cmdGitAdd := exec.Command("git", "add", "github-pat.txt", "jwt.txt")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to git add files: %s: %s", err, string(output))
		}

		cmdPreCommitIgnore := exec.Command("cx", "hooks", "pre-commit", "secrets-ignore", "--all")
		cmdPreCommitIgnore.Dir = tmpDir
		output, err := cmdPreCommitIgnore.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Added 5 new IDs to .checkmarx_ignore")

		ignorePath := filepath.Join(tmpDir, ".checkmarx_ignore")
		data, err := os.ReadFile(ignorePath)
		assert.NoError(t, err)
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		expectedShas := []string{
			"834a259ed3b76c6276d7e342a4018d3743b2926a",
			"250ab70ec2ffe57984fb72ad38e3d110429e99ef",
			"f4b68dd7bf7f3bf38ec1406f2c2efed37b109128",
			"d24005fcacdda80096118e33d082beff84f60041",
			"41a69f393a8cfc79d1ae072d545790967f057b2b",
		}
		assert.ElementsMatch(t, expectedShas, lines, "ignore file does not contain all expected shas")
	})
}

func TestPreCommitScan(t *testing.T) {
	t.Run("add files without secrets and commit", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// set dummy user.email
		cmdGitConfigEmail := exec.Command("git", "config", "user.email", "dummy@example.com")
		cmdGitConfigEmail.Dir = tmpDir
		if output, err := cmdGitConfigEmail.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy email: %s: %s", err, string(output))
		}

		// set dummy user.name
		cmdGitConfigName := exec.Command("git", "config", "user.name", "dummy Name")
		cmdGitConfigName.Dir = tmpDir
		if output, err := cmdGitConfigName.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy name: %s: %s", err, string(output))
		}

		// Install hook
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook")
		cmdPreCommitInstall.Dir = tmpDir
		_, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		// stage config file
		cmdGitAdd := exec.Command("git", "add", ".pre-commit-config.yaml")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage config file: %s: %s", err, string(output))
		}

		// Create files without secrets
		file1Path := filepath.Join(tmpDir, "no-secrets.txt")
		err = os.WriteFile(file1Path, []byte("dummy content"), 0644)
		assert.NoError(t, err)

		file2Path := filepath.Join(tmpDir, "no-secrets2.txt")
		err = os.WriteFile(file2Path, []byte("dummy content 2"), 0644)
		assert.NoError(t, err)

		// Stage the new file
		cmdGitAdd = exec.Command("git", "add", "no-secrets.txt")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage no-secrets file: %s: %s", err, string(output))
		}

		// commit changes, should not fail because no secrets were added
		cmdGitCommit := exec.Command("git", "commit", "-m", "no secrets")
		cmdGitCommit.Dir = tmpDir
		output, err := cmdGitCommit.CombinedOutput()
		if err != nil {
			t.Fatalf("should not fail to commit when no secrets are added: %s: %s", err, string(output))
		}

		assert.Contains(t, string(output), "Cx Secret Detection......................................................Passed")
	})
	t.Run("add files with secrets and commit then ignore all and commit again with local config", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Set dummy user.email.
		cmdGitConfigEmail := exec.Command("git", "config", "user.email", "dummy@example.com")
		cmdGitConfigEmail.Dir = tmpDir
		if output, err := cmdGitConfigEmail.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy email: %s: %s", err, string(output))
		}

		// Set dummy user.name.
		cmdGitConfigName := exec.Command("git", "config", "user.name", "dummy Name")
		cmdGitConfigName.Dir = tmpDir
		if output, err := cmdGitConfigName.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy name: %s: %s", err, string(output))
		}

		// Install the secrets git hook.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook")
		cmdPreCommitInstall.Dir = tmpDir
		_, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		// Stage the config file.
		cmdGitAdd := exec.Command("git", "add", ".pre-commit-config.yaml")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage config file: %s: %s", err, string(output))
		}

		// Create file1.txt with provided content.
		file1Content := `MOCK CONTENT 1
MOCK CONTENT 2
MOCK CONTENT 3
MOCK CONTENT 4

MOCK CONTENT TEST    -----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJAXWRPQyGlEY+SXz8Uslhe+MLjTgWd8lf/nA0hgCm9JFKC1tq1S73c
Q9naClNXsMqY7pwPt1bSY8jYRqHHbdoUvwIDAQABAkAfJkz1pCwtfkig8iZSEf2j
VUWBiYgUA9vizdJlsAZBLceLrdk8RZF2YOYCWHrpUtZVea37dzZJe99Dr53K0UZx
AiEAtyHQBGoCVHfzPM//a+4tv2ba3tx9at+3uzGR86YNMzcCIQCCjWHcLW/+sQTW
OXeXRrtxqHPp28ir8AVYuNX0nT1+uQIgJm158PMtufvRlpkux78a6mby1oD98Ecx
jp5AOhhF/NECICyHsQN69CJ5mt6/R01wMOt5u9/eubn76rbyhPgk0h7xAiEAjn6m
EmLwkIYD9VnZfp9+2UoWSh0qZiTIHyNwFpJH78o=
-----END RSA PRIVATE KEY-----

MOCK CONTENT 5
MOCK CONTENT 6
MOCK CONTENT 7
MOCK CONTENT 8`
		file1Path := filepath.Join(tmpDir, "file1.txt")
		err = os.WriteFile(file1Path, []byte(file1Content), 0644)
		assert.NoError(t, err)

		// Create file2.txt with provided content.
		file2Content := `SECRETS WITH DIFFERENT VALUE
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJtb2NrU3ViMiIsIm5hbWUiOiJtb2NrTmFtZTIifQ.dummysignature mongodb+srv://radar:mytoken@io.dbb.mongodb.net/?retryWrites=true&w=majority

SECRETS WITH THE SAME VALUE
ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD`
		file2Path := filepath.Join(tmpDir, "file2.txt")
		err = os.WriteFile(file2Path, []byte(file2Content), 0644)
		assert.NoError(t, err)

		// Stage the new files.
		cmdGitAdd = exec.Command("git", "add", "file1.txt", "file2.txt")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage files with secrets: %s: %s", err, string(output))
		}

		cmdGitCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdGitCommit.Dir = tmpDir
		output, err := cmdGitCommit.CombinedOutput()
		assert.Error(t, err)

		color.NoColor = false
		red := color.New(color.FgRed).SprintFunc()
		white := color.New(color.FgWhite).SprintFunc()
		hiYellow := color.New(color.FgHiYellow).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		hiBlue := color.New(color.FgHiBlue).SprintFunc()

		expectedOutput := "Cx Secret Detection......................................................Failed\n" +
			"- hook id: cx-secret-detection\n" +
			"- exit code: 1\n\n" +
			white("Commit scanned for secrets:\n\n") +
			white("Detected ") + red("5 secrets ") + white("in ") + red("2 files\n\n") +
			white("#1 File: ") + hiYellow("file1.txt\n") +
			red("1 ") + white("Secret detected in file\n") +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("private-key\n") +
			white("\tResult ID: ") + hiYellow("6690f3400f0b2445c71b44f9fa89308e35f7c822\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 6\n") +
			"\t           4 | MOCK CONTENT 4\n" +
			"\t           5 | \n" +
			hiYellow("\t           6 |") + " MOCK CONTENT TEST    " + red("----****** *** ******* ********") + "\n" +
			hiYellow("\t           7 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t           8 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t           9 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          10 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          11 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          12 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          13 |") + " " + red("****************************************") + "\n" +
			hiYellow("\t          14 |") + " " + red("******** *** ******* ********") + "\n" +
			"\t          15 | \n" +
			"\t          16 | MOCK CONTENT 5\n" +
			white("") + "\n" +
			white("#2 File: ") + hiYellow("file2.txt\n") +
			red("4 ") + white("Secrets detected in file\n") +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("jwt\n") +
			white("\tResult ID: ") + hiYellow("15db03900061beb413097e4e689b15d36efbb99b\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 2\n") +
			"\t           1 | SECRETS WITH DIFFERENT VALUE\n" +
			hiYellow("\t           2 |") + " " + red("eyJh**************************************************************************************************") + " mongodb+srv://rada*********@io.dbb.mongodb.net/?retryWrites=true&w=majority" + "\n" +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("authenticated-url\n") +
			white("\tResult ID: ") + hiYellow("f4dc5a5fd23a6308314898cb0af125e6423c213e\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 2\n") +
			"\t           1 | SECRETS WITH DIFFERENT VALUE\n" +
			hiYellow("\t           2 |") + " eyJh************************************************************************************************** mongodb+srv://" + red("rada*********") + "@io.dbb.mongodb.net/?retryWrites=true&w=majority" + "\n" +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("github-pat\n") +
			white("\tResult ID: ") + hiYellow("3b20d93240ebbe2566fd39a2fd7b456784502658\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 5\n") +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			hiYellow("\t           5 |") + " " + red("ghp_************************************") + " ghp_************************************" + "\n" +
			"\t           6 | \n" +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("github-pat\n") +
			white("\tResult ID: ") + hiYellow("3b20d93240ebbe2566fd39a2fd7b456784502658\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 5\n") +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			hiYellow("\t           5 |") + " ghp_************************************" + " " + red("ghp_************************************") + "\n" +
			"\t           6 | \n" +
			white("") + "\n" +
			white("Options for proceeding with the commit:\n\n") +
			white("  - Remediate detected secrets using the following workflow (") +
			green("recommended") +
			white("):\n") +
			white("      1. Remove detected secrets from files and store them securely. Options:\n") +
			white("         - Use environmental variables\n") +
			white("         - Use a secret management service\n") +
			white("         - Use a configuration management tool\n") +
			white("         - Encrypt files containing secrets (least secure method)\n") +
			white("      2. Commit fixed code.\n\n") +
			white("  - Ignore detected secrets (") +
			yellow("not recommended") +
			white("):\n") +
			white("      Use one of the following commands:\n") +
			hiBlue("          cx hooks pre-commit secrets-ignore --all\n") +
			hiBlue("          cx hooks pre-commit secrets-ignore --resultIds=id1,id2\n\n") +
			white("  - Bypass the pre-commit secret detection scanner (") +
			red("not recommended") +
			white("):\n") +
			white("      Use one of the following commands based on your OS:\n\n") +
			white("        Bash/Zsh:\n") +
			hiBlue("          SKIP=cx-secret-detection git commit -m \"<your message>\"\n\n") +
			white("        Windows CMD:\n") +
			hiBlue("          set SKIP=cx-secret-detection && git commit -m \"<your message>\"\n\n") +
			white("        PowerShell:\n") +
			hiBlue("          $env:SKIP=\"cx-secret-detection\"\n") +
			hiBlue("          git commit -m \"<your message>\"\n") +
			"\n\n"
		assert.Equal(t, fmt.Sprintf("%q", expectedOutput), fmt.Sprintf("%q", string(output)))

		// Run the command to ignore all detected secrets.
		cmdPreCommitIgnore := exec.Command("cx", "hooks", "pre-commit", "secrets-ignore", "--all")
		cmdPreCommitIgnore.Dir = tmpDir
		err = cmdPreCommitIgnore.Run()
		assert.NoError(t, err)

		// Verify that secret detection now passes.
		cmdGitCommit = exec.Command("git", "commit", "-m", "secrets")
		cmdGitCommit.Dir = tmpDir
		output, err = cmdGitCommit.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Cx Secret Detection......................................................Passed")
	})
	t.Run("add files with secrets and commit then ignore all and commit again with global config", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Install the global pre-commit hook.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook", "--global")
		_, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		defer func() {
			cmdUnset := exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
			if output, err := cmdUnset.CombinedOutput(); err != nil {
				t.Logf("failed to unset global hooks path: %s: %s", err, string(output))
			}
		}()

		// Initialize a Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Set dummy user.email.
		cmdGitConfigEmail := exec.Command("git", "config", "user.email", "dummy@example.com")
		cmdGitConfigEmail.Dir = tmpDir
		if output, err := cmdGitConfigEmail.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy email: %s: %s", err, string(output))
		}

		// Set dummy user.name.
		cmdGitConfigName := exec.Command("git", "config", "user.name", "dummy Name")
		cmdGitConfigName.Dir = tmpDir
		if output, err := cmdGitConfigName.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy name: %s: %s", err, string(output))
		}

		// Create file1.txt with provided content.
		file1Content := `MOCK CONTENT 1
MOCK CONTENT 2
MOCK CONTENT 3
MOCK CONTENT 4

MOCK CONTENT TEST    -----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJAXWRPQyGlEY+SXz8Uslhe+MLjTgWd8lf/nA0hgCm9JFKC1tq1S73c
Q9naClNXsMqY7pwPt1bSY8jYRqHHbdoUvwIDAQABAkAfJkz1pCwtfkig8iZSEf2j
VUWBiYgUA9vizdJlsAZBLceLrdk8RZF2YOYCWHrpUtZVea37dzZJe99Dr53K0UZx
AiEAtyHQBGoCVHfzPM//a+4tv2ba3tx9at+3uzGR86YNMzcCIQCCjWHcLW/+sQTW
OXeXRrtxqHPp28ir8AVYuNX0nT1+uQIgJm158PMtufvRlpkux78a6mby1oD98Ecx
jp5AOhhF/NECICyHsQN69CJ5mt6/R01wMOt5u9/eubn76rbyhPgk0h7xAiEAjn6m
EmLwkIYD9VnZfp9+2UoWSh0qZiTIHyNwFpJH78o=
-----END RSA PRIVATE KEY-----

MOCK CONTENT 5
MOCK CONTENT 6
MOCK CONTENT 7
MOCK CONTENT 8`
		file1Path := filepath.Join(tmpDir, "file1.txt")
		err = os.WriteFile(file1Path, []byte(file1Content), 0644)
		assert.NoError(t, err)

		// Create file2.txt with provided content.
		file2Content := `SECRETS WITH DIFFERENT VALUE
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJtb2NrU3ViMiIsIm5hbWUiOiJtb2NrTmFtZTIifQ.dummysignature mongodb+srv://radar:mytoken@io.dbb.mongodb.net/?retryWrites=true&w=majority

SECRETS WITH THE SAME VALUE
ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD`
		file2Path := filepath.Join(tmpDir, "file2.txt")
		err = os.WriteFile(file2Path, []byte(file2Content), 0644)
		assert.NoError(t, err)

		// Stage the new files.
		cmdGitAdd := exec.Command("git", "add", "file1.txt", "file2.txt")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage files with secrets: %s: %s", err, string(output))
		}

		cmdGitCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdGitCommit.Dir = tmpDir
		output, err := cmdGitCommit.CombinedOutput()
		assert.Error(t, err)

		color.NoColor = false
		red := color.New(color.FgRed).SprintFunc()
		white := color.New(color.FgWhite).SprintFunc()
		hiYellow := color.New(color.FgHiYellow).SprintFunc()
		green := color.New(color.FgGreen).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		hiBlue := color.New(color.FgHiBlue).SprintFunc()

		expectedOutput := white("Commit scanned for secrets:\n\n") +
			white("Detected ") + red("5 secrets ") + white("in ") + red("2 files\n\n") +
			white("#1 File: ") + hiYellow("file1.txt\n") +
			red("1 ") + white("Secret detected in file\n") +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("private-key\n") +
			white("\tResult ID: ") + hiYellow("6690f3400f0b2445c71b44f9fa89308e35f7c822\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 6\n") +
			"\t           4 | MOCK CONTENT 4\n" +
			"\t           5 | \n" +
			hiYellow("\t           6 |") + " MOCK CONTENT TEST    " + red("----****** *** ******* ********") + "\n" +
			hiYellow("\t           7 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t           8 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t           9 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          10 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          11 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          12 |") + " " + red("****************************************************************") + "\n" +
			hiYellow("\t          13 |") + " " + red("****************************************") + "\n" +
			hiYellow("\t          14 |") + " " + red("******** *** ******* ********") + "\n" +
			"\t          15 | \n" +
			"\t          16 | MOCK CONTENT 5\n" +
			white("") + "\n" +
			white("#2 File: ") + hiYellow("file2.txt\n") +
			red("4 ") + white("Secrets detected in file\n") +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("jwt\n") +
			white("\tResult ID: ") + hiYellow("15db03900061beb413097e4e689b15d36efbb99b\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 2\n") +
			"\t           1 | SECRETS WITH DIFFERENT VALUE\n" +
			hiYellow("\t           2 |") + " " + red("eyJh**************************************************************************************************") + " mongodb+srv://rada*********@io.dbb.mongodb.net/?retryWrites=true&w=majority" + "\n" +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("authenticated-url\n") +
			white("\tResult ID: ") + hiYellow("f4dc5a5fd23a6308314898cb0af125e6423c213e\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 2\n") +
			"\t           1 | SECRETS WITH DIFFERENT VALUE\n" +
			hiYellow("\t           2 |") + " eyJh************************************************************************************************** mongodb+srv://" + red("rada*********") + "@io.dbb.mongodb.net/?retryWrites=true&w=majority" + "\n" +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("github-pat\n") +
			white("\tResult ID: ") + hiYellow("3b20d93240ebbe2566fd39a2fd7b456784502658\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 5\n") +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			hiYellow("\t           5 |") + " " + red("ghp_************************************") + " ghp_************************************" + "\n" +
			"\t           6 | \n" +
			white("") + "\n" +
			white("\tSecret detected: ") + hiYellow("github-pat\n") +
			white("\tResult ID: ") + hiYellow("3b20d93240ebbe2566fd39a2fd7b456784502658\n") +
			white("\tRisk Score: ") + hiYellow("8.2\n") +
			white("\tLocation: ") + hiYellow("Line 5\n") +
			"\t           3 | \n" +
			"\t           4 | SECRETS WITH THE SAME VALUE\n" +
			hiYellow("\t           5 |") + " ghp_************************************" + " " + red("ghp_************************************") + "\n" +
			"\t           6 | \n" +
			white("") + "\n" +
			white("Options for proceeding with the commit:\n\n") +
			white("  - Remediate detected secrets using the following workflow (") +
			green("recommended") +
			white("):\n") +
			white("      1. Remove detected secrets from files and store them securely. Options:\n") +
			white("         - Use environmental variables\n") +
			white("         - Use a secret management service\n") +
			white("         - Use a configuration management tool\n") +
			white("         - Encrypt files containing secrets (least secure method)\n") +
			white("      2. Commit fixed code.\n\n") +
			white("  - Ignore detected secrets (") +
			yellow("not recommended") +
			white("):\n") +
			white("      Use one of the following commands:\n") +
			hiBlue("          cx hooks pre-commit secrets-ignore --all\n") +
			hiBlue("          cx hooks pre-commit secrets-ignore --resultIds=id1,id2\n\n") +
			white("  - Bypass the pre-commit secret detection scanner (") +
			red("not recommended") +
			white("):\n") +
			white("      Use one of the following commands based on your OS:\n\n") +
			white("        Bash/Zsh:\n") +
			hiBlue("          SKIP=cx-secret-detection git commit -m \"<your message>\"\n\n") +
			white("        Windows CMD:\n") +
			hiBlue("          set SKIP=cx-secret-detection && git commit -m \"<your message>\"\n\n") +
			white("        PowerShell:\n") +
			hiBlue("          $env:SKIP=\"cx-secret-detection\"\n") +
			hiBlue("          git commit -m \"<your message>\"\n")
		assert.Equal(t, fmt.Sprintf("%q", expectedOutput), fmt.Sprintf("%q", string(output)))

		// Run the command to ignore all detected secrets.
		cmdPreCommitIgnore := exec.Command("cx", "hooks", "pre-commit", "secrets-ignore", "--all")
		cmdPreCommitIgnore.Dir = tmpDir
		err = cmdPreCommitIgnore.Run()
		assert.NoError(t, err)

		// Verify that secret detection now passes.
		cmdGitCommit = exec.Command("git", "commit", "-m", "secrets")
		cmdGitCommit.Dir = tmpDir
		output, err = cmdGitCommit.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "secrets\n 2 files changed, 24 insertions(+)\n create mode 100644 file1.txt\n create mode 100644 file2.txt")
	})
	t.Run("add secrets over the max displayed results limit and commit", func(t *testing.T) {
		tmpDir, cleanup := setupTmpDir(t)
		defer cleanup()

		// Initialize a Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Set dummy user.email.
		cmdGitConfigEmail := exec.Command("git", "config", "user.email", "dummy@example.com")
		cmdGitConfigEmail.Dir = tmpDir
		if output, err := cmdGitConfigEmail.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy email: %s: %s", err, string(output))
		}

		// Set dummy user.name.
		cmdGitConfigName := exec.Command("git", "config", "user.name", "dummy Name")
		cmdGitConfigName.Dir = tmpDir
		if output, err := cmdGitConfigName.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to config git dummy name: %s: %s", err, string(output))
		}

		// Install the secrets git hook.
		cmdPreCommitInstall := exec.Command("cx", "hooks", "pre-commit", "secrets-install-git-hook")
		cmdPreCommitInstall.Dir = tmpDir
		_, err := cmdPreCommitInstall.CombinedOutput()
		assert.NoError(t, err, "pre-commit install should not fail in a git repo")

		// Stage the config file.
		cmdGitAdd := exec.Command("git", "add", ".pre-commit-config.yaml")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage config file: %s: %s", err, string(output))
		}

		// Create file.txt with a lot of secrets
		fileContent := ""
		for i := 0; i < 500; i++ {
			fileContent += "ghp_DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD\n"
		}
		filePath := filepath.Join(tmpDir, "file.txt")
		err = os.WriteFile(filePath, []byte(fileContent), 0644)
		assert.NoError(t, err)

		// Stage the new file
		cmdGitAdd = exec.Command("git", "add", "file.txt")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to stage files with secrets: %s: %s", err, string(output))
		}

		cmdGitCommit := exec.Command("git", "commit", "-m", "secrets")
		cmdGitCommit.Dir = tmpDir
		output, err := cmdGitCommit.CombinedOutput()
		assert.Error(t, err)

		color.NoColor = false
		red := color.New(color.FgRed).SprintFunc()
		white := color.New(color.FgWhite).SprintFunc()
		expectedTotalHeader := white("Detected ") + red("500 secrets ") + white("in ") + red("1 file\n\n")
		assert.Contains(t, string(output), expectedTotalHeader)
		assert.Contains(t, string(output), "Presenting first 100 results")
		assert.Equal(t, 100, strings.Count(string(output), "90ee853fb7bf125b6a42c7f4c8d8fcd7f9e8cfb5"))
	})
}

func setupTmpDir(t *testing.T) (tmpDir string, cleanup func()) {
	origWD, err := os.Getwd()
	assert.NoError(t, err)

	// Create a temporary directory for the test.
	tmpDir = t.TempDir()

	// Change working directory to the temporary directory.
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	// Return a cleanup function to restore the original working directory.
	cleanup = func() {
		assert.NoError(t, os.Chdir(origWD))
	}
	return
}
