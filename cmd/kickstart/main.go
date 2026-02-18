package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kickstartdev/kickstart/internal/debug"
	"github.com/kickstartdev/kickstart/ui"
)

func main() {
	debug.Init("debug.log")
	debug.Log("starting kickstart")

	p := tea.NewProgram(ui.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("something went wrong: %v", err)
		os.Exit(1)
	}
}
