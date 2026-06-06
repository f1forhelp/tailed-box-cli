package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type viewState struct {
	actions     []Action
	cursor      int
	width       int
	height      int
	mode        screenMode
	sectionOpen bool
	formAction  Action
	formCursor  int
	formValues  []string
	formError   string
	result      *CommandResult
}

type renderer struct {
	theme theme
}

func newRenderer(t theme) renderer {
	return renderer{theme: t}
}

func (r renderer) Render(state viewState) string {
	screenWidth := state.width
	if screenWidth <= 0 {
		screenWidth = 96
	}
	contentWidth := clamp(screenWidth-2, 58, 132)
	if screenWidth < 74 {
		contentWidth = max(40, screenWidth-2)
	}

	header := r.renderHeader(contentWidth)
	body := r.renderBody(state, contentWidth)
	footer := r.renderFooter(state, contentWidth)

	layout := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	return renderToWidth(lipgloss.NewStyle().Padding(0, 1), screenWidth, layout)
}

func (r renderer) renderHeader(width int) string {
	headerStyle := boxStyle(r.theme).Padding(0, 1)
	available := styleContentWidth(headerStyle, width)
	title := "tailedbox"
	gap := " "
	meta := "terminal control center  enter open/run  esc back"
	if available <= lipgloss.Width(title)+lipgloss.Width(gap) {
		return renderToWidth(headerStyle, width, r.theme.Title(truncateText(title, available)))
	}

	meta = truncateText(meta, available-lipgloss.Width(title)-lipgloss.Width(gap))
	line := lipgloss.JoinHorizontal(
		lipgloss.Center,
		r.theme.Title(title),
		gap,
		r.theme.Muted(meta),
	)
	return renderToWidth(headerStyle, width, line)
}

func (r renderer) renderBody(state viewState, width int) string {
	if state.mode == modeQuitConfirm {
		return renderBox(r.theme, width, r.renderQuitConfirm(boxContentWidth(width)))
	}
	if state.mode == modeResult {
		return renderBox(r.theme, width, r.renderResult(state, boxContentWidth(width)))
	}
	if state.mode == modeForm {
		return renderBox(r.theme, width, r.renderForm(state, boxContentWidth(width)))
	}

	innerWidth := boxContentWidth(width)
	if !state.sectionOpen {
		return renderBox(r.theme, width, r.renderSections(state, innerWidth))
	}
	return renderBox(r.theme, width, r.renderSectionActions(state, innerWidth))
}

func (r renderer) renderSections(state viewState, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", r.theme.Section("Sections"), r.theme.Label("up/down enter"))

	active := state.activeGroup()
	for i, group := range state.groups() {
		label := numberedLabel(i+1, compactGroupLabel(group))
		label = truncateText(label, width-2)
		cursor := " "
		if group == active {
			cursor = r.theme.Accent(">")
			label = selectedRowStyle(r.theme).
				Width(max(10, width-2)).
				Render(r.theme.Command(label))
		} else {
			label = r.theme.Section(label)
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, label)
	}

	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderSectionActions(state viewState, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", r.theme.Section("Sections / "+state.activeGroup()), r.theme.Label("esc back"))

	indexes, start, end := state.actionWindow()
	for position := start; position < end; position++ {
		i := indexes[position]
		action := state.actions[i]
		cursor := " "
		title := truncateText(numberedLabel(position+1, actionName(action)), width-2)
		if i == state.selectedIndex() {
			cursor = r.theme.Accent(">")
			title = selectedRowStyle(r.theme).
				Width(max(12, width-2)).
				Render(r.theme.Command(title))
		} else {
			title = r.theme.Section(title)
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, title)
	}
	if end < len(indexes) || start > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "%s\n", r.theme.Muted(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(indexes))))
	}

	if action, ok := state.selectedAction(); ok {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, r.theme.Title(actionName(action)))
		fmt.Fprintln(&b, r.theme.Subtitle(wrapLine(action.Description, width)))
		commandText := "No command"
		if len(action.Args) > 0 {
			commandText = "tailedbox " + strings.Join(action.PreviewArgs(nil), " ")
		}
		fmt.Fprintln(&b, r.theme.Label("Command"))
		fmt.Fprintln(&b, renderToWidth(commandStyle(r.theme), width, commandText))
	}

	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderSectionStrip(state viewState, width int) string {
	groups := state.groups()
	if len(groups) == 0 {
		return ""
	}
	active := state.activeGroup()
	var parts []string
	for _, group := range groups {
		label := group
		if group == active {
			label = "[" + group + "]"
		}
		parts = append(parts, label)
	}
	return truncateText("Sections: "+strings.Join(parts, "  "), width)
}

func compactGroupLabel(group string) string {
	switch group {
	case "PostgreSQL":
		return "Postgres"
	default:
		return group
	}
}

func numberedLabel(number int, label string) string {
	return fmt.Sprintf("%d. %s", number, label)
}

func (r renderer) renderMenu(state viewState, width int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", r.theme.Section(state.activeGroup()), r.theme.Muted("up/down"))

	indexes, start, end := state.actionWindow()
	for position := start; position < end; position++ {
		i := indexes[position]
		action := state.actions[i]
		cursor := " "
		title := truncateText(numberedLabel(position+1, actionName(action)), width-2)
		if i == state.selectedIndex() {
			cursor = r.theme.Accent(">")
			title = selectedRowStyle(r.theme).
				Width(max(12, width-2)).
				Render(r.theme.Command(title))
		} else {
			title = r.theme.Section(title)
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, title)
	}
	if end < len(indexes) || start > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "%s\n", r.theme.Muted(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(indexes))))
	}

	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderDetails(state viewState, width int) string {
	action, ok := state.selectedAction()
	if !ok {
		return r.theme.Muted("No actions available.")
	}

	commandText := "No command"
	if len(action.Args) > 0 {
		commandText = "tailedbox " + strings.Join(action.PreviewArgs(nil), " ")
	}

	var b strings.Builder
	fmt.Fprintln(&b, r.theme.Section(actionGroup(action)+" action"))
	fmt.Fprintln(&b, r.theme.Title(actionName(action)))
	fmt.Fprintln(&b, r.theme.Subtitle(wrapLine(action.Description, width)))
	fmt.Fprintln(&b)
	modeText := "Direct command"
	if action.HasInputs() {
		modeText = "Guided form"
	}
	fmt.Fprintf(&b, "%s %s\n", r.theme.Label("Mode:"), modeText)
	fmt.Fprintln(&b, r.theme.Label("Command"))
	fmt.Fprintln(&b, renderToWidth(commandStyle(r.theme), width, commandText))
	if action.HasInputs() {
		fmt.Fprintln(&b, r.theme.Muted("Enter opens a form; it still runs CLI args."))
	} else {
		fmt.Fprintln(&b, r.theme.Muted("Runs normal CLI args."))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderFooter(state viewState, width int) string {
	text := "up/down: sections   enter: open   q/esc: quit dialog"
	if state.sectionOpen {
		text = "up/down: actions   enter: run/open   esc/b: sections   q: quit dialog"
	}
	if state.mode == modeForm {
		text = "Enter: next/run/no   tab/up/down: fields   esc: back   ctrl+c: quit"
	} else if state.mode == modeResult {
		text = "esc/b: back to menu   q: quit dialog"
		if state.sectionOpen {
			text = "esc/b: back to actions   q: quit dialog"
		}
	} else if state.mode == modeQuitConfirm {
		text = "y: quit   enter/n: no, stay   esc/b: back"
	}
	footerStyle := lipgloss.NewStyle().Padding(0, 1)
	text = truncateText(text, styleContentWidth(footerStyle, width))
	return renderToWidth(footerStyle, width, r.theme.Muted(text))
}

func (r renderer) renderForm(state viewState, width int) string {
	action := state.formAction
	var b strings.Builder
	fmt.Fprintln(&b, r.theme.Section("Guided action"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Label(actionGroup(action)))
	fmt.Fprintln(&b, r.theme.Title(actionName(action)))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Subtitle(wrapLine(action.Description, width)))
	fmt.Fprintln(&b)

	for i, input := range action.Inputs {
		cursor := " "
		label := input.Label
		if input.Required {
			label += " *"
		}
		if i == state.formCursor {
			cursor = r.theme.Accent(">")
			label = r.theme.Command(label)
		} else {
			label = r.theme.Section(label)
		}
		fmt.Fprintf(&b, "%s %s\n", cursor, label)
		if input.Description != "" {
			fmt.Fprintln(&b, "  "+r.theme.Muted(wrapLine(input.Description, max(8, width-2))))
		}
		value := displayInputValue(state.formValues, i, input)
		fmt.Fprintln(&b, "  "+renderToWidth(fieldStyle(r.theme), max(12, width-2), value))
		fmt.Fprintln(&b)
	}

	cancelCursor := " "
	cancelLabel := "No / cancel"
	if state.formCursor == len(action.Inputs) {
		cancelCursor = r.theme.Accent(">")
		cancelLabel = selectedRowStyle(r.theme).
			Width(max(12, width-2)).
			Render(r.theme.Command(cancelLabel))
	} else {
		cancelLabel = r.theme.Section(cancelLabel)
	}
	fmt.Fprintf(&b, "%s %s\n", cancelCursor, cancelLabel)
	fmt.Fprintln(&b, "  "+r.theme.Muted("Go back without running this command."))
	fmt.Fprintln(&b)

	if state.formError != "" {
		fmt.Fprintln(&b, r.theme.Danger(state.formError))
		fmt.Fprintln(&b)
	}

	commandText := "tailedbox " + strings.Join(action.PreviewArgs(state.formValues), " ")
	fmt.Fprintln(&b, r.theme.Label("Command preview"))
	fmt.Fprintln(&b, renderToWidth(commandStyle(r.theme), width, commandText))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Muted("Enter: next field, run, or no/cancel   tab/up/down: move fields   esc: back"))
	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderQuitConfirm(width int) string {
	var b strings.Builder
	fmt.Fprintln(&b, r.theme.Section("Quit Tailedbox"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Warning("Are you sure?"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Subtitle(wrapLine("You are on the main control flow. Quit only if you are done managing this node.", width)))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, renderToWidth(commandStyle(r.theme), width, "y: quit   enter/n: no, stay   esc/b: back"))
	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderResult(state viewState, width int) string {
	result := state.result
	if result == nil {
		return r.theme.Muted("No command result yet.")
	}
	status := "Completed"
	if result.Error != "" {
		status = "Failed"
	} else if result.Stopped {
		status = "Stopped"
	}

	var b strings.Builder
	fmt.Fprintln(&b, r.theme.Section("Result"))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Title(result.Title))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, r.theme.Label("Status"))
	fmt.Fprintln(&b, r.renderStatus(status))
	fmt.Fprintln(&b)
	if len(result.Args) > 0 {
		fmt.Fprintln(&b, r.theme.Label("Command"))
		fmt.Fprintln(&b, renderToWidth(commandStyle(r.theme), width, "tailedbox "+strings.Join(result.Args, " ")))
		fmt.Fprintln(&b)
	}
	if result.Error != "" {
		fmt.Fprintln(&b, r.theme.Danger("Error"))
		fmt.Fprintln(&b, r.theme.Danger(result.Error))
		fmt.Fprintln(&b)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		fmt.Fprintln(&b, r.theme.Label("Output"))
		fmt.Fprintln(&b, outputBlock(width, result.Stdout))
		fmt.Fprintln(&b)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		fmt.Fprintln(&b, r.theme.Warning("Diagnostics"))
		fmt.Fprintln(&b, outputBlock(width, result.Stderr))
		fmt.Fprintln(&b)
	}
	if strings.TrimSpace(result.Stdout) == "" && strings.TrimSpace(result.Stderr) == "" && result.Error == "" {
		fmt.Fprintln(&b, r.theme.Muted("No output was produced."))
		fmt.Fprintln(&b)
	}
	fmt.Fprintln(&b, r.theme.Muted("Press Esc or b to return to the control center."))
	return strings.TrimRight(b.String(), "\n")
}

func (r renderer) renderStatus(status string) string {
	switch status {
	case "Completed":
		return r.theme.Success(status)
	case "Stopped":
		return r.theme.Warning(status)
	case "Failed":
		return r.theme.Danger(status)
	default:
		return status
	}
}

func displayInputValue(values []string, index int, input ActionInput) string {
	value := ""
	if index < len(values) {
		value = values[index]
	}
	if value == "" {
		return "(empty)"
	}
	if input.Secret {
		return strings.Repeat("*", len([]rune(value)))
	}
	return value
}

func outputBlock(width int, value string) string {
	value = strings.TrimRight(value, "\n")
	if value == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) > 18 {
		lines = append([]string{fmt.Sprintf("... showing last %d lines ...", 18)}, lines[len(lines)-18:]...)
	}
	for i, line := range lines {
		lines[i] = truncateText(line, width)
	}
	return strings.Join(lines, "\n")
}

func renderBox(t theme, totalWidth int, content string) string {
	return renderToWidth(boxStyle(t).Padding(0, 1), totalWidth, content)
}

func renderToWidth(style lipgloss.Style, totalWidth int, content string) string {
	return style.Width(styleContentWidth(style, totalWidth)).Render(content)
}

func styleContentWidth(style lipgloss.Style, totalWidth int) int {
	return max(1, totalWidth-style.GetHorizontalFrameSize())
}

func boxContentWidth(totalWidth int) int {
	return styleContentWidth(boxStyle(theme{}).Padding(0, 1), totalWidth)
}

func (s viewState) selectedAction() (Action, bool) {
	if len(s.actions) == 0 {
		return Action{}, false
	}
	return s.actions[s.selectedIndex()], true
}

func (s viewState) selectedIndex() int {
	return selectedActionIndex(s.actions, s.cursor)
}

func (s viewState) activeGroup() string {
	return selectedGroup(s.actions, s.cursor)
}

func (s viewState) groups() []string {
	return actionGroups(s.actions)
}

func (s viewState) actionWindow() ([]int, int, int) {
	indexes := actionIndexesForGroup(s.actions, s.activeGroup())
	total := len(indexes)
	if total == 0 {
		return nil, 0, 0
	}
	visible := s.visibleActionRows()
	if visible >= total {
		return indexes, 0, total
	}
	selected := s.selectedIndex()
	position := 0
	for i, index := range indexes {
		if index == selected {
			position = i
			break
		}
	}
	start := position - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return indexes, start, start + visible
}

func (s viewState) visibleActionRows() int {
	if s.height <= 0 {
		return len(s.actions)
	}
	rows := s.height - 14
	if rows < 8 {
		return min(8, len(s.actions))
	}
	return min(rows, len(s.actions))
}

func boxStyle(t theme) lipgloss.Style {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())
	if !t.color {
		return style
	}
	return style.
		BorderForeground(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))
}

func selectedRowStyle(t theme) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)
	if !t.color {
		return style
	}
	return style.
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235"))
}

func commandStyle(t theme) lipgloss.Style {
	style := lipgloss.NewStyle().Padding(0, 1)
	if !t.color {
		return style
	}
	return style.
		Foreground(lipgloss.Color("86")).
		Background(lipgloss.Color("235"))
}

func fieldStyle(t theme) lipgloss.Style {
	style := lipgloss.NewStyle().Padding(0, 1)
	if !t.color {
		return style
	}
	return style.
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235"))
}

func truncateText(value string, width int) string {
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

func wrapLine(value string, width int) string {
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
