package ui

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type theme struct {
	color bool

	title    lipgloss.Style
	subtitle lipgloss.Style
	section  lipgloss.Style
	command  lipgloss.Style
	accent   lipgloss.Style
	muted    lipgloss.Style
}

func newTheme(w io.Writer) theme {
	renderer := lipgloss.NewRenderer(w)
	term := strings.ToLower(os.Getenv("TERM"))
	color := os.Getenv("NO_COLOR") == "" && term != "" && term != "dumb"

	base := renderer.NewStyle()
	if !color {
		return theme{
			color:    false,
			title:    base.Copy(),
			subtitle: base.Copy(),
			section:  base.Copy(),
			command:  base.Copy(),
			accent:   base.Copy(),
			muted:    base.Copy(),
		}
	}

	return theme{
		color: color,

		title:    base.Copy().Bold(true).Foreground(lipgloss.Color("81")),
		subtitle: base.Copy().Foreground(lipgloss.Color("245")),
		section:  base.Copy().Bold(true).Foreground(lipgloss.Color("229")),
		command:  base.Copy().Bold(true).Foreground(lipgloss.Color("75")),
		accent:   base.Copy().Foreground(lipgloss.Color("86")),
		muted:    base.Copy().Foreground(lipgloss.Color("245")),
	}
}

func (t theme) Title(value string) string {
	return t.render(t.title, value)
}

func (t theme) Subtitle(value string) string {
	return t.render(t.subtitle, value)
}

func (t theme) Section(value string) string {
	return t.render(t.section, value)
}

func (t theme) Command(value string) string {
	return t.render(t.command, value)
}

func (t theme) Accent(value string) string {
	return t.render(t.accent, value)
}

func (t theme) Muted(value string) string {
	return t.render(t.muted, value)
}

func (t theme) render(style lipgloss.Style, value string) string {
	if !t.color || value == "" {
		return value
	}
	return style.Render(value)
}
