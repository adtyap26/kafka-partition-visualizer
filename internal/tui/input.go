package tui

import (
	"fmt"
	"strconv"

	"github.com/adtyap26/kafka-partition-visualizer/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
)

// setupInputsForStage configures the text input fields based on the current stage.
// This is an unexported method as it modifies the model's internal state.
func (m *Model) setupInputsForStage() {
	m.inputs = nil // Clear previous inputs
	m.focused = 0
	m.err = nil // Clear previous errors

	switch m.stage {
	case AskSingleConfig:
		m.inputs = make([]textinput.Model, 4)
		placeholders := []string{"Total Brokers", "Partitions", "Replication Factor", "Min ISR"}
		for i := range m.inputs {
			m.inputs[i] = textinput.New()
			m.inputs[i].Cursor.Style = CursorStyle // Use style from styles.go
			m.inputs[i].CharLimit = 5
			m.inputs[i].Placeholder = placeholders[i]
			m.inputs[i].Validate = isNumber // Basic validation
		}
		m.inputs[0].Focus() // Focus the first input
		m.inputs[0].PromptStyle = FocusedStyle
		m.inputs[0].TextStyle = FocusedStyle

	case AskMRCConfig:
		m.inputs = make([]textinput.Model, 5)
		placeholders := []string{"Data Centers", "Brokers per DC", "Partitions", "Replication Factor", "Min ISR"}
		for i := range m.inputs {
			m.inputs[i] = textinput.New()
			m.inputs[i].Cursor.Style = CursorStyle // Use style from styles.go
			m.inputs[i].CharLimit = 5
			m.inputs[i].Placeholder = placeholders[i]
			m.inputs[i].Validate = isNumber // Basic validation
		}
		m.inputs[0].Focus() // Focus the first input
		m.inputs[0].PromptStyle = FocusedStyle
		m.inputs[0].TextStyle = FocusedStyle
	}
}

// isNumber is a validation function for textinput, ensuring input is numeric.
// Kept unexported as it's a helper for input setup.
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

// parseAndValidateInputs attempts to parse integer values from the input fields
// and performs logical validation on the configuration.
// Returns an error if parsing fails or validation rules are violated.
// This is an unexported method modifying the model's state.
func (m *Model) parseAndValidateInputs() error {
	var err error
	values := make([]int, len(m.inputs))

	for i, input := range m.inputs {
		if input.Value() == "" {
			return fmt.Errorf("input for '%s' cannot be empty", input.Placeholder)
		}
		values[i], err = strconv.Atoi(input.Value())
		if err != nil {
			return fmt.Errorf("invalid number for '%s': %w", input.Placeholder, err)
		}
		if values[i] <= 0 {
			return fmt.Errorf("input for '%s' must be positive", input.Placeholder)
		}
	}

	// Assign parsed values to model fields based on stage
	if m.stage == AskSingleConfig {
		m.numBrokers = values[0] // Total brokers
		m.numPartitions = values[1]
		m.replicationFactor = values[2]
		m.minInSyncReplicas = values[3]
		m.numDCs = 1 // Implicitly 1 DC for single cluster
	} else { // AskMRCConfig
		m.numDCs = values[0]
		m.numBrokers = values[1] // Brokers *per DC*
		m.numPartitions = values[2]
		m.replicationFactor = values[3]
		m.minInSyncReplicas = values[4]
	}

	// --- Logical Validation ---
	totalBrokers := m.numBrokers
	if m.clusterType == config.MRC {
		totalBrokers *= m.numDCs // Calculate total brokers for MRC
		if m.numDCs <= 1 {
			return fmt.Errorf("MRC requires at least 2 Data Centers")
		}
	}

	if totalBrokers <= 0 {
		// This check should ideally be covered by individual broker count checks,
		// but good as a safeguard.
		return fmt.Errorf("total number of brokers must be positive")
	}
	if m.replicationFactor > totalBrokers {
		return fmt.Errorf("replication Factor (%d) cannot exceed total brokers (%d)", m.replicationFactor, totalBrokers)
	}
	if m.minInSyncReplicas > m.replicationFactor {
		return fmt.Errorf("min ISR (%d) cannot exceed Replication Factor (%d)", m.minInSyncReplicas, m.replicationFactor)
	}
	// Other checks (positivity) are handled by the loop above.

	return nil // No error
}
