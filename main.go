package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Configuration ---

type ClusterType int

const (
	SingleCluster ClusterType = iota
	MRC
)

type ReplicaRole string

const (
	Leader   ReplicaRole = "Leader"
	Follower ReplicaRole = "Follower"
	Observer ReplicaRole = "Observer" // As requested for color mapping
)

type ReplicaInfo struct {
	PartitionID int
	Role        ReplicaRole
}

type BrokerInfo struct {
	ID       int
	Replicas []ReplicaInfo
}

type DCInfo struct {
	ID      int
	Brokers map[int]*BrokerInfo // Map BrokerID -> BrokerInfo
}

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle.Copy()
	noStyle      = lipgloss.NewStyle()

	helpStyle = blurredStyle.Copy()

	// Replica Colors
	leaderColor   = lipgloss.Color("#00FF00") // Green
	followerColor = lipgloss.Color("#FFFF00") // Yellow
	observerColor = lipgloss.Color("#FF0000") // Red

	leaderStyle   = lipgloss.NewStyle().Foreground(leaderColor)
	followerStyle = lipgloss.NewStyle().Foreground(followerColor)
	observerStyle = lipgloss.NewStyle().Foreground(observerColor)

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")) // Red for errors

	dcHeaderStyle  = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	brokerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")). // Purple border
			Padding(0, 1).
			MarginRight(2).
			MarginBottom(1)
)

// --- Model ---

type Stage int

const (
	AskClusterType Stage = iota
	AskSingleConfig
	AskMRCConfig
	ShowPlacement
	ShowError
)

type model struct {
	stage         Stage
	clusterType   ClusterType
	inputs        []textinput.Model
	focused       int
	err           error
	width, height int // Terminal size

	// Config values
	numPartitions     int
	minInSyncReplicas int
	replicationFactor int
	numBrokers        int // Total for single, per DC for MRC
	numDCs            int

	// Placement results
	dcs               map[int]*DCInfo // Map DC ID -> DCInfo (for both single and MRC)
	mrcRecommendation string
}

func NewModel() model {
	m := model{
		stage:   AskClusterType,
		focused: 0,
		dcs:     make(map[int]*DCInfo),
	}
	// No inputs needed for the first stage
	return m
}

func (m *model) setupInputsForStage() {
	m.inputs = nil // Clear previous inputs
	m.focused = 0

	switch m.stage {
	case AskSingleConfig:
		m.inputs = make([]textinput.Model, 4)
		placeholders := []string{"Total Brokers", "Partitions", "Replication Factor", "Min ISR"}
		for i := range m.inputs {
			m.inputs[i] = textinput.New()
			m.inputs[i].Cursor.Style = cursorStyle
			m.inputs[i].CharLimit = 5
			m.inputs[i].Placeholder = placeholders[i]
			m.inputs[i].Validate = isNumber // Basic validation
		}
		m.inputs[0].Focus() // Focus the first input

	case AskMRCConfig:
		m.inputs = make([]textinput.Model, 5)
		placeholders := []string{"Data Centers", "Brokers per DC", "Partitions", "Replication Factor", "Min ISR"}
		for i := range m.inputs {
			m.inputs[i] = textinput.New()
			m.inputs[i].Cursor.Style = cursorStyle
			m.inputs[i].CharLimit = 5
			m.inputs[i].Placeholder = placeholders[i]
			m.inputs[i].Validate = isNumber // Basic validation
		}
		m.inputs[0].Focus() // Focus the first input
	}
}

// --- Bubble Tea Methods ---

func (m model) Init() tea.Cmd {
	rand.Seed(time.Now().UnixNano()) // Seed random for placement variation
	return textinput.Blink           // Start cursor blinking
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		// Set focus to next input
		case tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown:
			if m.stage == AskSingleConfig || m.stage == AskMRCConfig {
				s := msg.String()
				if s == "up" || s == "shift+tab" {
					m.focused--
				} else {
					m.focused++
				}

				if m.focused >= len(m.inputs) {
					m.focused = 0
				} else if m.focused < 0 {
					m.focused = len(m.inputs) - 1
				}

				for i := 0; i <= len(m.inputs)-1; i++ {
					if i == m.focused {
						// Set focused state
						cmds = append(cmds, m.inputs[i].Focus())
						m.inputs[i].PromptStyle = focusedStyle
						m.inputs[i].TextStyle = focusedStyle
					} else {
						// Remove focused state
						m.inputs[i].Blur()
						m.inputs[i].PromptStyle = noStyle
						m.inputs[i].TextStyle = noStyle
					}
				}
			}

		case tea.KeyEnter:
			switch m.stage {
			case AskClusterType:
				// This stage uses simple key presses, not text input
				// Handled below in the specific key checks

			case AskSingleConfig, AskMRCConfig:
				// If it's the last input field, try to process
				if m.focused == len(m.inputs)-1 {
					err := m.parseAndValidateInputs()
					if err != nil {
						m.err = err
						// Keep the current stage to show error
					} else {
						m.err = nil
						m.stage = ShowPlacement
						m.calculatePlacement() // Calculate placement
					}
				} else {
					// Move focus to the next field on Enter
					m.focused = (m.focused + 1) % len(m.inputs)
					for i := range m.inputs {
						if i == m.focused {
							cmds = append(cmds, m.inputs[i].Focus())
							m.inputs[i].PromptStyle = focusedStyle
							m.inputs[i].TextStyle = focusedStyle
						} else {
							m.inputs[i].Blur()
							m.inputs[i].PromptStyle = noStyle
							m.inputs[i].TextStyle = noStyle
						}
					}
				}

			case ShowPlacement, ShowError:
				// Allow Enter to go back or quit (optional)
				// For now, just quit on Enter from results/error
				// return m, tea.Quit
				// Or reset to start:
				return NewModel(), textinput.Blink
			}

		// Handle specific keys for the first stage
		default:
			if m.stage == AskClusterType {
				switch msg.String() {
				case "s", "S":
					m.clusterType = SingleCluster
					m.stage = AskSingleConfig
					m.setupInputsForStage()
					cmds = append(cmds, m.inputs[0].Focus()) // Focus first input
				case "m", "M":
					m.clusterType = MRC
					m.stage = AskMRCConfig
					m.setupInputsForStage()
					cmds = append(cmds, m.inputs[0].Focus()) // Focus first input
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Handle input field updates if applicable
	if m.stage == AskSingleConfig || m.stage == AskMRCConfig {
		for i := range m.inputs {
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Kafka Partition Visualizer"))
	b.WriteString("\n\n")

	switch m.stage {
	case AskClusterType:
		b.WriteString("Select cluster type:\n\n")
		b.WriteString("[S] Single Cluster\n")
		b.WriteString("[M] Multi-Region Cluster (MRC)\n\n")
		b.WriteString(helpStyle.Render("(Press S or M, then Enter is not needed. Ctrl+C to quit)"))

	case AskSingleConfig, AskMRCConfig:
		title := "Enter Single Cluster Configuration:"
		var labels []string
		if m.stage == AskMRCConfig {
			title = "Enter MRC Configuration:"
			// Labels for MRC config inputs
			labels = []string{"Data Centers:", "Brokers per DC:", "Partitions:", "Replication Factor:", "Min ISR:"}
		} else {
			// Labels for Single Cluster config inputs
			labels = []string{"Total Brokers:", "Partitions:", "Replication Factor:", "Min ISR:"}
		}
		b.WriteString(title + "\n\n")

		for i := range m.inputs {
			// Add the label before the input view
			b.WriteString(labels[i] + "\n")   // Display label on its own line
			b.WriteString(m.inputs[i].View()) // Display the input field below the label
			if i < len(m.inputs)-1 {
				b.WriteRune('\n') // Add space between input fields
			}
		}

		b.WriteString("\n\n")
		if m.err != nil {
			b.WriteString(errorStyle.Render("Error: " + m.err.Error()))
			b.WriteString("\n\n")
		}
		b.WriteString(helpStyle.Render("Use Tab/Shift+Tab or Up/Down to navigate. Enter to confirm or move to next. Ctrl+C to quit."))

	case ShowPlacement:
		b.WriteString("Partition Placement Visualization:\n\n")
		if m.clusterType == MRC {
			b.WriteString(fmt.Sprintf("MRC Recommendation: %s\n\n", m.mrcRecommendation))
		}

		// Sort DC IDs for consistent display
		dcIDs := make([]int, 0, len(m.dcs))
		for id := range m.dcs {
			dcIDs = append(dcIDs, id)
		}
		sort.Ints(dcIDs)

		var dcViews []string

		for _, dcID := range dcIDs {
			dc := m.dcs[dcID]
			var dcBuilder strings.Builder
			if m.clusterType == MRC {
				dcBuilder.WriteString(dcHeaderStyle.Render(fmt.Sprintf("Data Center %d:", dcID)))
				dcBuilder.WriteString("\n")
			}

			// Sort Broker IDs within DC
			brokerIDs := make([]int, 0, len(dc.Brokers))
			for id := range dc.Brokers {
				brokerIDs = append(brokerIDs, id)
			}
			sort.Ints(brokerIDs)

			var brokerViews []string
			for _, brokerID := range brokerIDs {
				broker := dc.Brokers[brokerID]
				var brokerBuilder strings.Builder
				brokerBuilder.WriteString(fmt.Sprintf("Broker %d:\n", broker.ID))
				if len(broker.Replicas) == 0 {
					brokerBuilder.WriteString(helpStyle.Render("  (empty)"))
				} else {
					// Sort replicas by partition ID for clarity
					sort.Slice(broker.Replicas, func(i, j int) bool {
						return broker.Replicas[i].PartitionID < broker.Replicas[j].PartitionID
					})
					for _, replica := range broker.Replicas {
						pStr := fmt.Sprintf(" p%d", replica.PartitionID)
						switch replica.Role {
						case Leader:
							brokerBuilder.WriteString(leaderStyle.Render(pStr))
						case Follower:
							brokerBuilder.WriteString(followerStyle.Render(pStr))
						case Observer:
							brokerBuilder.WriteString(observerStyle.Render(pStr))
						}
					}
				}
				brokerViews = append(brokerViews, brokerBoxStyle.Render(brokerBuilder.String()))
			}
			// Use lipgloss JoinHorizontal for better layout if needed, simple join for now
			dcBuilder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, brokerViews...))
			dcViews = append(dcViews, dcBuilder.String())
		}

		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, dcViews...))

		b.WriteString("\n\nLegend: ")
		b.WriteString(leaderStyle.Render("Leader (pX)"))
		b.WriteString("  ")
		b.WriteString(followerStyle.Render("Follower (pX)"))
		b.WriteString("  ")
		b.WriteString(observerStyle.Render("Observer (pX)"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("(Press Enter to restart. Ctrl+C to quit)"))

	case ShowError: // Should not happen if validation is good, but fallback
		b.WriteString(errorStyle.Render("An unexpected error occurred: " + m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("(Press Enter to restart. Ctrl+C to quit)"))
	}

	// Limit height if needed, basic full view for now
	return b.String()
}

// --- Helper Functions ---

func isNumber(s string) error {
	if s == "" {
		return nil // Allow empty while typing
	}
	_, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("must be a number")
	}
	return nil
}

func (m *model) parseAndValidateInputs() error {
	var err error
	values := make([]int, len(m.inputs))

	for i, input := range m.inputs {
		if input.Value() == "" {
			return fmt.Errorf("input for '%s' cannot be empty", input.Placeholder)
		}
		values[i], err = strconv.Atoi(input.Value())
		if err != nil {
			// Should be caught by Validate, but double-check
			return fmt.Errorf("invalid number for '%s': %w", input.Placeholder, err)
		}
		if values[i] <= 0 {
			return fmt.Errorf("input for '%s' must be positive", input.Placeholder)
		}
	}

	// Assign based on stage
	if m.stage == AskSingleConfig {
		m.numBrokers = values[0]
		m.numPartitions = values[1]
		m.replicationFactor = values[2]
		m.minInSyncReplicas = values[3]
		m.numDCs = 1 // Single DC
	} else { // AskMRCConfig
		m.numDCs = values[0]
		m.numBrokers = values[1] // Brokers *per DC*
		m.numPartitions = values[2]
		m.replicationFactor = values[3]
		m.minInSyncReplicas = values[4]
	}

	totalBrokers := m.numBrokers
	if m.clusterType == MRC {
		totalBrokers *= m.numDCs
		if m.numDCs <= 1 {
			return fmt.Errorf("MRC requires at least 2 Data Centers")
		}
	}

	// Logical validation
	if m.replicationFactor > totalBrokers {
		return fmt.Errorf("replication Factor (%d) cannot exceed total brokers (%d)", m.replicationFactor, totalBrokers)
	}
	if m.minInSyncReplicas > m.replicationFactor {
		return fmt.Errorf("min ISR (%d) cannot exceed Replication Factor (%d)", m.minInSyncReplicas, m.replicationFactor)
	}
	if m.minInSyncReplicas <= 0 {
		return fmt.Errorf("min ISR must be positive")
	}
	if m.replicationFactor <= 0 {
		return fmt.Errorf("replication Factor must be positive")
	}
	if m.numPartitions <= 0 {
		return fmt.Errorf("number of partitions must be positive")
	}
	if m.numBrokers <= 0 {
		return fmt.Errorf("number of brokers must be positive")
	}

	return nil // No error
}

// calculatePlacement simulates partition placement.
// This is a simplified round-robin approach.
func (m *model) calculatePlacement() {
	m.dcs = make(map[int]*DCInfo)
	brokerIDCounter := 0
	totalBrokers := 0

	// Initialize DCs and Brokers
	for dcIdx := 0; dcIdx < m.numDCs; dcIdx++ {
		dcID := dcIdx + 1 // 1-based DC IDs
		m.dcs[dcID] = &DCInfo{
			ID:      dcID,
			Brokers: make(map[int]*BrokerInfo),
		}
		for brokerIdx := 0; brokerIdx < m.numBrokers; brokerIdx++ {
			brokerID := brokerIDCounter
			m.dcs[dcID].Brokers[brokerID] = &BrokerInfo{
				ID:       brokerID,
				Replicas: []ReplicaInfo{},
			}
			brokerIDCounter++
		}
	}
	totalBrokers = brokerIDCounter

	// --- MRC Recommendation ---
	if m.clusterType == MRC {
		// Simple recommendation: Distribute evenly if possible.
		// More complex logic could consider RF vs DCs.
		m.mrcRecommendation = fmt.Sprintf("Distribute %d replicas across %d DCs for fault tolerance.", m.replicationFactor, m.numDCs)
		if m.replicationFactor <= m.numDCs {
			m.mrcRecommendation += " Aim for at most one replica per DC per partition."
		} else {
			minPerDC := m.replicationFactor / m.numDCs
			extra := m.replicationFactor % m.numDCs
			m.mrcRecommendation += fmt.Sprintf(" Aim for ~%d replicas per DC, with %d DCs having an extra replica.", minPerDC, extra)
		}
	}

	// --- Placement Logic ---
	allBrokerIDs := make([]int, 0, totalBrokers)
	for dcID := 1; dcID <= m.numDCs; dcID++ {
		for brokerID := range m.dcs[dcID].Brokers {
			allBrokerIDs = append(allBrokerIDs, brokerID)
		}
	}

	for p := 0; p < m.numPartitions; p++ {
		partitionID := p + 1 // 1-based partition IDs

		// Shuffle brokers for each partition for better distribution simulation
		// (Real Kafka uses more deterministic assignment)
		shuffledBrokerIDs := make([]int, len(allBrokerIDs))
		copy(shuffledBrokerIDs, allBrokerIDs)
		rand.Shuffle(len(shuffledBrokerIDs), func(i, j int) {
			shuffledBrokerIDs[i], shuffledBrokerIDs[j] = shuffledBrokerIDs[j], shuffledBrokerIDs[i]
		})

		// Determine leader broker (simple modulo for initial placement)
		leaderBrokerID := allBrokerIDs[p%totalBrokers] // Start leader assignment round-robin

		// Find the DC and Broker object for the leader
		leaderDC, leaderBroker := m.findBroker(leaderBrokerID)
		if leaderBroker == nil {
			continue
		} // Should not happen

		// Assign Leader
		leaderBroker.Replicas = append(leaderBroker.Replicas, ReplicaInfo{PartitionID: partitionID, Role: Leader})
		assignedBrokerIDs := map[int]bool{leaderBrokerID: true}
		assignedDCs := map[int]bool{leaderDC.ID: true}
		replicasPlaced := 1

		brokersToTry := shuffledBrokerIDs // Use shuffled list

		// Variables only needed for MRC role differentiation
		var numFollowers, numObservers, targetFollowers, targetObservers int
		if m.clusterType == MRC {
			targetFollowers = m.minInSyncReplicas - 1 // Followers needed for ISR quorum
			if targetFollowers < 0 {
				targetFollowers = 0
			}
			targetObservers = m.replicationFactor - 1 - targetFollowers // Remaining replicas
			if targetObservers < 0 {
				targetObservers = 0
			}
		}

		// First pass (try spreading across DCs for MRC)
		for _, brokerID := range brokersToTry {
			if replicasPlaced >= m.replicationFactor {
				break
			} // Stop if RF met
			if assignedBrokerIDs[brokerID] {
				continue
			} // Skip if broker already has a replica for this partition

			dc, broker := m.findBroker(brokerID)
			if broker == nil {
				continue
			}

			// MRC Placement Strategy: Try to place in different DCs first
			placeInThisDC := true
			if m.clusterType == MRC && len(assignedDCs) < m.numDCs {
				if assignedDCs[dc.ID] {
					// Check if we can place elsewhere before placing in an already used DC
					canPlaceElsewhere := false
					for _, otherBrokerID := range brokersToTry {
						if !assignedBrokerIDs[otherBrokerID] {
							otherDC, _ := m.findBroker(otherBrokerID)
							if otherDC != nil && !assignedDCs[otherDC.ID] { // Check otherDC is not nil
								canPlaceElsewhere = true
								break
							}
						}
					}
					if canPlaceElsewhere {
						placeInThisDC = false
					}
				}
			}

			if placeInThisDC {
				var role ReplicaRole
				if m.clusterType == SingleCluster {
					// In Single Cluster, all non-leaders are just Followers
					role = Follower
				} else { // MRC logic
					// Assign role based on ISR needs first, then observers
					if numFollowers < targetFollowers {
						role = Follower
						numFollowers++
					} else if numObservers < targetObservers {
						role = Observer
						numObservers++
					} else {
						// Fallback if RF > minISR + observers needed (shouldn't happen with calc above)
						role = Observer
						numObservers++
					}
				}

				broker.Replicas = append(broker.Replicas, ReplicaInfo{PartitionID: partitionID, Role: role})
				assignedBrokerIDs[brokerID] = true
				assignedDCs[dc.ID] = true // Track used DCs for MRC strategy
				replicasPlaced++
			}
		}

		// Second pass for MRC if needed (allow placing in same DC)
		if m.clusterType == MRC && replicasPlaced < m.replicationFactor {
			for _, brokerID := range brokersToTry {
				if replicasPlaced >= m.replicationFactor {
					break
				}
				if assignedBrokerIDs[brokerID] {
					continue
				}

				_, broker := m.findBroker(brokerID)
				if broker == nil {
					continue
				}

				// Assign role based on remaining needs for MRC
				var role ReplicaRole
				if numFollowers < targetFollowers {
					role = Follower
					numFollowers++
				} else if numObservers < targetObservers {
					role = Observer
					numObservers++
				} else {
					role = Observer // Assign remaining as Observers
					numObservers++
				}

				broker.Replicas = append(broker.Replicas, ReplicaInfo{PartitionID: partitionID, Role: role})
				assignedBrokerIDs[brokerID] = true
				// assignedDCs doesn't need update here
				replicasPlaced++
			}
		}
		// --- MODIFICATION END ---
	}
}

// findBroker searches all DCs to find the broker with the given ID.
func (m *model) findBroker(brokerID int) (*DCInfo, *BrokerInfo) {
	for _, dc := range m.dcs {
		if broker, ok := dc.Brokers[brokerID]; ok {
			return dc, broker
		}
	}
	return nil, nil // Not found
}

// --- Main ---

func main() {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen()) // Use AltScreen for cleaner exit
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
		os.Exit(1)
	}
}
