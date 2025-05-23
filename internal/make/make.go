// Package make provides the go-make command implementation.
package make //nolint:predeclared // package name is make.

import (
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tkrop/go-config/info"
	"github.com/tkrop/go-make/internal/cmd"
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
	// GoMakeTargetsFile provides the name of the file to store the go-make
	// targets.
	GoMakeTargetsFile = "${TMPDIR:-/tmp}/go-make-${USER:-$(whoami)}/${PWD}/targets"
	// CompleteTargetFunc provides the common target function to create
	// go-make targets for completion.
	CompleteTargetFunc = "_go-make-targets() {\n" +
		"    if [ -z \"$(grep '$(GOBIN)/go-make show-targets' Makefile)\" ]; then\n" +
		"        mkdir -p \"$(dirname ${1})\";\n" +
		"        make --no-builtin-rules --no-builtin-variables \\\n" +
		"            --print-data-base --question | awk -v RS=\"\" -F\":\" '\n" +
		"        /(^|\\n)# Files(\\n|$)/,/(^|\\n)# Finished / {\n" +
		"            if ($1 !~ \"^[#./]\") { print $1 }\n" +
		"        }' | LC_ALL=C sort --unique | tee ${1};\n" +
		"    else go-make show-targets; fi 2>/dev/null;\n" +
		"};\n"
	// CompleteFilterFunc provides the common filter function to filter
	// go-make targets before applying completion.
	CompleteFilterFunc = "_go-make-filter() {\n" +
		"    sed -E -e \"s|^(${1}[^/-]*[-/]?)?.*|\\1|g\"" +
		" | sort --unique;\n" +
		"};\n"
	// CompleteBash provides the bash completion setup for go-make.
	CompleteBash = "### bash completion for go-make\n" +
		"function " + CompleteTargetFunc +
		"function " + CompleteFilterFunc +
		"function __complete_go-make() {\n" +
		"    local WORD=\"${COMP_WORDS[COMP_CWORD]}\";\n" +
		"    local FILE=\"" + GoMakeTargetsFile + "\";\n" +
		"    if [ -f \"${FILE}\" ]; then\n" +
		"        local WORDS=\"$(cat \"${FILE}\" | _go-make-filter \"${WORD}\")\";\n" +
		"        ( _go-make-targets \"${FILE}\" >/dev/null & ) 2>/dev/null;\n" +
		"    else\n" +
		"        local WORDS=\"$(_go-make-targets \"${FILE}\" | _go-make-filter \"${WORD}\")\";\n" +
		"    fi;\n" +
		"    COMPREPLY=($(compgen -W \"${WORDS}\" -- \"${WORD}\"));\n" +
		"};\n" +
		"complete -F __complete_go-make go-make;\n"
	// CompleteZsh provides the zsh completion setup for go-make.
	CompleteZsh = "### zsh completion for make/go-make\n" +
		CompleteTargetFunc + CompleteFilterFunc +
		"__complete_go-make() {\n" +
		"    local targets=();\n" +
		"    local FILE=\"" + GoMakeTargetsFile + "\";\n" +
		"    if [ -f \"${FILE}\" ]; then\n" +
		"        targets=($(cat \"${FILE}\" | _go-make-filter \"${words[-1]}\"));\n" +
		"        ( _go-make-targets \"${FILE}\" >/dev/null & ) 2>/dev/null;\n" +
		"    else\n" +
		"        targets=($(_go-make-targets \"${FILE}\" | _go-make-filter \"${words[-1]}\"));\n" +
		"    fi;\n" +
		"    _describe 'go-make' targets;\n" +
		"};\n" +
		"compdef __complete_go-make go-make;\n" +
		"compdef __complete_go-make make;\n"
)

// Available exit code constants.
const (
	ExitSuccess       int = 0
	ExitConfigFailure int = 2
	ExitTargetFailure int = 3
)

// AbsPath returns the absolute path of given directory.
func AbsPath(path string) string {
	path, _ = filepath.Abs(path)
	return path
}

// EvalSymlinks returns the evaluated path of given directory.
func EvalSymlinks(path string) string {
	path, _ = filepath.EvalSymlinks(path)
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
	// Stdin provides the standard input reader.
	Stdin io.Reader
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

// NewGoMake returns a new default `go-make` service context with given
// standard input reader, standard output writer, standard error writer, build
// information, config setup, working directory and environment variables.
func NewGoMake( //revive:disable-line:argument-limit // kiss.
	stdin io.Reader, stdout, stderr io.Writer,
	info *info.Info, config, wd string, env ...string,
) *GoMake {
	return (&GoMake{
		Info:     info,
		Executor: cmd.NewExecutor(),
		Logger:   log.NewLogger(),
		Stdin:    stdin,
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
func (gm *GoMake) setupWorkDir() {
	buffer := &strings.Builder{}
	if err := gm.exec(nil, buffer, gm.Stderr,
		gm.WorkDir, gm.Env, CmdGitTop()...); err == nil {
		gm.WorkDir = strings.TrimSpace(buffer.String())
	}
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
	if err := gm.exec(nil, gm.Stderr, gm.Stderr,
		gm.WorkDir, gm.Env, CmdTestDir(path)...); err != nil {
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

	if err := gm.exec(nil, gm.Stderr, gm.Stderr,
		gm.WorkDir, gm.Env, CmdTestDir(gm.ConfigDir)...); err != nil {
		if err := gm.exec(nil, gm.Stderr, gm.Stderr, gm.WorkDir,
			gm.Env, CmdGoInstall(gm.Info.Path, gm.ConfigVersion)...); err != nil {
			return NewErrNotFound(gm.Info.Path, gm.ConfigVersion, err)
		}
	}
	return nil
}

// CmdArgRegex is the regular expression that match commands with arguments
// that will be transformed into a `ARGS` variable.
var CmdArgRegex = regexp.MustCompile(
	`^(call|show|git-|test-|lint|run-|version-|update).*$`)

// makeTargets executes the provided make targets.
func (gm *GoMake) makeTargets(args ...string) error {
	for index, arg := range args {
		if CmdArgRegex.MatchString(arg) && index < len(args)-1 {
			return gm.exec(gm.Stdin, gm.Stdout, gm.Stderr, gm.WorkDir,
				append(gm.Env, "ARGS="+strings.Join(args[index+1:], " ")),
				CmdMakeTargets(gm.Makefile, args[0:index+1]...)...)
		}
	}
	return gm.exec(gm.Stdin, gm.Stdout, gm.Stderr,
		gm.WorkDir, gm.Env, CmdMakeTargets(gm.Makefile, args...)...)
}

// Executes the command with given name and arguments in given directory
// calling the command executor taking care to wrap the resulting error.
func (gm *GoMake) exec(
	stdin io.Reader, stdout, stderr io.Writer,
	dir string, env []string, args ...string,
) error {
	if gm.Trace {
		gm.Logger.Exec(stderr, dir, args...)
	}

	if err := gm.Executor.Exec(stdin, stdout, stderr,
		dir, env, args...); err != nil {
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

		case strings.HasPrefix(arg, "--completion=bash"):
			gm.Logger.Message(gm.Stdout, CompleteBash)
			return 0, nil

		case strings.HasPrefix(arg, "--completion=zsh"):
			gm.Logger.Message(gm.Stdout, CompleteZsh)
			return 0, nil

		case strings.HasPrefix(arg, "--config="):
			gm.Config = arg[len("--config="):]

		default:
			targets = append(targets, arg)
		}
	}

	gm.setupWorkDir()
	if err := gm.setupConfig(); err != nil {
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
	stdin io.Reader, stdout, stderr io.Writer, info *info.Info,
	config, wd string, env []string, args ...string,
) int {
	exit, _ := NewGoMake(
		stdin, stdout, stderr, info, config, wd, env...,
	).Make(args[1:]...)

	return exit
}
