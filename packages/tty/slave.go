package tty

import (
	"io"
)

// Slave represents a PTY slave, typically it's a local command.
type Slave interface {
	io.ReadWriter

	// WindowTitleVariables returns any values that can be used to fill out
	// the title of a terminal.
	WindowTitleVariables() map[string]interface{}

	// ResizeTerminal sets a new size of the terminal.
	ResizeTerminal(columns int, rows int) error

	Close() error
}

type Factory interface {
	Name() string
	New(params map[string][]string) (Slave, error)
}
