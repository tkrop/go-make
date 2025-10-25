// Package cmd provides a common interface for executing commands.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
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

// Cmd represents a command to be executed.
type Cmd struct {
	// Mode contains the execution mode.
	Mode Mode
	// Dir contains the working directory.
	Dir string
	// Env contains the environment variables.
	Env []string
	// Args contains the command arguments.
	Args []string
	// Stdin is the input stream for the command.
	Stdin io.Reader
	// Stdout is the output stream for the command.
	Stdout io.Writer
	// Stderr is the error stream for the command.
	Stderr io.Writer

	// exec is the command exec.
	exec Executor
}

// New creates a new command with the given arguments.
func New(args ...string) *Cmd {
	return &Cmd{
		Mode:   Attached,
		Dir:    ".",
		Env:    []string{},
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// IsMode returns whether the execution mode of the command includes the
// specified mode.
func (c *Cmd) IsMode(mode Mode) bool {
	if c != nil {
		return c.Mode&mode == mode
	}
	return false
}

// WithMode sets the execution mode for the command.
func (c *Cmd) WithMode(mode Mode) *Cmd {
	if c != nil {
		c.Mode = mode
	}
	return c
}

// WithWorkDir sets the working directory for the command.
func (c *Cmd) WithWorkDir(dir string) *Cmd {
	if c != nil {
		c.Dir = dir
	}
	return c
}

// WithEnv adds environment variables to the command.
func (c *Cmd) WithEnv(env ...string) *Cmd {
	if c != nil {
		c.Env = append(c.Env, env...)
	}
	return c
}

// WithArgs adds arguments to the command.
func (c *Cmd) WithArgs(args ...string) *Cmd {
	if c != nil {
		c.Args = append(c.Args, args...)
	}
	return c
}

// WithIO sets the input, output, and error streams for the command.
func (c *Cmd) WithIO(stdin io.Reader, stdout, stderr io.Writer) *Cmd {
	if c != nil {
		c.Stdin = stdin
		c.Stdout = stdout
		c.Stderr = stderr
	}
	return c
}

// WithStdin sets the input stream for the command.
func (c *Cmd) WithStdin(stdin io.Reader) *Cmd {
	if c != nil {
		c.Stdin = stdin
	}
	return c
}

// WithStdout sets the output stream for the command.
func (c *Cmd) WithStdout(stdout io.Writer) *Cmd {
	if c != nil {
		c.Stdout = stdout
	}
	return c
}

// WithStderr sets the error stream for the command.
func (c *Cmd) WithStderr(stderr io.Writer) *Cmd {
	if c != nil {
		c.Stderr = stderr
	}
	return c
}

// WithExecutor sets the executor for the command.
func (c *Cmd) WithExecutor(exec Executor) *Cmd {
	if c != nil {
		c.exec = exec
	}
	return c
}

// Copy creates a copy of the command to allow command templating.
func (c *Cmd) Copy() *Cmd {
	if c == nil {
		return nil
	}

	return &Cmd{
		Args:   append([]string{}, c.Args...),
		Env:    append([]string{}, c.Env...),
		Dir:    c.Dir,
		Mode:   c.Mode,
		Stdin:  c.Stdin,
		Stdout: c.Stdout,
		Stderr: c.Stderr,
		exec:   c.exec,
	}
}

// Executor returns the executor of the command. If no executor is set, a new
// default executor is created.
func (c *Cmd) Executor() Executor {
	if c == nil || c.exec == nil || reflect.ValueOf(c.exec).IsNil() {
		return NewExecutor()
	}
	return c.exec
}

// Exec executes the command using the provided context. If the command has no
// executor set, a new default executor is created for execution.
func (c *Cmd) Exec(ctx context.Context) error {
	return c.Executor().Exec(ctx, c) //nolint:wrapcheck // is wrapped.
}

// Error creates a new command error with error message and causing error. The
// basic command in the command error is stripped of I/O streams and executor
// for clarity and testability.
func (c *Cmd) Error(msg string, cause error) error {
	if c == nil {
		return &CmdError{
			Message: msg,
			Cause:   cause,
		}
	}
	return &CmdError{
		Cmd: New(c.Args...).WithMode(c.Mode).
			WithWorkDir(c.Dir).WithEnv(c.Env...),
		Message: msg,
		Cause:   cause,
	}
}

// ErrCmd is a sentinel error for command execution failures.
var ErrCmd = errors.New("command")

// CmdError represents a command execution error with full context information.
type CmdError struct {
	// Cmd is the command that caused the failure.
	Cmd *Cmd
	// Message provides additional context about the failure
	Message string
	// Cause is the underlying error that caused the failure
	Cause error
}

// NewCmdError creates a new command error with the given message, command,
// and cause.
func NewCmdError(message string, cmd *Cmd, cause error) *CmdError {
	return &CmdError{
		Cmd:     cmd,
		Message: message,
		Cause:   cause,
	}
}

// Error returns a formatted error message with full context.
func (e *CmdError) Error() string {
	if e.Cmd == nil {
		return fmt.Sprintf("%v - %s: %v", ErrCmd, e.Message, e.Cause)
	}
	return fmt.Sprintf("%v - %s [dir=%s, env=%v, call=%v]: %v",
		ErrCmd, e.Message, e.Cmd.Dir, e.Cmd.Env, e.Cmd.Args, e.Cause)
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
	// Exec executes the given command with provided context using the executor.
	Exec(ctx context.Context, cmd *Cmd) error
	// New creates a new command with the given arguments.
	New(args ...string) *Cmd
}

// CmdExecutor provides a default command CmdExecutor using `os/exec`
// supporting optional tracing.
type CmdExecutor struct {
	devnull string
	start   func(mode Mode, cmd *exec.Cmd) error
	finish  func(mode Mode, cmd *exec.Cmd) error
}

// NewExecutor creates a new default command process.
func NewExecutor() *CmdExecutor {
	return &CmdExecutor{
		devnull: os.DevNull,
		start: func(mode Mode, cmd *exec.Cmd) error {
			if mode&Background == Background {
				cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			}
			return cmd.Start()
		},
		finish: func(mode Mode, cmd *exec.Cmd) error {
			if mode&Background != Background {
				return cmd.Wait()
			}
			return cmd.Process.Release()
		},
	}
}

// Exec executes the given command with provided context using the executor.
func (e *CmdExecutor) Exec(ctx context.Context, cmd *Cmd) error {
	if e == nil {
		return cmd.Error("nil executor", nil)
	} else if cmd == nil {
		return cmd.Error("nil command", nil)
	}

	// #nosec G204 -- caller ensures safe commands
	cc := exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	cc.Dir, cc.Env = cmd.Dir, os.Environ()
	cc.Env = append(cc.Env, cmd.Env...)

	if cmd.IsMode(Detached) {
		devnull, err := os.OpenFile(e.devnull, os.O_RDWR, 0)
		if err != nil {
			return cmd.Error("opening /dev/null", err)
		}
		cc.Stdin, cc.Stdout, cc.Stderr = devnull, devnull, devnull
		defer devnull.Close()
	} else {
		cc.Stdin, cc.Stdout, cc.Stderr = cmd.Stdin, cmd.Stdout, cmd.Stderr
	}

	if err := e.start(cmd.Mode, cc); err != nil {
		return cmd.Error("starting process", err)
	} else if err := e.finish(cmd.Mode, cc); err != nil {
		return cmd.Error("releasing process", err)
	}
	return nil
}

// New creates a new command with the given arguments.
func (e *CmdExecutor) New(args ...string) *Cmd {
	return New(args...).WithExecutor(e)
}
