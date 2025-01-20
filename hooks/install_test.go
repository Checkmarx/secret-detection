package hooks

import (
	"testing"
)

func TestIgnore(t *testing.T) {
	err := Install()
	if err != nil {
		t.Errorf("Ignore() error = %v, wantErr %v", err, nil)
	}
}
