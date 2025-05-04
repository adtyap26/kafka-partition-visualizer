package tui

import (
	"fmt"
	"sort"
	"strings"

	// Use the full module path for your internal packages
	"github.com/adtyap26/kafka-partition-visualizer/internal/config"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI based on the current model state. Required by Bubble Tea.
func (m Model) View() string {
	var b strings.Builder

	// --- Title ---
	b.WriteString(TitleStyle.Render("Kafka Partition Visualizer"))
	b.WriteString("\n\n")

	// --- Content Based on Stage ---
	switch m.stage {
	case AskClusterType:
		b.WriteString("Select cluster type:\n\n")
		b.WriteString("[S] Single Cluster\n")
		b.WriteString("[M] Multi-Region Cluster (MRC)\n\n")
		b.WriteString(HelpStyle.Render("(Press S or M. Ctrl+C to quit)"))

	case AskSingleConfig, AskMRCConfig:
		title := "Enter Single Cluster Configuration:"
		var labels []string
		if m.stage == AskMRCConfig {
			title = "Enter MRC Configuration:"
			labels = []string{"Data Centers:", "Brokers per DC:", "Partitions:", "Replication Factor:", "Min ISR:"}
		} else {
			labels = []string{"Total Brokers:", "Partitions:", "Replication Factor:", "Min ISR:"}
		}
		b.WriteString(title + "\n\n")

		// Display labels and input fields
		for i := range m.inputs {
			b.WriteString(labels[i] + "\n")
			b.WriteString(m.inputs[i].View())
			// Add spacing between input fields, but not after the last one
			if i < len(m.inputs)-1 {
				b.WriteString("\n\n") // More spacing
			} else {
				b.WriteRune('\n') // Single newline after the last input
			}
		}

		// Display error if present
		if m.err != nil {
			b.WriteString("\n") // Add space before error
			b.WriteString(ErrorStyle.Render("Error: " + m.err.Error()))
			b.WriteString("\n\n")
		} else {
			b.WriteString("\n\n") // Add spacing even if no error
		}

		b.WriteString(HelpStyle.Render("Use Tab/Shift+Tab or Up/Down to navigate. Enter to confirm/move next. Ctrl+C to quit."))

	case ShowPlacement:
		b.WriteString("Partition Placement Visualization:\n\n")
		if m.clusterType == config.MRC && m.mrcRecommendation != "" {
			b.WriteString(fmt.Sprintf("MRC Recommendation: %s\n\n", m.mrcRecommendation))
		}

		// Sort DC IDs for consistent display order
		dcIDs := make([]int, 0, len(m.dcs))
		for id := range m.dcs {
			dcIDs = append(dcIDs, id)
		}
		sort.Ints(dcIDs)

		var dcViews []string // Store rendered views for each DC

		for _, dcID := range dcIDs {
			dc := m.dcs[dcID]
			var dcBuilder strings.Builder

			// Add DC header only for MRC setups
			if m.clusterType == config.MRC {
				dcBuilder.WriteString(DCHeaderStyle.Render(fmt.Sprintf("Data Center %d:", dcID)))
				// No newline needed here, header style has margin
			}

			// Sort Broker IDs within the DC
			brokerIDs := make([]int, 0, len(dc.Brokers))
			for id := range dc.Brokers {
				brokerIDs = append(brokerIDs, id)
			}
			sort.Ints(brokerIDs)

			var brokerViews []string // Store rendered views for each broker box

			for _, brokerID := range brokerIDs {
				broker := dc.Brokers[brokerID]
				var brokerBuilder strings.Builder
				brokerBuilder.WriteString(fmt.Sprintf("Broker %d:\n", broker.ID)) // Add newline after Broker ID

				if len(broker.Replicas) == 0 {
					brokerBuilder.WriteString(HelpStyle.Render("  (empty)"))
				} else {
					// Sort replicas by partition ID within the broker for clarity
					sort.Slice(broker.Replicas, func(i, j int) bool {
						return broker.Replicas[i].PartitionID < broker.Replicas[j].PartitionID
					})

					// Render each replica with appropriate style
					for _, replica := range broker.Replicas {
						pStr := fmt.Sprintf(" p%d", replica.PartitionID) // Add space before pX
						switch replica.Role {
						case config.Leader:
							brokerBuilder.WriteString(LeaderStyle.Render(pStr))
						case config.Follower:
							brokerBuilder.WriteString(FollowerStyle.Render(pStr))
						case config.Observer:
							// Only show observer style if it's actually MRC
							if m.clusterType == config.MRC {
								brokerBuilder.WriteString(ObserverStyle.Render(pStr))
							} else {
								// Should not happen based on placement logic, but fallback
								brokerBuilder.WriteString(FollowerStyle.Render(pStr))
							}
						}
					}
				}
				// Apply box style to the individual broker's content
				brokerViews = append(brokerViews, BrokerBoxStyle.Render(brokerBuilder.String()))
			}

			// Join broker boxes horizontally for the current DC
			// Add newline after header if MRC
			if m.clusterType == config.MRC {
				dcBuilder.WriteString("\n") // Add space below DC header
			}
			dcBuilder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, brokerViews...))
			dcViews = append(dcViews, dcBuilder.String())
		}

		// Join all DC views vertically
		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, dcViews...))

		// --- Legend ---
		b.WriteString("\n\nLegend: ")
		b.WriteString(LeaderStyle.Render("Leader (pX)"))
		b.WriteString("  ")
		b.WriteString(FollowerStyle.Render("Follower (pX)"))
		// Only show Observer in legend if MRC is possible
		if m.clusterType == config.MRC { // Check the *potential* type, not just current selection
			b.WriteString("  ")
			b.WriteString(ObserverStyle.Render("Observer (pX)"))
		}
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("(Press Enter to restart. Ctrl+C to quit)"))

	case ShowError:
		// Display a general error message if we land in this state
		// Specific validation errors are shown in the input stages
		errMsg := "An unexpected error occurred."
		if m.err != nil {
			errMsg = m.err.Error()
		}
		b.WriteString(ErrorStyle.Render("Error: " + errMsg))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("(Press Enter to restart. Ctrl+C to quit)"))
	}

	return b.String()
}
