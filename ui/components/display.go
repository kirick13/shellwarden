package components

import (
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/charmbracelet/x/exp/charmtone"
	view "github.com/kirick13/shellwarden/ui/components/view"
)

// hasDarkBG = lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
// lightDark = lipgloss.LightDark(hasDarkBG)

type Display struct {
	CurrentView view.View
	width       int
	height      int
}

var physicalWidth, physicalHeight, _ = term.GetSize(os.Stdout.Fd())

var bgStyle = lipgloss.NewStyle().
		Foreground(charmtone.Iron).
		Height(physicalWidth).
		Width(physicalHeight).
		Align(lipgloss.Center)
var bgString = makeBg()

func NewDisplay() *Display {
	d := &Display{
		width:  physicalWidth,
		height: physicalHeight,
	}

	d.SetCurrentView(view.NewSpinnerView("starting Shellwarden..."))
	return d
}

func (d *Display) SetCurrentView(current view.View) {
	d.CurrentView = current
	if d.CurrentView != nil {
		d.CurrentView.SetDisplay(d)
		inputs := d.CurrentView.Inputs()
		if len(inputs) > 0 {
			focused := clamp(d.CurrentView.FocusedInput(), 0, len(inputs)-1)
			for i := range inputs {
				inputs[i].Blur()
			}
			inputs[focused].Focus()
			d.CurrentView.SetFocusedInput(focused)
		}
	}
}

func (d *Display) MoveFocus(delta int) {
	if d == nil || d.CurrentView == nil {
		return
	}

	inputs := d.CurrentView.Inputs()
	if len(inputs) == 0 {
		return
	}

	for i := range inputs {
		inputs[i].Blur()
	}

	focused := d.CurrentView.FocusedInput() + delta
	if focused < 0 {
		focused = len(inputs) - 1
	}
	if focused >= len(inputs) {
		focused = 0
	}

	inputs[focused].Focus()
	d.CurrentView.SetFocusedInput(focused)
}

func (d *Display) InnerSize() (int, int) {
	return max(d.width, 0), max(d.height, 0)
}

func (d *Display) Render() string {
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(makeBg()),
	}

	if d != nil && d.CurrentView != nil {
		contentWidth, contentHeight := d.InnerSize()
		viewContent := lipgloss.NewStyle().
			MaxWidth(contentWidth).
			MaxHeight(contentHeight).
			Render(d.CurrentView.Render())
		viewWidth := min(lipgloss.Width(viewContent), contentWidth)
		viewHeight := min(lipgloss.Height(viewContent), contentHeight)
		x := max((contentWidth-viewWidth)/2, 0)
		y := max((contentHeight-viewHeight)/2, 0)
		layers = append(layers, lipgloss.NewLayer(viewContent).X(x).Y(y))
	}

	return lipgloss.NewCompositor(layers...).Render()
}

func (d *Display) UpdateDocumentSize(width, height int) {
	physicalWidth = width
	physicalHeight = height

	if d != nil {
		d.width = width
		d.height = height
	}

	bgStyle = bgStyle.
		Height(physicalHeight).
		Width(physicalWidth)

	bgString = makeBg()
}

func makeBg() string {
	b := strings.Builder{}
	for i := range physicalHeight {
		for range physicalWidth {
			b.WriteString("/")
		}
		if i < physicalHeight - 1 {
			b.WriteString("\n")
		}
	}

	return bgStyle.Render(b.String())
}

func clamp(value, minValue, maxValue int) int {
	return min(max(value, minValue), maxValue)
}
