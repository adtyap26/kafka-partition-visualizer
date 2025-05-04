package tui

import (
	// Use the full module path for your internal packages
	"github.com/adtyap26/kafka-partition-visualizer/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Stage defines the current view/state of the TUI application.
type Stage int

const (
	AskClusterType Stage = iota
	AskSingleConfig
	AskMRCConfig
	ShowPlacement
	ShowError // Represents a state where a known error is displayed
)

// Model holds the state for the TUI application. Exported for use in main.go.
type Model struct {
	stage         Stage
	clusterType   config.ClusterType
	inputs        []textinput.Model
	focused       int
	err           error // To store validation or processing errors
	width, height int   // Terminal size

	// Config values gathered from inputs
	numPartitions     int
	minInSyncReplicas int
	replicationFactor int
	numBrokers        int // Represents Total Brokers for Single, Brokers Per DC for MRC
	numDCs            int

	// Placement results from the placement package
	dcs               map[int]*config.DCInfo // Map DC ID -> DCInfo
	mrcRecommendation string
}

// NewModel creates the initial state of the TUI model. Exported for use in main.go.
func NewModel() Model {
	m := Model{
		stage:   AskClusterType,
		focused: 0,
		dcs:     make(map[int]*config.DCInfo),
	}
	// No inputs needed for the first stage, they are setup in Update
	return m
}

// Init initializes the TUI model. Required by Bubble Tea.
// Currently, just starts the cursor blinking.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}
