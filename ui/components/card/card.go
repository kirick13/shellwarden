package card

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

func New() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#a3a3a3")).
		BorderForegroundBlend(
			charmtone.Cherry,
			charmtone.Charple,
			charmtone.Guac,
			charmtone.Charple,
			charmtone.Sriracha,
		).
		Margin(0, 1)
}
