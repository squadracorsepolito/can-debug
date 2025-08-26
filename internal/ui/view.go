package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// wrapStatus wraps a status message to fit within the terminal width
func (m Model) wrapStatus(status string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80 // fallback width
	}

	// Reserve space for "Status: " prefix (11 characters)
	availableWidth := maxWidth - 11
	if availableWidth <= 20 {
		availableWidth = 20
	}

	words := strings.Fields(status)
	if len(words) == 0 {
		return status
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) <= availableWidth {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	// Join lines with newline and proper indentation for continuation lines
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n           " + lines[i] // 11 spaces to align with "💬 Status: "
	}

	return result
}

// View renders the current state of the UI
func (m Model) View() string {
	switch m.State {
	case StateFilePicker:
		return m.filePickerView()
	case StateSendReceiveSelector:
		return m.sendReceiveSelectorView()
	case StateMessageSelector:
		return m.messageSelectorView()
	case StateMonitoring:
		return m.monitoringView()
	case StateSendConfiguration:
		return m.sendConfigurationView()
	default:
		return "Not recognized state"
	}
}

// filePickerView renders the file picker view
func (m Model) filePickerView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("🔧 CAN Debug Tool - Select a DBC file"))
	s.WriteString("\n\n")

	s.WriteString("↑/k up • ↓/j down • Enter: open directory/select .dbc file • q quit")
	s.WriteString("\n\n")

	if m.Err != nil {
		s.WriteString(fmt.Sprintf("Error: %v", m.Err))
		s.WriteString("\n\n")
	}

	s.WriteString(m.FilePicker.View())

	return s.String()
}

// messageSelectorView renders the message selector view
func (m Model) messageSelectorView() string {
	var s strings.Builder

	// Display the message list
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📋 Select signals:"))
	s.WriteString("\n\n")

	// Status bar with commands for the message list
	s.WriteString("Navigation: ↑/k up • ↓/j down • / filter • Tab back to mode selection • q quit\n")
	s.WriteString("Actions: Space select/deselect • Enter start monitoring\n")
	s.WriteString("\n") // Single newline instead of double

	s.WriteString(m.MessageList.View())

	return s.String()
}

// monitoringView renders the monitoring view
func (m Model) monitoringView() string {
	var s strings.Builder

	// Show error if candump failed, otherwise show normal title
	if m.Err != nil {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render("📡 Real time CAN monitoring"))
		s.WriteString("\n\n")
		wrappedStatus := m.wrapStatus(m.Err.Error(), m.Width)
		s.WriteString(fmt.Sprintf("💬 Status: %s", wrappedStatus))
		s.WriteString("\n\n")
		// Show simplified commands when there's an error
		s.WriteString("Tab back to message selection • q quit")
	} else {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render("📡 Real time CAN monitoring"))
		s.WriteString(fmt.Sprintf(" | Last update: %s", m.LastUpdate.Format("15:04:05.000")))
		s.WriteString("\n\n")

		// Status bar with commands for the monitoring table
		s.WriteString("↑/k up • ↓/j down • Tab back to message selection • q quit")
		s.WriteString("\n\n")

		s.WriteString(m.MonitoringTable.View())
	}

	return s.String()
}

// sendReceiveSelectorView renders the send/receive selector view
func (m Model) sendReceiveSelectorView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("🔧 CAN Debug Tool - Choose mode"))
	s.WriteString(fmt.Sprintf(" (File: %s)", m.DBCPath))
	s.WriteString("\n\n")

	s.WriteString("Select use mode:\n\n")

	// Display the send/receive options
	if m.SendReceiveChoice == 0 {
		s.WriteString("> 📤 Send CAN messages\n")
		s.WriteString("  📥 Receive and monitor CAN messages\n")
	} else {
		s.WriteString("  📤 Send CAN messages\n")
		s.WriteString("> 📥 Receive and monitor CAN messages\n")
	}

	// Show navigation instructions based on how DBC was loaded
	if m.DBCFromCommandLine {
		s.WriteString("\n↑/k up • ↓/j down • Enter confirm • q quit")
	} else {
		s.WriteString("\n↑/k up • ↓/j down • Enter confirm • Tab back to file selection • q quit")
	}

	return s.String()
}

// sendingView renders the sending view
func (m Model) sendingView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📤 Send CAN messages"))
	s.WriteString(fmt.Sprintf(" (File: %s)", m.DBCPath))
	s.WriteString("\n\n")

	s.WriteString("Enter send • Tab back to selection mode • q quit")
	s.WriteString("\n\n")

	s.WriteString(m.TextInput.View())
	s.WriteString("\n\n")

	// Status and last sent message
	if m.LastSentMessage != "" {
		// Show success message with last sent message
		s.WriteString(fmt.Sprintf("✅ Last sent message: \"%s\"", m.LastSentMessage))
		s.WriteString("\n")

		if m.SendStatus != "" {
			wrappedStatus := m.wrapStatus(m.SendStatus, m.Width)
			s.WriteString(fmt.Sprintf("\n💬 Status: %s", wrappedStatus))
		}
	} else if m.SendStatus != "" {
		// Show only status (error or initial state)
		wrappedStatus := m.wrapStatus(m.SendStatus, m.Width)
		s.WriteString(fmt.Sprintf("💬 Status: %s", wrappedStatus))
	} else {
		// Initial state
		s.WriteString("💡 No message sent yet.")
	}

	return s.String()
}

// sendConfigurationView renders the send configuration view
func (m Model) sendConfigurationView() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📤 Configure message values to send"))
	s.WriteString(fmt.Sprintf(" (File: %s)", m.DBCPath))
	s.WriteString("\n\n")

	// Instructions organized by category
	s.WriteString("Navigation: ↑/k up • ↓/j down • Tab back • q quit\n")
	s.WriteString("Action: Enter send message • Space toggle message • ←→ adjust message cycle • s stop all")
	s.WriteString("\n\n")

	// Show individual signal status if any are active
	activeCount := 0
	for _, signal := range m.SendSignals {
		if signal.IsActive {
			activeCount++
		}
	}
	if activeCount > 0 {
		s.WriteString(fmt.Sprintf("🎯 Continuous signals active: %d/%d\n", activeCount, len(m.SendSignals)))
		s.WriteString("\n")
	}

	// Show the send table
	if len(m.SendSignals) > 0 {
		s.WriteString(m.SendTable.View())
		s.WriteString("\n\n")
	}

	// Status
	if m.SendStatus != "" {
		wrappedStatus := m.wrapStatus(m.SendStatus, m.Width)
		s.WriteString(fmt.Sprintf("💬 Status: %s", wrappedStatus))
	} else if activeCount > 0 {
		s.WriteString("🎯 Continuous signals are running. Use Space to start/stop, Enter to send once.")
	} else {
		s.WriteString("💡 Enter values, set cycle times. Use Enter to send once or Space for continuous sending.")
	}

	return s.String()
}
