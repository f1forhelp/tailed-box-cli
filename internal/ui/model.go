package ui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type screenMode int

const (
	modeMenu screenMode = iota
	modeForm
	modeResult
	modeQuitConfirm
)

type Selection struct {
	Action Action
	Args   []string
	Quit   bool
}

type CommandResult struct {
	Title   string
	Args    []string
	Stdout  string
	Stderr  string
	Error   string
	Kind    ActionKind
	Stopped bool
}

type model struct {
	theme              theme
	actions            []Action
	cursor             int
	width              int
	height             int
	mode               screenMode
	formAction         Action
	formCursor         int
	formValues         []string
	formError          string
	result             *CommandResult
	sectionOpen        bool
	confirmBack        screenMode
	confirmSectionOpen bool
	chosenAction       Action
	chosen             []string
	quit               bool
}

func Run(stdin io.Reader, stdout io.Writer, result *CommandResult) (Selection, error) {
	initial := newModelWithResult(stdout, DefaultActions(), result)
	program := tea.NewProgram(initial, tea.WithInput(stdin), tea.WithOutput(stdout), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return Selection{}, err
	}
	menu, ok := finalModel.(model)
	if !ok || menu.quit {
		return Selection{Quit: true}, nil
	}
	fmt.Fprintln(stdout)
	return Selection{
		Action: menu.chosenAction,
		Args:   append([]string(nil), menu.chosen...),
	}, nil
}

func newModel(stdout io.Writer, actions []Action) model {
	return newModelWithResult(stdout, actions, nil)
}

func newModelWithResult(stdout io.Writer, actions []Action, result *CommandResult) model {
	clonedActions := cloneActions(actions)
	cursor := 0
	sectionOpen := false
	mode := modeMenu
	if result != nil {
		mode = modeResult
		cursor = actionIndexForResult(clonedActions, result)
		sectionOpen = true
	}
	return model{
		theme:       newTheme(stdout),
		actions:     clonedActions,
		cursor:      cursor,
		mode:        mode,
		result:      cloneCommandResult(result),
		sectionOpen: sectionOpen,
	}
}

func actionIndexForResult(actions []Action, result *CommandResult) int {
	if result == nil {
		return 0
	}
	for i, action := range actions {
		if action.Title == result.Title {
			return i
		}
	}
	for i, action := range actions {
		if argsMatchAction(action, result.Args) {
			return i
		}
	}
	return 0
}

func argsMatchAction(action Action, args []string) bool {
	if len(action.Args) == 0 || len(args) < len(action.Args) {
		return false
	}
	for i, arg := range action.Args {
		if args[i] != arg {
			return false
		}
	}
	return true
}

func cloneCommandResult(result *CommandResult) *CommandResult {
	if result == nil {
		return nil
	}
	cloned := *result
	cloned.Args = append([]string(nil), result.Args...)
	return &cloned
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m.startQuitConfirm(), nil
		}
		if m.mode == modeQuitConfirm {
			return m.updateQuitConfirm(msg)
		}
		if m.mode == modeResult {
			return m.updateResult(msg)
		}
		if m.mode == modeForm {
			return m.updateForm(msg)
		}
		return m.updateMenu(msg)
	}
	return m, nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m.startQuitConfirm(), nil
	case "esc", "backspace", "b", "left", "h":
		if m.sectionOpen {
			m.sectionOpen = false
			return m, nil
		}
		return m.startQuitConfirm(), nil
	case "up", "k":
		if len(m.actions) == 0 {
			return m, nil
		}
		if m.sectionOpen {
			m.moveAction(-1)
		} else {
			m.moveGroup(-1)
		}
	case "down", "j":
		if len(m.actions) == 0 {
			return m, nil
		}
		if m.sectionOpen {
			m.moveAction(1)
		} else {
			m.moveGroup(1)
		}
	case "right", "l":
		if len(m.actions) == 0 {
			return m, nil
		}
		m.sectionOpen = true
	case "enter":
		if len(m.actions) == 0 {
			return m, nil
		}
		if !m.sectionOpen {
			m.sectionOpen = true
			return m, nil
		}
		selected := m.actions[m.cursor]
		if selected.Args == nil {
			m.quit = true
			return m, tea.Quit
		}
		if selected.HasInputs() {
			return m.startForm(selected), nil
		}
		m.chosenAction = selected
		m.chosen = append([]string(nil), selected.Args...)
		return m, tea.Quit
	}
	return m, nil
}

func (m model) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m.startQuitConfirm(), nil
	case "esc", "backspace", "b":
		m.mode = modeMenu
		m.result = nil
		return m, nil
	}
	return m, nil
}

func (m model) updateQuitConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		m.quit = true
		return m, tea.Quit
	case "n", "enter", "esc", "backspace", "b":
		m.mode = m.confirmBack
		if m.mode == modeQuitConfirm {
			m.mode = modeMenu
		}
		m.sectionOpen = m.confirmSectionOpen
		return m, nil
	}
	return m, nil
}

func (m model) startQuitConfirm() model {
	if m.mode != modeQuitConfirm {
		m.confirmBack = m.mode
		m.confirmSectionOpen = m.sectionOpen
	}
	m.mode = modeQuitConfirm
	return m
}

func (m *model) moveAction(delta int) {
	group := selectedGroup(m.actions, m.cursor)
	indexes := actionIndexesForGroup(m.actions, group)
	if len(indexes) == 0 {
		return
	}
	position := 0
	selected := selectedActionIndex(m.actions, m.cursor)
	for i, index := range indexes {
		if index == selected {
			position = i
			break
		}
	}
	position = (position + delta + len(indexes)) % len(indexes)
	m.cursor = indexes[position]
}

func (m *model) moveGroup(delta int) {
	groups := actionGroups(m.actions)
	if len(groups) == 0 {
		return
	}
	active := selectedGroup(m.actions, m.cursor)
	position := 0
	for i, group := range groups {
		if group == active {
			position = i
			break
		}
	}
	position = (position + delta + len(groups)) % len(groups)
	m.cursor = firstActionIndexForGroup(m.actions, groups[position])
}

func (m model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.cancelForm(), nil
	case "tab", "down":
		m.nextField()
		return m, nil
	case "shift+tab", "up":
		m.previousField()
		return m, nil
	case "enter":
		if m.formCursor == len(m.formAction.Inputs) {
			return m.cancelForm(), nil
		}
		if m.formCursor < len(m.formAction.Inputs)-1 {
			m.nextField()
			return m, nil
		}
		return m.submitForm()
	case "backspace", "ctrl+h", "delete":
		m.backspaceField()
		return m, nil
	case "space":
		if m.formCursor >= len(m.formValues) {
			return m, nil
		}
		m.formValues[m.formCursor] += " "
		m.formError = ""
		return m, nil
	}
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		if m.formCursor >= len(m.formValues) {
			return m, nil
		}
		m.formValues[m.formCursor] += string(msg.Runes)
		m.formError = ""
	}
	return m, nil
}

func (m model) startForm(action Action) model {
	m.mode = modeForm
	m.formAction = action
	m.formCursor = 0
	m.formValues = make([]string, len(action.Inputs))
	for i, input := range action.Inputs {
		m.formValues[i] = input.Default
	}
	if action.DefaultCancel && len(action.Inputs) > 0 {
		m.formCursor = len(action.Inputs)
	}
	m.formError = ""
	return m
}

func (m *model) nextField() {
	total := m.formItemCount()
	if total == 0 {
		return
	}
	m.formCursor = (m.formCursor + 1) % total
	m.formError = ""
}

func (m *model) previousField() {
	total := m.formItemCount()
	if total == 0 {
		return
	}
	if m.formCursor == 0 {
		m.formCursor = total - 1
	} else {
		m.formCursor--
	}
	m.formError = ""
}

func (m *model) backspaceField() {
	if len(m.formValues) == 0 || m.formCursor >= len(m.formValues) {
		return
	}
	value := m.formValues[m.formCursor]
	if value == "" {
		return
	}
	runes := []rune(value)
	m.formValues[m.formCursor] = string(runes[:len(runes)-1])
	m.formError = ""
}

func (m model) formItemCount() int {
	if len(m.formAction.Inputs) == 0 {
		return 0
	}
	return len(m.formAction.Inputs) + 1
}

func (m model) cancelForm() model {
	m.mode = modeMenu
	m.sectionOpen = true
	m.formAction = Action{}
	m.formCursor = 0
	m.formValues = nil
	m.formError = ""
	return m
}

func (m model) submitForm() (tea.Model, tea.Cmd) {
	if index, message := m.formAction.ValidateInputs(m.formValues); message != "" {
		m.formCursor = index
		m.formError = message
		return m, nil
	}
	m.chosenAction = m.formAction
	m.chosen = m.formAction.BuildArgs(m.formValues)
	return m, tea.Quit
}

func (m model) View() string {
	return newRenderer(m.theme).Render(viewState{
		actions:     m.actions,
		cursor:      m.cursor,
		width:       m.width,
		height:      m.height,
		mode:        m.mode,
		sectionOpen: m.sectionOpen,
		formAction:  m.formAction,
		formCursor:  m.formCursor,
		formValues:  append([]string(nil), m.formValues...),
		formError:   strings.TrimSpace(m.formError),
		result:      cloneCommandResult(m.result),
	})
}
