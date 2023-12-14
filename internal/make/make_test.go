// Package make contains the main logic of the go-make command.
package make_test

import (
	"embed"
	"errors"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-make/internal/info"
	"github.com/tkrop/go-make/internal/log"
	"github.com/tkrop/go-make/internal/make"
	"github.com/tkrop/go-testing/mock"
	"github.com/tkrop/go-testing/test"
)

//revive:disable:line-length-limit // go:generate line length

//go:generate mockgen -package=make_test -destination=mock_cmd_test.go -source=../cmd/cmd.go CmdExecutor
//go:generate mockgen -package=make_test -destination=mock_logger_test.go -source=../log/logger.go Logger

//revive:enable:line-length-limit

const (
	// DirExist contains an arbitrary execution directory (use '.').
	dirExec = "."
	// goMakeDirNew contains an arbitrary non-existing directory.
	goMakeDirNew = "new-dir"
	// goMakeDirExist contains an arbitrary existing directory (use build).
	goMakeDirExist = "../../build"
	// goMakePath contains an arbitrary source path for go-make.
	goMakePath = "github.com/tkrop/go-make"
	// goMakeGit contains an arbitrary source repository for go-make.
	goMakeGit = "git@github.com:tkrop/go-make"
	// goMakeHTTP contains an arbitrary source repository for go-make.
	goMakeHTTP = "https://github.com/tkrop/go-make.git"
	// revisionHead contains an arbitrary head revision.
	revisionHead = "1b66f320c950b25fa63b81fd4e660c5d1f9d758e"
	// HeadRevision contains an arbitrary default revision.
	revisionDefault = "c0a7f81b82937ffe379ac39ece2925fa4d19fd40"
	// revisionOther contains an arbitrary different revision.
	revisionOther = "fbb61d4981d22b94412b906ea4d7ae3302d860d0"
)

var (
	// infoDirty without version and revision hash.
	infoDirty = info.NewInfo(goMakePath, "", "", "", "", true)

	// infoTag with version and revision hash from git.
	infoTag = info.NewInfo(goMakePath,
		"v1.1.1",
		"v1.1.1",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	// infoHash with revision hash from git.
	infoHash = info.NewInfo(goMakePath,
		"v0.0.0-20231110152254-1b66f320c950",
		revisionDefault,
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	// infoHead without revision hash from git.
	infoHead = info.NewInfo(goMakePath,
		"v0.0.0-20231110152254-1b66f320c950",
		"",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	argsVersion      = []string{"--version"}
	argsBash         = []string{"--completion=bash"}
	argsTarget       = []string{"target"}
	argsTraceVersion = []string{"--trace", "--version"}
	argsTraceBash    = []string{"--trace", "--completion=bash"}
	argsTraceTarget  = []string{"--trace", "target"}

	// Any error that can happen.
	errAny = errors.New("any error")
)

// NewWriter creates a new writer with the given id.
func NewWriter(id string) io.Writer {
	builder := &strings.Builder{}
	builder.Write([]byte(id))
	return builder
}

// GoMakeSetup sets up a new go-make test with mocks.
func GoMakeSetup(
	t test.Test, param MakeParams,
) (*make.GoMake, *mock.Mocks) {
	mocks := mock.NewMocks(t).
		SetArg("stdout", NewWriter("stdout")).
		SetArg("stderr", NewWriter("stderr")).
		SetArg("builder", &strings.Builder{}).
		Expect(param.mockSetup)

	gm := make.NewGoMake(
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
		param.info,
	)

	if param.goMakeDir == "" {
		param.goMakeDir = goMakeDirExist
	}

	gm.Executor = mock.Get(mocks, NewMockExecutor)
	gm.Logger = mock.Get(mocks, NewMockLogger)
	gm.Makefile = make.Makefile
	gm.MakeDir = param.goMakeDir
	gm.WorkDir = dirExec

	return gm, mocks
}

func ToAny(args ...any) []any {
	return args
}

func Exec( //revive:disable:argument-limit
	stdout, stderr string, dir string,
	args []string, err error, sout, serr string,
) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockExecutor).EXPECT().
			Exec(mocks.GetArg(stdout), mocks.GetArg(stderr), dir, ToAny(args)...).
			DoAndReturn(mocks.Call(cmd.Executor.Exec,
				func(args ...any) []any {
					if _, err := args[0].(io.Writer).Write([]byte(sout)); err != nil {
						assert.Fail(mocks.Ctrl.T, "failed to write to stdout", err)
					}
					if _, err := args[1].(io.Writer).Write([]byte(serr)); err != nil {
						assert.Fail(mocks.Ctrl.T, "failed to write to stderr", err)
					}
					return []any{err}
				}))
	}
}

func LogCall(writer string, args []string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Call(mocks.GetArg(writer), ToAny(args)...).
			DoAndReturn(mocks.Do(log.Logger.Call))
	}
}

func LogExec(writer string, dir string, args []string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Exec(mocks.GetArg(writer), dir, ToAny(args)...).
			DoAndReturn(mocks.Do(log.Logger.Exec))
	}
}

func LogInfo(writer string, info *info.Info, raw bool) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Info(mocks.GetArg(writer), info, raw).
			DoAndReturn(mocks.Do(log.Logger.Info))
	}
}

func LogError(writer string, message string, err error) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Error(mocks.GetArg(writer), message, err).
			DoAndReturn(mocks.Do(log.Logger.Error))
	}
}

func LogMessage(writer string, message string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Message(mocks.GetArg(writer), message).
			DoAndReturn(mocks.Do(log.Logger.Message))
	}
}

type MakeParams struct {
	mockSetup   mock.SetupFunc
	info        *info.Info
	goMakeDir   string
	args        []string
	expectError error
	expectExit  int
}

var testMakeParams = map[string]MakeParams{
	// dirty option targets without trace.
	"check go-make completion bash": {
		mockSetup: mock.Chain(
			LogMessage("stdout", make.BashCompletion),
		),
		args: argsBash,
	},
	"check go-make version": {
		mockSetup: mock.Chain(
			LogInfo("stdout", infoDirty, true),
		),
		info: infoDirty,
		args: argsVersion,
	},
	"dirty go-make to run target": {
		mockSetup: mock.Chain(
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...), nil, "", ""),
		),
		info: infoDirty,
		args: argsTarget,
	},

	"dirty go-make failed": {
		mockSetup: mock.Chain(
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoDirty, false),
			LogError("stderr", "execute make", make.NewErrCallFailed(
				make.CmdMakeTargets(make.Makefile, argsTarget...), errAny)),
		),
		info: infoDirty,
		args: argsTarget,
		expectError: make.NewErrCallFailed(
			make.CmdMakeTargets(make.Makefile, argsTarget...), errAny),
		expectExit: 2,
	},

	"clone go-make clone failed": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), errAny, "", ""),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeHTTP, goMakeDirNew), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoDirty, false),
			LogError("stderr", "update config", make.NewErrCallFailed(
				make.CmdGitClone(goMakeHTTP, goMakeDirNew), errAny)),
		),
		info:      infoDirty,
		args:      argsTarget,
		goMakeDir: goMakeDirNew,
		expectError: make.NewErrCallFailed(
			make.CmdGitClone(goMakeHTTP, goMakeDirNew), errAny),
		expectExit: 1,
	},

	// dirty option targets with trace.
	"check go-make completion bash traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceBash),
			LogInfo("stderr", infoDirty, false),
			LogMessage("stdout", make.BashCompletion),
		),
		info: infoDirty,
		args: argsTraceBash,
	},
	"check go-make version traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceVersion),
			LogInfo("stderr", infoDirty, false),
			LogInfo("stdout", infoDirty, true),
		),
		info: infoDirty,
		args: argsTraceVersion,
	},
	"dirty go-make to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoDirty, false),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...), nil, "", ""),
		),
		info: infoDirty,
		args: argsTraceTarget,
	},
	"dirty go-make failed traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoDirty, false),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...), errAny, "", ""),
			LogError("stderr", "execute make", make.NewErrCallFailed(
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...), errAny)),
		),
		info: infoDirty,
		args: argsTraceTarget,
		expectError: make.NewErrCallFailed(
			make.CmdMakeTargets(make.Makefile, argsTraceTarget...), errAny),
		expectExit: 2,
	},
	"clone go-make failed traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoDirty, false),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), errAny, "", ""),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeHTTP, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeHTTP, goMakeDirNew), errAny, "", ""),
			LogError("stderr", "update config", make.NewErrCallFailed(
				make.CmdGitClone(goMakeHTTP, goMakeDirNew), errAny)),
		),
		info:      infoDirty,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirNew,
		expectError: make.NewErrCallFailed(
			make.CmdGitClone(goMakeHTTP, goMakeDirNew), errAny),
		expectExit: 1,
	},

	// clone targets without trace.
	"clone go-make reset failed": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionOther, ""),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoHash, false),
			LogError("stderr", "update config",
				make.NewErrNotFound(goMakeDirNew, infoHash.Revision,
					make.NewErrCallFailed(make.CmdGitHashReset(infoHash.Revision),
						errAny))),
		),
		info:      infoHash,
		args:      argsTarget,
		goMakeDir: goMakeDirNew,
		expectError: make.NewErrNotFound(goMakeDirNew, infoHash.Revision,
			make.NewErrCallFailed(make.CmdGitHashReset(infoHash.Revision), errAny)),
		expectExit: 1,
	},

	"clone go-make head to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashHead(), nil, revisionHead, ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoHead,
		args:      argsTarget,
		goMakeDir: goMakeDirNew,
	},

	"clone go-make hash to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionOther, ""),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoHash,
		args:      argsTarget,
		goMakeDir: goMakeDirNew,
	},

	"clone go-make tag to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionOther, ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(revisionOther), nil, "", ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoTag,
		args:      argsTarget,
		goMakeDir: goMakeDirNew,
	},

	"clone go-make fallback to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), errAny, "", ""),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeHTTP, goMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionOther, ""),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoHash,
		args:      argsTarget,
		goMakeDir: goMakeDirNew,
	},

	// clone targets with trace.
	"clone go-make head to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoHead, false),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			LogExec("stderr", goMakeDirNew, make.CmdGitHashHead()),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashHead(), nil, revisionHead, ""),
			LogExec("stderr", goMakeDirNew, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoHead,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirNew,
	},

	"clone go-make hash to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoHash, false),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			LogExec("stderr", goMakeDirNew, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionOther, ""),
			LogExec("stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision)),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision), nil, "", ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoHash,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirNew,
	},

	"clone go-make tag to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoTag, false),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), nil, "", ""),
			LogExec("stderr", goMakeDirNew,
				make.CmdGitHashTag(infoTag.Revision)),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionOther, ""),
			LogExec("stderr", goMakeDirNew, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			LogExec("stderr", goMakeDirNew,
				make.CmdGitHashReset(revisionOther)),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(revisionOther), nil, "", ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoTag,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirNew,
	},

	"clone go-make fallback to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoHash, false),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeGit, goMakeDirNew), errAny, "", ""),
			LogExec("stderr", dirExec,
				make.CmdGitClone(goMakeHTTP, goMakeDirNew)),
			Exec("stderr", "stderr", dirExec,
				make.CmdGitClone(goMakeHTTP, goMakeDirNew), nil, "", ""),
			LogExec("stderr", goMakeDirNew, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirNew,
				make.CmdGitHashNow(), nil, revisionOther, ""),
			LogExec("stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision)),
			Exec("stderr", "stderr", goMakeDirNew,
				make.CmdGitHashReset(infoHash.Revision), nil, "", ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoHash,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirNew,
	},

	// check targets without trace.
	"check go-make head hash failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashHead(), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoHead, false),
			LogError("stderr", "update config", make.NewErrCallFailed(
				make.CmdGitHashHead(), errAny)),
		),
		info:        infoHead,
		args:        argsTarget,
		goMakeDir:   goMakeDirExist,
		expectError: make.NewErrCallFailed(make.CmdGitHashHead(), errAny),
		expectExit:  1,
	},

	"check go-make head now failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashHead(), nil, revisionHead, ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoHead, false),
			LogError("stderr", "update config", make.NewErrCallFailed(
				make.CmdGitHashNow(), errAny)),
		),
		info:        infoHead,
		args:        argsTarget,
		goMakeDir:   goMakeDirExist,
		expectError: make.NewErrCallFailed(make.CmdGitHashNow(), errAny),
		expectExit:  1,
	},

	"check go-make tag log failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision), errAny, revisionDefault, ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoTag, false),
			LogError("stderr", "update config", make.NewErrCallFailed(
				make.CmdGitHashTag(infoTag.Revision), errAny)),
		),
		info:      infoTag,
		args:      argsTarget,
		goMakeDir: goMakeDirExist,
		expectError: make.NewErrCallFailed(
			make.CmdGitHashTag(infoTag.Revision), errAny),
		expectExit: 1,
	},

	"check go-make tag fetch failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionDefault, ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(revisionDefault),
				make.NewErrNotFound(goMakeDirExist, revisionDefault, errAny), "", ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitFetch(infoTag.Repo), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoTag, false),
			LogError("stderr", "update config", make.NewErrCallFailed(
				make.CmdGitFetch(infoTag.Repo), errAny)),
		),
		info:        infoTag,
		args:        argsTarget,
		goMakeDir:   goMakeDirExist,
		expectError: make.NewErrCallFailed(make.CmdGitFetch(infoTag.Repo), errAny),
		expectExit:  1,
	},

	"check go-make head to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashHead(), nil, revisionHead, ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoHead,
		args:      argsTarget,
		goMakeDir: goMakeDirExist,
	},

	"check go-make hash to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(infoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoHash,
		args:      argsTarget,
		goMakeDir: goMakeDirExist,
	},

	"check go-make tag to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionDefault, ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(revisionDefault), nil, "", ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoTag,
		args:      argsTarget,
		goMakeDir: goMakeDirExist,
	},

	"check go-make tag fetch to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionDefault, ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(revisionDefault),
				make.NewErrNotFound(goMakeDirExist, revisionDefault, errAny), "", ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitFetch(infoTag.Repo), nil, "", ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionDefault, ""),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(revisionDefault), nil, "", ""),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTarget...),
				nil, "", ""),
		),
		info:      infoTag,
		args:      argsTarget,
		goMakeDir: goMakeDirExist,
	},

	// check targets with trace.
	"check go-make head to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoHead, false),
			LogExec("stderr", goMakeDirExist, make.CmdGitHashHead()),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashHead(), nil, revisionHead, ""),
			LogExec("stderr", goMakeDirExist, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoHead,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirExist,
	},

	"check go-make hash to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoHash, false),
			LogExec("stderr", goMakeDirExist, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			LogExec("stderr", goMakeDirExist,
				make.CmdGitHashReset(infoHash.Revision)),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(infoHash.Revision), nil, "", ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoHash,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirExist,
	},

	"check go-make tag to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoTag, false),
			LogExec("stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision)),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashTag(infoTag.Revision), nil, revisionDefault, ""),
			LogExec("stderr", goMakeDirExist, make.CmdGitHashNow()),
			Exec("builder", "stderr", goMakeDirExist,
				make.CmdGitHashNow(), nil, revisionHead, ""),
			LogExec("stderr", goMakeDirExist,
				make.CmdGitHashReset(revisionDefault)),
			Exec("stderr", "stderr", goMakeDirExist,
				make.CmdGitHashReset(revisionDefault), nil, "", ""),
			LogExec("stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...)),
			Exec("stdout", "stderr", dirExec,
				make.CmdMakeTargets(make.Makefile, argsTraceTarget...),
				nil, "", ""),
		),
		info:      infoTag,
		args:      argsTraceTarget,
		goMakeDir: goMakeDirExist,
	},
}

func TestMakeMock(t *testing.T) {
	test.Map(t, testMakeParams).
		Run(func(t test.Test, param MakeParams) {
			// Given
			//revive:disable-next-line:redefines-builtin-id // Is package name.
			make, _ := GoMakeSetup(t, param) //nolint:predeclared // Is package name.

			// When
			exit, err := make.Make(param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectExit, exit)
		})
}

//go:embed fixtures/*
var fixtures embed.FS

func ReadFile(fs embed.FS, name string) string {
	out, err := fs.ReadFile(name)
	if err == nil && out != nil {
		return string(out)
	} else if err != nil {
		panic(err)
	}
	panic("no output")
}

func SetupMakeConfig(t test.Test, src string) {
	t.Helper()

	if dir, err := os.Executable(); err == nil {
		dst := dir + ".config"
		cmd := exec.Command("cp", "--recursive", src, dst)
		out := &strings.Builder{}
		cmd.Stdout, cmd.Stderr = out, out
		require.NoError(t, cmd.Run(), "copying failed", dst, out.String())
	} else {
		require.NoError(t, err, "executable failed", dir)
	}
}

var (
	// regexMatchTestDir is the regular expression that is used to remove the
	// test execution path dependent parts.
	regexMatchTestDir = regexp.MustCompile(
		"(?m)/?/tmp/go-build.*/make.test.config/")
	// regexMatchBuildDir is the regular expression that is used to remove the
	// build path dependent parts.
	regexMatchSourceDir = regexp.MustCompile( //nolint:gosimple // Just wrong!
		"(?m)(['\\[])([^'\\]]*/)(go-make/[^'\\]]*)(['\\]])")
)

func FilterMakeOutput(str string) string {
	str = regexMatchTestDir.ReplaceAllString(str, "")
	return regexMatchSourceDir.ReplaceAllString(str, "$1$3$4")
}

type MakeExecParams struct {
	info         *info.Info
	args         []string
	expectExit   int
	expectStdout string
	expectStderr string
}

var testMakeExecParams = map[string]MakeExecParams{
	"go-make version": {
		info:         infoDirty,
		args:         []string{"go-make", "--version"},
		expectStdout: ReadFile(fixtures, "fixtures/version.out"),
	},
	"go-make version trace": {
		info:         infoDirty,
		args:         []string{"go-make", "--trace", "--version"},
		expectStdout: ReadFile(fixtures, "fixtures/version.out"),
		expectStderr: ReadFile(fixtures, "fixtures/version-trace.err"),
	},

	"go-make bash": {
		info:         infoDirty,
		args:         []string{"go-make", "--completion=bash"},
		expectStdout: ReadFile(fixtures, "fixtures/bash.out"),
	},
	"go-make bash trace": {
		info:         infoDirty,
		args:         []string{"go-make", "--trace", "--completion=bash"},
		expectStdout: ReadFile(fixtures, "fixtures/bash.out"),
		expectStderr: ReadFile(fixtures, "fixtures/bash-trace.err"),
	},

	"go-make targets": {
		info:         infoDirty,
		args:         []string{"go-make", "targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets.err"),
	},
	"go-make targets trace": {
		args:         []string{"go-make", "targets", "--trace"},
		expectStdout: ReadFile(fixtures, "fixtures/targets-trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets-trace.err"),
	},
}

func TestMakeExec(t *testing.T) {
	SetupMakeConfig(t, "../..")

	test.Map(t, testMakeExecParams).
		Run(func(t test.Test, param MakeExecParams) {
			// Given
			info := infoDirty
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}

			// When
			exit := make.Make(info, stdout, stderr, param.args...)

			// Then
			assert.Equal(t, param.expectExit, exit)
			assert.Equal(t, param.expectStdout,
				FilterMakeOutput(stdout.String()))
			assert.Equal(t, param.expectStderr,
				FilterMakeOutput(stderr.String()))
		})
}
