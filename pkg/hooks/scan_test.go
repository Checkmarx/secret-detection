package hooks

import (
	twoms "github.com/checkmarx/2ms/pkg"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseGitDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected []twoms.ScanItem
	}{
		{
			name:     "Empty diff",
			diff:     "",
			expected: nil,
		},
		{
			name: "rename file without changes",
			diff: `diff --git a/oldname.txt b/newname.txt
similarity index 100%
rename from oldname.txt
rename to newname.txt`,
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-newname.txt",
					Source:  "newname.txt",
					Content: strPtr(""),
				},
			},
		},
		{
			name: "new file without content",
			diff: `diff --git a/newfile.txt b/newfile.txt
new file mode 100644
index 0000000..abc1234`,
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-newfile.txt",
					Source:  "newfile.txt",
					Content: strPtr(""),
				},
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
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-newfile.txt",
					Source:  "newfile.txt",
					Content: strPtr("line1\nline2\n"),
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
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-file.txt",
					Source:  "file.txt",
					Content: strPtr(""),
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
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-newname.txt",
					Source:  "newname.txt",
					Content: strPtr("lineB-modified\n"),
				},
			},
		},
		{
			name: "single file changes",
			diff: `diff --git a/file1.txt b/file1.txt
index abc1234..0000000 111111
--- a/file1.txt
+++ b/file1.txt
@@ -0,0 +1,2 @@
+line1
+line2`,
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-file1.txt",
					Source:  "file1.txt",
					Content: strPtr("line1\nline2\n"),
				},
			},
		},
		{
			name: "Multiple file changes",
			diff: `diff --git a/file1.txt b/file2.txt
similarity index 100%
rename from file1.txt
rename to file2.txt
diff --git a/file3.txt b/file3.txt
new file mode 100644
index 0000000..abc1234
diff --git a/file4.txt b/file4.txt
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/file4.txt
@@ -0,0 +1,2 @@
+line1
+line2
diff --git a/file5.txt b/file5.txt
deleted file mode 100644
index abc1234..0000000
--- a/file5.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
diff --git a/file6.txt b/file7.txt
similarity index 80%
rename from file6.txt
rename to file7.txt
index abc1234..0000000 111111
--- a/file6.txt
+++ b/file7.txt
@@ -1,3 +1,3 @@
-lineB
+lineB-modified
diff --git a/file8.txt b/file8.txt
index abc1234..0000000 111111
--- a/file8.txt
+++ b/file8.txt
@@ -0,0 +1,2 @@
+line1
+line2`,
			expected: []twoms.ScanItem{
				{
					ID:      "pre-commit-file2.txt",
					Source:  "file2.txt",
					Content: strPtr(""),
				},
				{
					ID:      "pre-commit-file3.txt",
					Source:  "file3.txt",
					Content: strPtr(""),
				},
				{
					ID:      "pre-commit-file4.txt",
					Source:  "file4.txt",
					Content: strPtr("line1\nline2\n"),
				},
				{
					ID:      "pre-commit-file5.txt",
					Source:  "file5.txt",
					Content: strPtr(""),
				},
				{
					ID:      "pre-commit-file7.txt",
					Source:  "file7.txt",
					Content: strPtr("lineB-modified\n"),
				},
				{
					ID:      "pre-commit-file8.txt",
					Source:  "file8.txt",
					Content: strPtr("line1\nline2\n"),
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseGitDiff(tc.diff)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
