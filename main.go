package main //revive:disable:max-public-structs // keep it simple

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
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
	return NewInfo(Path, Version, Revision, Build, Commit, true)
}

// NewInfo returns the build information of a command or module using given
// custom version and custom build time using RFC3339 format. The provided
// version must follow semantic versioning as supported by go.
func NewInfo(path, version, revision, build, commit string, dirty bool) *Info {
	info := &Info{
		Go:       runtime.Version()[2:],
		Compiler: runtime.Compiler,
		Platform: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	info.Version = version
	info.Revision = revision
	info.Build, _ = time.Parse(time.RFC3339, build)
	info.Commit, _ = time.Parse(time.RFC3339, commit)
	info.Dirty = dirty
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

// Printer provides a common interface for printing.
type Printer interface {
	Fprintf(io.Writer, string, ...any)
}

type DefaultPrinter struct{}

func (*DefaultPrinter) Fprintf(writer io.Writer, format string, args ...any) {
	fmt.Fprintf(writer, format, args...)
}

// Write writes the given data to the underlying writer.
func (f *MakeFilter) Write(data []byte) (int, error) {
	return f.writer.Write(f.filter.ReplaceAll(data, []byte{})) //nolint:wrapcheck // not used
}

// CmdExecutor provides a common interface for executing commands.
type CmdExecutor interface {
	// Exec executes the command with given name and arguments in given
	// directory redirecting stdout and stderr to given writers.
	Exec(stdout, stderr io.Writer, dir string, args ...string) error
}

// DefaultCmdExecutor provides a default command executor using `os/exec`
// supporting optional tracing.
type DefaultCmdExecutor struct{}

// NewCmdExecutor creates a new default command executor.
func NewCmdExecutor() CmdExecutor {
	return &DefaultCmdExecutor{}
}

// Exec executes the command with given name and arguments in given directory
// redirecting stdout and stderr to given writers.
func (*DefaultCmdExecutor) Exec(
	stdout, stderr io.Writer, dir string, args ...string,
) error {
	//#nosec G204 -- caller ensures safe commands
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir, cmd.Env = dir, os.Environ()
	cmd.Env = append(cmd.Env, "MAKE=go-make")
	cmd.Stdout, cmd.Stderr = stdout, stderr

	return cmd.Run() //nolint:wrapcheck // checked on next layer
}

// Logger provides a common interface for logging.
type Logger interface {
	// Logs the build information of the command or module to the given writer.
	Info(writer io.Writer, info *Info, raw bool)
	// Exec logs the internal command execution for debugging to the given writer.
	Exec(writer io.Writer, dir string, args ...string)
	// Logs the call of the command to the given writer.
	Call(writer io.Writer, args ...string)
	// Logs the given error message and error to the given writer.
	Error(writer io.Writer, message string, err error)
	// Logs the given message to the given writer.
	Message(writer io.Writer, message string)
}

// DefaultLogger provides a default logger using `fmt` and `json` package.
type DefaultLogger struct {
	// fmt provides the print formatter.
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

// Exec logs the internal command execution for debugging to the given writer.
func (log *DefaultLogger) Exec(writer io.Writer, dir string, args ...string) {
	log.fmt.Fprintf(writer, "exec: %s [%s]\n", strings.Join(args, " "), dir)
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

var (
	// Base command array for `git clone` arguments.
	cmdGitClone = []string{"git", "clone", "--depth=1"}
	// Base command array for `git fetch` arguments.
	cmdGitFetch = []string{"git", "fetch"}
	// Base command array for `git reset --hard` arguments.
	cmdGitHashReset = []string{"git", "reset", "--hard"}
	// Base command array for `git rev-list --max-count=1` arguments.
	cmdGitHashTag = []string{"git", "rev-list", "--max-count=1"}
	// Base command array for `git rev-parse HEAD` arguments.
	cmdGitHashHead = []string{"git", "rev-parse", "HEAD"}
	// Base command array for `git log --max-count=1 --format="%H"` arguments.
	cmdGitHashNow = []string{"git", "log", "--max-count=1", "--format=\"%H\""}
)

// CmdGitClone creates the argument array of a `git clone` command of the given
// repository into the target directory.
func CmdGitClone(repo, dir string) []string {
	return append(cmdGitClone, repo, dir)
}

// CmdGitFetch creates the argument array of a `git fetch` command using the
// given source repository.
func CmdGitFetch(repo string) []string {
	return append(cmdGitFetch, repo)
}

// CmdGitHashReset creates the argument array of a `git reset --hard` command
// using the given revision hash.
func CmdGitHashReset(hash string) []string {
	return append(cmdGitHashReset, hash)
}

// CmdGitHashTag creates the argument array of a `git rev-list --max-count=1`
// command using the given target tag.
func CmdGitHashTag(tag string) []string {
	return append(cmdGitHashTag, "tags/"+tag)
}

func CmdGitHashHead() []string {
	return cmdGitHashHead
}

func CmdGitHashNow() []string {
	return cmdGitHashNow
}

// CmdMakeTargets creates the argument array of a `make --file <Makefile>
// <targets...>` command using the given makefile name amd argument list.
func CmdMakeTargets(file string, args ...string) []string {
	return append([]string{"make", "--file", file}, args...)
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
	// Trace provides the flags to trace commands.
	Trace bool
	// Debug provides the flags to debug commands
	Debug bool
}

// NewGoMake returns a new default `go-make` application context with given
// standard output writer, standard error writer, and trace flag.
func NewGoMake(
	stdout, stderr io.Writer, info *Info,
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
		Executor: NewCmdExecutor(),
		Logger:   NewLogger(),
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

// Updates the go-make command repository.
func (gm *GoMake) updateGoMakeRepo() error {
	// Update by cloning the current revision.
	if _, err := os.Stat(gm.CmdDir); os.IsNotExist(err) {
		return gm.cloneGoMakeRepo()
	} else if err != nil {
		return NewErrNotFound(gm.CmdDir, gm.Info.Revision, err)
	}

	// Do never update on dirty revisions.
	if gm.Info.Dirty {
		return nil
	}

	// Update revision checking for new commits.
	if err := gm.updateRevision(); errors.Is(err, ErrNotFound) {
		// Update revision again after fetching latest commits.
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.CmdDir,
			CmdGitFetch(gm.Info.Repo)...); err != nil {
			return err
		}
		return gm.updateRevision()
	} else if err != nil {
		return err
	}

	return nil
}

// cloneGoMakeRepo clones the go-make command repository.
func (gm *GoMake) cloneGoMakeRepo() error {
	if err := gm.cloneGoMakeExec(gm.Info.Repo); err != nil {
		repo := "https://" + gm.Info.Path + ".git"
		if err := gm.cloneGoMakeExec(repo); err != nil {
			return err
		}
	}
	return gm.updateRevision()
}

// cloneGoMakeExec executes the cloning of the go-make command repository.
func (gm *GoMake) cloneGoMakeExec(repo string) error {
	return gm.exec(gm.Stderr, gm.Stderr, gm.WorkDir,
		CmdGitClone(repo, gm.CmdDir)...)
}

// updateRevision updates the current revision of the go-make command
// repository as required by the go-make command. If the update fails the
// an error is returned.
func (gm *GoMake) updateRevision() error {
	if revision, err := gm.findRevision(); err != nil {
		return err
	} else if ok, err := gm.isOnRevision(revision); err != nil {
		return err
	} else if !ok {
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.CmdDir,
			CmdGitHashReset(revision)...); err != nil {
			return NewErrNotFound(gm.CmdDir, revision, err)
		}
	}

	return nil
}

// findRevision returns the current revision required by the go-make command
// as git commit hash. If the revision is a git tag, it is resolved to the
// respective git commit hash of the tag. If the resolution of the git hash
// fails the error is returned.
func (gm *GoMake) findRevision() (string, error) {
	if gm.Info.Revision == "" {
		builder := strings.Builder{}
		if err := gm.exec(&builder, gm.Stderr, gm.CmdDir,
			CmdGitHashHead()...); err != nil {
			return "", err
		}
		return builder.String(), nil
	} else if gm.Info.Version == gm.Info.Revision {
		builder := strings.Builder{}
		if err := gm.exec(&builder, gm.Stderr, gm.CmdDir,
			CmdGitHashTag(gm.Info.Revision)...); err != nil {
			return "", err
		}
		return builder.String(), nil
	}
	return gm.Info.Revision, nil
}

// isOnRevision returns whether the go-make config repository head commit hash
// is matching the revision required by the go-make command. If the resolution
// of the current hash fails the error is returned.
func (gm *GoMake) isOnRevision(hash string) (bool, error) {
	builder := strings.Builder{}
	err := gm.exec(&builder, gm.Stderr, gm.CmdDir, CmdGitHashNow()...)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(builder.String(), hash), nil
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
	if gm.Debug {
		gm.Logger.Exec(stdout, dir, args...)
	}

	if err := gm.Executor.Exec(stdout, stderr, dir, args...); err != nil {
		return NewErrCallFailed(args, err)
	}
	return nil
}

// RunCmd runs the go-make command with given arguments.
func (gm *GoMake) RunCmd(args ...string) error {
	for _, arg := range args {
		switch arg {
		case "--debug":
			gm.Debug = true

		case "--trace":
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
			gm.Trace = true

		case "--version":
			gm.Logger.Info(gm.Stdout, gm.Info, false)
			return nil

		case "--completion=bash":
			gm.Logger.Message(gm.Stdout, BashCompletion)
			return nil
		}
	}

	if err := gm.updateGoMakeRepo(); err != nil {
		if !gm.Trace {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "update config", err)
		return err
	}
	if err := gm.makeTargets(args...); err != nil {
		if !gm.Trace {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "execute make", err)
		return err
	}
	return nil
}

// ErrNotFound represent a revision not found error.
var ErrNotFound = errors.New("revision not found")

// NewErrNotFound wraps the error of failed command unable to find a revision.
func NewErrNotFound(dir, revision string, err error) error {
	return fmt.Errorf("%w [dir=%s, revision=%s]: %w",
		ErrNotFound, dir, revision, err)
}

// NewErrCallFailed wraps the error of a failed command call.
func NewErrCallFailed(args []string, err error) error {
	return fmt.Errorf("call failed [name=%s, args=%v]: %w",
		args[0], args[1:], err)
}

// Run runs the go-make command with given build information, standard output
// writer, standard error writer, and command arguments.
func Run(info *Info, stdout, stderr io.Writer, args ...string) error {
	return NewGoMake(
		// TODO we would like to filter some make file startup specific
		// output that creates hard to validate output.
		// NewMakeFilter(stdout), NewMakeFilter(stderr),
		stdout, stderr, info,
	).RunCmd(args[1:]...)
}

// main is the main entry point of the go-make command.
func main() {
	if Run(NewDefaultInfo(), os.Stdout, os.Stderr, os.Args...) != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
