package main //revive:disable:max-public-structs // keep it simple

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

var (
	// Path contains the package path.
	Path string
	// Version contains the custom version.
	Version string
	// Build contains the custom build time.
	Build string
	// Revision contains the custom revision.
	Revision string
	// Commit contains the custom commit time.
	Commit string
	// Dirty contains the custom dirty flag.
	Dirty string
)

const (
	// RepoPathSepNum is the number of the repository path separator.
	RepoPathSepNum = 3
	// GitSha1HashLen is the full length of sha1-hashes used in git.
	GitFullHashLen = 40
	// GitShortHashLen is the short length of sha1-hashes used in git.
	GitShortHashLen = 7

	// Makefile base makefile to be executed by go-make.
	Makefile = "Makefile.base"
	// bashCompletion contains the bash completion script for go-make.
	BashCompletion = `### bash completion for go-make
	function __complete_go-make() {
		COMPREPLY=($(compgen -W "$(go-make targets)" -- "${COMP_WORDS[COMP_CWORD]}"));
	}
	complete -F __complete_go-make go-make;
	`
)

var (
	// Regexp for semantic versioning as supported by go as tag.
	semVersionTagRegex = regexp.MustCompile(
		`v(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)` +
			`(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
			`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
			`(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	// Regexp for filtering entering and leaving lines from make output.
	makeFilterRegexp = regexp.MustCompile(
		`(?s)make\[[0-9]+\]: (Entering|Leaving) directory '[^\n]*'\n?`)
)

// Info provides the build information of a command or module.
type Info struct {
	// Path contains the package path of the command or module.
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
	// Repo contains the repository of the command or module.
	Repo string `yaml:"repo,omitempty" json:"repo,omitempty"`
	// Version contains the actual version of the command or module.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	// Revision contains the revision of the command or module from version
	// control system.
	Revision string `yaml:"revision,omitempty" json:"revision,omitempty"`
	// Build contains the build time of the command or module.
	Build time.Time `yaml:"build,omitempty" json:"build,omitempty"`
	// Commit contains the last commit time of the command or module from the
	// version control system.
	Commit time.Time `yaml:"commit,omitempty" json:"commit,omitempty"`
	// Dirty flags whether the build of the command or module is based on a
	// dirty local repository state.
	Dirty bool `yaml:"dirty,omitempty" json:"dirty,omitempty"`
	// Checksum contains the check sum of the command or module.
	Checksum string `yaml:"checksum,omitempty" json:"checksum,omitempty"`

	// Go contains the go version the command or module was build with.
	Go string `yaml:"go,omitempty" json:"go,omitempty"`
	// Platform contains the build platform the command or module was build
	// for.
	Platform string `yaml:"platform,omitempty" json:"platform,omitempty"`
	// Compiler contains the actual compiler the command or module was build
	// with.
	Compiler string `yaml:"compiler,omitempty" json:"compiler,omitempty"`
}

// NewDefaultInfo returns the build information of a command or module with
// default values.
func NewDefaultInfo() *Info {
	return NewInfo(Path, Version, Revision, Build, Commit, Dirty)
}

// NewInfo returns the build information of a command or module using given
// custom version and custom build time using RFC3339 format. The provided
// version must follow semantic versioning as supported by go.
func NewInfo(path, version, revision, build, commit, dirty string) *Info {
	info := &Info{
		Go:       runtime.Version()[2:],
		Compiler: runtime.Compiler,
		Platform: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	info.Version = version
	info.Revision = revision
	info.Build, _ = time.Parse(time.RFC3339, build)
	info.Commit, _ = time.Parse(time.RFC3339, commit)
	info.Dirty, _ = strconv.ParseBool(dirty)
	if buildinfo, ok := debug.ReadBuildInfo(); ok {
		if path != "" {
			info.Path = path
			info.Repo = "git@" + strings.Replace(
				info.splitRuneN(path, '/', RepoPathSepNum), "/", ":", 1)
		} else {
			info.Path = buildinfo.Main.Path
			info.Repo = "git@" + strings.Replace(buildinfo.Main.Path, "/", ":", 1)
		}

		if semVersionTagRegex.MatchString(buildinfo.Main.Version) {
			info.Version = buildinfo.Main.Version
			index := strings.LastIndex(buildinfo.Main.Version, "-")
			info.Revision = buildinfo.Main.Version[index+1:]
		}

		info.Checksum = buildinfo.Main.Sum
		for _, kv := range buildinfo.Settings {
			switch kv.Key {
			case "vcs.revision":
				info.Revision = kv.Value
			case "vcs.time":
				info.Commit, _ = time.Parse(time.RFC3339, kv.Value)
			case "vcs.modified":
				info.Dirty = kv.Value == "true"
			}
		}
	}

	if !semVersionTagRegex.MatchString(info.Version) {
		if info.Revision != "" && !info.Commit.Equal(time.Time{}) {
			info.Version = fmt.Sprintf("v0.0.0-%s-%s",
				info.Commit.UTC().Format("20060102150405"), info.Revision[0:12])
		}
	}

	return info
}

// splitRuneN splits the string s at the `n`th occurrence of the rune ch.
func (*Info) splitRuneN(s string, ch rune, n int) string {
	count := 0
	index := strings.IndexFunc(s, func(is rune) bool {
		if ch == is {
			count++
		}
		return count == n
	})
	if index >= 0 {
		return s[0:index]
	}
	return s
}

// MakeFilter is a custom filter that implements the io.Writer interface.
type MakeFilter struct {
	writer io.Writer
	filter *regexp.Regexp
}

func NewMakeFilter(writer io.Writer) *MakeFilter {
	return &MakeFilter{
		writer: writer,
		filter: makeFilterRegexp,
	}
}

// Write writes the given data to the underlying writer.
func (f *MakeFilter) Write(data []byte) (int, error) {
	return f.writer.Write(f.filter.ReplaceAll(data, []byte{})) //nolint:wrapcheck // not used
}

// CmdExecutor provides a common interface for executing commands.
type CmdExecutor interface {
	// Exec executes the command with given name and arguments in given
	// directory redirecting stdout and stderr to given writers.
	Exec(stdout, stderr io.Writer, dir, name string, args ...string) error
	// Trace returns flags whether the command executor traces the command
	// execution.
	Trace() bool
}

// DefaultCmdExecutor provides a default command executor using `os/exec`
// supporting optional tracing.
type DefaultCmdExecutor struct {
	trace bool
}

// NewCmdExecutor creates a new default command executor with given trace flag.
func NewCmdExecutor(trace bool) CmdExecutor {
	return &DefaultCmdExecutor{trace: trace}
}

// Exec executes the command with given name and arguments in given directory
// redirecting stdout and stderr to given writers.
func (e *DefaultCmdExecutor) Exec(
	stdout, stderr io.Writer, dir, name string, args ...string,
) error {
	if e.trace {
		fmt.Fprintf(stdout, "%s %s [%s]\n",
			name, strings.Join(args, " "), dir)
	}

	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Env = dir, os.Environ()
	//	cmd.Env = append(cmd.Env, "MAKE=go-make")
	cmd.Stdout, cmd.Stderr = stdout, stderr

	return cmd.Run() //nolint:wrapcheck // checked on next layer
}

// Trace returns flags whether the command executor traces the command
// execution.
func (e *DefaultCmdExecutor) Trace() bool {
	return e.trace
}

// Printer provides a common interface for printing.
type Printer interface {
	Fprintf(io.Writer, string, ...any)
}

type DefaultPrinter struct{}

func (*DefaultPrinter) Fprintf(writer io.Writer, format string, args ...any) {
	fmt.Fprintf(writer, format, args...)
}

// Logger provides a common interface for logging.
type Logger interface {
	// Logs the build information of the command or module to the given writer.
	Info(writer io.Writer, info *Info, raw bool)
	// Logs the call of the command to the given writer.
	Call(writer io.Writer, args ...string)
	// Logs the given error message and error to the given writer.
	Error(writer io.Writer, message string, err error)
	// Logs the given message to the given writer.
	Message(writer io.Writer, message string)
}

// DefaultLogger provides a default logger using `fmt` and `json` package.
type DefaultLogger struct {
	// fmt provides the print formater.
	fmt Printer
}

// NewLogger creates a new default logger.
func NewLogger() Logger {
	return &DefaultLogger{fmt: &DefaultPrinter{}}
}

// Info logs the build information of the command or module to the given
// writer.
func (log *DefaultLogger) Info(writer io.Writer, info *Info, raw bool) {
	if out, err := json.Marshal(info); err != nil {
		log.fmt.Fprintf(writer, "info: %v\n", err)
	} else if raw {
		log.fmt.Fprintf(writer, "info: %s\n", out)
	} else {
		log.fmt.Fprintf(writer, "%s\n", out)
	}
}

// Call logs the call of the command to the given writer.
func (log *DefaultLogger) Call(writer io.Writer, args ...string) {
	log.fmt.Fprintf(writer, "call: %s\n", strings.Join(args, " "))
}

// Error logs the given error message and error to the given writer.
func (log *DefaultLogger) Error(writer io.Writer, message string, err error) {
	if err != nil {
		log.fmt.Fprintf(writer, "error: %s: %v\n", message, err)
	} else {
		log.fmt.Fprintf(writer, "error: %s\n", message)
	}
}

// Message logs the given message to the given writer.
func (*DefaultLogger) Message(writer io.Writer, message string) {
	fmt.Fprintf(writer, "%s\n", message)
}

// GoMake provides the default `go-make` application context.
type GoMake struct {
	// Info provides the build information of go-make.
	Info *Info
	// The directory of the go-make command.
	CmdDir string
	// The actual working directory.
	WorkDir string
	// The path to the go-make command Makefile.
	Makefile string
	// Executor provides the command executor.
	Executor CmdExecutor
	// Logger provides the logger.
	Logger Logger
	// Stdout provides the standard output writer.
	Stdout io.Writer
	// Stderr provides the standard error writer.
	Stderr io.Writer
}

// NewGoMake returns a new default `go-make` application context with given
// standard output writer, standard error writer, and trace flag.
func NewGoMake(
	stdout, stderr io.Writer, info *Info, trace bool,
) *GoMake {
	cmd, _ := os.Executable()
	cmdDir := cmd + ".config"
	makefile := filepath.Join(cmdDir, Makefile)
	workdir, _ := os.Getwd()

	return &GoMake{
		Info:     info,
		CmdDir:   cmdDir,
		WorkDir:  workdir,
		Makefile: makefile,
		Executor: NewCmdExecutor(trace),
		Logger:   NewLogger(),
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

// Updates the go-make command.
func (gm *GoMake) updateGoMake() error {
	if _, err := os.Stat(gm.CmdDir); os.IsNotExist(err) {
		if err := gm.cloneRepo(); err != nil {
			return err
		} else if err := gm.setRevision(); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return ErrConfigFailure(gm.CmdDir, err)
	}

	if gm.Info.Revision != "" {
		if revision, err := gm.getRevision(); err != nil {
			return err
		} else if strings.HasPrefix(revision, gm.Info.Revision) {
			return nil
		}
	}

	return gm.updateRevision()
}

// Clones the go-make command repository.
func (gm *GoMake) cloneRepo() error {
	if err := gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir,
		"git", "clone", "--depth=1", gm.Info.Repo, gm.CmdDir); err != nil {
		repo := "https://" + gm.Info.Path + ".git"
		return gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir,
			"git", "clone", "--depth=1", repo, gm.CmdDir)
	}
	return nil
}

// Updates the go-make command repository.
func (gm *GoMake) updateRepo() error {
	return gm.exec(gm.Stderr, gm.Stderr, gm.CmdDir,
		"git", "fetch", gm.Info.Repo)
}

// Updates the go-make command repository revision.
func (gm *GoMake) updateRevision() error {
	if gm.Info.Dirty {
		return nil
	} else if err := gm.updateRepo(); err != nil {
		return err
	}
	return gm.setRevision()
}

// Returns the current revision of the go-make command repository.
func (gm *GoMake) getRevision() (string, error) {
	builder := strings.Builder{}
	if err := gm.exec(&builder, gm.Stderr, gm.CmdDir,
		"git", "rev-parse", "HEAD"); err != nil {
		return "", err
	}
	return builder.String()[0:GitFullHashLen], nil
}

// Sets the current revision of the go-make command repository.
func (gm *GoMake) setRevision() error {
	revision := gm.Info.Revision
	if revision == "" {
		revision = "HEAD"
	} else if len(revision) < GitFullHashLen {
		revision = revision[0:GitShortHashLen]
	}

	return gm.exec(gm.Stderr, gm.Stderr, gm.CmdDir,
		"git", "reset", "--hard", revision)
}

// Executes the provided make targets.
func (gm *GoMake) makeTarget(args ...string) error {
	args = append([]string{"--file", gm.Makefile}, args...)
	return gm.exec(gm.Stdout, gm.Stderr, gm.WorkDir, "make", args...)
}

// Executes the command with given name and arguments in given directory
// calling the command executor taking care to wrap the resulting error.
func (gm *GoMake) exec(
	stdout, stderr io.Writer, dir, name string, args ...string,
) error {
	err := gm.Executor.Exec(stdout, stderr, dir, name, args...)
	if err != nil {
		return ErrCallFailed(name, args, err)
	}
	return nil
}

// RunCmd runs the go-make command with given arguments.
func (gm *GoMake) RunCmd(args ...string) error {
	if gm.Executor.Trace() {
		gm.Logger.Call(gm.Stderr, args...)
		gm.Logger.Info(gm.Stderr, gm.Info, false)
	}

	switch {
	case slices.Contains(args, "--version"):
		gm.Logger.Info(gm.Stdout, gm.Info, false)
		return nil

	case slices.Contains(args, "--completion=bash"):
		gm.Logger.Message(gm.Stdout, BashCompletion)
		return nil
	}

	if err := gm.updateGoMake(); err != nil {
		if !gm.Executor.Trace() {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "update config", err)
		return err
	}
	if err := gm.makeTarget(args...); err != nil {
		if !gm.Executor.Trace() {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "execute make", err)
		return err
	}
	return nil
}

// ErrCallFailed wraps the error of a failed command call.
func ErrCallFailed(name string, args []string, err error) error {
	return fmt.Errorf("call failed [name=%s, args=%v]: %w",
		name, args, err)
}

// ErrConfigFailure wraps the error of a failed config update.
func ErrConfigFailure(dir string, err error) error {
	return fmt.Errorf("config failure [dir=%s]: %w", dir, err)
}

// Run runs the go-make command with given build information, standard output
// writer, standard error writer, and command arguments.
func Run(info *Info, stdout, stderr io.Writer, args ...string) error {
	return NewGoMake(
		// TODO we would like to filter some make file startup specific
		// output that creates hard to validate output.
		// NewMakeFilter(stdout), NewMakeFilter(stderr),
		stdout, stderr,
		info, slices.Contains(args, "--trace"),
	).RunCmd(args[1:]...)
}

// main is the main entry point of the go-make command.
func main() {
	if Run(NewDefaultInfo(), os.Stdout, os.Stderr, os.Args...) != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
