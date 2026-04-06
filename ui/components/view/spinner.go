package view

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	card "github.com/kirick13/shellwarden/ui/components/card"
)

type SpinnerView struct {
	BaseView
	spinner spinner.Model
	text    string
}

func NewSpinnerView(text string) *SpinnerView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa"))

	return &SpinnerView{
		BaseView: BaseView{},
		spinner:  s,
		text:     text,
	}
}

func (v *SpinnerView) Render() string {
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d4d4d4"))

	content := []string{
		v.spinner.View() + " " + bodyStyle.Render(v.text),
	}

	return card.New().Padding(1, 2).Render(strings.Join(content, "\n"))
}

func (v *SpinnerView) OnMsg(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.spinner, cmd = v.spinner.Update(msg)
	return cmd
}

func (v *SpinnerView) Init() tea.Cmd {
	return v.spinner.Tick
}
