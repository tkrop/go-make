// Package make provides the go-make command implementation.
package make //nolint:predeclared // package name is make.

import (
	"context"
	"errors"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/tkrop/go-config/info"
	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-make/internal/log"
	"github.com/tkrop/go-make/internal/sys"
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
	// GoMakeCompletion provides the common completion options for the
	// go-make command.
	GoMakeCompletion = "bash zsh"
	// GoMakeOutputSync provides the common output sync options for the
	// go-make command.
	GoMakeOutputSync = "none line target recurse"
	// GoMakeTargetsDir provides the base name of the file to cache the
	// go-make targets.
	GoMakeTargetsDir = "${TMPDIR:-/tmp}/go-make-${USER:-$(whoami)}${PWD}"
	// CompleteFilterFunc provides the common filter function to filter
	// go-make targets before applying completion.
	CompleteFilterFunc = "_go-make-filter() {\n" +
		"    awk -v prefix=\"^${1}\" -v pat=\"[-/]\" '\n" +
		"        function min(x, y, l) {\n" +
		"            while (substr(x, 0, l) != substr(y, 0, l)) { l--; }\n" +
		"            return l;\n" +
		"        }\n" +
		"        function short(x, l) {\n" +
		"            return (substr(x, 0, 2) == \"--\") ? x :\n" +
		"                (m = match(substr(x, l+1), pat)) ? substr(x, 0, l+m) : x;\n" +
		"        }\n" +
		"        ($0 ~ prefix) {\n" +
		"            if (n == 0) { array[n++] = $0; l = length($0); next; }\n" +
		"            k = min(array[n-1], $0, l); y = short($0, k); a = 0;\n" +
		"            for (i = n-1; i >= 0; i--) {\n" +
		"                if (l != k) {\n" +
		"                    x = short(array[i], k); array[i] = x;\n" +
		"                } else { x = array[i]; }\n" +
		"                if (x == y) { a++; }\n" +
		"            }\n" +
		"            if (a == 0) { array[n++] = y; }; l = k;\n" +
		"         }\n" +
		"         END {\n" +
		"             for (key in array) print array[key];\n" +
		"         }';\n" +
		"};\n"
	// CompleteShowTargetsFunc provides the common target function to create
	// go-make targets for completion.
	CompleteShowTargetsFunc = "_go-make-show-targets() {\n" +
		"    local CMD=\"${1}\"; local WORD=\"${2}\";\n" +
		"    local FILE=\"" + GoMakeTargetsDir + "/targets.${CMD}\";\n" +
		"    if [ -f \"${FILE}\" ]; then cat \"${FILE}\";\n" +
		"        (go-make show-targets-${CMD} >/dev/null &);\n" +
		"    else go-make show-targets-${CMD}; fi 2>/dev/null |\n" +
		"        _go-make-filter \"${WORD}\";\n" +
		"};\n"
	// CompleteCPUCountFunc provides the common function to get the number of
	// CPUs available on the system used for parallel execution of `--jobs`.
	CompleteCPUCountFunc = "_go-make-cpu-count() {\n" +
		"    case \"$(uname -s)\" in\n" +
		"        ( Linux* ) nproc;;\n" +
		"        ( Darwin* ) sysctl -n hw.ncpu;;\n" +
		"        (*) echo \"1\";;\n" +
		"    esac;\n" +
		"};\n"
	// CompleteBash provides the bash completion setup for go-make.
	CompleteBash = "### bash completion for go-make\n" +
		"function " + CompleteFilterFunc +
		"function " + CompleteShowTargetsFunc +
		"function " + CompleteCPUCountFunc +
		"function __complete_go-make() {\n" +
		"    if [ \"${COMP_WORDS[COMP_CWORD]}\" == \"=\" ]; then\n" +
		"        WORD=\"${COMP_WORDS[COMP_CWORD-1]}=\";\n" +
		"        COMP_WORDS=(\"${COMP_WORDS[@]:0:COMP_CWORD-1}\" \"${WORD}\");\n" +
		"    elif [ \"${COMP_WORDS[COMP_CWORD-1]}\" == \"=\" ]; then\n" +
		"        WORD=\"${COMP_WORDS[COMP_CWORD-2]}=${COMP_WORDS[COMP_CWORD]}\";\n" +
		"        COMP_WORDS=(\"${COMP_WORDS[@]:0:COMP_CWORD-2}\" \"${WORD}\");\n" +
		"    else local WORD=\"${COMP_WORDS[COMP_CWORD]}\"; fi;\n" +
		"    case \"${WORD}\" in\n" +
		"    ( --directory=* | --include-dir=* )\n" +
		"        COMPREPLY=($(compgen -d -- \"${WORD#*=}\"));;\n" +
		"    ( --file=* | --makefile=* | --config=* | --what-if=* )\n" +
		"        COMPREPLY=($(compgen -df -- \"${WORD#*=}\"));;\n" +
		"    ( --assume-new=* | --assume-old=* | --old-file=* | --new-file=* )\n" +
		"        COMPREPLY=($(compgen -df -- \"${WORD#*=}\"));;\n" +
		"    ( --completion=* )\n" +
		"        COMPREPLY=($(compgen -W \"" + GoMakeCompletion + "\" -- \"${WORD#*=}\"));;\n" +
		"    ( --output-sync=* )\n" +
		"        COMPREPLY=($(compgen -W \"" + GoMakeOutputSync + "\" -- \"${WORD#*=}\"));;\n" +
		"    ( --jobs=* )\n" +
		"        COMPREPLY=($(compgen -W \"$(_go-make-cpu-count)\" -- \"${WORD#*=}\"));;\n" +
		"    ( * )\n" +
		"        local WORDS=\"$(_go-make-show-targets \"${COMP_WORDS[0]}\" \"${WORD}\")\";\n" +
		"        COMPREPLY=($(compgen -W \"${WORDS}\" -- \"${WORD}\"));;\n" +
		"    esac;\n " +
		"    if [ \"${#COMPREPLY[@]}\" == \"1\" ] &&\n" +
		"        [[ \"${COMPREPLY[0]}\" == \"--\"*\"=\" ]]; then\n" +
		"        COMPREPLY=(\"${COMPREPLY[0]}\" \"${COMPREPLY[0]}*\");\n" +
		"    fi;\n" +
		"};\n" +
		"complete -F __complete_go-make go-make;\n\n"
	// CompleteZsh provides the zsh completion setup for go-make.
	CompleteZsh = "### zsh completion for make/go-make\n" +
		CompleteFilterFunc + CompleteShowTargetsFunc +
		"__complete_go-make() {\n" +
		"    local targets=($(_go-make-show-targets \"${words[1]}\" \"${words[-1]}\"));\n" +
		"    _describe 'go-make' targets;\n" +
		"};\n" +
		"compdef __complete_go-make go-make;\n" +
		"compdef __complete_go-make make;\n\n"
)

// Available exit code constants.
const (
	// ExitSuccess indicates that the command completed successfully.
	ExitSuccess int = 0
	// ExitConfigFailure indicates that finding the configuration failed.
	ExitConfigFailure int = 2
	// ExitTargetFailure indicates that executing targets failed.
	ExitTargetFailure int = 3
)

var (
	// SuffixTargetsGoMake provides the suffix for the go-make targets file.
	SuffixTargetsGoMake = ptr("go-make")
	// SuffixTargetsMake provides the suffix for the make targets file.
	SuffixTargetsMake = ptr("make")
	// SuffixTargets provides the suffix for the general targets file.
	SuffixTargets = ptr("")
)

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// AbsPath returns the absolute path of given directory.
func AbsPath(path string) string {
	path, _ = filepath.Abs(path)
	return path
}

// CmdGoInstall creates the argument array of a `go install <path>@<version>`
// command with the given working directory and environment variables.
func CmdGoInstall(path, version, dir string, env ...string) *cmd.Cmd {
	return cmd.New("go", "install", "-v", "-mod=readonly",
		"-buildvcs=true", path+"@"+version).WithEnv(env...).WithWorkDir(dir)
}

// CmdTestDir creates the argument array of a `test -d <path>` command with the
// given working directory and environment variables. We use this to test if a
// directory exists, since it allows us to mock the check.
func CmdTestDir(path, dir string, env ...string) *cmd.Cmd {
	return cmd.New("test", "-d", path).WithEnv(env...).WithWorkDir(dir)
}

// CmdMakeTargets creates the argument array of a `make --file <Makefile>
// <targets...>` command using the given makefile name amd argument list with
// the given working directory and environment variables.
func CmdMakeTargets(
	file string, targets []string, dir string, env ...string,
) *cmd.Cmd {
	return cmd.New(append([]string{
		"make", "--file", file, "--no-print-directory",
	}, targets...)...).WithEnv(env...).WithWorkDir(dir)
}

// CmdGitTop creates the argument array of a `git rev-parse` command to
// get the root path of the current git repository.
func CmdGitTop(dir string, env ...string) *cmd.Cmd {
	return cmd.New("git", "rev-parse", "--show-toplevel").
		WithEnv(env...).WithWorkDir(dir)
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

// ErrNotFound represent a version not found error.
var ErrNotFound = errors.New("version not found")

// NewErrNotFound wraps the error of failed command to install the requested
// go-make config version.
func NewErrNotFound(dir, version string, err error) error {
	return fmt.Errorf("%w [dir=%s, version=%s]: %w",
		ErrNotFound, dir, version, err)
}

// ErrCallFailed represent a version not found error.
var ErrCallFailed = errors.New("call failed")

// NewErrCallFailed wraps the error of a failed command call.
func NewErrCallFailed(cmd *cmd.Cmd, err error) error {
	return fmt.Errorf("%w [dir=%s, call=%s]: %w",
		ErrCallFailed, cmd.Dir, cmd.Args, err)
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

	// Aborted indicates whether go-make was Aborted.
	Aborted atomic.Bool
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
func (gm *GoMake) setupWorkDir(ctx context.Context) {
	buffer := &strings.Builder{}
	if err := gm.exec(ctx, CmdGitTop(gm.WorkDir, gm.Env...).
		WithIO(nil, buffer, gm.Stderr)); err == nil {
		gm.WorkDir = strings.TrimSpace(buffer.String())
	}
}

// setupConfig sets up the go-make config by evaluating the director or version
// as provided by the command line arguments, the environment variables, or the
// context of the executed go-make command. The setup ensures that the expected
// go-make config is installed and the correct Makefile is referenced.
func (gm *GoMake) setupConfig(ctx context.Context) error {
	if gm.Config == "" {
		return gm.ensureConfig(ctx, gm.Info.Version,
			GoMakePath(gm.Info.Path, gm.Info.Version))
	}

	path := AbsPath(gm.Config)
	if err := gm.exec(ctx, CmdTestDir(path, gm.WorkDir, gm.Env...).
		WithIO(nil, gm.Stderr, gm.Stderr)); err != nil {
		return gm.ensureConfig(ctx, gm.Config,
			GoMakePath(gm.Info.Path, gm.Config))
	}
	return gm.ensureConfig(ctx, "custom", path)
}

// ensureConfig ensures that the go-make config is valid and installed and
// the correct Makefile is referenced.
func (gm *GoMake) ensureConfig(
	ctx context.Context, version, dir string,
) error {
	gm.ConfigVersion, gm.ConfigDir = version, dir
	gm.Makefile = filepath.Join(dir, Makefile)
	if version == "custom" {
		return nil
	}

	if err := gm.exec(ctx,
		CmdTestDir(gm.ConfigDir, gm.WorkDir, gm.Env...).
			WithIO(nil, gm.Stderr, gm.Stderr)); err != nil {
		if err := gm.exec(ctx,
			CmdGoInstall(gm.Info.Path, gm.ConfigVersion, gm.WorkDir, gm.Env...).
				WithIO(nil, gm.Stderr, gm.Stderr)); err != nil {
			return NewErrNotFound(gm.Info.Path, gm.ConfigVersion, err)
		}
	}
	return nil
}

// Executes given command using given context calling the command executor and
// taking care to wrap the resulting error.
func (gm *GoMake) exec(ctx context.Context, cmd *cmd.Cmd) error {
	if gm.Trace {
		gm.Logger.Exec(cmd.Stderr, cmd.Dir, cmd.Args...)
	}

	if err := gm.Executor.Exec(ctx, cmd); err != nil {
		return NewErrCallFailed(cmd, errors.Unwrap(err))
	}
	return nil
}

// Make runs the go-make command with given arguments and return the exit code
// and error.
func (gm *GoMake) Make(args ...string) (int, error) {
	var mode cmd.Mode
	var suffix *string
	var targets []string
	for _, arg := range args[1:] {
		switch {
		case arg == "--trace":
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
			targets = append(targets, arg)
			gm.Trace = true

		case arg == "--version":
			gm.Logger.Info(gm.Stdout, gm.Info, true)
			return ExitSuccess, nil

		case strings.HasPrefix(arg, "--completion=bash"):
			gm.Logger.Message(gm.Stdout, CompleteBash)
			return ExitSuccess, nil

		case strings.HasPrefix(arg, "--completion=zsh"):
			gm.Logger.Message(gm.Stdout, CompleteZsh)
			return ExitSuccess, nil

		case strings.HasPrefix(arg, "--config="):
			gm.Config = arg[len("--config="):]

		// case arg == "--async":
		// 	mode |= cmd.Detached | cmd.Background
		// case arg == "--detached":
		// 	mode |= cmd.Detached
		// case arg == "--background":
		// 	mode |= cmd.Background

		case arg == "show-targets-go-make":
			targets = append(targets, arg)
			suffix = SuffixTargetsGoMake

		case arg == "show-targets-make":
			targets = append(targets, arg)
			suffix = SuffixTargetsMake

		case arg == "show-targets":
			targets = append(targets, arg)
			suffix = SuffixTargets

		default:
			targets = append(targets, arg)
		}
	}

	return gm.makeTargets(mode, suffix, targets)
}

// makeTargets executes the provided make targets with given command mode and
// targets suffix. If the targets suffix indicates that the targets should be
// shown, it displays them and returns immediately. Otherwise, it calls the
// targets and returns the exit code and error.
func (gm *GoMake) makeTargets(
	mode cmd.Mode, suffix *string, targets []string,
) (int, error) {
	if gm.showTargets(suffix) {
		mode = cmd.Detached | cmd.Background
	}

	ctx := sys.NewSignaler(gm.HandleSignal, sys.Signals...).
		Signal(context.Background())

	return gm.callTargets(ctx, mode, targets)
}

// HandleSignal handles received OS signals during go-make execution.
func (gm *GoMake) HandleSignal(cancel context.CancelFunc, signal os.Signal) {
	if signal == syscall.SIGABRT {
		gm.Aborted.Store(true)
	}
	cancel()
}

// showTargets reads the targets from the go-make targets file and displays
// them via standard output. If the file does not exist, it returns false
// indicating that no targets were found.
func (gm *GoMake) showTargets(suffix *string) bool {
	if gm.Trace || suffix == nil {
		return false
	}

	// Read the targets file, iff it exists.
	file := gm.fileTargets(*suffix)
	// #nosec G304 -- file is safe to dump.
	if content, err := os.ReadFile(file); err == nil {
		gm.Logger.Message(gm.Stdout, string(content))
		return true
	}
	return false
}

// fileTargets returns the path to the go-make targets file based on the
// provided suffix. It checks the environment variables for a custom file path
// or defaults to a temporary directory structure based on a user name. The
// file path is cleaned and returned as an absolute path.
func (gm *GoMake) fileTargets(suffix string) string {
	var file string
	switch suffix {
	case "go-make":
		file = gm.GetEnvDefault("FILE_TARGETS_GOMAKE", "")
	case "make":
		file = gm.GetEnvDefault("FILE_TARGETS_MAKE", "")
	default:
		file = gm.GetEnvDefault("FILE_TARGETS", "")
	}

	if file == "" {
		file = filepath.Join(
			gm.GetEnvDefault("TMPDIR", os.TempDir()),
			"go-make-"+gm.GetEnvDefault("USER", "unknown"),
			AbsPath(gm.WorkDir), "targets."+suffix)
	}
	return filepath.Clean(file)
}

// GetEnvDefault returns the value of the environment variable with given name
// or the given default value, if the environment variable is not set. The
// function checks the go-make context environment variables backwards first
// and falls back to the system environment variables, before falling back to
// the provided default value.
func (gm *GoMake) GetEnvDefault(name, deflt string) string {
	for i := len(gm.Env) - 1; i >= 0; i-- {
		if env := gm.Env[i]; strings.HasPrefix(env, name+"=") {
			return env[len(name)+1:]
		}
	}
	return GetEnvDefault(name, deflt)
}

// callTargets executes the provided make targets after setting up the working
// directory and the go-make config. It returns the exit code and error if any
// step of the setup or the targets execution fails.
func (gm *GoMake) callTargets(
	ctx context.Context, mode cmd.Mode, targets []string,
) (int, error) {
	gm.setupWorkDir(ctx)
	if err := gm.setupConfig(ctx); err != nil {
		gm.Logger.Error(gm.Stderr, "ensure config", err)
		return ExitConfigFailure, err
	} else if err := gm.exec(ctx,
		CmdMakeTargets(gm.Makefile, targets, gm.WorkDir, gm.Env...).
			WithMode(mode).WithIO(gm.Stdin, gm.Stdout, gm.Stderr)); err != nil {
		if !gm.Aborted.Load() {
			gm.Logger.Error(gm.Stderr, "execute make", err)
			return ExitTargetFailure, err
		}
	}
	return ExitSuccess, nil
}

// Make runs the go-make command with given build information, standard output
// writer, standard error writer, and command arguments.
func Make( //revive:disable-line:argument-limit // ensures testability.
	stdin io.Reader, stdout, stderr io.Writer, info *info.Info,
	config, wd string, env []string, args ...string,
) int {
	exit, _ := NewGoMake(
		stdin, stdout, stderr, info, config, wd, env...,
	).Make(args...)

	return exit
}
