package ui

import (
	"fmt"
	"net"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/squadracorsepolito/acmelib"

	"github.com/carolabonamico/can-debug/internal/can"
)

// State represents the current state of the UI
type State int

const (
	StateFilePicker State = iota
	StateMessageSelector
	StateMonitoring
)

// CANMessage represents a message in the CAN bus
type CANMessage struct {
	ID       uint32
	Name     string
	Selected bool
	Message  *acmelib.Message
}

func (c CANMessage) Title() string {
	if c.Selected {
		return "✅ " + c.Name
	}
	return c.Name
}

func (c CANMessage) Description() string {
	desc := fmt.Sprintf("ID: 0x%X", c.ID)
	if c.Selected {
		desc += " (selected)"
	}
	return desc
}

func (c CANMessage) FilterValue() string { return c.Name }

// Main model of the application
type Model struct {
	State            State
	FilePicker       filepicker.Model
	MessageList      list.Model
	MonitoringTable  table.Model
	SelectedMessages []CANMessage
	DBCPath          string
	Messages         []*acmelib.Message
	Decoder          *can.Decoder
	LastUpdate       time.Time
	Width            int
	Height           int
	Err              error
	CanNetwork		 net.Conn
}

// Message for updating real-time data
type TickMsg time.Time
