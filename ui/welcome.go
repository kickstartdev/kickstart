package ui

import "github.com/charmbracelet/lipgloss"


var logo = `
 ██╗  ██╗██╗ ██████╗██╗  ██╗███████╗████████╗ █████╗ ██████╗ ████████╗   ███████╗██╗  ██╗
 ██║ ██╔╝██║██╔════╝██║ ██╔╝██╔════╝╚══██╔══╝██╔══██╗██╔══██╗╚══██╔══╝   ██╔════╝██║  ██║
 █████╔╝ ██║██║     █████╔╝ ███████╗   ██║   ███████║██████╔╝   ██║      ███████╗███████║
 ██╔═██╗ ██║██║     ██╔═██╗ ╚════██║   ██║   ██╔══██║██╔══██╗   ██║      ╚════██║██╔══██║
 ██║  ██╗██║╚██████╗██║  ██╗███████║   ██║   ██║  ██║██║  ██║   ██║   ██╗███████║██║  ██║
 ╚═╝  ╚═╝╚═╝ ╚═════╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝╚══════╝╚═╝  ╚═╝`


 func (m *Model) ViewWelcome() string {
	s := logoStyle.Render(logo) + "\n"
	s += subtitleStyle.Render("scaffold projects from templates, fast.") + "\n\n"
	s += "  Press " + accentStyle.Render("enter") + " to get started\n"
	s += helpStyle.Render("  press q to quit")
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, s)
}