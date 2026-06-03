package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	title       string
	description string
	args        []string
}

type menuModel struct {
	theme  theme
	items  []menuItem
	cursor int
	width  int
	height int
	chosen []string
	quit   bool
}

func (a *app) runMenu(_ context.Context) ([]string, error) {
	model := newMenuModel(a.theme)
	program := tea.NewProgram(model, tea.WithInput(a.stdin), tea.WithOutput(a.stdout), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}
	menu, ok := finalModel.(menuModel)
	if !ok || menu.quit {
		return nil, nil
	}
	fmt.Fprintln(a.stdout)
	return menu.chosen, nil
}

func newMenuModel(t theme) menuModel {
	return menuModel{
		theme: t,
		items: []menuItem{
			{
				title:       "Status",
				description: "Show role-aware local status for this node.",
				args:        []string{"status"},
			},
			{
				title:       "Agent status",
				description: "Show local agent heartbeat, uptime, and memory usage.",
				args:        []string{"agent", "status"},
			},
			{
				title:       "Initialize as master",
				description: "Create master role metadata, identity, and local state.",
				args:        []string{"init", "--role", "master"},
			},
			{
				title:       "Initialize as worker",
				description: "Create worker role metadata, identity, and local state.",
				args:        []string{"init", "--role", "worker"},
			},
			{
				title:       "Master status",
				description: "Show the current master and trusted cluster nodes.",
				args:        []string{"master", "status"},
			},
			{
				title:       "Worker status",
				description: "Show this worker's local join and mesh readiness.",
				args:        []string{"worker", "status"},
			},
			{
				title:       "Create worker join code",
				description: "Issue a one-time 15 minute enrollment code for a worker.",
				args:        []string{"master", "join-code", "create", "--role", "worker", "--ttl", "15m"},
			},
			{
				title:       "Create master join code",
				description: "Issue a one-time 15 minute enrollment code for another master.",
				args:        []string{"master", "join-code", "create", "--role", "master", "--ttl", "15m"},
			},
			{
				title:       "Recent logs",
				description: "Show recent local Tailedbox JSONL logs.",
				args:        []string{"logs", "--lines", "50"},
			},
			{
				title:       "Version",
				description: "Show build and Go runtime version information.",
				args:        []string{"version"},
			},
			{
				title:       "Help",
				description: "Print the full command reference.",
				args:        []string{"--help"},
			},
			{
				title:       "Exit",
				description: "Close the Tailedbox menu.",
				args:        nil,
			},
		},
	}
}

func (m menuModel) Init() tea.Cmd {
	return nil
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if len(m.items) == 0 {
				return m, nil
			}
			if m.cursor == 0 {
				m.cursor = len(m.items) - 1
			} else {
				m.cursor--
			}
		case "down", "j":
			if len(m.items) == 0 {
				return m, nil
			}
			m.cursor = (m.cursor + 1) % len(m.items)
		case "enter":
			if len(m.items) == 0 {
				return m, nil
			}
			selected := m.items[m.cursor]
			if selected.args == nil {
				m.quit = true
				return m, tea.Quit
			}
			m.chosen = append([]string(nil), selected.args...)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	return newMenuRenderer(m.theme).Render(menuViewState{
		items:  m.items,
		cursor: m.cursor,
		width:  m.width,
		height: m.height,
	})
}
