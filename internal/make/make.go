// Package make provides the go-make command implementation.
package make //nolint:predeclared // package name is make.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-make/internal/info"
	"github.com/tkrop/go-make/internal/log"
)

const (
	GoMakeConfig = "GOMAKE_CONFIG"
	// GitSha1HashLen is the full length of sha1-hashes used in git.
	GitFullHashLen = 40

	// Makefile base makefile to be executed by go-make.
	Makefile = "Makefile.base"
	// bashCompletion contains the bash completion script for go-make.
	BashCompletion = "### bash completion for go-make\n" +
		"function __complete_go-make() {\n" +
		"	COMPREPLY=($(compgen -W \"$(go-make targets 2>/dev/null)\"" +
		" -- \"${COMP_WORDS[COMP_CWORD]}\"));\n" +
		"}\n" +
		"complete -F __complete_go-make go-make;\n"
)

// Available exit code constants.
const (
	ExitSuccess       int = 0
	ExitConfigFailure int = 1
	ExitExecFailure   int = 2
)

// Regexp for filtering entering and leaving lines from make output.
// var makeFilterRegexp = regexp.MustCompile(
// 	"make\\[[0-9]+\\]: (Entering|Leaving) directory '[^\n]*'\n")

// MakeFilter is a custom filter that implements the io.Writer interface.
// type MakeFilter struct {
// 	writer io.Writer
// 	filter *regexp.Regexp
// 	buf    []byte
// }

// NewMakeFilter creates a new make filter using the given writer.
// func NewMakeFilter(writer io.Writer) *MakeFilter {
// 	return &MakeFilter{
// 		writer: writer,
// 		filter: makeFilterRegexp,
// 		buf:    []byte{},
// 	}
// }

// Write writes the given data to the underlying writer.
// func (f *MakeFilter) Write(data []byte) (int, error) {
// 	s := 0
// 	for c := 0; c < len(data); c++ {
// 		if data[c] != '\n' {
// 			continue
// 		}

// 		f.buf = append(f.buf, data[s:c+1]...)
// 		if f.filter.Match(f.buf) {
// 			fmt.Fprintf(os.Stdout, "skip: %s", string(f.buf))
// 			f.buf = []byte{}
// 			continue
// 		} else if i, err := f.writer.Write(f.buf); err != nil {
// 			fmt.Fprintf(os.Stdout, "error: %d %v", i, err)
// 			return i, err
// 		}
// 		fmt.Fprintf(os.Stdout, "write: %s", string(f.buf))
// 		f.buf = []byte{}
// 		s = c + 1
// 	}
// 	f.buf = append(f.buf, data[s:]...)
// 	return s, nil
// }

// CmdGoInstall creates the argument array of a `go install <path>@<version>`
// command.
func CmdGoInstall(path, version string) []string {
	return []string{
		"go", "install", "-v", "-mod=readonly",
		"-buildvcs=true", path + "@" + version,
	}
}

// CmdTestDir creates the argument array of a `test -d <path>` command. We
// majorly use this to test if a directory exists, since it allows us to mock
// the check.
func CmdTestDir(path string) []string {
	return []string{"test", "-d", path}
}

// CmdMakeTargets creates the argument array of a `make --file <Makefile>
// <targets...>` command using the given makefile name amd argument list.
func CmdMakeTargets(file string, args ...string) []string {
	return append([]string{
		"make", "--file", file, "--no-print-directory",
	}, args...)
}

// GoMakePath returns the path to the go-make config directory.
func GoMakePath(path, version string) string {
	return filepath.Join(os.Getenv("GOPATH"),
		"pkg", "mod", path+"@"+version, "config")
}

// GoMake provides the default `go-make` application context.
type GoMake struct {
	// Info provides the build information of go-make.
	Info *info.Info
	// Executor provides the command executor.
	Executor cmd.Executor
	// Logger provides the logger.
	Logger log.Logger
	// Stdout provides the standard output writer.
	Stdout io.Writer
	// Stderr provides the standard error writer.
	Stderr io.Writer

	// The actual working directory.
	WorkDir string
	// The version of the go-make config.
	ConfigVersion string
	// The directory of the go-make config.
	ConfigDir string
	// The path to the go-make config Makefile.
	Makefile string
	// Trace provides the flags to trace calls.
	Trace bool
}

// NewGoMake returns a new default `go-make` application context with given
// standard output writer, standard error writer, and trace flag.
func NewGoMake(
	info *info.Info, config string, stdout, stderr io.Writer,
) *GoMake {
	return (&GoMake{
		Info:     info,
		Executor: cmd.NewExecutor(),
		Logger:   log.NewLogger(),
		Stdout:   stdout,
		Stderr:   stderr,
	}).setupConfig(config)
}

// setupConfig sets up the go-make config directory and base makefile.
func (gm *GoMake) setupConfig(config string) *GoMake {
	// ---revive:disable-next-line:redefines-builtin-id // Is package name.
	gm.WorkDir, _ = os.Getwd()
	if config != "" {
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir,
			CmdTestDir(config)...); err != nil {
			gm.ConfigVersion = config
			gm.ConfigDir = GoMakePath(gm.Info.Path, config)
		} else {
			gm.ConfigVersion = "latest"
			gm.ConfigDir = config
		}
	} else {
		gm.ConfigVersion = gm.Info.Version
		gm.ConfigDir = GoMakePath(gm.Info.Path, gm.Info.Version)
	}
	gm.Makefile = filepath.Join(gm.ConfigDir, Makefile)

	return gm
}

// ensureConfig ensures the go-make config is available.
func (gm *GoMake) ensureConfig() error {
	if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir,
		CmdTestDir(gm.ConfigDir)...); err != nil {
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir,
			CmdGoInstall(gm.Info.Path, gm.ConfigVersion)...); err != nil {
			return NewErrNotFound(gm.Info.Path, gm.ConfigVersion, err)
		}
	}
	return nil
}

// makeTargets executes the provided make targets.
func (gm *GoMake) makeTargets(args ...string) error {
	return gm.exec(gm.Stdout, gm.Stderr, gm.WorkDir,
		CmdMakeTargets(gm.Makefile, args...)...)
}

// Executes the command with given name and arguments in given directory
// calling the command executor taking care to wrap the resulting error.
func (gm *GoMake) exec(
	stdout, stderr io.Writer, dir string, args ...string,
) error {
	if gm.Trace {
		gm.Logger.Exec(stderr, dir, args...)
	}

	if err := gm.Executor.Exec(stdout, stderr, dir, args...); err != nil {
		return NewErrCallFailed(args, err)
	}
	return nil
}

// Make runs the go-make command with given arguments and return exit code.
func (gm *GoMake) Make(args ...string) (int, error) {
	var targets []string
	for _, arg := range args {
		switch {
		case arg == "--trace":
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
			targets = append(targets, arg)
			gm.Trace = true

		case arg == "--version":
			gm.Logger.Info(gm.Stdout, gm.Info, true)
			return 0, nil

		case strings.HasPrefix(arg, "--completion"):
			gm.Logger.Message(gm.Stdout, BashCompletion)
			return 0, nil

		case strings.HasPrefix(arg, "--config="):
			gm.setupConfig(arg[len("--config="):])

		default:
			targets = append(targets, arg)
		}
	}

	if err := gm.ensureConfig(); err != nil {
		if !gm.Trace {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "ensure config", err)
		return ExitConfigFailure, err
	} else if err := gm.makeTargets(targets...); err != nil {
		if !gm.Trace {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "execute make", err)
		return ExitExecFailure, err
	}
	return ExitSuccess, nil
}

// ErrNotFound represent a version not found error.
var ErrNotFound = errors.New("version not found")

// NewErrNotFound wraps the error of failed command to install the requested
// go-mock config version.
func NewErrNotFound(path, version string, err error) error {
	return fmt.Errorf("%w [path=%s, version=%s]: %w",
		ErrNotFound, path, version, err)
}

// NewErrCallFailed wraps the error of a failed command call.
func NewErrCallFailed(args []string, err error) error {
	return fmt.Errorf("call failed [name=%s, args=%v]: %w",
		args[0], args[1:], err)
}

// Make runs the go-make command with given build information, standard output
// writer, standard error writer, and command arguments.
func Make(
	info *info.Info, config string,
	stdout, stderr io.Writer, args ...string,
) int {
	exit, _ := NewGoMake(
		info, config,
		// TODO we would like to filter some make file startup specific
		// output that creates hard to validate output.
		// NewMakeFilter(stdout), NewMakeFilter(stderr), info,
		stdout, stderr,
	).Make(args[1:]...)

	return exit
}
