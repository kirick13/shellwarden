package view

import (
	"charm.land/lipgloss/v2"
	card "github.com/kirick13/shellwarden/ui/components/card"
	keys "github.com/kirick13/shellwarden/ui/components/keys"
)

type DialogServerRestartView struct {
	BaseView
	Previous      View
	OnRestartHook func()
	OnKillHook    func()
}

func NewDialogServerRestartView(previous View, onRestart func(), onKill func()) *DialogServerRestartView {
	return &DialogServerRestartView{
		BaseView:      BaseView{},
		Previous:      previous,
		OnRestartHook: onRestart,
		OnKillHook:    onKill,
	}
}

func (v *DialogServerRestartView) Render() string {
	maxWidth := 50
	if v.Display != nil {
		displayWidth, _ := v.Display.InnerSize()
		maxWidth = max(displayWidth-10, 20)
	}

	cardStyle := card.New().
		MaxWidth(maxWidth).
		Padding(1, 2)
	titleStyle := lipgloss.NewStyle().Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d4d4d4"))

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Server actions"),
		"",
		bodyStyle.Render("What would you like to do with the Shellwarden server?"),
		"",
		keys.RenderKeys([]keys.Keys{
			{Key: "r", Title: "restart"},
			{Key: "k", Title: "kill"},
			{Key: "esc", Title: "cancel"},
		}),
	)

	return cardStyle.Render(content)
}

func (v *DialogServerRestartView) OnKey(key string) {
	switch key {
	case "r":
		if v.OnRestartHook != nil {
			v.OnRestartHook()
		}
	case "k":
		if v.OnKillHook != nil {
			v.OnKillHook()
		}
	}
}

func (v *DialogServerRestartView) OnEsc() {
	if v.Display != nil && v.Previous != nil {
		v.Display.SetCurrentView(v.Previous)
	}
}
