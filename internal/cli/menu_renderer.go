package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type menuViewState struct {
	items  []menuItem
	cursor int
	width  int
	height int
}

type menuRenderer struct {
	theme theme
}

func newMenuRenderer(t theme) menuRenderer {
	return menuRenderer{theme: t}
}

func (r menuRenderer) Render(state menuViewState) string {
	screenWidth := state.width
	if screenWidth <= 0 {
		screenWidth = 96
	}
	contentWidth := clamp(screenWidth-6, 68, 110)
	if screenWidth < 74 {
		contentWidth = max(44, screenWidth-4)
	}

	header := r.renderHeader(contentWidth)
	body := r.renderBody(state, contentWidth)
	footer := r.renderFooter(contentWidth)

	layout := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	return lipgloss.NewStyle().
		Width(screenWidth).
		Padding(1, 2).
		Render(layout)
}

func (r menuRenderer) renderHeader(width int) string {
	title := lipgloss.JoinHorizontal(
		lipgloss.Center,
		r.theme.Title("tailedbox"),
		"  ",
		r.theme.Muted("CLI control plane"),
	)
	subtitle := r.theme.Subtitle("Secure VPS control from one lightweight binary")
	return menuBoxStyle(r.theme).
		Width(width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, subtitle))
}

func (r menuRenderer) renderBody(state menuViewState, width int) string {
	if width < 86 {
		return menuBoxStyle(r.theme).
			Width(width).
			Padding(1, 2).
			Render(lipgloss.JoinVertical(lipgloss.Left, r.renderMenu(state, width-4), "", r.renderDetails(state, width-4)))
	}

	menuWidth := 42
	detailWidth := width - menuWidth - 3
	menu := menuBoxStyle(r.theme).
		Width(menuWidth).
		Padding(1, 2).
		Render(r.renderMenu(state, menuWidth-4))
	details := menuBoxStyle(r.theme).
		Width(detailWidth).
		Padding(1, 2).
		Render(r.renderDetails(state, detailWidth-4))

	return lipgloss.JoinHorizontal(lipgloss.Top, menu, "   ", details)
}

func (r menuRenderer) renderMenu(state menuViewState, width int) string {
	var b strings.Builder
	fmt.Fprintln(&b, r.theme.Section("Actions"))
	fmt.Fprintln(&b, r.theme.Muted("Move with up/down or j/k. Press Enter."))
	fmt.Fprintln(&b)

	for i, item := range state.items {
		cursor := " "
		title := truncateMenuText(item.title, width-4)
		if i == state.selectedIndex() {
			cursor = r.theme.Accent(">")
			title = selectedMenuRowStyle(r.theme).
				Width(max(12, width-2)).
				Render(r.theme.Command(title))
		} else {
			title = " " + r.theme.Section(title)
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, title)
	}

	return strings.TrimRight(b.String(), "\n")
}

func (r menuRenderer) renderDetails(state menuViewState, width int) string {
	item, ok := state.selectedItem()
	if !ok {
		return r.theme.Muted("No actions available.")
	}

	commandText := "No command"
	if len(item.args) > 0 {
		commandText = "tailedbox " + strings.Join(item.args, " ")
	}

	var b strings.Builder
	fmt.Fprintln(&b, r.theme.Section("Selected"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Title(item.title))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, wrapMenuLine(item.description, width))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Muted("Command"))
	fmt.Fprintln(&b, menuCommandStyle(r.theme).Width(width).Render(commandText))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Muted("This menu is a launcher. Every action still maps to a normal CLI command."))
	return strings.TrimRight(b.String(), "\n")
}

func (r menuRenderer) renderFooter(width int) string {
	return lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Render(r.theme.Muted("Enter: run selected action   q/esc/ctrl+c: quit"))
}

func (s menuViewState) selectedItem() (menuItem, bool) {
	if len(s.items) == 0 {
		return menuItem{}, false
	}
	return s.items[s.selectedIndex()], true
}

func (s menuViewState) selectedIndex() int {
	if len(s.items) == 0 {
		return 0
	}
	if s.cursor < 0 {
		return 0
	}
	if s.cursor >= len(s.items) {
		return len(s.items) - 1
	}
	return s.cursor
}

func menuBoxStyle(t theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))
}

func selectedMenuRowStyle(t theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Bold(true)
}

func menuCommandStyle(t theme) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)
}

func truncateMenuText(value string, width int) string {
	if width <= 0 || lipgloss.Width(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	runes := []rune(value)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "."
}

func wrapMenuLine(value string, width int) string {
	if width <= 0 {
		return value
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return value
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		next := current + " " + word
		if lipgloss.Width(next) > width {
			lines = append(lines, current)
			current = word
			continue
		}
		current = next
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}

func clamp(value, low, high int) int {
	return min(max(value, low), high)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
