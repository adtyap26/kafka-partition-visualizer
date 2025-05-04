package tui

import (
	// Use the full module path for your internal packages
	"github.com/adtyap26/kafka-partition-visualizer/internal/config"
	"github.com/adtyap26/kafka-partition-visualizer/internal/placement"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the TUI model. Required by Bubble Tea.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Potentially update layout constraints here if needed

	case tea.KeyMsg:
		switch m.stage {
		// --- Handling Keys in Input Stages ---
		case AskSingleConfig, AskMRCConfig:
			switch msg.Type {
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit

			case tea.KeyEnter:
				// Check if focused on the last input field
				if m.focused == len(m.inputs)-1 {
					// Attempt to parse and validate all inputs
					err := m.parseAndValidateInputs() // This now updates model fields directly
					if err != nil {
						m.err = err // Store error to display in View
					} else {
						// Validation successful, calculate placement
						m.err = nil
						m.stage = ShowPlacement
						// Call placement logic from the placement package
						m.dcs, m.mrcRecommendation = placement.CalculatePlacement(
							config.PlacementConfig{
								m.clusterType,
								m.numPartitions,
								m.replicationFactor,
								m.minInSyncReplicas,
								m.numBrokers, // Pass BrokersPerDC or TotalBrokers based on type
								m.numDCs,
							},
						)
						// No command needed here, view will update based on new stage
					}
				} else {
					// Move focus to the next input field
					m.focused = (m.focused + 1) % len(m.inputs)
					// Update focus styles for all inputs
					for i := range m.inputs {
						if i == m.focused {
							cmds = append(cmds, m.inputs[i].Focus())
							m.inputs[i].PromptStyle = FocusedStyle
							m.inputs[i].TextStyle = FocusedStyle
						} else {
							m.inputs[i].Blur()
							m.inputs[i].PromptStyle = NoStyle
							m.inputs[i].TextStyle = NoStyle
						}
					}
				}
				// Prevent Enter from being processed by the text input itself
				return m, tea.Batch(cmds...)

			// Handle navigation keys (Tab, Shift+Tab, Up, Down)
			case tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown:
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					m.focused--
				} else {
					m.focused++
				}

				// Wrap focus around
				if m.focused >= len(m.inputs) {
					m.focused = 0
				} else if m.focused < 0 {
					m.focused = len(m.inputs) - 1
				}

				// Update focus styles
				for i := range m.inputs {
					if i == m.focused {
						cmds = append(cmds, m.inputs[i].Focus())
						m.inputs[i].PromptStyle = FocusedStyle
						m.inputs[i].TextStyle = FocusedStyle
					} else {
						m.inputs[i].Blur()
						m.inputs[i].PromptStyle = NoStyle
						m.inputs[i].TextStyle = NoStyle
					}
				}
			} // End switch msg.Type for input stages

		// --- Handling Keys in Other Stages ---
		case AskClusterType:
			switch msg.String() { // Use String() for simple key checks
			case "s", "S":
				m.clusterType = config.SingleCluster
				m.stage = AskSingleConfig
				m.setupInputsForStage()                  // Setup inputs for the new stage
				cmds = append(cmds, m.inputs[0].Focus()) // Focus first input
			case "m", "M":
				m.clusterType = config.MRC
				m.stage = AskMRCConfig
				m.setupInputsForStage()                  // Setup inputs for the new stage
				cmds = append(cmds, m.inputs[0].Focus()) // Focus first input
			case "ctrl+c": // Explicitly handle Ctrl+C here too
				return m, tea.Quit
			}

		case ShowPlacement, ShowError:
			// On Enter, reset to the beginning. On Esc/Ctrl+C, quit.
			switch msg.Type {
			case tea.KeyEnter:
				// Reset the model to its initial state
				return NewModel(), textinput.Blink // Return new model and blink command
			case tea.KeyEsc, tea.KeyCtrlC:
				return m, tea.Quit
			}
		} // End switch m.stage
	} // End switch msg.(type)

	// --- Handle Input Field Updates ---
	// This needs to happen regardless of the key pressed if inputs are active
	if m.stage == AskSingleConfig || m.stage == AskMRCConfig {
		// Only update the focused input field? No, update all to handle blur/focus cmds.
		for i := range m.inputs {
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
