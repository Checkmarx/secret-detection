package pre_receive

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigExcludesToGitExcludes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single filename", "a.txt", []string{":(exclude)a.txt"}},
		{"single path", "dir/a.txt", []string{":(exclude)dir/a.txt"}},
		{"windows backslash", `"\folder\file.txt"`, []string{":(exclude)folder/file.txt"}},
		{"leading slash", "/root.txt", []string{":(exclude)root.txt"}},
		{"multiple patterns", "a.txt, dir/b.log", []string{":(exclude)a.txt", ":(exclude)dir/b.log"}},
		{"trims spaces", " a.txt ,dir/c.md ", []string{":(exclude)a.txt", ":(exclude)dir/c.md"}},
		{"filename with space", "my file.txt", []string{":(exclude)my file.txt"}},
		{"path with spaces", "dir name/file name.txt", []string{":(exclude)dir name/file name.txt"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := configExcludesToGitExcludes(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
