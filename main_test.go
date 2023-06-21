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
	// GoMakeDirNew contains an arbitrary non-existing directory.
	GoMakeDirNew = "new-dir"
	// GoMakeDirExist contains an arbitrary existing directory (use build).
	GoMakeDirExist = "build"
	// GoMakeGit contains an arbitrary source repository for go-make.
	GoMakeGit = "git@github.com:tkrop/go-make"
	// GoMakeHTTP contains an arbitrary source repository for go-make.
	GoMakeHTTP = "https://github.com/tkrop/go-make.git"
	// OtherRevision contains an arbitrary revision different from default.
	OtherRevision = "x1b66f320c950b25fa63b81fd4e660c5d1f9d758"
)

var (
	// DefaultInfo captured from once existing state.
	DefaultInfo = main.NewInfo("github.com/tkrop/go-make",
		"v0.0.0-20231110152254-1b66f320c950",
		"1b66f320c950b25fa63b81fd4e660c5d1f9d758e",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		"false")

	// ShortInfo with hash to short captured from once existing state.
	ShortInfo = main.NewInfo("github.com/tkrop/go-make",
		"v0.0.0-20231110152254-1b66f320c950",
		"1b66f320c950b25fa63b81fd4e660c5d1f9d758",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		"false")

	// HeadInfo without hash captured from once existing state.
	HeadInfo = main.NewInfo("github.com/tkrop/go-make",
		"v0.0.0-20231110152254-1b66f320c950",
		"",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		"false")

	// NewInfo captured created after installing.
	NewInfo = main.NewDefaultInfo()

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

	if param.info == nil {
		param.info = DefaultInfo
	}

	gm := main.NewGoMake(
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
		param.info, false,
	)

	if param.goMakeDir == "" {
		param.goMakeDir = GoMakeDirExist
	}

	gm.Executor = mock.Get(mocks, NewMockCmdExecutor)
	gm.Logger = mock.Get(mocks, NewMockLogger)
	gm.Makefile = main.Makefile
	gm.CmdDir = param.goMakeDir
	gm.WorkDir = "."

	return gm, mocks
}

func ToAny(args ...any) []any {
	return args
}

func Trace(trace bool) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockCmdExecutor).EXPECT().
			Trace().DoAndReturn(mocks.Do(main.CmdExecutor.Trace, trace))
	}
}

func Exec( //revive:disable:argument-limit
	stdout, stderr string, dir, name string,
	args []string, err error, sout, serr string,
) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockCmdExecutor).EXPECT().
			Exec(mocks.GetArg(stdout), mocks.GetArg(stderr), dir, name, ToAny(args)...).
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
	// successful options without trace.
	"check go-make version": {
		mockSetup: mock.Chain(Trace(false),
			LogInfo("stdout", DefaultInfo, false),
		),
		args: []string{"--version"},
	},
	"check new go-make version": {
		mockSetup: mock.Chain(Trace(false),
			LogInfo("stdout", NewInfo, false),
		),
		info: NewInfo,
		args: []string{"--version"},
	},

	"check go-make completion bash": {
		mockSetup: mock.Chain(Trace(false),
			LogMessage("stdout", main.BashCompletion),
		),
		args: []string{"--completion=bash"},
	},

	// successful options without trace.
	"check go-make version with trace": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "--version"}),
			LogInfo("stderr", DefaultInfo, false),
			LogInfo("stdout", DefaultInfo, false),
		),
		args: []string{"--trace", "--version"},
	},

	"check go-make completion bash with trace": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "--completion=bash"}),
			LogInfo("stderr", DefaultInfo, false),
			LogMessage("stderr", main.BashCompletion),
		),
		args: []string{"--trace", "--completion=bash"},
	},

	// successful targets without trace.
	"check go-make to run target": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, DefaultInfo.Revision, ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, nil, "", ""),
		),
		args: []string{"target"},
	},

	"fetch and reset go-make to run target": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, OtherRevision, ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, nil, "", ""),
		),
		args: []string{"target"},
	},

	"fetch and reset head go-make to run target": {
		mockSetup: mock.Chain(Trace(false),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"reset", "--hard", "HEAD"}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, nil, "", ""),
		),
		info: HeadInfo,
		args: []string{"target"},
	},

	"clone and reset go-make to run target": {
		mockSetup: mock.Chain(Trace(false),
			Exec("stderr", "stderr", ".", "git",
				[]string{"clone", "--depth=1", GoMakeGit, GoMakeDirNew},
				nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, nil, "", ""),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"target"},
	},

	"clone fallback and reset go-make to run target": {
		mockSetup: mock.Chain(Trace(false),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeGit, GoMakeDirNew,
			}, errAny, "", ""),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeHTTP, GoMakeDirNew,
			}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, nil, "", ""),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"target"},
	},

	// successful targets with trace.
	"check go-make to run target with trace": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, DefaultInfo.Revision, ""),
			Exec("stdout", "stderr", ".", "make", []string{
				"--file", main.Makefile, "--trace", "target",
			}, nil, "", ""),
		),
		args: []string{"--trace", "target"},
	},

	"fetch and reset go-make to run target with trace": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, OtherRevision, ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "--trace", "target"}, nil, "", ""),
		),
		args: []string{"--trace", "target"},
	},

	"clone and reset go-make to run target with trace": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("stderr", "stderr", ".", "git",
				[]string{"clone", "--depth=1", GoMakeGit, GoMakeDirNew},
				nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "--trace", "target"}, nil, "", ""),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"--trace", "target"},
	},

	"clone fallback and reset go-make to run target with trace": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeGit, GoMakeDirNew,
			}, errAny, "", ""),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeHTTP, GoMakeDirNew,
			}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "--trace", "target"}, nil, "", ""),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"--trace", "target"},
	},

	// failed targets without trace.
	"check go-make to run target failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, DefaultInfo.Revision, ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "execute make", main.ErrCallFailed("make",
				[]string{"--file", main.Makefile, "target"}, errAny)),
		),
		args: []string{"target"},
		expectError: main.ErrCallFailed("make", []string{
			"--file", main.Makefile, "target",
		}, errAny),
	},

	"fetch and reset go-make to run target failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, OtherRevision, ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "execute make", main.ErrCallFailed("make",
				[]string{"--file", main.Makefile, "target"}, errAny)),
		),
		args: []string{"target"},
		expectError: main.ErrCallFailed("make", []string{
			"--file", main.Makefile, "target",
		}, errAny),
	},

	"clone and reset go-make to run target failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("stderr", "stderr", ".", "git",
				[]string{"clone", "--depth=1", GoMakeGit, GoMakeDirNew},
				nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "target"}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "execute make", main.ErrCallFailed("make",
				[]string{"--file", main.Makefile, "target"}, errAny)),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"target"},
		expectError: main.ErrCallFailed("make", []string{
			"--file", main.Makefile, "target",
		}, errAny),
	},

	// failed targets with trace.
	"check go-make to run target with trace failed": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, DefaultInfo.Revision, ""),
			Exec("stdout", "stderr", ".", "make", []string{
				"--file", main.Makefile, "--trace", "target",
			}, errAny, "", ""),
			Trace(true),
			LogError("stderr", "execute make", main.ErrCallFailed("make",
				[]string{"--file", main.Makefile, "--trace", "target"}, errAny)),
		),
		args: []string{"--trace", "target"},
		expectError: main.ErrCallFailed("make", []string{
			"--file", main.Makefile, "--trace", "target",
		}, errAny),
	},

	"fetch and reset go-make to run target with trace failed": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, OtherRevision, ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "--trace", "target"}, errAny, "", ""),
			Trace(true),
			LogError("stderr", "execute make", main.ErrCallFailed("make",
				[]string{"--file", main.Makefile, "--trace", "target"}, errAny)),
		),
		args: []string{"--trace", "target"},
		expectError: main.ErrCallFailed("make", []string{
			"--file", main.Makefile, "--trace", "target",
		}, errAny),
	},

	"clone and reset go-make to run target with trace failed": {
		mockSetup: mock.Chain(Trace(true),
			LogCall("stderr", []string{"--trace", "target"}),
			LogInfo("stderr", DefaultInfo, false),
			Exec("stderr", "stderr", ".", "git",
				[]string{"clone", "--depth=1", GoMakeGit, GoMakeDirNew},
				nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, nil, "", ""),
			Exec("stdout", "stderr", ".", "make",
				[]string{"--file", main.Makefile, "--trace", "target"}, errAny, "", ""),
			Trace(true),
			LogError("stderr", "execute make", main.ErrCallFailed("make",
				[]string{"--file", main.Makefile, "--trace", "target"}, errAny)),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"--trace", "target"},
		expectError: main.ErrCallFailed("make", []string{
			"--file", main.Makefile, "--trace", "target",
		}, errAny),
	},

	// failed setup without trace.
	"check go-make failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "update config", main.ErrCallFailed("git",
				[]string{"rev-parse", "HEAD"}, errAny)),
		),
		args: []string{"target"},
		expectError: main.ErrCallFailed("git",
			[]string{"rev-parse", "HEAD"}, errAny),
	},

	"fetch go-make failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, OtherRevision, ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "update config", main.ErrCallFailed("git",
				[]string{"fetch", GoMakeGit}, errAny)),
		),
		args:        []string{"target"},
		expectError: main.ErrCallFailed("git", []string{"fetch", GoMakeGit}, errAny),
	},

	"fetch and reset go-make failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("builder", "stderr", GoMakeDirExist, "git",
				[]string{"rev-parse", "HEAD"}, nil, OtherRevision, ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"fetch", GoMakeGit}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirExist, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "update config", main.ErrCallFailed("git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, errAny)),
		),
		goMakeDir: GoMakeDirExist,
		args:      []string{"target"},
		expectError: main.ErrCallFailed("git",
			[]string{"reset", "--hard", DefaultInfo.Revision}, errAny),
	},

	"clone fallback go-make failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeGit, GoMakeDirNew,
			}, errAny, "", ""),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeHTTP, GoMakeDirNew,
			}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "update config", main.ErrCallFailed("git",
				[]string{"clone", "--depth=1", GoMakeHTTP, GoMakeDirNew}, errAny)),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"target"},
		expectError: main.ErrCallFailed("git",
			[]string{"clone", "--depth=1", GoMakeHTTP, GoMakeDirNew}, errAny),
	},

	"clone fallback and reset go-make failed": {
		mockSetup: mock.Chain(Trace(false),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeGit, GoMakeDirNew,
			}, errAny, "", ""),
			Exec("stderr", "stderr", ".", "git", []string{
				"clone", "--depth=1",
				GoMakeHTTP, GoMakeDirNew,
			}, nil, "", ""),
			Exec("stderr", "stderr", GoMakeDirNew, "git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, errAny, "", ""),
			Trace(false),
			LogCall("stderr", []string{"target"}),
			LogInfo("stderr", DefaultInfo, false),
			LogError("stderr", "update config", main.ErrCallFailed("git",
				[]string{"reset", "--hard", DefaultInfo.Revision}, errAny)),
		),
		goMakeDir: GoMakeDirNew,
		args:      []string{"target"},
		expectError: main.ErrCallFailed("git",
			[]string{"reset", "--hard", DefaultInfo.Revision}, errAny),
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
	err = exec.Command("cp", "--recursive", ".", cmdDir).Run()
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
