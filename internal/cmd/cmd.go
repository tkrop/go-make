// Package cmd provides a common interface for executing commands.
package cmd

import (
	"io"
	"os"
	"os/exec"
)

// Executor provides a common interface for executing commands.
type Executor interface {
	// Exec executes the command provided by given arguments in given working
	// directory using stdin as input while redirecting stdout and stderr to
	// given writers.
	Exec(stdin io.Reader, stdout, stderr io.Writer,
		dir string, env []string, args ...string) error
	// ExecPty executes the command with given name and arguments in given
	// directory using a pseudo terminal defined by the given file for input
	// and output.
	// ExecPty(stdin, stdout, stderr *os.File,
	// 	dir string, env []string, args ...string) error
}

// defaultExecutor provides a default command executor using `os/exec`
// supporting optional tracing.
type defaultExecutor struct{}

// NewExecutor creates a new default command executor.
func NewExecutor() Executor {
	return &defaultExecutor{}
}

// Exec executes the command provided by given arguments in given working
// directory using stdin as input while redirecting stdout and stderr to given
// writers.
func (*defaultExecutor) Exec(
	stdin io.Reader, stdout, stderr io.Writer,
	dir string, env []string, args ...string,
) error {
	// #nosec G204 -- caller ensures safe commands
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir, cmd.Env = dir, os.Environ()
	cmd.Env = append(cmd.Env, env...)

	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	return cmd.Run() //nolint:wrapcheck // checked on next layer
}
