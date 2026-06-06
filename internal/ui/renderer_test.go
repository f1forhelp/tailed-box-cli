package ui

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
		sectionOpen: true,
		width:       96,
	})
	for _, want := range []string{"Sections / System", "Agent status", "tailedbox agent status"} {
		if !strings.Contains(view, want) {
			t.Fatalf("menu renderer output missing %q:\n%s", want, view)
		}
	}
}

func TestRendererShowsSectionsFirst(t *testing.T) {
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		actions: []Action{
			{Title: "Core: status", Args: []string{"status"}},
			{Title: "Core: version", Args: []string{"version"}},
			{Title: "Mesh: status", Args: []string{"mesh", "status"}},
		},
		width: 96,
	})
	for _, want := range []string{"Sections", "1. Core", "2. Mesh"} {
		if !strings.Contains(view, want) {
			t.Fatalf("section renderer output missing %q:\n%s", want, view)
		}
	}
	for _, hidden := range []string{"Core (2)", "Mesh (1)", "tailedbox status", "version"} {
		if strings.Contains(view, hidden) {
			t.Fatalf("section renderer should not show action %q before opening a section:\n%s", hidden, view)
		}
	}
}

func TestRendererWindowsLongActionLists(t *testing.T) {
	actions := make([]Action, 20)
	for i := range actions {
		actions[i] = Action{
			Title:       fmt.Sprintf("Action %02d", i),
			Description: "Test action",
			Args:        []string{"version"},
		}
	}
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		actions:     actions,
		cursor:      12,
		width:       96,
		height:      20,
		sectionOpen: true,
	})
	for _, want := range []string{"13. Action 12", "Showing"} {
		if !strings.Contains(view, want) {
			t.Fatalf("menu renderer output missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Action 00") {
		t.Fatalf("menu renderer did not window long action list:\n%s", view)
	}
}

func TestRendererDoesNotOverflowWideLayout(t *testing.T) {
	actions := DefaultActions()
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		actions: actions,
		cursor:  len(actions) - 2,
		width:   180,
		height:  40,
	})

	assertRenderedWidth(t, view, 180)
}

func TestRendererDoesNotOverflowDefaultLayout(t *testing.T) {
	actions := DefaultActions()
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		actions: actions,
		cursor:  0,
		width:   96,
		height:  30,
	})

	assertRenderedWidth(t, view, 96)
}

func assertRenderedWidth(t *testing.T, view string, width int) {
	t.Helper()

	for i, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line %d width=%d exceeds %d:\n%s", i+1, got, width, line)
		}
	}
}
