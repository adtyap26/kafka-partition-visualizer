package main

import (
	"log"
	"os"

	// Use the full module path for internal packages
	"github.com/adtyap26/kafka-partition-visualizer/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create the initial TUI model
	m := tui.NewModel()

	// Create and run the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen()) // Use AltScreen for cleaner exit
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
		os.Exit(1)
	}
}
