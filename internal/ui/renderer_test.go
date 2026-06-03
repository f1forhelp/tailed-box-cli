package ui

import (
	"io"
	"strings"
	"testing"
)

func TestRendererShowsSelectedCommand(t *testing.T) {
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		actions: []Action{
			{
				Title:       "Agent status",
				Description: "Show local agent heartbeat, uptime, and memory usage.",
				Args:        []string{"agent", "status"},
			},
		},
		width: 96,
	})
	for _, want := range []string{"Actions", "Selected", "Agent status", "tailedbox agent status"} {
		if !strings.Contains(view, want) {
			t.Fatalf("menu renderer output missing %q:\n%s", want, view)
		}
	}
}
