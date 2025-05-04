package tui

import "github.com/charmbracelet/lipgloss"

// Package tui contains all the Bubble Tea related code for the
// terminal user interface, including the model, update logic, view rendering,
// input handling, and styling.

// --- Styles ---
// Define lipgloss styles for the TUI elements. Exported so they can be used
// by the view logic (potentially in view.go).

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	FocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	BlurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	CursorStyle  = FocusedStyle.Copy()
	NoStyle      = lipgloss.NewStyle()

	HelpStyle = BlurredStyle.Copy()

	// Replica Colors
	leaderColor   = lipgloss.Color("#00FF00") // Green
	followerColor = lipgloss.Color("#FFFF00") // Yellow
	observerColor = lipgloss.Color("#FF0000") // Red

	LeaderStyle   = lipgloss.NewStyle().Foreground(leaderColor)
	FollowerStyle = lipgloss.NewStyle().Foreground(followerColor)
	ObserverStyle = lipgloss.NewStyle().Foreground(observerColor)

	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")) // Red for errors

	DCHeaderStyle  = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	BrokerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")). // Purple border
			Padding(0, 1).
			MarginRight(2).
			MarginBottom(1)
)
