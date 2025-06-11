package pre_receive

import (
	"embed"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/fixsures/skip_*.log
var expectedFiles embed.FS

func TestLogSkip(t *testing.T) {
	cases := []struct {
		name        string
		envKey      string
		envValue    string
		fixturePath string
	}{
		{"GitHub", envGitHubUserLogin, "githubuser", "testdata/fixtures/skip_github.log"},
		{"GitLab", envGitLabUsername, "gitlabuser", "testdata/fixtures/skip_gitlab.log"},
		{"Bitbucket", envBitbucketUserName, "bbuser", "testdata/fixtures/skip_bitbucket.log"},
		{"Unknown", "", "", "testdata/fixtures/skip_unknown.log"},
	}

	refs := []string{"old1 new1 ref1", "old2 new2 ref2"}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(envGitHubUserLogin)
			os.Unsetenv(envGitLabUsername)
			os.Unsetenv(envBitbucketUserName)
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envValue)
			}
			defer os.Unsetenv(tc.envKey)

			// Temporary directory for log output
			dir := t.TempDir()

			// Run logSkip
			err := logSkip(dir, refs)
			assert.NoError(t, err, "logSkip should not return an error")

			// Verify one log file created
			pattern := filepath.Join(dir, "skip_*.log")
			files, err := filepath.Glob(pattern)
			assert.NoError(t, err, "glob should not error")
			assert.Len(t, files, 1, "expected exactly one skip log file")
			logPath := files[0]

			// Read generated content
			gotBytes, err := os.ReadFile(logPath)
			assert.NoError(t, err, "reading generated log should not error")

			// Read expected fixture
			expBytes, err := fs.ReadFile(expectedFiles, tc.fixturePath)
			assert.NoError(t, err, "reading fixture file should not error")

			// Normalize line endings to LF for comparison
			got := strings.ReplaceAll(string(gotBytes), "\r\n", "\n")
			exp := strings.ReplaceAll(string(expBytes), "\r\n", "\n")

			// Compare content
			assert.Equal(t, exp, got, "content mismatch for %s", tc.name)
		})
	}
}
