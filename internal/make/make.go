// Package make provides the go-make command implementation.
package make //nolint:predeclared // package name is make.

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-make/internal/info"
	"github.com/tkrop/go-make/internal/log"
)

const (
	// EnvGoMakeConfig provides the name of the go-make config environment
	// variable.
	EnvGoMakeConfig = "GOMAKE_CONFIG"
	// EnvGoPath provides the name of the genera go path environment variable.
	EnvGoPath = "GOPATH"
	// Makefile provides the name of the base makefile to be executed by
	// go-make.
	Makefile = "Makefile.base"
	// bashCompletion provides the bash completion script for go-make.
	BashCompletion = "### bash completion for go-make\n" +
		"function __complete_go-make() {\n" +
		"	COMPREPLY=($(compgen -W \"$(go-make show-targets 2>/dev/null)\" \\\n" +
		"		-- \"${COMP_WORDS[COMP_CWORD]}\"));\n" +
		"}\n" +
		"complete -F __complete_go-make go-make;\n"
)

// Available exit code constants.
const (
	ExitSuccess       int = 0
	ExitGitFailure    int = 1
	ExitConfigFailure int = 2
	ExitTargetFailure int = 3
)

// AbsPath returns the absolute path of given directory.
func AbsPath(dir string) string {
	path, _ := filepath.Abs(dir)
	return path
}

// CmdGoInstall creates the argument array of a `go install <path>@<version>`
// command.
func CmdGoInstall(path, version string) []string {
	return []string{
		"go", "install", "-v", "-mod=readonly",
		"-buildvcs=true", path + "@" + version,
	}
}

// CmdTestDir creates the argument array of a `test -d <path>` command. We
// use this to test if a directory exists, since it allows us to mock the
// check.
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

// CmdGitTop creates the argument array of a `git rev-parse` command to
// get the root path of the current git repository.
func CmdGitTop() []string {
	return []string{"git", "rev-parse", "--show-toplevel"}
}

// GetEnvDefault returns the value of the environment variable with given key
// or the given default value, if the environment variable is not set.
func GetEnvDefault(key, value string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return value
}

// GoMakePath returns the path to the go-make config directory.
func GoMakePath(path, version string) string {
	return filepath.Join(GetEnvDefault(EnvGoPath, build.Default.GOPATH),
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
	// Config provides the go-make config argument.
	Config string
	// Env provides the additional environment variables.
	Env []string

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
// standard output writer, standard error writer, config setup, and working
// directory.
func NewGoMake(
	stdout, stderr io.Writer, info *info.Info, config, wd string,
	env ...string,
) *GoMake {
	return (&GoMake{
		Info:     info,
		Executor: cmd.NewExecutor(),
		Logger:   log.NewLogger(),
		Stdout:   stdout,
		Stderr:   stderr,
		Config:   config,
		WorkDir:  wd,
		Env:      env,
	})
}

// setupWorkDir ensures that the working directory is setup to the root of the
// current git repository since this is where the go-make targets should be
// executed.
func (gm *GoMake) setupWorkDir() error {
	buffer := &strings.Builder{}
	if err := gm.exec(buffer, gm.Stderr, gm.WorkDir, gm.Env,
		CmdGitTop()...); err != nil {
		return err
	}
	gm.WorkDir = strings.TrimSpace(buffer.String())
	return nil
}

// setupConfig sets up the go-make config by evaluating the director or version
// as provided by the command line arguments, the environment variables, or the
// context of the executed go-make command. The setup ensures that the expected
// go-make config is installed and the correct Makefile is referenced.
func (gm *GoMake) setupConfig() error {
	if gm.Config == "" {
		return gm.ensureConfig(gm.Info.Version,
			GoMakePath(gm.Info.Path, gm.Info.Version))
	}

	path := AbsPath(gm.Config)
	if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir, gm.Env,
		CmdTestDir(path)...); err != nil {
		return gm.ensureConfig(gm.Config,
			GoMakePath(gm.Info.Path, gm.Config))
	}
	return gm.ensureConfig("custom", path)
}

// ensureConfig ensures that the go-make config is valid and installed and
// the correct Makefile is referenced.
func (gm *GoMake) ensureConfig(version, dir string) error {
	gm.ConfigVersion, gm.ConfigDir = version, dir
	gm.Makefile = filepath.Join(dir, Makefile)
	if version == "custom" {
		return nil
	}

	if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir, gm.Env,
		CmdTestDir(gm.ConfigDir)...); err != nil {
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir, gm.Env,
			CmdGoInstall(gm.Info.Path, gm.ConfigVersion)...); err != nil {
			return NewErrNotFound(gm.Info.Path, gm.ConfigVersion, err)
		}
	}
	return nil
}

// makeTargets executes the provided make targets.
func (gm *GoMake) makeTargets(args ...string) error {
	return gm.exec(gm.Stdout, gm.Stderr, gm.WorkDir, gm.Env,
		CmdMakeTargets(gm.Makefile, args...)...)
}

// Executes the command with given name and arguments in given directory
// calling the command executor taking care to wrap the resulting error.
func (gm *GoMake) exec(
	stdout, stderr io.Writer, dir string, env []string, args ...string,
) error {
	if gm.Trace {
		gm.Logger.Exec(stderr, dir, args...)
	}

	if err := gm.Executor.Exec(stdout, stderr, dir, env, args...); err != nil {
		return NewErrCallFailed(dir, args, err)
	}
	return nil
}

// Make runs the go-make command with given arguments and return the exit code
// and error.
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
			gm.Config = arg[len("--config="):]

		default:
			targets = append(targets, arg)
		}
	}

	if err := gm.setupWorkDir(); err != nil {
		gm.Logger.Error(gm.Stderr, "ensure top", err)
		return ExitGitFailure, err
	} else if err := gm.setupConfig(); err != nil {
		gm.Logger.Error(gm.Stderr, "ensure config", err)
		return ExitConfigFailure, err
	} else if err := gm.makeTargets(targets...); err != nil {
		gm.Logger.Error(gm.Stderr, "execute make", err)
		return ExitTargetFailure, err
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
func NewErrCallFailed(path string, args []string, err error) error {
	return fmt.Errorf("call failed [path=%s, call=%s]: %w", path, args, err)
}

// Make runs the go-make command with given build information, standard output
// writer, standard error writer, and command arguments.
func Make( //revive:disable-line:argument-limit // ensures testability.
	stdout, stderr io.Writer, info *info.Info,
	config, wd string, env []string, args ...string,
) int {
	exit, _ := NewGoMake(
		stdout, stderr, info, config, wd, env...,
	).Make(args[1:]...)

	return exit
}
