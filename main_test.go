package main_test

import (
	"embed"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	main "github.com/tkrop/go-make"
	"github.com/tkrop/go-testing/mock"
	"github.com/tkrop/go-testing/test"
)

//revive:disable:line-length-limit // go:generate line length

//go:generate mockgen -package=main_test -destination=mock_gomake_test.go -source=main.go CmdExecutor Logger

//revive:enable:line-length-limit

const (
	// DirExist contains an arbitrary execution directory (use '.').
	DirExec = "."
	// GoMakeDirNew contains an arbitrary non-existing directory.
	GoMakeDirNew = "new-dir"
	// GoMakeDirExist contains an arbitrary existing directory (use build).
	GoMakeDirExist = "build"
	// GoMakeGit contains an arbitrary source repository for go-make.
	GoMakeGit = "git@github.com:tkrop/go-make"
	// GoMakeHTTP contains an arbitrary source repository for go-make.
	GoMakeHTTP = "https://github.com/tkrop/go-make.git"
	// RevisionHead contains an arbitrary head revision.
	RevisionHead = "1b66f320c950b25fa63b81fd4e660c5d1f9d758e"
	// HeadRevision contains an arbitrary default revision.
	RevisionDefault = "c0a7f81b82937ffe379ac39ece2925fa4d19fd40"
	// RevisionOther contains an arbitrary different revision.
	RevisionOther = "fbb61d4981d22b94412b906ea4d7ae3302d860d0"
)

var (
	// InfoDirty without version and revision hash.
	InfoDirty = main.NewDefaultInfo()

	// InfoTag with version and revision hash from git.
	InfoTag = main.NewInfo("github.com/tkrop/go-make",
		"v1.1.1",
		"v1.1.1",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	// InfoHash with revision hash from git.
	InfoHash = main.NewInfo("github.com/tkrop/go-make",
		"v0.0.0-20231110152254-1b66f320c950",
		RevisionDefault,
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	// InfoShort with revision hash from git to short.
	InfoShort = main.NewInfo("github.com/tkrop/go-make",
		"v0.0.0-20231110152254-1b66f320c950",
		RevisionDefault[:20],
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	// InfoHead without revision hash from git.
	InfoHead = main.NewInfo("github.com/tkrop/go-make",
		"v0.0.0-20231110152254-1b66f320c950",
		"",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	ArgsVersion      = []string{"--version"}
	ArgsBash         = []string{"--completion=bash"}
	ArgsTarget       = []string{"target"}
	ArgsDebugTarget  = []string{"--debug", "target"}
	ArgsTraceVersion = []string{"--trace", "--version"}
	ArgsTraceBash    = []string{"--trace", "--completion=bash"}
	ArgsTraceTarget  = []string{"--trace", "target"}

	// Any error that can happen.
	errAny = errors.New("any error")
)

// GoMakeSetup sets up a new go-make test with mocks.
func GoMakeSetup(
	t test.Test, param MainParams,
) (*main.GoMake, *mock.Mocks) {
	mocks := mock.NewMocks(t).
		SetArg("stdout", &strings.Builder{}).
		SetArg("stderr", &strings.Builder{}).
		SetArg("builder", &strings.Builder{}).
		Expect(param.mockSetup)

	gm := main.NewGoMake(
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
		param.info,
	)

	if param.goMakeDir == "" {
		param.goMakeDir = GoMakeDirExist
	}

	gm.Executor = mock.Get(mocks, NewMockCmdExecutor)
	gm.Logger = mock.Get(mocks, NewMockLogger)
	gm.Makefile = main.Makefile
	gm.CmdDir = param.goMakeDir
	gm.WorkDir = DirExec

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
		return mock.Get(mocks, NewMockCmdExecutor).EXPECT().
			Exec(mocks.GetArg(stdout), mocks.GetArg(stderr), dir, ToAny(args)...).
			DoAndReturn(mocks.Call(main.CmdExecutor.Exec,
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
			DoAndReturn(mocks.Do(main.Logger.Call))
	}
}

func LogExec(writer string, dir string, args []string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Exec(mocks.GetArg(writer), dir, ToAny(args)...).
			DoAndReturn(mocks.Do(main.Logger.Exec))
	}
}

func LogInfo(writer string, info *main.Info, raw bool) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Info(mocks.GetArg(writer), info, raw).
			DoAndReturn(mocks.Do(main.Logger.Info))
	}
}

func LogError(writer string, message string, err error) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Error(mocks.GetArg(writer), message, err).
			DoAndReturn(mocks.Do(main.Logger.Error))
	}
}

func LogMessage(writer string, message string) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Message(mocks.GetArg(writer), message).
			DoAndReturn(mocks.Do(main.Logger.Message))
	}
}

type MainParams struct {
	mockSetup   mock.SetupFunc
	info        *main.Info
	goMakeDir   string
	args        []string
	expectError error
}

var testMainParams = map[string]MainParams{
	// dirty option targets without trace.
	"check go-make completion bash": {
		mockSetup: mock.Chain(
			LogMessage("stdout", main.BashCompletion),
		),
		args: ArgsBash,
	},
	"check go-make version": {
		mockSetup: mock.Chain(
			LogInfo("stdout", InfoDirty, false),
		),
		info: InfoDirty,
		args: ArgsVersion,
	},
	"dirty go-make to run target": {
		mockSetup: mock.Chain(
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...), nil, "", ""),
		),
		info: InfoDirty,
		args: ArgsTarget,
	},

	"dirty go-make failed": {
		mockSetup: mock.Chain(
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...), errAny, "", ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoDirty, false),
			LogError("stderr", "execute make", main.NewErrCallFailed(
				main.CmdMakeTargets(main.Makefile, ArgsTarget...), errAny)),
		),
		info: InfoDirty,
		args: ArgsTarget,
		expectError: main.NewErrCallFailed(
			main.CmdMakeTargets(main.Makefile, ArgsTarget...), errAny),
	},

	"clone go-make clone failed": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), errAny, "", ""),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny, "", ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoDirty, false),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny)),
		),
		info:      InfoDirty,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirNew,
		expectError: main.NewErrCallFailed(
			main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny),
	},

	// dirty option targets with trace.
	"check go-make completion bash traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceBash),
			LogInfo("stderr", InfoDirty, false),
			LogMessage("stderr", main.BashCompletion),
		),
		info: InfoDirty,
		args: ArgsTraceBash,
	},
	"check go-make version traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceVersion),
			LogInfo("stderr", InfoDirty, false),
			LogInfo("stderr", InfoDirty, false),
		),
		info: InfoDirty,
		args: ArgsTraceVersion,
	},
	"dirty go-make to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoDirty, false),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...), nil, "", ""),
		),
		info: InfoDirty,
		args: ArgsTraceTarget,
	},
	"dirty go-make failed traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoDirty, false),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...), errAny, "", ""),
			LogError("stderr", "execute make", main.NewErrCallFailed(
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...), errAny)),
		),
		info: InfoDirty,
		args: ArgsTraceTarget,
		expectError: main.NewErrCallFailed(
			main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...), errAny),
	},
	"clone go-make failed traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoDirty, false),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), errAny, "", ""),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny, "", ""),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny)),
		),
		info:      InfoDirty,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirNew,
		expectError: main.NewErrCallFailed(
			main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny),
	},

	// dirty option targets with debug.
	"dirty go-make to run target debugged": {
		mockSetup: mock.Chain(
			LogExec("stdout", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsDebugTarget...)),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsDebugTarget...), nil, "", ""),
		),
		info: InfoDirty,
		args: ArgsDebugTarget,
	},

	"dirty go-make failed debugged": {
		mockSetup: mock.Chain(
			LogExec("stdout", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsDebugTarget...)),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsDebugTarget...), errAny, "", ""),
			LogCall("stderr", ArgsDebugTarget),
			LogInfo("stderr", InfoDirty, false),
			LogError("stderr", "execute make", main.NewErrCallFailed(
				main.CmdMakeTargets(main.Makefile, ArgsDebugTarget...), errAny)),
		),
		info: InfoDirty,
		args: ArgsDebugTarget,
		expectError: main.NewErrCallFailed(
			main.CmdMakeTargets(main.Makefile, ArgsDebugTarget...), errAny),
	},

	"clone go-make clone failed debugged": {
		mockSetup: mock.Chain(
			LogExec("stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew)),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), errAny, "", ""),
			LogExec("stderr", DirExec,
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew)),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny, "", ""),
			LogCall("stderr", ArgsDebugTarget),
			LogInfo("stderr", InfoDirty, false),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny)),
		),
		info:      InfoDirty,
		args:      ArgsDebugTarget,
		goMakeDir: GoMakeDirNew,
		expectError: main.NewErrCallFailed(
			main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), errAny),
	},

	// clone targets without trace.
	"clone go-make reset failed": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionOther, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(InfoHash.Revision), errAny, "", ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoHash, false),
			LogError("stderr", "update config",
				main.NewErrNotFound(GoMakeDirNew, InfoHash.Revision,
					main.NewErrCallFailed(main.CmdGitHashReset(InfoHash.Revision),
						errAny))),
		),
		info:      InfoHash,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirNew,
		expectError: main.NewErrNotFound(GoMakeDirNew, InfoHash.Revision,
			main.NewErrCallFailed(main.CmdGitHashReset(InfoHash.Revision), errAny)),
	},

	"clone go-make head to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashHead(), nil, RevisionHead, ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoHead,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirNew,
	},

	"clone go-make hash to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionOther, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(InfoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoHash,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirNew,
	},

	"clone go-make tag to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionOther, ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(RevisionOther), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoTag,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirNew,
	},

	"clone go-make fallback to run target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), errAny, "", ""),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionOther, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(InfoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoHash,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirNew,
	},

	// clone targets with trace.
	"clone go-make head to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoHead, false),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashHead(), nil, RevisionHead, ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoHead,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirNew,
	},

	"clone go-make hash to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoHash, false),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionOther, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(InfoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoHash,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirNew,
	},

	"clone go-make tag to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoTag, false),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionOther, ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(RevisionOther), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoTag,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirNew,
	},

	"clone go-make fallback to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoHash, false),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeGit, GoMakeDirNew), errAny, "", ""),
			Exec("stderr", "stderr", DirExec,
				main.CmdGitClone(GoMakeHTTP, GoMakeDirNew), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirNew,
				main.CmdGitHashNow(), nil, RevisionOther, ""),
			Exec("stderr", "stderr", GoMakeDirNew,
				main.CmdGitHashReset(InfoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoHash,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirNew,
	},

	// check targets without trace.
	"check go-make head hash failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashHead(), errAny, "", ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoHead, false),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitHashHead(), errAny)),
		),
		info:        InfoHead,
		args:        ArgsTarget,
		goMakeDir:   GoMakeDirExist,
		expectError: main.NewErrCallFailed(main.CmdGitHashHead(), errAny),
	},

	"check go-make head now failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashHead(), nil, RevisionHead, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), errAny, "", ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoHead, false),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitHashNow(), errAny)),
		),
		info:        InfoHead,
		args:        ArgsTarget,
		goMakeDir:   GoMakeDirExist,
		expectError: main.NewErrCallFailed(main.CmdGitHashNow(), errAny),
	},

	"check go-make tag log failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashTag(InfoTag.Revision), errAny, RevisionDefault, ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoTag, false),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitHashTag(InfoTag.Revision), errAny)),
		),
		info:      InfoTag,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirExist,
		expectError: main.NewErrCallFailed(
			main.CmdGitHashTag(InfoTag.Revision), errAny),
	},

	"check go-make tag fetch failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionDefault, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(RevisionDefault),
				main.NewErrNotFound(GoMakeDirExist, RevisionDefault, errAny), "", ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitFetch(InfoTag.Repo), errAny, "", ""),
			LogCall("stderr", ArgsTarget),
			LogInfo("stderr", InfoTag, false),
			LogError("stderr", "update config", main.NewErrCallFailed(
				main.CmdGitFetch(InfoTag.Repo), errAny)),
		),
		info:        InfoTag,
		args:        ArgsTarget,
		goMakeDir:   GoMakeDirExist,
		expectError: main.NewErrCallFailed(main.CmdGitFetch(InfoTag.Repo), errAny),
	},

	"check go-make head to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashHead(), nil, RevisionHead, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoHead,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirExist,
	},

	"check go-make hash to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(InfoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoHash,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirExist,
	},

	"check go-make tag to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionDefault, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(RevisionDefault), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoTag,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirExist,
	},

	"check go-make tag fetch to run target": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionDefault, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(RevisionDefault),
				main.NewErrNotFound(GoMakeDirExist, RevisionDefault, errAny), "", ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitFetch(InfoTag.Repo), nil, "", ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionDefault, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(RevisionDefault), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTarget...),
				nil, "", ""),
		),
		info:      InfoTag,
		args:      ArgsTarget,
		goMakeDir: GoMakeDirExist,
	},

	// check targets with trace.
	"check go-make head to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoHead, false),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashHead(), nil, RevisionHead, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoHead,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirExist,
	},

	"check go-make hash to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoHash, false),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(InfoHash.Revision), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoHash,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirExist,
	},

	"check go-make tag to run target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", ArgsTraceTarget),
			LogInfo("stderr", InfoTag, false),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashTag(InfoTag.Revision), nil, RevisionDefault, ""),
			Exec("builder", "stderr", GoMakeDirExist,
				main.CmdGitHashNow(), nil, RevisionHead, ""),
			Exec("stderr", "stderr", GoMakeDirExist,
				main.CmdGitHashReset(RevisionDefault), nil, "", ""),
			Exec("stdout", "stderr", DirExec,
				main.CmdMakeTargets(main.Makefile, ArgsTraceTarget...),
				nil, "", ""),
		),
		info:      InfoTag,
		args:      ArgsTraceTarget,
		goMakeDir: GoMakeDirExist,
	},
}

func TestMainMock(t *testing.T) {
	test.Map(t, testMainParams).
		Run(func(t test.Test, param MainParams) {
			// Given
			gomake, _ := GoMakeSetup(t, param)

			// When
			err := gomake.RunCmd(param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
		})
}

type MainTargetParams struct {
	args         []string
	expectError  error
	expectStdout string
	expectStderr string
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

var testMainTargetParams = map[string]MainTargetParams{
	"go-make targets": {
		args:         []string{"go-make", "targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets.out"),
	},
	// TODO: figure out how to test trace output.
	// "go-make targets trace": {
	// 	args:         []string{"go-make", "targets", "--trace"},
	// 	expectStdout: ReadFile(fixtures, "fixtures/targets-trace.out"),
	// },
}

func TestMainTargets(t *testing.T) {
	// Prepare go-make config.
	cmd, err := os.Executable()
	assert.NoError(t, err)
	cmdDir := cmd + ".config"
	t.Cleanup(func() {
		os.RemoveAll(cmdDir)
	})
	err = exec.Command("cp", "--recursive", DirExec, cmdDir).Run()
	assert.NoError(t, err)

	test.Map(t, testMainTargetParams).
		Run(func(t test.Test, param MainTargetParams) {
			// Given
			info := main.NewDefaultInfo()
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			info.Dirty = true

			// When
			err := main.Run(info, stdout, stderr, param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
			// TODO: fix output!
			assert.Equal(t, param.expectStdout, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}
