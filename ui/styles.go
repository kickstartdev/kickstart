package ui

import "github.com/charmbracelet/lipgloss"

var (
    logoStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#f0883e")).
        Bold(true).
        MarginBottom(1)

    subtitleStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#7d8590")).
        Italic(true).
        MarginBottom(2)

    accentStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#f0883e")).
        Bold(true)

    greenStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#3fb950"))

    dimStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#7d8590"))

    helpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#7d8590"))

    spinnerStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#f0883e"))
    
    headerStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#f0883e")).
        Bold(true).
        BorderStyle(lipgloss.NormalBorder()).
        BorderBottom(true).
        BorderForeground(lipgloss.Color("#30363d")).
        Width(0). // we'll set this dynamically
        Align(lipgloss.Center).
        Padding(1, 0)

    redStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#f85149"))

)