package input

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/rivo/uniseg"
)

var (
	focusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#171717")).
		Background(lipgloss.Color("#f5f5f5"))
	blurredStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#171717")).
		Background(lipgloss.Color("#a3a3a3"))
	placeholderColor = lipgloss.Color("#525252")
)

type Model struct {
	textinput.Model
}

func New(placeholder string) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetVirtualCursor(false)
	ti.SetWidth(45)

	s := ti.Styles()
	s.Cursor.Color = lipgloss.Color("#2563eb")
	s.Cursor.Blink = false
	s.Focused.Text = focusedStyle
	s.Focused.Placeholder = focusedStyle.Foreground(placeholderColor)
	s.Blurred.Text = blurredStyle
	s.Blurred.Placeholder = blurredStyle.Foreground(placeholderColor)
	ti.SetStyles(s)

	return Model{Model: ti}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Model.Update(msg)
	m.Model = next
	return m, cmd
}

func (m Model) Render() string {
	if m.Focused() {
		return m.View()
	}

	styles := m.Styles()
	contentStyle := styles.Blurred.Placeholder
	content := m.Placeholder

	if value := m.Value(); value != "" {
		contentStyle = styles.Blurred.Text
		content = value
	}

	content = trimToWidth(content, m.Width() + 1)
	content = contentStyle.Width(m.Width() + 1).MaxWidth(m.Width() + 1).Render(content)

	return content
}

func trimToWidth(s string, width int) string {
	if width <= 0 || uniseg.StringWidth(s) <= width {
		return s
	}

	runes := []rune(s)
	for len(runes) > 0 && uniseg.StringWidth(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}

	return string(runes)
}
