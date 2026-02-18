package ui

import (
	"strings"

	lipgloss "github.com/charmbracelet/lipgloss"
)

func (m *Model) Layout(content string, helpText string) string {
	// header
	header := headerStyle.Width(m.Width).Render("kickstart.sh")

	// help bar
	help := helpStyle.Render(helpText)
	helpBar := lipgloss.PlaceHorizontal(m.Width, lipgloss.Center, help)

	// center content
	headerHeight := lipgloss.Height(header)
	helpHeight := lipgloss.Height(helpBar)
	middleHeight := m.Height - headerHeight - helpHeight
	if middleHeight < 0 {
		middleHeight = 0
	}
	centered := lipgloss.Place(m.Width, middleHeight, lipgloss.Center, lipgloss.Center, content)

	// join and pad to exact terminal height
	output := lipgloss.JoinVertical(lipgloss.Left, header, centered, helpBar)

	// ensure output is exactly m.Height lines to prevent bleed-through
	lines := strings.Split(output, "\n")
	for len(lines) < m.Height {
		lines = append(lines, "")
	}
	lines = lines[:m.Height]

	return strings.Join(lines, "\n")
}
