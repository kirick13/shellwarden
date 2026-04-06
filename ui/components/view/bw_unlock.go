package view

import (
	"strings"

	"charm.land/lipgloss/v2"
	card "github.com/kirick13/shellwarden/ui/components/card"
	keys "github.com/kirick13/shellwarden/ui/components/keys"
	input "github.com/kirick13/shellwarden/ui/elements"
)

type BwUnlockView struct {
	BaseView
	errText string
}

func NewBwUnlockView() *BwUnlockView {
	password := input.New("Bitwarden master password")
	password.EchoMode = 1

	v := &BwUnlockView{
		BaseView: BaseView{},
	}
	v.SetInputs([]input.Model{password})
	v.SetFocusedInput(0)

	return v
}

func (v *BwUnlockView) Render() string {
	cardStyle := card.New().Padding(1, 2)
	titleStyle := lipgloss.NewStyle().Bold(true)
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d4d4d4"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))

	lines := []string{
		titleStyle.Render("Unlock Bitwarden"),
		"",
		bodyStyle.Render("Enter your Bitwarden master password."),
		"",
		v.passwordInput().Render(),
		"",
		keys.RenderKeys([]keys.Keys{
			{Key: "enter", Title: "unlock"},
			{Key: "esc", Title: "clear"},
		}),
	}

	if v.errText != "" {
		lines = append(lines, "", errorStyle.Render(v.errText))
	}

	return cardStyle.Render(strings.Join(lines, "\n"))
}

func (v *BwUnlockView) OnEnter() {
	if strings.TrimSpace(v.passwordInput().Value()) == "" {
		v.errText = "Password is required."
	}
}

func (v *BwUnlockView) OnEsc() {
	password := v.passwordInput()
	password.SetValue("")
	v.replacePasswordInput(password)
	v.errText = ""
}

func (v *BwUnlockView) passwordInput() input.Model {
	return v.Inputs()[0]
}

func (v *BwUnlockView) replacePasswordInput(next input.Model) {
	inputs := v.Inputs()
	if len(inputs) == 0 {
		return
	}
	inputs[0] = next
	v.SetInputs(inputs)
}

func (v *BwUnlockView) Password() string {
	return v.passwordInput().Value()
}

func (v *BwUnlockView) SetError(errText string) {
	v.errText = errText
}
