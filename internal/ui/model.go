package ui

import (
	"net"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"go.einride.tech/can/pkg/socketcan"
)

// NewModel create a new model for the UI
func NewModel(CanNet net.Conn) Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".dbc"}
	fp.CurrentDirectory = "."
	fp.ShowHidden = false
	fp.DirAllowed = true
	fp.FileAllowed = true

	return Model{
		State:                     StateFilePicker,
		FilePicker:                fp,
		SelectedMessages:          make([]CANMessage, 0),
		LastUpdate:                time.Now(),
		CanNetwork:                CanNet,
		Transmitter:               socketcan.NewTransmitter(CanNet),
		SendReceiveChoice:         0,
		PreviousSendReceiveChoice: 0, // Initialize to same as current
		SendSignals:               make([]SendSignal, 0),
		CurrentInputIndex:         -1,
		ActiveMessages:            make(map[int]infoSending),
	}
}

// NewModelWithDBC creates a new model with the specified DBC file
func NewModelWithDBC(dbcPath string, CanNet net.Conn) Model {
	m := NewModel(CanNet)

	if dbcPath != "" {
		m.DBCPath = dbcPath
		m.DBCFromCommandLine = true // Set flag when DBC is from command line
		m.Err = m.loadDBC()
		if m.Err == nil {
			m.State = StateSendReceiveSelector
		}
	}

	return m
}

// TickCmd returns a command that ticks every 100 milliseconds
func TickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.FilePicker.Init(),
		TickCmd(),
	)
}
