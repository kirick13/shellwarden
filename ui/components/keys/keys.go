package keys

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type Keys struct {
	Key   string
	Title string
}

var (
	keyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#737373"))
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#e5e5e5"))
	separator  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#737373")).Render(" · ")
)

func RenderKeys(keys []Keys) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, keyStyle.Render(key.Key)+" "+titleStyle.Render(key.Title))
	}

	return strings.Join(parts, separator)
}
