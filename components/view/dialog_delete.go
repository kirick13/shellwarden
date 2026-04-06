package view

import (
	"fmt"

	"charm.land/lipgloss/v2"
	card "github.com/kirick13/shellwarden/components/card"
	keys "github.com/kirick13/shellwarden/components/keys"
)

type DialogDeleteView struct {
	BaseView
	BookmarkName string
	Previous     View
	OnDeleteHook func()
}

func NewDialogDeleteView(bookmarkName string, previous View, onDelete func()) *DialogDeleteView {
	return &DialogDeleteView{
		BaseView:     BaseView{},
		BookmarkName: bookmarkName,
		Previous:     previous,
		OnDeleteHook: onDelete,
	}
}

func (v *DialogDeleteView) Render() string {
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
		titleStyle.Render("Delete bookmark"),
		"",
		bodyStyle.Render(fmt.Sprintf("Delete bookmark \"%s\"?", v.BookmarkName)),
		"",
		keys.RenderKeys([]keys.Keys{
			{Key: "enter", Title: "delete"},
			{Key: "esc", Title: "cancel"},
		}),
	)

	return cardStyle.Render(content)
}

func (v *DialogDeleteView) OnEnter() {
	if v.OnDeleteHook != nil {
		v.OnDeleteHook()
	}
	if v.Display != nil && v.Previous != nil {
		v.Display.SetCurrentView(v.Previous)
	}
}

func (v *DialogDeleteView) OnEsc() {
	if v.Display != nil && v.Previous != nil {
		v.Display.SetCurrentView(v.Previous)
	}
}
