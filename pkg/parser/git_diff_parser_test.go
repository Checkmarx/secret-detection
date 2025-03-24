package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDiffStream(t *testing.T) {
	tests := []struct {
		name              string
		diff              string
		expectedFileDiffs map[string][]Hunk
	}{
		{
			name:              "Empty diff",
			diff:              "",
			expectedFileDiffs: map[string][]Hunk{},
		},
		{
			name: "rename file without changes",
			diff: `diff --git a/oldname.txt b/newname.txt
similarity index 100%
rename from oldname.txt
rename to newname.txt`,
			expectedFileDiffs: map[string][]Hunk{
				"newname.txt": nil,
			},
		},
		{
			name: "new file without content",
			diff: `diff --git a/newfile.txt b/newfile.txt
new file mode 100644
index 0000000..abc1234`,
			expectedFileDiffs: map[string][]Hunk{
				"newfile.txt": nil,
			},
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
			expectedFileDiffs: map[string][]Hunk{
				"newfile.txt": {
					{
						StartLine: 1,
						Content:   "line1\nline2\n",
						Size:      2,
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
+++ b/dev/null
@@ -1,2 +0,0 @@
-line1
-line2`,
			expectedFileDiffs: map[string][]Hunk{
				"file.txt": {
					{
						StartLine: 0,
						Content:   "",
						Size:      0,
					},
				},
			},
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
			expectedFileDiffs: map[string][]Hunk{
				"newname.txt": {
					{
						StartLine: 1,
						Content:   "lineB-modified\n",
						Size:      3,
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
 context4`,
			expectedFileDiffs: map[string][]Hunk{
				"file1.txt": {
					{
						StartLine: 8,
						Content:   "line1\nline2\n",
						Size:      2,
					},
					{
						StartLine: 16,
						Content:   "line3\nline4\n",
						Size:      2,
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
 context2`,
			expectedFileDiffs: map[string][]Hunk{
				"file_😃.txt": {
					{
						StartLine: 8,
						Content:   "line1\nline2\n",
						Size:      2,
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
+++ b/dev/null
@@ -1,2 +0,0 @@
-line1
-line2`,
			expectedFileDiffs: map[string][]Hunk{
				"file1.txt": {
					{
						StartLine: 8,
						Content:   "line1\nline2\n",
						Size:      2,
					},
					{
						StartLine: 16,
						Content:   "line3\nline4\n",
						Size:      2,
					},
				},
				"newname.txt": {
					{
						StartLine: 1,
						Content:   "lineB-modified\n",
						Size:      3,
					},
				},
				"file.txt": {
					{
						StartLine: 0,
						Content:   "",
						Size:      0,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewDiffParser()
			err := parser.ParseDiffStream(strings.NewReader(tc.diff))
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedFileDiffs, parser.FileDiffs)
		})
	}
}
