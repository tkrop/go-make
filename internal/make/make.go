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

var (
	// Base command array for `git clone` arguments.
	cmdGitClone = []string{"git", "clone", "--depth=1"}
	// Base command array for `git fetch` arguments.
	cmdGitFetch = []string{"git", "fetch"}
	// Base command array for `git status`` arguments.
	cmdGitStatus = []string{"git", "status", "--short"}
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

// CmdGitStatus creates the argument array of a `git status` command.
func CmdGitStatus() []string {
	return cmdGitStatus
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
	Info *info.Info
	// The directory of the go-make command.
	MakeDir string
	// The actual working directory.
	WorkDir string
	// The path to the go-make command Makefile.
	Makefile string
	// Executor provides the command executor.
	Executor cmd.Executor
	// Logger provides the logger.
	Logger log.Logger
	// Stdout provides the standard output writer.
	Stdout io.Writer
	// Stderr provides the standard error writer.
	Stderr io.Writer
	// Trace provides the flags to trace commands.
	Trace bool
}

// NewGoMake returns a new default `go-make` application context with given
// standard output writer, standard error writer, and trace flag.
func NewGoMake(
	stdout, stderr io.Writer, info *info.Info,
) *GoMake {
	//revive:disable-next-line:redefines-builtin-id // Is package name.
	make, _ := os.Executable() //nolint:predeclared // Is package name.
	makeDir := make + ".config"
	workdir, _ := os.Getwd()

	return &GoMake{
		Info:     info,
		MakeDir:  makeDir,
		WorkDir:  workdir,
		Makefile: filepath.Join(makeDir, Makefile),
		Executor: cmd.NewExecutor(),
		Logger:   log.NewLogger(),
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

// Updates the go-make command repository.
func (gm *GoMake) updateGoMakeRepo() error {
	// Update by cloning the current revision.
	if _, err := os.Stat(gm.MakeDir); os.IsNotExist(err) {
		return gm.cloneGoMakeRepo()
		// I'm not sure how this can happen, so I commented it out.
		// } else if err != nil {
		// 	return NewErrNotFound(gm.MakeDir, gm.Info.Revision, err)
	}

	// Do never update on dirty revisions.
	if ok, err := gm.repoIsDirty(); err != nil {
		return err
	} else if ok {
		return nil
	}

	// Update revision checking for new commits.
	if err := gm.updateRevision(); errors.Is(err, ErrNotFound) {
		// Update revision again after fetching latest commits.
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.MakeDir,
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
		CmdGitClone(repo, gm.MakeDir)...)
}

// repoIsDirty returns whether the go-make command repository is dirty.
func (gm *GoMake) repoIsDirty() (bool, error) {
	builder := strings.Builder{}
	if err := gm.exec(&builder, gm.Stderr, gm.MakeDir,
		CmdGitStatus()...); err != nil {
		return false, err
	}
	return strings.TrimSpace(builder.String()) != "", nil
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
		if err := gm.exec(gm.Stderr, gm.Stderr, gm.MakeDir,
			CmdGitHashReset(revision)...); err != nil {
			return NewErrNotFound(gm.MakeDir, revision, err)
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
		if err := gm.exec(&builder, gm.Stderr, gm.MakeDir,
			CmdGitHashHead()...); err != nil {
			return "", err
		}
		return strings.TrimSpace(builder.String()), nil
	} else if gm.Info.Version == gm.Info.Revision {
		builder := strings.Builder{}
		if err := gm.exec(&builder, gm.Stderr, gm.MakeDir,
			CmdGitHashTag(gm.Info.Revision)...); err != nil {
			return "", err
		}
		return strings.TrimSpace(builder.String()), nil
	}
	return gm.Info.Revision, nil
}

// isOnRevision returns whether the go-make config repository head commit hash
// is matching the revision required by the go-make command. If the resolution
// of the current hash fails the error is returned.
func (gm *GoMake) isOnRevision(hash string) (bool, error) {
	builder := strings.Builder{}
	if err := gm.exec(&builder, gm.Stderr, gm.MakeDir,
		CmdGitHashNow()...); err != nil {
		return false, err
	}
	return hash == strings.TrimSpace(builder.String()), nil
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
	for _, arg := range args {
		switch arg {
		case "--trace":
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
			gm.Trace = true

		case "--version":
			gm.Logger.Info(gm.Stdout, gm.Info, true)
			return 0, nil

		case "--completion=bash":
			gm.Logger.Message(gm.Stdout, BashCompletion)
			return 0, nil
		}
	}

	if err := gm.updateGoMakeRepo(); err != nil {
		if !gm.Trace {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "update config", err)
		return ExitConfigFailure, err
	}
	if err := gm.makeTargets(args...); err != nil {
		if !gm.Trace {
			gm.Logger.Call(gm.Stderr, args...)
			gm.Logger.Info(gm.Stderr, gm.Info, false)
		}
		gm.Logger.Error(gm.Stderr, "execute make", err)
		return ExitExecFailure, err
	}
	return ExitSuccess, nil
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

// Make runs the go-make command with given build information, standard output
// writer, standard error writer, and command arguments.
func Make(
	info *info.Info, stdout, stderr io.Writer, args ...string,
) int {
	exit, _ := NewGoMake(
		// TODO we would like to filter some make file startup specific
		// output that creates hard to validate output.
		// NewMakeFilter(stdout), NewMakeFilter(stderr), info,
		stdout, stderr, info,
	).Make(args[1:]...)

	return exit
}
