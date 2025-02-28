package integration

import (
	"fmt"
	"github.com/Checkmarx/secret-detection/pkg/config"
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
		_, destBinary, cleanup := setupCxBinary(t)
		defer cleanup()

		// Run the pre-commit install and expect failure because it's not a git repo.
		cmdPreCommitInstall := exec.Command(destBinary, "pre-commit", "install")
		output, err := cmdPreCommitInstall.CombinedOutput()
		assert.Error(t, err, "should fail because it is not a git repo")
		assert.Contains(t, string(output), "current directory is not a Git repository")
	})
	t.Run("Git repository with no config file", func(t *testing.T) {
		tmpDir, destBinary, cleanup := setupCxBinary(t)
		defer cleanup()

		// Initialize Git repository.
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		output, err := cmdGitInit.CombinedOutput()
		assert.NoError(t, err, "git init should not fail")
		t.Logf("Git init output: %s", string(output))

		// Run the pre-commit install and expect success.
		cmdPreCommitInstall := exec.Command(destBinary, "pre-commit", "install")
		cmdPreCommitInstall.Dir = tmpDir
		output, err = cmdPreCommitInstall.CombinedOutput()
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
		assert.Contains(t, string(output), "Installing pre-commit hooks...")
		assert.Contains(t, string(output), "pre-commit installed at .git")
	})
	t.Run("Git repository with config file", func(t *testing.T) {
		tmpDir, destBinary, cleanup := setupCxBinary(t)
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
							Args:                    []string{"pre-commit", "scan"},
							Language:                "system",
							PassFilenames:           true,
							MinimumPreCommitVersion: "3.2.0",
						},
					},
				},
			},
		}

		cmdPreCommitInstall := exec.Command(destBinary, "pre-commit", "install")
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
		assert.Contains(t, string(output), "Installing pre-commit hooks...")
		assert.Contains(t, string(output), "pre-commit installed at .git")
	})
}

func TestPreCommitUninstall(t *testing.T) {
	t.Run("Remove cx-secret-detection hook from config", func(t *testing.T) {
		tmpDir, destBinary, cleanup := setupCxBinary(t)
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

		cmdPreCommitUninstall := exec.Command(destBinary, "pre-commit", "uninstall")
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
}

func TestPreCommitUpdate(t *testing.T) {
	t.Run("Update cx-secret-detection hook from config", func(t *testing.T) {
		tmpDir, destBinary, cleanup := setupCxBinary(t)
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

		cmdPreCommitUpdate := exec.Command(destBinary, "pre-commit", "update")
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

		assert.Contains(t, string(output), "Updating cx-secret-detection hook")
		assert.Contains(t, string(output), "cx-secret-detection hook updated successfully")
	})
}

func TestPreCommitIgnore(t *testing.T) {
	t.Run("Ignore resultIds", func(t *testing.T) {
		tmpDir, destBinary, cleanup := setupCxBinary(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		mockSha1, mockSha2, mockSha3 := "sha1", "sha2", "sha3"
		cmdPreCommitIgnore := exec.Command(destBinary, "pre-commit", "ignore", "--resultIds", fmt.Sprintf("%s,%s,%s", mockSha1, mockSha2, mockSha3))
		cmdPreCommitIgnore.Dir = tmpDir
		output, err := cmdPreCommitIgnore.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Added 3 new IDs to .checkmarx_ignore.txt")

		ignorePath := filepath.Join(tmpDir, ".checkmarx_ignore.txt")

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
		cmdPreCommitIgnore = exec.Command(destBinary, "pre-commit", "ignore", "--resultIds", fmt.Sprintf("%s,%s", mockSha4, mockSha5))
		cmdPreCommitIgnore.Dir = tmpDir
		output, err = cmdPreCommitIgnore.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Added 2 new IDs to .checkmarx_ignore.txt")

		// Read from file and verify expected shas
		data, err = os.ReadFile(ignorePath)
		assert.NoError(t, err)
		lines = strings.Split(strings.TrimSpace(string(data)), "\n")
		expectedShas = []string{mockSha1, mockSha2, mockSha3, mockSha4, mockSha5}
		assert.Equal(t, expectedShas, lines, "ignore file does not contain all expected shas")
	})
	t.Run("Ignore all results", func(t *testing.T) {
		tmpDir, destBinary, cleanup := setupCxBinary(t)
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

		cmdPreCommitIgnore := exec.Command(destBinary, "pre-commit", "ignore", "--all")
		cmdPreCommitIgnore.Dir = tmpDir
		output, err := cmdPreCommitIgnore.CombinedOutput()
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Added 5 new IDs to .checkmarx_ignore.txt")

		ignorePath := filepath.Join(tmpDir, ".checkmarx_ignore.txt")
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
	t.Run("add files and commit", func(t *testing.T) {
		tmpDir, _, cleanup := setupCxBinary(t)
		defer cleanup()

		// Initialize a Git repository
		cmdGitInit := exec.Command("git", "init")
		cmdGitInit.Dir = tmpDir
		if output, err := cmdGitInit.CombinedOutput(); err != nil {
			t.Fatalf("failed to initialize git repository: %s: %s", err, string(output))
		}

		// Create file without secrets
		file1Path := filepath.Join(tmpDir, "no-secrets.txt")
		err := os.WriteFile(file1Path, []byte("dummy content"), 0644)
		assert.NoError(t, err)

		// Stage the new file
		cmdGitAdd := exec.Command("git", "add", "no-secrets.txt")
		cmdGitAdd.Dir = tmpDir
		if output, err := cmdGitAdd.CombinedOutput(); err != nil {
			t.Fatalf("failed to git add files: %s: %s", err, string(output))
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

		// commit changes, should not fail because no secrets were added
		cmdGitCommit := exec.Command("git", "commit", "-m", "no secrets")
		cmdGitCommit.Dir = tmpDir
		if output, err := cmdGitCommit.CombinedOutput(); err != nil {
			t.Fatalf("should not fail to commit when no secrets are added: %s: %s", err, string(output))
		}

		// TODO add secrets to a file and commit, should fail
		// TODO compare expected with actual report
		// TODO ignore all results and commit again, should pass
	})
}

func setupCxBinary(t *testing.T) (tmpDir, destBinary string, cleanup func()) {
	origWD, err := os.Getwd()
	assert.NoError(t, err)

	// Create a temporary directory for the test.
	tmpDir = t.TempDir()

	// Define the original binary location and the destination path.
	origBinary := filepath.Join("/app/integration/bin", "cx")
	destBinary = filepath.Join(tmpDir, "cx")

	// Read the binary from the original location.
	input, err := os.ReadFile(origBinary)
	assert.NoError(t, err)

	// Write the binary to the temporary directory with executable permissions.
	err = os.WriteFile(destBinary, input, 0755)
	assert.NoError(t, err)

	// Change working directory to the temporary directory.
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	// Return a cleanup function to restore the original working directory.
	cleanup = func() {
		assert.NoError(t, os.Chdir(origWD))
	}
	return
}
