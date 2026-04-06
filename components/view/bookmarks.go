package view

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	keys "github.com/kirick13/shellwarden/components/keys"
)
type Bookmark struct {
	Name string
	IP   string
}

type BookmarksView struct {
	BaseView
	Rows    []Bookmark
	Table   table.Model
	Reorder bool
}

var (
	keysBrowse = keys.RenderKeys([]keys.Keys{
		{Key: "n", Title: "new"},
		{Key: "e", Title: "edit"},
		{Key: "c", Title: "clone"},
		{Key: "r", Title: "reorder"},
		{Key: "backspace", Title: "delete"},
		{Key: "enter", Title: "connect"},
	})
	keysReorder = keys.RenderKeys([]keys.Keys{
		{Key: "esc", Title: "exit reorder"},
	})
)

func NewBookmarksView() *BookmarksView {
	rows := make([]Bookmark, 100)
	for i := range rows {
		rows[i] = Bookmark{
			Name: randomName(),
			IP:   randomIP(),
		}
	}

	v := &BookmarksView{
		BaseView: BaseView{},
		Rows:     rows,
		Table: table.New(
			table.WithFocused(true),
		),
	}

	v.Table.SetStyles(table.Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			PaddingRight(1),
		Cell: lipgloss.NewStyle().
			PaddingRight(1),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#171717")).
			Background(lipgloss.Color("#f5f5f5")),
	})
	v.syncTable(60, 12)

	return v
}

func (v *BookmarksView) OnKey(key string) {
	switch key {
	case "up":
		if v.Reorder {
			v.moveRow(-1)
			return
		}
		v.Table.MoveUp(1)
	case "down":
		if v.Reorder {
			v.moveRow(1)
			return
		}
		v.Table.MoveDown(1)
	case "r":
		v.Reorder = true
	case "home":
		v.Table.GotoTop()
	case "end":
		v.Table.GotoBottom()
	case "backspace":
		v.openDeleteDialog()
	}
}

func (v *BookmarksView) OnEsc() {
	v.Reorder = false
}

func (v *BookmarksView) Render() string {
	width, height := 60, 12
	if v.Display != nil {
		width, height = v.Display.InnerSize()
	}

	if width <= 0 || height <= 0 {
		return ""
	}

	v.syncTable(width, height)

	mode := "Browse"
	if v.Reorder {
		mode = "Reorder"
	}

	left := lipgloss.NewStyle().Bold(true).Render("Bookmarks") +
		"  " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#a3a3a3")).Render(
			fmt.Sprintf("%s %d/%d", mode, min(v.Table.Cursor()+1, max(len(v.Rows), 1)), len(v.Rows)),
		)

	right := keysBrowse
	if v.Reorder {
		right = keysReorder
	}

	status := left + strings.Repeat(" ", max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right

	return trimToWidth(status, width) + "\n\n" + v.Table.View()
}

func (v *BookmarksView) moveRow(delta int) {
	if len(v.Rows) == 0 {
		return
	}

	current := v.Table.Cursor()
	next := clamp(current+delta, 0, len(v.Rows)-1)
	if next == current {
		return
	}

	v.Rows[current], v.Rows[next] = v.Rows[next], v.Rows[current]
	v.syncRows()
	v.Table.SetCursor(next)
}

func (v *BookmarksView) openDeleteDialog() {
	if v.Display == nil || len(v.Rows) == 0 {
		return
	}

	current := clamp(v.Table.Cursor(), 0, len(v.Rows)-1)
	bookmarkName := v.Rows[current].Name
	v.Display.SetCurrentView(NewDialogDeleteView(bookmarkName, v, func() {
		v.deleteRow(current)
	}))
}

func (v *BookmarksView) deleteRow(index int) {
	if index < 0 || index >= len(v.Rows) {
		return
	}

	v.Rows = append(v.Rows[:index], v.Rows[index+1:]...)
	v.syncRows()

	if len(v.Rows) == 0 {
		v.Table.SetCursor(0)
		return
	}

	v.Table.SetCursor(clamp(index, 0, len(v.Rows)-1))
}

func (v *BookmarksView) syncTable(width, height int) {
	columnGap := 1
	globeWidth := 3
	usableWidth := max(width-globeWidth-(columnGap*2), 0)
	nameWidth := max((usableWidth*3)/5, 8)
	ipWidth := max(usableWidth-nameWidth, 7)
	nameWidth = max(usableWidth-ipWidth, 8)

	v.Table.SetColumns([]table.Column{
		{Title: "", Width: globeWidth},
		{Title: "Name", Width: nameWidth},
		{Title: "IP", Width: ipWidth},
	})
	v.Table.SetWidth(width)
	v.Table.SetHeight(max(height-2, 1))
	v.syncRows()
}

func (v *BookmarksView) syncRows() {
	rows := make([]table.Row, len(v.Rows))
	for i, row := range v.Rows {
		rows[i] = table.Row{
			" 🌐",
			row.Name,
			row.IP,
		}
	}
	v.Table.SetRows(rows)
}

// func (v *BookmarksView) SetDisplay(display Display) {
// 	v.BaseView.SetDisplay(display)
// 	if display == nil {
// 		return
// 	}

// 	width, height := display.InnerSize()
// 	v.syncTable(width, height)
// }

func randomName() string {
	adjectives := []string{
		"blue", "quick", "silent", "north", "solid", "brisk", "calm", "bright",
	}
	nouns := []string{
		"fox", "node", "shell", "gate", "box", "link", "host", "cloud",
	}

	return fmt.Sprintf("%s-%s-%02d", adjectives[rand.IntN(len(adjectives))], nouns[rand.IntN(len(nouns))], rand.IntN(100))
}

func randomIP() string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		rand.IntN(223-10)+10,
		rand.IntN(256),
		rand.IntN(256),
		rand.IntN(254)+1,
	)
}

func clamp(value, minValue, maxValue int) int {
	return min(max(value, minValue), maxValue)
}

func trimToWidth(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}

	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}

	return string(runes)
}
