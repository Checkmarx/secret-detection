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

//go:embed testdata/fixtures/report_sample.json
var sampleFiles embed.FS

//go:embed testdata/fixtures/skip_*.json
var expectedSkipFiles embed.FS

func TestLogJSONReport(t *testing.T) {
	// Read our sample JSON fixture
	expBytes, err := fs.ReadFile(sampleFiles, "testdata/fixtures/report_sample.json")
	assert.NoError(t, err, "reading embedded sample JSON")

	// Create a temp directory
	dir := t.TempDir()

	// Call the function under test
	err = logJSONReport(dir, expBytes)
	assert.NoError(t, err, "logJSONReport should not error")

	// There should be exactly one file named report_*.json
	pattern := filepath.Join(dir, "report_*.json")
	files, err := filepath.Glob(pattern)
	assert.NoError(t, err, "glob should not error")
	assert.Len(t, files, 1, "expected exactly one JSON report file")

	// Read its contents
	gotBytes, err := os.ReadFile(files[0])
	assert.NoError(t, err, "reading generated JSON file")

	// Compare byte-for-byte to the fixture
	assert.Equal(t, expBytes, gotBytes, "written JSON must exactly match fixture")
}

func TestLogSkip(t *testing.T) {
	cases := []struct {
		name        string
		envKey      string
		envValue    string
		fixtureFile string
	}{
		{"GitHub", envGitHubUserLogin, "githubuser", "skip_github.json"},
		{"GitLab", envGitLabUsername, "gitlabuser", "skip_gitlab.json"},
		{"Bitbucket", envBitbucketUserName, "bbuser", "skip_bitbucket.json"},
		{"Unknown", "", "", "skip_unknown.json"},
	}

	refs := []string{"old1 new1 ref1", "old2 new2 ref2"}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear all upstream env vars, then set the one we care about
			os.Unsetenv(envGitHubUserLogin)
			os.Unsetenv(envGitLabUsername)
			os.Unsetenv(envBitbucketUserName)
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envValue)
			}
			defer os.Unsetenv(tc.envKey)

			dir := t.TempDir()
			err := logSkip(dir, refs)
			assert.NoError(t, err)

			// There should be exactly one skip_*.json file
			pattern := filepath.Join(dir, "skip_*.json")
			files, err := filepath.Glob(pattern)
			assert.NoError(t, err)
			assert.Len(t, files, 1, "expected one skip JSON file")
			logPath := files[0]

			gotBytes, err := os.ReadFile(logPath)
			assert.NoError(t, err)

			expBytes, err := fs.ReadFile(expectedSkipFiles, "testdata/fixtures/"+tc.fixtureFile)
			assert.NoError(t, err)

			// Compare as JSON
			assert.JSONEq(t,
				strings.ReplaceAll(string(expBytes), "\r", ""),
				strings.ReplaceAll(string(gotBytes), "\r", ""),
				"mismatch for %s", tc.name,
			)
		})
	}
}
