package make_test

import (
	"embed"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

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
	// goMakePath contains an arbitrary source path for go-make.
	goMakePath = "github.com/tkrop/go-make"
	// latest contains the latest version.
	latest = "latest"
)

var (
	// dirWork contains an arbitrary working directory (uses current).
	dirWork, _ = os.Getwd()
	// infoBase with version and revision.
	infoBase = info.NewInfo(goMakePath,
		"v0.0.25",
		"ba4ff068e795443f256caa06180d976a0fb244e9",
		"2024-01-09T13:02:46+01:00",
		"2024-01-10T16:22:54+01:00",
		true)

	infoNew = info.NewInfo(goMakePath, latest,
		"ba4ff068e795443f256caa06180d976a0fb244e9",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	argsVersion      = []string{"--version"}
	argsBash         = []string{"--completion=bash"}
	argsTarget       = []string{"target"}
	argsConfigTarget = []string{"--config=" + latest, "target"}
	argsTraceVersion = []string{"--trace", "--version"}
	argsTraceBash    = []string{"--trace", "--completion=bash"}
	argsTraceTarget  = []string{"--trace", "target"}

	// Any error that can happen.
	errAny = errors.New("any error")
)

func Makefile(path string, version string) string {
	return filepath.Join(make.GoMakePath(path, version), make.Makefile)
}

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

	gm := make.NewGoMake(param.info, "",
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
	)

	gm.Executor = mock.Get(mocks, NewMockExecutor)
	gm.Logger = mock.Get(mocks, NewMockLogger)

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
	args        []string
	expectError error
	expectExit  int
}

var testMakeParams = map[string]MakeParams{
	// targets without trace.
	"go-make version": {
		mockSetup: mock.Chain(
			LogInfo("stdout", infoBase, true),
		),
		info: infoBase,
		args: argsVersion,
	},
	"go-make completion bash": {
		mockSetup: mock.Chain(
			LogMessage("stdout", make.BashCompletion),
		),
		info: infoBase,
		args: argsBash,
	},

	"go-make target": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			Exec("stdout", "stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTarget...),
				nil, "", ""),
		),
		info: infoBase,
		args: argsTarget,
	},
	"go-make target failed": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			Exec("stdout", "stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTarget...),
				errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoBase, false),
			LogError("stderr", "execute make", make.NewErrCallFailed(
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTarget...), errAny)),
		),
		info: infoBase,
		args: argsTarget,
		expectError: make.NewErrCallFailed(make.CmdMakeTargets(
			Makefile(infoBase.Path, infoBase.Version), argsTarget...),
			errAny),
		expectExit: 2,
	},

	"go-make target install": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoNew.Path, infoNew.Version)),
				errAny, "", ""),
			Exec("stderr", "stderr", dirWork, make.CmdGoInstall(
				infoNew.Path, infoNew.Version), nil, "", ""),
			Exec("stdout", "stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoNew.Path, infoNew.Version), argsTarget...),
				nil, "", ""),
		),
		info: infoNew,
		args: argsTarget,
	},
	"go-make target install failed": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoNew.Path, infoNew.Version)),
				errAny, "", ""),
			Exec("stderr", "stderr", dirWork, make.CmdGoInstall(
				infoNew.Path, infoNew.Version), errAny, "", ""),
			LogCall("stderr", argsTarget),
			LogInfo("stderr", infoNew, false),
			LogError("stderr", "ensure config", make.NewErrNotFound(
				infoNew.Path, infoNew.Version, make.NewErrCallFailed(
					make.CmdGoInstall(infoNew.Path, infoNew.Version),
					errAny))),
		),
		info: infoNew,
		args: argsTarget,
		expectError: make.NewErrNotFound(
			infoNew.Path, infoNew.Version, make.NewErrCallFailed(
				make.CmdGoInstall(infoNew.Path, infoNew.Version),
				errAny)),
		expectExit: 1,
	},

	"go-make target config": {
		mockSetup: mock.Chain(
			Exec("stderr", "stderr", dirWork,
				make.CmdTestDir(latest), errAny, "", ""),
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, latest)), errAny, "", ""),
			Exec("stderr", "stderr", dirWork, make.CmdGoInstall(
				infoBase.Path, latest), nil, "", ""),
			Exec("stdout", "stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, latest), argsTarget...),
				nil, "", ""),
		),
		info: infoBase,
		args: argsConfigTarget,
	},

	// targets without trace.
	"go-make version traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceVersion),
			LogInfo("stderr", infoBase, false),
			LogInfo("stdout", infoBase, true),
		),
		info: infoBase,
		args: argsTraceVersion,
	},
	"go-make completion bash traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceBash),
			LogInfo("stderr", infoBase, false),
			LogMessage("stdout", make.BashCompletion),
		),
		info: infoBase,
		args: argsTraceBash,
	},
	"go-make target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoBase, false),
			LogExec("stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version))),
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			LogExec("stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...)),
			Exec("stdout", "stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...),
				nil, "", ""),
		),
		info: infoBase,
		args: argsTraceTarget,
	},
	"go-make target failed traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceTarget),
			LogInfo("stderr", infoBase, false),
			LogExec("stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version))),
			Exec("stderr", "stderr", dirWork, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			LogExec("stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...)),
			Exec("stdout", "stderr", dirWork, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...),
				errAny, "", ""),
			LogError("stderr", "execute make", make.NewErrCallFailed(
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTraceTarget...), errAny)),
		),
		info: infoBase,
		args: argsTraceTarget,
		expectError: make.NewErrCallFailed(make.CmdMakeTargets(
			Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...),
			errAny),
		expectExit: 2,
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

var (
	// regexMatchBuildDir is the regular expression that is used to remove the
	// build path dependent parts.
	//lint:ignore S1007 // Escaping makes it less readable.
	regexMatchSourceDir = regexp.MustCompile( //nolint:gosimple // Just wrong!
		"(?m)(['\\[])([^'\\]]*/)(go-make/[^'\\]]*)(['\\]])")
	// regexMatchMakeLog is the regular expression that is used to remove the
	// make specific output in the  log parts.
	//lint:ignore S1007 // Escaping makes it less readable.
	regexMatchMakeLog = regexp.MustCompile( //nolint:gosimple // Just wrong!
		"(?m)make\\[[0-9]*\\]: (Entering|Leaving) directory [^\\n]*\\n")
	// replaceFixture replaces the placeholders in the fixture with the values
	// provided to the replacer.
	replacerFixture = strings.NewReplacer(
		"{{GOVERSION}}", runtime.Version()[2:],
		"{{PLATFORM}}", runtime.GOOS+"/"+runtime.GOARCH,
		"{{COMPILER}}", runtime.Compiler)
)

func FilterMakeOutput(str string) string {
	str = regexMatchMakeLog.ReplaceAllString(str, "")
	str = regexMatchSourceDir.ReplaceAllString(str, "$1$3$4")
	return str
}

func SetupMakeFixture(str string) string {
	return replacerFixture.Replace(str)
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
		info:         infoBase,
		args:         []string{"go-make", "--version"},
		expectStdout: ReadFile(fixtures, "fixtures/version.out"),
	},
	"go-make version trace": {
		info:         infoBase,
		args:         []string{"go-make", "--trace", "--version"},
		expectStdout: ReadFile(fixtures, "fixtures/version.out"),
		expectStderr: ReadFile(fixtures, "fixtures/version-trace.err"),
	},

	"go-make bash": {
		info:         infoBase,
		args:         []string{"go-make", "--completion=bash"},
		expectStdout: ReadFile(fixtures, "fixtures/bash.out"),
	},
	"go-make bash trace": {
		info:         infoBase,
		args:         []string{"go-make", "--trace", "--completion=bash"},
		expectStdout: ReadFile(fixtures, "fixtures/bash.out"),
		expectStderr: ReadFile(fixtures, "fixtures/bash-trace.err"),
	},

	"go-make targets": {
		info:         infoBase,
		args:         []string{"go-make", "targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets.err"),
	},
	"go-make targets trace": {
		args: []string{
			"go-make", "--trace", "targets",
		},
		expectStdout: ReadFile(fixtures, "fixtures/targets-trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets-trace.err"),
	},
}

// TODO: this test is sensitive to the execution parameters and fails if the
// make target is started with option `--trace`. We need to figure out how this
// influences the execution of the test and the output.
func TestMakeExec(t *testing.T) {
	test.Map(t, testMakeExecParams).
		Run(func(t test.Test, param MakeExecParams) {
			// Given
			info := infoBase
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}

			// When
			exit := make.Make(info, "../../config",
				stdout, stderr, param.args...)

			// Then
			assert.Equal(t, param.expectExit, exit)
			assert.Equal(t, SetupMakeFixture(param.expectStdout),
				FilterMakeOutput(stdout.String()))
			assert.Equal(t, SetupMakeFixture(param.expectStderr),
				FilterMakeOutput(stderr.String()))
		})
}
