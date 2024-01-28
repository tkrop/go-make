package make_test

import (
	"embed"
	"errors"
	"io"
	"os/exec"
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
)

var (
	// dirWork contains an arbitrary working directory (uses current).
	dirWork = "."
	// dirRoot contains an arbitrary root directory (use go-make root).
	dirRoot = filepath.Dir(filepath.Dir(make.AbsPath(dirWork)))
	// infoBase with version and revision.
	infoBase = info.NewInfo(goMakePath,
		"v0.0.25",
		"ba4ff068e795443f256caa06180d976a0fb244e9",
		"2024-01-09T13:02:46+01:00",
		"2024-01-10T16:22:54+01:00",
		true)

	infoNew = info.NewInfo(goMakePath, "latest",
		"ba4ff068e795443f256caa06180d976a0fb244e9",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00",
		false)

	argsVersion      = []string{"--version"}
	argsBash         = []string{"--completion=bash"}
	argsTarget       = []string{"target"}
	argsTargetCustom = []string{"--config=custom", "target"}
	argsTargetLatest = []string{"--config=latest", "target"}
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

	gm := make.NewGoMake(
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
		// Filling the test coverage gap of returning the default.
		param.info, make.GetEnvDefault(make.EnvGoMakeConfig, ""), ".",
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
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTarget...),
				nil, "", ""),
		),
		info: infoBase,
		args: argsTarget,
	},
	"go-make target install": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoNew.Path, infoNew.Version)),
				errAny, "", ""),
			Exec("stderr", "stderr", dirRoot, make.CmdGoInstall(
				infoNew.Path, infoNew.Version), nil, "", ""),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoNew.Path, infoNew.Version), argsTarget...),
				nil, "", ""),
		),
		info: infoNew,
		args: argsTarget,
	},
	"go-make target config custom": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.AbsPath("custom")), nil, "", ""),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
				filepath.Join(make.AbsPath("custom"), make.Makefile),
				argsTarget...),
				nil, "", ""),
		),
		info: infoBase,
		args: argsTargetCustom,
	},
	"go-make target config version latest": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.AbsPath("latest")), errAny, "", ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, "latest")), errAny, "", ""),
			Exec("stderr", "stderr", dirRoot, make.CmdGoInstall(
				infoBase.Path, "latest"), nil, "", ""),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, "latest"), argsTarget...),
				nil, "", ""),
		),
		info: infoBase,
		args: argsTargetLatest,
	},

	"go-make target top failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				errAny, "", ""),
			LogError("stderr", "ensure top", make.NewErrCallFailed(
				dirWork, make.CmdGitTop(), errAny)),
		),
		info: infoBase,
		args: argsTarget,
		expectError: make.NewErrCallFailed(dirWork,
			make.CmdGitTop(), errAny),
		expectExit: make.ExitGitFailure,
	},
	"go-make target install failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoNew.Path, infoNew.Version)),
				errAny, "", ""),
			Exec("stderr", "stderr", dirRoot, make.CmdGoInstall(
				infoNew.Path, infoNew.Version), errAny, "", ""),
			LogError("stderr", "ensure config", make.NewErrNotFound(
				infoNew.Path, infoNew.Version, make.NewErrCallFailed(dirRoot,
					make.CmdGoInstall(infoNew.Path, infoNew.Version), errAny))),
		),
		info: infoNew,
		args: argsTarget,
		expectError: make.NewErrNotFound(
			infoNew.Path, infoNew.Version, make.NewErrCallFailed(dirRoot,
				make.CmdGoInstall(infoNew.Path, infoNew.Version), errAny)),
		expectExit: make.ExitConfigFailure,
	},
	"go-make target failed": {
		mockSetup: mock.Chain(
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTarget...),
				errAny, "", ""),
			LogError("stderr", "execute make", make.NewErrCallFailed(dirRoot,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTarget...), errAny)),
		),
		info: infoBase,
		args: argsTarget,
		expectError: make.NewErrCallFailed(dirRoot, make.CmdMakeTargets(
			Makefile(infoBase.Path, infoBase.Version), argsTarget...), errAny),
		expectExit: make.ExitTargetFailure,
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
			LogExec("stderr", dirWork, make.CmdGitTop()),
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			LogExec("stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version))),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			LogExec("stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...)),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
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
			LogExec("stderr", dirWork, make.CmdGitTop()),
			Exec("builder", "stderr", dirWork, make.CmdGitTop(),
				nil, dirRoot, ""),
			LogExec("stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version))),
			Exec("stderr", "stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version)),
				nil, "", ""),
			LogExec("stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...)),
			Exec("stdout", "stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...),
				errAny, "", ""),
			LogError("stderr", "execute make", make.NewErrCallFailed(dirRoot,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTraceTarget...), errAny)),
		),
		info: infoBase,
		args: argsTraceTarget,
		expectError: make.NewErrCallFailed(dirRoot, make.CmdMakeTargets(
			Makefile(infoBase.Path, infoBase.Version), argsTraceTarget...),
			errAny),
		expectExit: make.ExitTargetFailure,
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
	// regexMakePrintDir is the regular expression used to remove the make
	// print directory specific output. At the moment printing the directory
	// is disabled by default and the filter is matching nothing.
	regexMakePrintDir = regexp.MustCompile(
		`(?m)make\[[0-9]*\]: (Entering|Leaving) directory [^\n]*\n`)
	// regexGoMakeWarning is the regular expression that is used to remove the
	// go-make version mismatch warning that happens after bumping the version.
	regexGoMakeWarning = regexp.MustCompile(
		`(?m).*warning:.*go-make version.*\n`)
	// regexGoMakeMakePath is the regular expression used to remove the go-make
	// source specific path information.
	regexGoMakeSource = regexp.MustCompile(`(?m)` + make.AbsPath(dirRoot))
	// regexMakeTrace is the regular expression used to match the make trace
	// output and to remove the line number to match resiliently when make
	// targets are moved around.
	regexMakeTrace = regexp.MustCompile(
		`(?m)(go-make/config/Makefile.base:)[0-9]+:`)
	// replaceFixture replaces the placeholders in the fixture with the values
	// provided to the replacer.
	replacerFixture = strings.NewReplacer(
		"{{GOVERSION}}", runtime.Version()[2:],
		"{{PLATFORM}}", runtime.GOOS+"/"+runtime.GOARCH,
		"{{COMPILER}}", runtime.Compiler)
)

func FilterMakeOutput(str string) string {
	str = regexMakePrintDir.ReplaceAllString(str, "")
	str = regexGoMakeWarning.ReplaceAllString(str, "")
	str = regexGoMakeSource.ReplaceAllString(str, "go-make")
	str = regexMakeTrace.ReplaceAllString(str, "$1")
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

	"go-make show targets": {
		info:         infoBase,
		args:         []string{"go-make", "show-targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets.err"),
	},
	"go-make show targets trace": {
		args:         []string{"go-make", "--trace", "show-targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets-trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets-trace.err"),
	},
}

// TODO: this test is sensitive to the execution parameters and fails if the
// make target is started with option `--trace`. We need to figure out how this
// influences the execution of the test and the output.
func TestMakeExec(t *testing.T) {
	workDir := make.AbsPath("../../run")
	configDir := make.AbsPath("../../config")

	cmd := exec.Command("mkdir", "-p", workDir)
	assert.NoError(t, cmd.Run())
	cmd = exec.Command("git", "init", workDir)
	assert.NoError(t, cmd.Run())

	test.Map(t, testMakeExecParams).
		Run(func(t test.Test, param MakeExecParams) {
			// Given
			info := infoBase
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}

			// When
			exit := make.Make(stdout, stderr, info,
				configDir, workDir, param.args...)

			// Then
			assert.Equal(t, param.expectExit, exit)
			assert.Equal(t, SetupMakeFixture(param.expectStdout),
				FilterMakeOutput(stdout.String()))
			assert.Equal(t, SetupMakeFixture(param.expectStderr),
				FilterMakeOutput(stderr.String()))
		})
}
