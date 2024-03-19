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

// // ExecPty executes the command with given name and arguments in given
// // directory using a pseudo terminal defined by the given file for input
// // and output.
// func (e *defaultExecutor) ExecPty(
// 	stdin, stdout, stderr *os.File,
// 	dir string, env []string, args ...string,
// ) error {
// 	// #nosec G204 -- caller ensures safe commands
// 	cmd := exec.Command(args[0], args[1:]...)
// 	cmd.Dir, cmd.Env = dir, os.Environ()
// 	cmd.Env = append(cmd.Env, env...)

// 	if ptmx, err := pty.Start(cmd); err == nil {
// 		resetTermResize := e.ptyTermResize(ptmx, stdin, stderr)
// 		resetTermRaw := e.ptyTermRaw(stdin)

// 		defer func() {
// 			resetTermResize()
// 			resetTermRaw()
// 			_ = ptmx.Close()
// 		}()

// 		eg := errgroup.Group{}
// 		eg.Go(func() error {
// 			_, err := io.Copy(stdin, ptmx)
// 			return err //nolint:wrapcheck // checked on next layer
// 		})
// 		eg.Go(func() error {
// 			_, err := io.Copy(ptmx, stdout)
// 			return err //nolint:wrapcheck // checked on next layer
// 		})
// 		return eg.Wait() //nolint:wrapcheck // checked on next layer
// 	} else {
// 		return err //nolint:wrapcheck // checked on next layer
// 	}
// }

// // ptyTermResize connects the pseudo terminal to the given file descriptor to
// // support resizing of the terminal. It returns a cleanup function that needs
// // to be called when the pseudo terminal is closed.
// func (*defaultExecutor) ptyTermResize(
// 	ptmx *os.File, stdin, stderr *os.File,
// ) func() {
// 	ch := make(chan os.Signal, 1)
// 	signal.Notify(ch, syscall.SIGWINCH)

// 	go func() {
// 		for range ch {
// 			if err := pty.InheritSize(stdin, ptmx); err != nil {
// 				fmt.Fprintf(stderr, "error resizing pty: %s", err)
// 			}
// 		}
// 	}()

// 	ch <- syscall.SIGWINCH

// 	return func() { signal.Stop(ch); close(ch) }
// }

// // ptyTermRaw enables the raw mode of the given file descriptor and returns a
// // clean up function that needs to be called when the pseudo terminal is
// // closed.
// func (*defaultExecutor) ptyTermRaw(file *os.File) func() {
// 	if state, err := term.MakeRaw(int(file.Fd())); err == nil {
// 		return func() { _ = term.Restore(int(file.Fd()), state) }
// 	} else {
// 		panic(err)
// 	}
// }
