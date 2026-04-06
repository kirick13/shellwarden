package view

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"github.com/kirick13/shellwarden/bw"
	"github.com/kirick13/shellwarden/components/card"
	keys "github.com/kirick13/shellwarden/components/keys"
)

type HostsView struct {
	BaseView
	Rows    []bw.Host
	Table   table.Model
	Reorder bool
}

var (
	keysBrowse = keys.RenderKeys([]keys.Keys{
		{Key: "r", Title: "reload"},
		{Key: "enter", Title: "connect"},
	})
	keysReorder = keys.RenderKeys([]keys.Keys{
		{Key: "esc", Title: "exit reorder"},
	})
)

func NewHostsView(rows []bw.Host) *HostsView {
	v := &HostsView{
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

func NewHostsViewWithSelection(rows []bw.Host, selectedID string) *HostsView {
	v := NewHostsView(rows)
	v.SelectHostByID(selectedID)
	return v
}

func (v *HostsView) OnKey(key string) {
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
	// case "home":
	// 	v.Table.GotoTop()
	// case "end":
	// 	v.Table.GotoBottom()
	case "backspace":
		v.openDeleteDialog()
	}
}

func (v *HostsView) OnEsc() {
	v.Reorder = false
}

func (v *HostsView) Render() string {
	width, height := 60, 12
	if v.Display != nil {
		width, height = v.Display.InnerSize()
	}

	if width <= 0 || height <= 0 {
		return ""
	}

	contentWidth := max(width-6, 1)
	contentHeight := max(height-2, 1)
	v.syncTable(contentWidth, contentHeight)

	mode := "Browse"
	if v.Reorder {
		mode = "Reorder"
	}

	left := lipgloss.NewStyle().Bold(true).Render("Shellwarden") +
		"  " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#a3a3a3")).Render(
			fmt.Sprintf("%s %d/%d", mode, min(v.Table.Cursor()+1, max(len(v.Rows), 1)), len(v.Rows)),
		)

	right := keysBrowse
	if v.Reorder {
		right = keysReorder
	}

	status := left + strings.Repeat(" ", max(contentWidth-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right
	content := trimToWidth(status, contentWidth) + "\n\n" + v.Table.View()

	return card.New().
		Margin(0, 1).
		Padding(0, 1).
		Render(content)
}

func (v *HostsView) moveRow(delta int) {
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

func (v *HostsView) openDeleteDialog() {
	if v.Display == nil || len(v.Rows) == 0 {
		return
	}

	current := clamp(v.Table.Cursor(), 0, len(v.Rows)-1)
	bookmarkName := v.Rows[current].Name
	v.Display.SetCurrentView(NewDialogDeleteView(bookmarkName, v, func() {
		v.deleteRow(current)
	}))
}

func (v *HostsView) deleteRow(index int) {
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

func (v *HostsView) syncTable(width, height int) {
	columnGap := 1
	globeWidth := 3
	usableWidth := max(width-globeWidth-(columnGap*2), 0)
	nameWidth := max((usableWidth*3)/5, 8)
	hostWidth := max(usableWidth-nameWidth, 7)
	nameWidth = max(usableWidth-hostWidth, 8)

	v.Table.SetColumns([]table.Column{
		{Title: "", Width: globeWidth},
		{Title: "Name", Width: nameWidth},
		{Title: "Host", Width: hostWidth},
	})
	v.Table.SetWidth(width)
	v.Table.SetHeight(max(height-2, 1))
	v.syncRows()
}

func (v *HostsView) syncRows() {
	rows := make([]table.Row, len(v.Rows))
	for i, row := range v.Rows {
		rows[i] = table.Row{
			" 🌐",
			row.Name,
			formatHost(row),
		}
	}
	v.Table.SetRows(rows)
}

func (v *HostsView) SelectedHostID() string {
	if len(v.Rows) == 0 {
		return ""
	}

	index := clamp(v.Table.Cursor(), 0, len(v.Rows)-1)
	return v.Rows[index].ID
}

func (v *HostsView) SelectHostByID(hostID string) {
	if hostID == "" || len(v.Rows) == 0 {
		return
	}

	for i, host := range v.Rows {
		if host.ID == hostID {
			v.Table.SetCursor(i)
			return
		}
	}
}

func clamp(value, minValue, maxValue int) int {
	return min(max(value, minValue), maxValue)
}

func formatHost(host bw.Host) string {
	if host.SSHPort == "" {
		return host.IPv4
	}

	return fmt.Sprintf("%s:%s", host.IPv4, host.SSHPort)
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
