package report

import (
	"fmt"
	"github.com/checkmarx/2ms/lib/reporting"
	"github.com/checkmarx/2ms/lib/secrets"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"strings"
	"testing"
)

const (
	exceedsMaxDisplayedResultsSource     = "testdata/fixtures/exceeds_max_displayed_results.txt"
	multipleFilesCommitsAndSecretsSource = "testdata/fixtures/multiple_files_commits_and_secrets.txt"
	singleFileCommitAndSecretSource      = "testdata/fixtures/single_file_commit_and_secret.txt"
)

// makeReport generates a reporting.Report containing exactly totalSecrets dummy
// secrets, evenly distributed across numFiles files and numCommits commits.
// Each secret is assigned its own unique resultID and pseudo-random metadata,
// but because we use an RNG with a fixed seed, the entire report is fully
// deterministic. We can safely generate fixtures once and rely on
// makeReport(…) to reproduce the same output every time in the tests.
func makeReport(totalSecrets, numFiles, numCommits int) *reporting.Report {
	rng := rand.New(rand.NewSource(42))

	// pre-generate commit IDs: COMMIT000, COMMIT001, …
	commitIDs := make([]string, numCommits)
	for j := 0; j < numCommits; j++ {
		commitIDs[j] = fmt.Sprintf("COMMIT%03d", j)
	}

	results := make(map[string][]*secrets.Secret, totalSecrets)
	usedIDs := make(map[string]struct{}, totalSecrets)

	for i := 0; i < totalSecrets; i++ {
		// round-robin assign file and commit
		fileName := fmt.Sprintf("file%02d.txt", i%numFiles)
		commitID := commitIDs[i%numCommits]

		// generate a unique random resultID of length 8
		var resultID string
		for {
			resultID = randomValue(rng, 8)
			if _, exists := usedIDs[resultID]; !exists {
				usedIDs[resultID] = struct{}{}
				break
			}
		}

		s := &secrets.Secret{
			ID:        fmt.Sprintf("ID%03d", i),
			Source:    fmt.Sprintf("Added:%s:%s", commitID, fileName),
			RuleID:    fmt.Sprintf("RULE%02d", rng.Intn(10)),
			StartLine: rng.Intn(200),
			Value:     randomValue(rng, 12),
		}

		// each resultID gets its own slice (one secret)
		results[resultID] = []*secrets.Secret{s}
	}

	return &reporting.Report{
		TotalItemsScanned: numFiles,
		TotalSecretsFound: totalSecrets,
		Results:           results,
	}
}

// randomValue returns an alphanumeric string of length n, using the provided rng.
// We use this for both resultIDs and secret Values.
func randomValue(rng *rand.Rand, n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func TestGroupSecretsByFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    []*SecretInfo
		wantKeys []string
		wantLens map[string]int
	}{
		{
			name:     "nil input",
			input:    nil,
			wantKeys: []string{},
			wantLens: map[string]int{},
		},
		{
			name:     "empty slice",
			input:    []*SecretInfo{},
			wantKeys: []string{},
			wantLens: map[string]int{},
		},
		{
			name: "group by single file",
			input: []*SecretInfo{
				{secret: &secrets.Secret{ID: "s1"}, source: SourceInfo{fileName: "a.txt"}},
				{secret: &secrets.Secret{ID: "s2"}, source: SourceInfo{fileName: "a.txt"}},
				{secret: &secrets.Secret{ID: "s3"}, source: SourceInfo{fileName: "c.txt"}},
				{secret: &secrets.Secret{ID: "s4"}, source: SourceInfo{fileName: "b.txt"}},
				{secret: &secrets.Secret{ID: "s5"}, source: SourceInfo{fileName: "c.txt"}},
				{secret: &secrets.Secret{ID: "s6"}, source: SourceInfo{fileName: "c.txt"}},
			},
			wantKeys: []string{"a.txt", "b.txt", "c.txt"},
			wantLens: map[string]int{"a.txt": 2, "b.txt": 1, "c.txt": 3},
		},
		{
			name: "empty filename",
			input: []*SecretInfo{
				{secret: &secrets.Secret{ID: "s7"}, source: SourceInfo{fileName: ""}},
				{secret: &secrets.Secret{ID: "s8"}, source: SourceInfo{fileName: ""}},
			},
			wantKeys: []string{""},
			wantLens: map[string]int{"": 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := groupSecretsByFileName(tc.input)

			// number of keys
			assert.Lenf(t, got, len(tc.wantKeys),
				"expected %d keys, got %d", len(tc.wantKeys), len(got),
			)

			// each expected key exists and has correct length
			for _, key := range tc.wantKeys {
				list, found := got[key]
				assert.Truef(t, found, "expected key %q to be present", key)
				assert.Lenf(t, list, tc.wantLens[key],
					"for key %q expected %d items, got %d",
					key, tc.wantLens[key], len(list),
				)
			}

			// no unexpected keys
			for key := range got {
				assert.Containsf(t, tc.wantKeys, key,
					"did not expect key %q", key,
				)
			}

			// order and pointer identity
			for _, si := range tc.input {
				list := got[si.source.fileName]
				assert.Containsf(t, list, si,
					"secret ID=%s should be in group %q",
					si.secret.ID, si.source.fileName,
				)
			}
		})
	}
}

func TestGroupReportResultsByCommitID(t *testing.T) {
	tests := []struct {
		name       string
		report     *reporting.Report
		wantKeys   []string
		wantCounts map[string]int
	}{
		{
			name:       "empty results",
			report:     &reporting.Report{Results: map[string][]*secrets.Secret{}},
			wantKeys:   []string{},
			wantCounts: map[string]int{},
		},
		{
			name: "single valid entry",
			report: &reporting.Report{
				Results: map[string][]*secrets.Secret{
					"any": {
						{
							ID:     "s1",
							Source: "fs:commitA:file1.txt",
							Value:  "mock_password",
						},
					},
				},
			},
			wantKeys:   []string{"commitA"},
			wantCounts: map[string]int{"commitA": 1},
		},
		{
			name: "source with filename containing ´:´",
			report: &reporting.Report{
				Results: map[string][]*secrets.Secret{
					"any": {
						{
							ID:     "s1",
							Source: "fs:commitA:file1:name.txt",
							Value:  "mock_password",
						},
					},
				},
			},
			wantKeys:   []string{"commitA"},
			wantCounts: map[string]int{"commitA": 1},
		},
		{
			name: "multiple commits and skip invalid",
			report: &reporting.Report{
				Results: map[string][]*secrets.Secret{
					"r1": {
						{ID: "s2", Source: "db:commitX:db.conf", Value: "secretX"},
						{ID: "s3", Source: "badformat", Value: "shouldSkip"},
						{ID: "s4", Source: "api:commitY:config.yaml", Value: "tokenY"},
					},
					"r2": {
						{ID: "s5", Source: "fs:commitX:other.txt", Value: "anotherX"},
					},
				},
			},
			wantKeys: []string{"commitX", "commitY"},
			wantCounts: map[string]int{
				"commitX": 2,
				"commitY": 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := groupReportResultsByCommitID(tc.report)

			// check number of distinct commit IDs
			assert.Lenf(t, got, len(tc.wantKeys),
				"expected %d commit IDs, got %d",
				len(tc.wantKeys), len(got),
			)

			for _, commitID := range tc.wantKeys {
				list, found := got[commitID]
				assert.Truef(t, found, "expected commit ID %q to be present", commitID)
				assert.Lenf(t, list, tc.wantCounts[commitID],
					"for commit %q expected %d secrets, got %d",
					commitID, tc.wantCounts[commitID], len(list),
				)
				// verify each SecretInfo
				for _, si := range list {
					// ID is preserved
					assert.NotNil(t, si.secret, "secret pointer should not be nil")
					// Value should be obfuscated
					expectedOb := obfuscateSecret(si.secret.Value)
					assert.Equalf(t, expectedOb, si.secret.Value,
						"secret ID=%s: Value should be obfuscated", si.secret.ID,
					)
					// SourceInfo fields from split
					parts := strings.SplitN(si.secret.Source, ":", 3)
					assert.Equalf(t, parts[0], si.source.contentType,
						"secret ID=%s: contentType mismatch", si.secret.ID,
					)
					assert.Equalf(t, parts[2], si.source.fileName,
						"secret ID=%s: fileName mismatch", si.secret.ID,
					)
				}
			}

			// ensure no unexpected commit IDs
			for commitID := range got {
				assert.Containsf(t, tc.wantKeys, commitID,
					"unexpected commit ID %q", commitID,
				)
			}
		})
	}
}

func TestObfuscateSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short string under max length",
			input:    "abc",
			expected: "abc",
		},
		{
			name:     "string exactly max length",
			input:    "abcd", // length == secretMaxCharacters
			expected: "abcd",
		},
		{
			name:     "string longer than max length",
			input:    "abcdef",
			expected: "abcd***", // truncated to 4 + obfuscatedSecretString
		},
		{
			name: "private key with newline prefix",
			input: `-----BEGIN KEY-----` +
				"\n" +
				"SECRETVALUE",
			// split on "-----": ["", "BEGIN KEY", "\nSECRETVALUE"]
			// take index 2, trim "\n" => "SECRETVALUE", then truncate to "SECR***"
			expected: "SECR***",
		},
		{
			name: "private key with literal \\n prefix",
			input: `-----BEGIN KEY-----` +
				`\n` +
				"LONGSECRET",
			// split gives ["", "BEGIN KEY", `\nLONGSECRET`]
			// trimPrefix `\n` => "LONGSECRET", truncate => "LONG***"
			expected: "LONG***",
		},
		{
			name: "private key small secret",
			input: `-----BEGIN KEY-----` +
				"\n" +
				"AB",
			// after trim => "AB" (length < max), no truncation
			expected: "AB",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := obfuscateSecret(tc.input)
			assert.Equal(t, tc.expected, got, "unexpected obfuscated value")
		})
	}
}

func TestPreReceiveReport(t *testing.T) {
	tests := []struct {
		name         string
		totalSecrets int
		numFiles     int
		numCommits   int
		expectedFile string
	}{
		{
			name:         "single file commit and secret",
			totalSecrets: 1,
			numFiles:     1,
			numCommits:   1,
			expectedFile: singleFileCommitAndSecretSource,
		},
		{
			name:         "multiple files and commits",
			totalSecrets: 8,
			numFiles:     4,
			numCommits:   2,
			expectedFile: multipleFilesCommitsAndSecretsSource,
		},
		{
			name:         "exceeds maxDisplayedResults",
			totalSecrets: 150,
			numFiles:     10,
			numCommits:   4,
			expectedFile: exceedsMaxDisplayedResultsSource,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rpt := makeReport(tc.totalSecrets, tc.numFiles, tc.numCommits)

			wantBytes, err := os.ReadFile(tc.expectedFile)
			assert.NoError(t, err, "reading %s", tc.expectedFile)
			want := strings.ReplaceAll(string(wantBytes), "\r", "")

			got := PreReceiveReport(rpt)

			assert.Equal(t, want, got, "output mismatch for %s", tc.name)
		})
	}
}
