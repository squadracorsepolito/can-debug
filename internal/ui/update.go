package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// Update updates the model based on the received message
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Update the height of the file picker based on the window size
		switch m.State {
		case StateFilePicker:
			m.FilePicker.Height = msg.Height - 4
		case StateSendReceiveSelector:
			// No handling needed
		case StateMessageSelector:
			m.MessageList.SetWidth(msg.Width)
			m.MessageList.SetHeight(msg.Height - 6)
		case StateMonitoring:
			m.MonitoringTable.SetWidth(msg.Width)
			m.MonitoringTable.SetHeight(msg.Height - 4)
		case StateSendConfiguration:
			m.SendTable.SetWidth(msg.Width)
			// Calculate available height for the table
			// Title+file (1) + spacing (2) + Navigation (1) + spacing (2) + Send mode (3-4) + spacing (1) + table spacing (2) + status (1) = ~12-13 lines
			availableHeight := msg.Height - 13
			if availableHeight < 3 {
				availableHeight = 3 // Minimum height
			}
			m.SendTable.SetHeight(availableHeight)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Tab sempre torna indietro alla schermata precedente
			switch m.State {
			case StateSendReceiveSelector:
				// Da send/receive selector, torna a file picker solo se il DBC non è da riga di comando
				if !m.DBCFromCommandLine {
					m.State = StateFilePicker
				}
			case StateMessageSelector:
				// Da message selector, torna a send/receive selector
				m.State = StateSendReceiveSelector
			case StateMonitoring:
				// Da monitoring, torna a message selector
				m.State = StateMessageSelector
				// Update the message list when returning from monitoring mode
				m.updateMessageListItems()
				// Reset the table to avoid it being visible
				m.MonitoringTable = table.Model{}
			case StateSendConfiguration:
				// Da send configuration, torna a message selector
				m.State = StateMessageSelector
				// Update the message list when returning from send configuration
				m.updateMessageListItems()
			}
		}

	case TickMsg:
		m.LastUpdate = time.Now()
		return m, TickCmd()
	}

	switch m.State {
	case StateFilePicker:
		m.FilePicker, cmd = m.FilePicker.Update(msg)
		cmds = append(cmds, cmd)

		// Check if a file was selected
		if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
			// Verify that it is a .dbc file
			if strings.HasSuffix(strings.ToLower(path), ".dbc") {
				m.DBCPath = path
				m.Err = m.loadDBC()
				if m.Err == nil {
					m.State = StateSendReceiveSelector
				}
			} else {
				m.Err = fmt.Errorf("select a .dbc file, not %s", path)
			}
		}

	case StateSendReceiveSelector:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.SendReceiveChoice > 0 {
					m.SendReceiveChoice--
				}
			case "down", "j":
				if m.SendReceiveChoice < 1 {
					m.SendReceiveChoice++
				}
			case "enter":
				if m.SendReceiveChoice == 0 {
					// Send mode - go to message selector
					m.State = StateMessageSelector
					m.setupMessageList()
				} else {
					// Receive mode
					m.State = StateMessageSelector
					m.setupMessageList()
				}
			}
		}

	case StateMessageSelector:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if len(m.SelectedMessages) > 0 {
					if m.SendReceiveChoice == 0 {
						// Send mode - vai a send configuration
						m.State = StateSendConfiguration
						m.setupSendConfiguration()
					} else {
						// Receive mode - vai a monitoring
						m.setupMonitoringTable()
						m.initializesTableDBCSignals()
						m.State = StateMonitoring
						go m.startReceavingMessages()
					}
				}
			case " ":
				// Toggle message selection
				m.toggleMessageSelection()
			}
		}
		m.MessageList, cmd = m.MessageList.Update(msg)
		cmds = append(cmds, cmd)

	case StateMonitoring:
		m.MonitoringTable, cmd = m.MonitoringTable.Update(msg)
		cmds = append(cmds, cmd)

	case StateSendConfiguration:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				// Send the configured signals
				if m.SendMode == 0 {
					// Single send
					m.sendConfiguredSignals()
				} else {
					// Start/stop cyclical sending
					if m.IsSendingCyclical {
						m.stopCyclicalSending()
					} else {
						m.startCyclicalSending()
					}
				}
			case "s":
				// Switch send mode
				if m.SendMode == 0 {
					m.SendMode = 1
				} else {
					m.SendMode = 0
				}
			case "-":
				// Decrease interval (only in cyclical mode)
				if m.SendMode == 1 && m.SendInterval > 50 {
					m.SendInterval -= 10
				}
			case "+", "=":
				// Increase interval (only in cyclical mode)
				if m.SendMode == 1 && m.SendInterval < 2000 {
					m.SendInterval += 10
				}
			case "up", "k":
				if m.CurrentInputIndex > 0 {
					// Remove focus from current input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Blur()
					}
					// Move to previous input
					m.CurrentInputIndex--
					// Sync table cursor
					m.SendTable.SetCursor(m.CurrentInputIndex)
					// Set focus to new input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Focus()
					}
				}
			case "down", "j":
				if m.CurrentInputIndex < len(m.SendSignals)-1 {
					// Remove focus from current input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Blur()
					}
					// Move to next input
					m.CurrentInputIndex++
					// Sync table cursor
					m.SendTable.SetCursor(m.CurrentInputIndex)
					// Set focus to new input
					if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
						m.SendSignals[m.CurrentInputIndex].TextInput.Focus()
					}
				}
			default:
				// Update the current input field
				if m.CurrentInputIndex >= 0 && m.CurrentInputIndex < len(m.SendSignals) {
					m.SendSignals[m.CurrentInputIndex].TextInput, cmd = m.SendSignals[m.CurrentInputIndex].TextInput.Update(msg)
					cmds = append(cmds, cmd)
					// Update the table rows to reflect the new input value
					m.updateSendTableRows()
				}
			}
		}
		// Don't let the table handle navigation autonomously for this state
		// We manage cursor manually to sync with input focus
	}

	return m, tea.Batch(cmds...)
}
