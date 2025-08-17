package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel create a new model for the UI
func NewModel() Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".dbc"}
	fp.CurrentDirectory = "."
	fp.ShowHidden = false
	fp.DirAllowed = true
	fp.FileAllowed = true

	return Model{
		State:            StateFilePicker,
		FilePicker:       fp,
		SelectedMessages: make([]CANMessage, 0),
		LastUpdate:       time.Now(),
	}
}

// NewModelWithDBC creates a new model with the specified DBC file
func NewModelWithDBC(dbcPath string) Model {
	m := NewModel()

	if dbcPath != "" {
		m.DBCPath = dbcPath
		m.Err = m.loadDBC()
		if m.Err == nil {
			m.State = StateMessageSelector
			m.setupMessageList()
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
