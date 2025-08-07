// Package cmd provides a common interface for executing commands.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Mode represents the execution mode for commands.
type Mode int

const (
	// Attached mode - process output is attached to parent.
	Attached Mode = 0x00
	// Detached mode - process output is detached from parent.
	Detached Mode = 0x01
	// Background mode - process runs in the background.
	Background Mode = 0x02
)

// ErrCmd is a sentinel error for command execution failures.
var ErrCmd = errors.New("command")

// CmdError represents a command execution error with full context information.
type CmdError struct {
	// Message provides additional context about the failure
	Message string
	// Dir is the working directory where the command was executed
	Dir string
	// Env contains the environment variables passed to the command
	Env []string
	// Args contains the command arguments
	Args []string
	// Cause is the underlying error that caused the failure
	Cause error
}

// NewCmdError creates a new command execution error with failure context
// information, including an error message, as well as the working directory,
// environment variables, command arguments, and underlying error.
func NewCmdError(
	msg, dir string, env, args []string, cause error,
) error {
	return &CmdError{
		Message: msg,
		Dir:     dir,
		Env:     env,
		Args:    args,
		Cause:   cause,
	}
}

// Error returns a formatted error message with full context.
func (e *CmdError) Error() string {
	return fmt.Sprintf("%v - %s [dir=%s, env=%v, call=%v]: %v",
		ErrCmd, e.Message, e.Dir, e.Env, e.Args, e.Cause)
}

// Unwrap returns the underlying cause error for error unwrapping.
func (e *CmdError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for `errors.Is()`.
func (*CmdError) Is(target error) bool {
	return errors.Is(target, ErrCmd)
}

// Executor provides a common interface for executing commands.
type Executor interface {
	// Exec executes the command provided by given arguments in given working
	// directory using stdin as input while redirecting stdout and stderr to
	// given writers.
	Exec(mode Mode, stdin io.Reader, stdout, stderr io.Writer,
		dir string, env []string, args ...string) error
}

// executor provides a default command executor using `os/exec`
// supporting optional tracing.
type executor struct {
	devnull string
}

// NewExecutor creates a new default command process.
func NewExecutor() Executor {
	return &executor{
		devnull: os.DevNull,
	}
}

// Exec executes the command provided by given arguments in given working
// directory using stdin as input while redirecting stdout and stderr to given
// writers.
func (e *executor) Exec( //revive:disable-line:argument-limit
	mode Mode, stdin io.Reader, stdout, stderr io.Writer,
	dir string, env []string, args ...string,
) error {
	// #nosec G204 -- caller ensures safe commands
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir, cmd.Env = dir, os.Environ()
	cmd.Env = append(cmd.Env, env...)

	if mode&0x1 == Detached {
		devnull, err := os.OpenFile(e.devnull, os.O_RDWR, 0)
		if err != nil {
			return NewCmdError("opening /dev/null", dir, env, args, err)
		}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = devnull, devnull, devnull
		defer devnull.Close()
	} else {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	}

	if mode&0x2 == Background {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			return NewCmdError("starting process", dir, env, args, err)
		} else if err := cmd.Process.Release(); err != nil {
			// #no-cover: difficult to test release error handling is it
			// requires a process that is already terminated before being
			// released reliably.
			return NewCmdError("releasing process", dir, env, args, err)
		}
	} else if err := cmd.Run(); err != nil {
		return NewCmdError("starting process", dir, env, args, err)
	}

	return nil
}
