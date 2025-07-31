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
	// GoMakeOptions provides the common options for the go-make command.
	GoMakeOptions = "--async --completion= --config="
	// GoMakeCompletion provides the common completion options for the
	// go-make command.
	GoMakeCompletion = "bash zsh"
	// GoMakeOutputSync provides the common output sync options for the
	// go-make command.
	GoMakeOutputSync = "none line target recurse"
	// CompleteFilterFunc provides the common filter function to filter
	// go-make targets before applying completion.
	CompleteFilterFunc = "_go-make-filter() {\n" +
		"    awk -v prefix=^${1} -v pat=\"[-/]\" '\n" +
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
		"    local WORD=\"${1:-${WORD}}\";\n" +
		"echo -ne \"\n$(date '+%F %T.%3N') show-targets start ${COMP_WORDS[@]}\" >&2;\n" +
		"    if [ ${COMP_WORDS[0]} != \"go-make\" ]; then\n" +
		"       go-make show-targets-make;\n" +
		"    else go-make show-targets; fi |\n" +
		"       _go-make-filter \"${WORD}\";\n" +
		"echo -ne \"\n$(date '+%F %T.%3N') show-targets stop ${COMP_WORDS[@]}\" >&2;\n" +
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
		"echo -ne \"\n$(date '+%F %T.%3N') complete start ${COMP_WORDS[@]}\" >&2;\n" +
		"    if [ \"${COMP_WORDS[COMP_CWORD]}\" == \"=\" ]; then\n" +
		"        WORD=\"${COMP_WORDS[COMP_CWORD-1]}=\";\n" +
		"        COMP_WORDS=(\"${COMP_WORDS[@]:0:COMP_CWORD-1}\" \"${WORD}\");\n" +
		"    elif [ \"${COMP_WORDS[COMP_CWORD-1]}\" == \"=\" ]; then\n" +
		"        WORD=\"${COMP_WORDS[COMP_CWORD-2]}=${COMP_WORDS[COMP_CWORD]}\";\n" +
		"        COMP_WORDS=(\"${COMP_WORDS[@]:0:COMP_CWORD-2}\" \"${WORD}\");\n" +
		"    else local WORD=\"${COMP_WORDS[COMP_CWORD]}\"; fi;\n" +
		"echo -ne \"\n$(date '+%F %T.%3N') complete word ${COMP_WORDS[@]}\" >&2;\n" +
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
		"echo -ne \"\n$(date '+%F %T.%3N') complete * start ${COMP_WORDS[@]}\" >&2;\n" +
		"        local WORDS=\"$(_go-make-show-targets \"${WORD}\")\";\n" +
		"echo -ne \"\n$(date '+%F %T.%3N') complete * targets ${COMP_WORDS[@]}\" >&2;\n" +
		"        if [ \"${COMP_WORDS[0]}\" == \"go-make\" ]; then\n" +
		"            local WORDS=\"" + GoMakeOptions + " ${WORDS}\";\n" +
		"        fi;\n" +
		"echo -ne \"\n$(date '+%F %T.%3N') complete * end ${COMP_WORDS[@]}\" >&2;\n" +
		"        COMPREPLY=($(compgen -W \"${WORDS}\" -- \"${WORD}\"));;\n" +
		"    esac;\n " +
		"echo -ne \"\n$(date '+%F %T.%3N') complete reply ${COMP_WORDS[@]}\" >&2;\n" +
		"    if [ \"${#COMPREPLY[@]}\" == \"1\" ] &&\n" +
		"        [[ \"${COMPREPLY[0]}\" == \"--\"*\"=\" ]]; then\n" +
		"        COMPREPLY=(\"${COMPREPLY[0]}\" \"${COMPREPLY[0]}*\");\n" +
		"    fi;\n" +
		"echo -ne \"\n$(date '+%F %T.%3N') complete stop ${COMP_WORDS[@]}\" >&2;\n" +
		"};\n" +
		"complete -F __complete_go-make go-make;\n"
	// CompleteZsh provides the zsh completion setup for go-make.
	CompleteZsh = "### zsh completion for make/go-make\n" +
		CompleteFilterFunc + CompleteShowTargetsFunc +
		"__complete_go-make() {\n" +
		"    local targets=($(_go-make-show-targets \"${words[-1]}\"));\n" +
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

	if err := gm.Executor.Exec(cmd.Attached, stdin, stdout, stderr,
		dir, env, args...); err != nil {
		return NewErrCallFailed(dir, args, errors.Unwrap(err))
	}
	return nil
}

// Make runs the go-make command with given arguments and return the exit code
// and error.
func (gm *GoMake) Make(args ...string) (int, error) {
	var targets []string
	for _, arg := range args {
		switch {
		case arg == "--async":
			targets = append(targets, "&")

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

		case arg == "show-targets":
			targets = append(targets, arg)
			if gm.showTargets("go-make") {
				targets = append(targets, "&")
			}

		case arg == "show-targets-make":
			targets = append(targets, arg)
			if gm.showTargets("make") {
				targets = append(targets, "&")
			}

		default:
			targets = append(targets, arg)
		}
	}

	return gm.callTargets(targets)
}

// showTargets reads the targets from the go-make targets file and displays
// them via standard output. If the file does not exist, it returns false
// indicating that no targets were found.
func (gm *GoMake) showTargets(suffix string) bool {
	file := filepath.
		Clean(filepath.Join(GetEnvDefault("TMPDIR", "/tmp"),
			"go-make-"+GetEnvDefault("USER", "unknown"),
			AbsPath(gm.WorkDir), "targets."+suffix))

	// Read the targets file if it exists.
	if content, err := os.ReadFile(file); err == nil {
		gm.Logger.Message(gm.Stdout, string(content))
		return true
	}
	return false
}

// callTargets executes the provided make targets after setting up the working
// directory and the go-make config. It returns the exit code and error if any
// step of the setup or the targets execution fails.
func (gm *GoMake) callTargets(targets []string) (int, error) {
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
func NewErrNotFound(dir, version string, err error) error {
	return fmt.Errorf("%w [dir=%s, version=%s]: %w",
		ErrNotFound, dir, version, err)
}

// ErrCallFailed represent a version not found error.
var ErrCallFailed = errors.New("call failed")

// NewErrCallFailed wraps the error of a failed command call.
func NewErrCallFailed(dir string, args []string, err error) error {
	return fmt.Errorf("%w [dir=%s, call=%s]: %w",
		ErrCallFailed, dir, args, err)
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
