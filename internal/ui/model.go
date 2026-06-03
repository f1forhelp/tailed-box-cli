package ui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	theme   theme
	actions []Action
	cursor  int
	width   int
	height  int
	chosen  []string
	quit    bool
}

func Run(stdin io.Reader, stdout io.Writer) ([]string, error) {
	initial := newModel(stdout, DefaultActions())
	program := tea.NewProgram(initial, tea.WithInput(stdin), tea.WithOutput(stdout), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}
	menu, ok := finalModel.(model)
	if !ok || menu.quit {
		return nil, nil
	}
	fmt.Fprintln(stdout)
	return append([]string(nil), menu.chosen...), nil
}

func newModel(stdout io.Writer, actions []Action) model {
	return model{
		theme:   newTheme(stdout),
		actions: cloneActions(actions),
	}
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
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if len(m.actions) == 0 {
				return m, nil
			}
			if m.cursor == 0 {
				m.cursor = len(m.actions) - 1
			} else {
				m.cursor--
			}
		case "down", "j":
			if len(m.actions) == 0 {
				return m, nil
			}
			m.cursor = (m.cursor + 1) % len(m.actions)
		case "enter":
			if len(m.actions) == 0 {
				return m, nil
			}
			selected := m.actions[m.cursor]
			if selected.Args == nil {
				m.quit = true
				return m, tea.Quit
			}
			m.chosen = append([]string(nil), selected.Args...)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return newRenderer(m.theme).Render(viewState{
		actions: m.actions,
		cursor:  m.cursor,
		width:   m.width,
		height:  m.height,
	})
}
