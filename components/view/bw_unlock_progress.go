package view

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	card "github.com/kirick13/shellwarden/components/card"
)

type BwUnlockProgressView struct {
	BaseView
	spinner spinner.Model
}

func NewBwUnlockProgressView() *BwUnlockProgressView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa"))

	return &BwUnlockProgressView{
		BaseView: BaseView{},
		spinner:  s,
	}
}

func (v *BwUnlockProgressView) Render() string {
	cardStyle := card.New().Padding(1, 2)
	titleStyle := lipgloss.NewStyle().Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d4d4d4"))

	content := []string{
		titleStyle.Render("Unlock Bitwarden"),
		"",
		v.spinner.View() + " " + bodyStyle.Render("Unlocking Bitwarden"),
	}

	return cardStyle.Render(strings.Join(content, "\n"))
}

func (v *BwUnlockProgressView) OnMsg(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.spinner, cmd = v.spinner.Update(msg)
	return cmd
}

func (v *BwUnlockProgressView) Init() tea.Cmd {
	return v.spinner.Tick
}
