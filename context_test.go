package tailedboxcli_test

import (
	"os"
	"strings"
	"testing"
)

func TestContextExistsAndTracksMilestone(t *testing.T) {
	data, err := os.ReadFile("context.md")
	if err != nil {
		t.Fatalf("read context.md: %v", err)
	}
	content := string(data)
	for _, want := range []string{"# Project Context", "## Goal", "## Boundaries", "## Current Status", "## Next Action"} {
		if !strings.Contains(content, want) {
			t.Fatalf("context.md missing %q", want)
		}
	}
}
