// Package log provides a common interface for logging.
package log

import (
	"fmt"
	"io"
	"strings"

	"github.com/tkrop/go-config/info"
)

// Logger provides a common interface for logging.
type Logger interface {
	// Logs the build information of the command or module to the given writer.
	Info(writer io.Writer, info *info.Info, raw bool)
	// Exec logs the internal command execution for debugging to the given writer.
	Exec(writer io.Writer, dir string, args ...string)
	// Logs the call of the command to the given writer.
	Call(writer io.Writer, args ...string)
	// Logs the given error message and error to the given writer.
	Error(writer io.Writer, message string, err error)
	// Logs the given message to the given writer.
	Message(writer io.Writer, message string)
}

// defaultLogger provides a default logger using `fmt` and `json` package.
type defaultLogger struct{}

// NewLogger creates a new default logger.
func NewLogger() Logger {
	return &defaultLogger{}
}

// Info logs the build information of the command or module to the given
// writer.
func (*defaultLogger) Info(writer io.Writer, info *info.Info, raw bool) {
	if !raw {
		fmt.Fprintf(writer, "info: %s\n", info)
	} else {
		fmt.Fprintf(writer, "%s\n", info)
	}
}

// Exec logs the internal command execution for debugging to the given writer.
func (*defaultLogger) Exec(writer io.Writer, dir string, args ...string) {
	if len(args) != 0 {
		fmt.Fprintf(writer, "exec: %s [%s]\n", strings.Join(args, " "), dir)
	} else {
		fmt.Fprintf(writer, "exec: [%s]\n", dir)
	}
}

// Call logs the call of the command to the given writer.
func (*defaultLogger) Call(writer io.Writer, args ...string) {
	if len(args) != 0 {
		fmt.Fprintf(writer, "call: %s\n", strings.Join(args, " "))
	} else {
		fmt.Fprintf(writer, "call: %s\n", "<no-args>")
	}
}

// Error logs the given error message and error to the given writer.
func (*defaultLogger) Error(writer io.Writer, message string, err error) {
	switch {
	case err != nil && message != "":
		fmt.Fprintf(writer, "error: %s: %v\n", message, err)
	case message != "":
		fmt.Fprintf(writer, "error: %s\n", message)
	case err != nil:
		fmt.Fprintf(writer, "error: %v\n", err)
	default:
		fmt.Fprintf(writer, "error: %s\n", "<no-error>")
	}
}

// Message logs the given message to the given writer.
func (*defaultLogger) Message(writer io.Writer, message string) {
	if len(message) == 0 || message[len(message)-1] != '\n' {
		fmt.Fprintf(writer, "%s\n", message)
	} else {
		fmt.Fprint(writer, message)
	}
}
