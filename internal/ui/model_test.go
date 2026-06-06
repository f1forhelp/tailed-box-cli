package ui

import (
	"io"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestActionBuildArgsWithInputs(t *testing.T) {
	action := guidedJoinAction()
	got := action.BuildArgs([]string{"tbxjc1.code.secret", "/tmp/master-state"})
	want := []string{"worker", "join", "--code", "tbxjc1.code.secret", "--master-state-dir", "/tmp/master-state"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildArgs() = %#v, want %#v", got, want)
	}
}

func TestDefaultUninstallActionsDefaultToCancel(t *testing.T) {
	for _, title := range []string{
		"System: uninstall local files",
		"System: uninstall service and local files",
		"System: uninstall everything",
	} {
		action, ok := findActionForTest(DefaultActions(), title)
		if !ok {
			t.Fatalf("missing default action %q", title)
		}
		if !action.DefaultCancel {
			t.Fatalf("%q should default guided form focus to no/cancel", title)
		}
	}
}

func TestModelGuidedActionBuildsCLIArgs(t *testing.T) {
	m := newModel(io.Discard, []Action{guidedJoinAction()})

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.sectionOpen || m.mode != modeMenu {
		t.Fatalf("first enter should open the section, mode=%v sectionOpen=%v", m.mode, m.sectionOpen)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeForm {
		t.Fatalf("expected form mode, got %v", m.mode)
	}

	m = updateModelForTest(t, m, keyRunes("tbxjc1.code.secret"))
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModelForTest(t, m, keyRunes("/tmp/master-state"))
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	want := []string{"worker", "join", "--code", "tbxjc1.code.secret", "--master-state-dir", "/tmp/master-state"}
	if !reflect.DeepEqual(m.chosen, want) {
		t.Fatalf("chosen args = %#v, want %#v", m.chosen, want)
	}
}

func TestModelGuidedActionRequiresInput(t *testing.T) {
	m := newModel(io.Discard, []Action{guidedJoinAction()})

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.formCursor != 0 {
		t.Fatalf("expected cursor to return to missing join code, got %d", m.formCursor)
	}
	if !strings.Contains(m.formError, "Join code is required") {
		t.Fatalf("expected required input error, got %q", m.formError)
	}
	if len(m.chosen) != 0 {
		t.Fatalf("expected no chosen args on validation failure, got %#v", m.chosen)
	}
}

func TestModelGuidedActionCanCancelFromVisibleNoOption(t *testing.T) {
	m := newModel(io.Discard, []Action{guidedJoinAction()})

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeForm {
		t.Fatalf("expected form mode, got %v", m.mode)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.formCursor != len(m.formAction.Inputs) {
		t.Fatalf("expected visible no/cancel option to be selected, cursor=%d", m.formCursor)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeMenu || !m.sectionOpen {
		t.Fatalf("cancel should return to opened action section, mode=%v sectionOpen=%v", m.mode, m.sectionOpen)
	}
	if len(m.chosen) != 0 {
		t.Fatalf("cancel should not choose command args, got %#v", m.chosen)
	}
	if m.formAction.Title != "" || len(m.formValues) != 0 {
		t.Fatalf("cancel should clear form state, action=%#v values=%#v", m.formAction, m.formValues)
	}
}

func TestModelGuidedActionCanDefaultToCancel(t *testing.T) {
	action := guidedJoinAction()
	action.DefaultCancel = true
	m := newModel(io.Discard, []Action{action})

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeForm {
		t.Fatalf("expected form mode, got %v", m.mode)
	}
	if m.formCursor != len(m.formAction.Inputs) {
		t.Fatalf("expected no/cancel to be selected by default, cursor=%d", m.formCursor)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeMenu || len(m.chosen) != 0 {
		t.Fatalf("default cancel should return to menu without command, mode=%v chosen=%#v", m.mode, m.chosen)
	}
}

func TestModelNavigatesBySectionAndAction(t *testing.T) {
	m := newModel(io.Discard, []Action{
		{Title: "Core: status", Args: []string{"status"}},
		{Title: "Core: version", Args: []string{"version"}},
		{Title: "Mesh: status", Args: []string{"mesh", "status"}},
		{Title: "Mesh: peers", Args: []string{"mesh", "peers"}},
	})

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := actionGroup(m.actions[m.cursor]); got != "Mesh" {
		t.Fatalf("down should move between sections before one is open, got %q", got)
	}
	if got := actionName(m.actions[m.cursor]); got != "status" {
		t.Fatalf("new section should select first action, got %q", got)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.sectionOpen {
		t.Fatalf("enter should open the selected section")
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := actionName(m.actions[m.cursor]); got != "peers" {
		t.Fatalf("down should move inside opened section, got %q", got)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.sectionOpen {
		t.Fatalf("esc should return from actions to sections")
	}
}

func TestModelResultScreenReturnsToMenu(t *testing.T) {
	m := newModelWithResult(io.Discard, []Action{{Title: "Core: status", Args: []string{"status"}}}, &CommandResult{
		Title:  "Core: status",
		Args:   []string{"status"},
		Stdout: "ok\n",
	})
	if m.mode != modeResult {
		t.Fatalf("expected result mode, got %v", m.mode)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeResult {
		t.Fatalf("enter should not leave result mode, got %v", m.mode)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeMenu {
		t.Fatalf("expected menu mode after esc, got %v", m.mode)
	}
	if m.result != nil {
		t.Fatalf("expected result to be cleared after returning to menu")
	}
	if !m.sectionOpen {
		t.Fatalf("expected result back to keep the originating section open")
	}
}

func TestModelResultScreenReturnsToOriginatingActionSection(t *testing.T) {
	m := newModelWithResult(io.Discard, []Action{
		{Title: "Core: status", Args: []string{"status"}},
		{Title: "Core: version", Args: []string{"version"}},
		{Title: "Mesh: status", Args: []string{"mesh", "status"}},
		{Title: "Mesh: peers", Args: []string{"mesh", "peers"}},
	}, &CommandResult{
		Title:  "Mesh: peers",
		Args:   []string{"mesh", "peers"},
		Stdout: "ok\n",
	})

	if got := actionGroup(m.actions[m.cursor]); got != "Mesh" {
		t.Fatalf("expected result screen to remember Mesh section, got %q", got)
	}
	if got := actionName(m.actions[m.cursor]); got != "peers" {
		t.Fatalf("expected result screen to remember peers action, got %q", got)
	}
	if !m.sectionOpen {
		t.Fatalf("expected result screen to keep originating section open")
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeMenu {
		t.Fatalf("expected menu mode after esc, got %v", m.mode)
	}
	if !m.sectionOpen {
		t.Fatalf("expected esc from result to show originating action list")
	}
	if got := actionGroup(m.actions[m.cursor]); got != "Mesh" {
		t.Fatalf("expected esc from result to show Mesh section, got %q", got)
	}
	if got := actionName(m.actions[m.cursor]); got != "peers" {
		t.Fatalf("expected esc from result to keep peers selected, got %q", got)
	}
}

func TestModelQuitConfirmFromMenu(t *testing.T) {
	m := newModel(io.Discard, []Action{{Title: "Core: status", Args: []string{"status"}}})

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeQuitConfirm {
		t.Fatalf("expected quit confirmation after menu esc, got %v", m.mode)
	}
	if m.quit {
		t.Fatalf("menu esc should not quit immediately")
	}

	m = updateModelForTest(t, m, keyRunes("n"))
	if m.mode != modeMenu || m.quit {
		t.Fatalf("n should return to menu without quitting, mode=%v quit=%v", m.mode, m.quit)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeMenu || m.quit {
		t.Fatalf("enter should choose no/stay by default, mode=%v quit=%v", m.mode, m.quit)
	}

	m = updateModelForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	m = updateModelForTest(t, m, keyRunes("y"))
	if !m.quit {
		t.Fatalf("y should confirm quit")
	}
}

func TestRendererShowsCommandResult(t *testing.T) {
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		mode: modeResult,
		result: &CommandResult{
			Title:  "Core: status",
			Args:   []string{"status"},
			Stdout: "node ready\n",
		},
		width:  96,
		height: 28,
	})
	for _, want := range []string{"Result", "Core: status", "tailedbox status", "node ready"} {
		if !strings.Contains(view, want) {
			t.Fatalf("result output missing %q:\n%s", want, view)
		}
	}
}

func TestRendererShowsQuitConfirm(t *testing.T) {
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		mode:   modeQuitConfirm,
		width:  96,
		height: 28,
	})
	for _, want := range []string{"Quit Tailedbox", "Are you sure?", "y: quit", "enter/n: no, stay"} {
		if !strings.Contains(view, want) {
			t.Fatalf("quit confirmation missing %q:\n%s", want, view)
		}
	}
}

func TestRendererMasksGuidedSecretInputs(t *testing.T) {
	action := guidedJoinAction()
	view := newRenderer(newTheme(io.Discard)).Render(viewState{
		mode:       modeForm,
		formAction: action,
		formValues: []string{"tbxjc1.code.secret", "/tmp/master-state"},
		width:      96,
		height:     28,
	})
	for _, want := range []string{"Guided action", "Worker", "join cluster", "[hidden]", "/tmp/master-state", "No / cancel"} {
		if !strings.Contains(view, want) {
			t.Fatalf("guided form output missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "tbxjc1.code.secret") {
		t.Fatalf("guided form leaked secret input:\n%s", view)
	}
}

func guidedJoinAction() Action {
	return Action{
		Title:       "Worker: join cluster",
		Description: "Join this worker to a master cluster with a one-time code.",
		Args:        []string{"worker", "join"},
		Inputs: []ActionInput{
			{
				Label:    "Join code",
				Flag:     "code",
				Required: true,
				Secret:   true,
			},
			{
				Label:    "Master state dir",
				Flag:     "master-state-dir",
				Required: true,
			},
		},
	}
}

func findActionForTest(actions []Action, title string) (Action, bool) {
	for _, action := range actions {
		if action.Title == title {
			return action, true
		}
	}
	return Action{}, false
}

func updateModelForTest(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()
	next, _ := m.Update(msg)
	value, ok := next.(model)
	if !ok {
		t.Fatalf("expected model, got %T", next)
	}
	return value
}

func keyRunes(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}
