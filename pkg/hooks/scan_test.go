package hooks

import (
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseGitDiff(t *testing.T) {
	tests := []struct {
		name              string
		diff              string
		expectedScanItems []twoms.ScanItem
		expectedContext   map[string][]LineContext
	}{
		{
			name:              "Empty diff",
			diff:              "",
			expectedScanItems: nil,
			expectedContext:   map[string][]LineContext{},
		},
		{
			name: "rename file without changes",
			diff: `diff --git a/oldname.txt b/newname.txt
similarity index 100%
rename from oldname.txt
rename to newname.txt`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-newname.txt",
					Source:  "newname.txt",
					Content: strPtr(""),
				},
			},
			expectedContext: map[string][]LineContext{},
		},
		{
			name: "new file without content",
			diff: `diff --git a/newfile.txt b/newfile.txt
new file mode 100644
index 0000000..abc1234`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-newfile.txt",
					Source:  "newfile.txt",
					Content: strPtr(""),
				},
			},
			expectedContext: map[string][]LineContext{},
		},
		{
			name: "new file with content",
			diff: `diff --git a/newfile.txt b/newfile.txt
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/newfile.txt
@@ -0,0 +1,2 @@
+line1
+line2`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-newfile.txt",
					Source:  "newfile.txt",
					Content: strPtr("line1\nline2\n"),
				},
			},
			expectedContext: map[string][]LineContext{
				"newfile.txt": {
					{
						hunkStartLine: 1,
						index:         0,
						context:       strPtr("line1\nline2\n"),
					},
					{
						hunkStartLine: 1,
						index:         1,
						context:       strPtr("line1\nline2\n"),
					},
				},
			},
		},
		{
			name: "delete file",
			diff: `diff --git a/file.txt b/file.txt
deleted file mode 100644
index abc1234..0000000
--- a/file.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-file.txt",
					Source:  "file.txt",
					Content: strPtr(""),
				},
			},
			expectedContext: map[string][]LineContext{},
		},
		{
			name: "rename file with changes",
			diff: `diff --git a/oldname.txt b/newname.txt
similarity index 80%
rename from oldname.txt
rename to newname.txt
index abc1234..0000000 111111
--- a/oldname.txt
+++ b/newname.txt
@@ -1,3 +1,3 @@
-lineB
+lineB-modified`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-newname.txt",
					Source:  "newname.txt",
					Content: strPtr("lineB-modified\n"),
				},
			},
			expectedContext: map[string][]LineContext{
				"newname.txt": {
					{
						hunkStartLine: 1,
						index:         0,
						context:       strPtr("lineB-modified\n"),
					},
				},
			},
		},
		{
			name: "single file changes",
			diff: `diff --git a/file1.txt b/file1.txt
index abc1234..0000000 111111
--- a/file1.txt
+++ b/file1.txt
@@ -0,0 +8,2 @@
 context1
+line1
+line2
 context2
@@ -0,0 +16,2 @@
 context3
+line3
+line4
 context4
`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-file1.txt",
					Source:  "file1.txt",
					Content: strPtr("line1\nline2\nline3\nline4\n"),
				},
			},
			expectedContext: map[string][]LineContext{
				"file1.txt": {
					{
						hunkStartLine: 8,
						index:         1,
						context:       strPtr("context1\nline1\nline2\ncontext2\n"),
					},
					{
						hunkStartLine: 8,
						index:         2,
						context:       strPtr("context1\nline1\nline2\ncontext2\n"),
					},
					{
						hunkStartLine: 16,
						index:         1,
						context:       strPtr("context3\nline3\nline4\ncontext4\n"),
					},
					{
						hunkStartLine: 16,
						index:         2,
						context:       strPtr("context3\nline3\nline4\ncontext4\n"),
					},
				},
			},
		},
		{
			name: "a file with an emoji in its name",
			diff: `diff --git a/file_😃.txt b/file_😃.txt
index abc1234..0000000 111111
--- a/file_😃.txt
+++ b/file_😃.txt
@@ -0,0 +8,2 @@
 context1
+line1
+line2
 context2
`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-file_😃.txt",
					Source:  "file_😃.txt",
					Content: strPtr("line1\nline2\n"),
				},
			},
			expectedContext: map[string][]LineContext{
				"file_😃.txt": {
					{
						hunkStartLine: 8,
						index:         1,
						context:       strPtr("context1\nline1\nline2\ncontext2\n"),
					},
					{
						hunkStartLine: 8,
						index:         2,
						context:       strPtr("context1\nline1\nline2\ncontext2\n"),
					},
				},
			},
		},
		{
			name: "Multiple file changes",
			diff: `diff --git a/file1.txt b/file1.txt
index abc1234..0000000 111111
--- a/file1.txt
+++ b/file1.txt
@@ -0,0 +8,2 @@
 context1
+line1
+line2
 context2
@@ -0,0 +16,2 @@
 context3
+line3
+line4
 context4
diff --git a/oldname.txt b/newname.txt
similarity index 80%
rename from oldname.txt
rename to newname.txt
index abc1234..0000000 111111
--- a/oldname.txt
+++ b/newname.txt
@@ -1,3 +1,3 @@
-lineB
+lineB-modified
diff --git a/file.txt b/file.txt
deleted file mode 100644
index abc1234..0000000
--- a/file.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`,
			expectedScanItems: []twoms.ScanItem{
				{
					ID:      "pre-commit-file1.txt",
					Source:  "file1.txt",
					Content: strPtr("line1\nline2\nline3\nline4\n"),
				},
				{
					ID:      "pre-commit-newname.txt",
					Source:  "newname.txt",
					Content: strPtr("lineB-modified\n"),
				},
				{
					ID:      "pre-commit-file.txt",
					Source:  "file.txt",
					Content: strPtr(""),
				},
			},
			expectedContext: map[string][]LineContext{
				"file1.txt": {
					{
						hunkStartLine: 8,
						index:         1,
						context:       strPtr("context1\nline1\nline2\ncontext2\n"),
					},
					{
						hunkStartLine: 8,
						index:         2,
						context:       strPtr("context1\nline1\nline2\ncontext2\n"),
					},
					{
						hunkStartLine: 16,
						index:         1,
						context:       strPtr("context3\nline3\nline4\ncontext4\n"),
					},
					{
						hunkStartLine: 16,
						index:         2,
						context:       strPtr("context3\nline3\nline4\ncontext4\n"),
					},
				},
				"newname.txt": {
					{
						hunkStartLine: 1,
						index:         0,
						context:       strPtr("lineB-modified\n"),
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actualScanItems, actualContext, err := parseGitDiff(tc.diff)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedScanItems, actualScanItems)
			assert.Equal(t, tc.expectedContext, actualContext)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
