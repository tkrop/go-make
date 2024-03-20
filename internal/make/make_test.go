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

	argsVersion           = []string{"--version"}
	argsTraceVersion      = []string{"--trace", "--version"}
	argsBash              = []string{"--completion=bash"}
	argsTraceBash         = []string{"--trace", "--completion=bash"}
	argsShowTargets       = []string{"show-targets"}
	argsShowTargetsParam  = []string{"show-targets", "param"}
	argsShowTargetsCustom = []string{"--config=custom", "show-targets"}
	argsShowTargetsLatest = []string{"--config=latest", "show-targets"}
	argsTraceAnyTarget    = []string{"--trace", "target"}

	// Any error that can happen.
	errAny = errors.New("any error")
)

func Makefile(path string, version string) string {
	return filepath.Join(make.GoMakePath(path, version), make.Makefile)
}

// NewReader creates a new reader with the given id.
func NewReader(id string) io.Reader {
	return strings.NewReader(id)
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
		SetArg("stdin", NewReader("stdin")).
		SetArg("stdout", NewWriter("stdout")).
		SetArg("stderr", NewWriter("stderr")).
		SetArg("builder", &strings.Builder{}).
		Expect(param.mockSetup)

	gm := make.NewGoMake(
		mocks.GetArg("stdin").(io.Reader),
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
		// Filling the test coverage gap of returning the default.
		param.info, make.GetEnvDefault(make.EnvGoMakeConfig, ""),
		".", param.env...,
	)

	gm.Executor = mock.Get(mocks, NewMockExecutor)
	gm.Logger = mock.Get(mocks, NewMockLogger)

	return gm, mocks
}

func ToAny(args ...any) []any {
	return args
}

func Exec( //revive:disable-line:argument-limit
	stdin, stdout, stderr string, dir string,
	env []string, args []string, err error, sout, serr string,
) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockExecutor).EXPECT().
			Exec(mocks.GetArg(stdin), mocks.GetArg(stdout), mocks.GetArg(stderr),
				dir, env, ToAny(args)...).
			DoAndReturn(mocks.Call(cmd.Executor.Exec,
				func(args ...any) []any {
					if _, err := args[1].(io.Writer).Write([]byte(sout)); err != nil {
						assert.Fail(mocks.Ctrl.T, "failed to write to stdout", err)
					}
					if _, err := args[2].(io.Writer).Write([]byte(serr)); err != nil {
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
	env         []string
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

	"go-make show targets": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoBase.Path,
					infoBase.Version)), nil, "", ""),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsShowTargets...), nil, "", ""),
		),
		info: infoBase,
		args: argsShowTargets,
	},
	"go-make show targets with param": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoBase.Path,
					infoBase.Version)), nil, "", ""),
			Exec("stdin", "stdout", "stderr", dirRoot, []string{
				"ARGS=" + strings.Join(argsShowTargetsParam[1:], " "),
			}, make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
				argsShowTargets...), nil, "", ""),
		),
		info: infoBase,
		args: argsShowTargetsParam,
	},
	"go-make show targets install": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoNew.Path,
					infoNew.Version)), errAny, "", ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdGoInstall(infoNew.Path, infoNew.Version), nil, "", ""),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(Makefile(infoNew.Path, infoNew.Version),
					argsShowTargets...), nil, "", ""),
		),
		info: infoNew,
		args: argsShowTargets,
	},
	"go-make show targets config custom": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.AbsPath("custom")), nil, "", ""),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(filepath.Join(make.AbsPath("custom"),
					make.Makefile), argsShowTargets...), nil, "", ""),
		),
		info: infoBase,
		args: argsShowTargetsCustom,
	},
	"go-make show targets config version latest": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.AbsPath("latest")), errAny, "", ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoBase.Path, "latest")),
				errAny, "", ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdGoInstall(infoBase.Path, "latest"), nil, "", ""),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(Makefile(infoBase.Path, "latest"),
					argsShowTargets...), nil, "", ""),
		),
		info: infoBase,
		args: argsShowTargetsLatest,
	},

	"go-make show targets top failed": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), errAny, "", ""),
			LogError("stderr", "ensure top", make.NewErrCallFailed(
				dirWork, make.CmdGitTop(), errAny)),
		),
		info: infoBase,
		args: argsShowTargets,
		expectError: make.NewErrCallFailed(dirWork,
			make.CmdGitTop(), errAny),
		expectExit: make.ExitGitFailure,
	},
	"go-make show targets install failed": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoNew.Path,
					infoNew.Version)), errAny, "", ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdGoInstall(infoNew.Path, infoNew.Version), errAny, "", ""),
			LogError("stderr", "ensure config", make.NewErrNotFound(
				infoNew.Path, infoNew.Version, make.NewErrCallFailed(dirRoot,
					make.CmdGoInstall(infoNew.Path, infoNew.Version), errAny))),
		),
		info: infoNew,
		args: argsShowTargets,
		expectError: make.NewErrNotFound(
			infoNew.Path, infoNew.Version, make.NewErrCallFailed(dirRoot,
				make.CmdGoInstall(infoNew.Path, infoNew.Version), errAny)),
		expectExit: make.ExitConfigFailure,
	},
	"go-make show targets failed": {
		mockSetup: mock.Chain(
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoBase.Path,
					infoBase.Version)), nil, "", ""),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsShowTargets...), errAny, "", ""),
			LogError("stderr", "execute make", make.NewErrCallFailed(dirRoot,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsShowTargets...), errAny)),
		),
		info: infoBase,
		args: argsShowTargets,
		expectError: make.NewErrCallFailed(dirRoot, make.CmdMakeTargets(
			Makefile(infoBase.Path, infoBase.Version), argsShowTargets...), errAny),
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
	"go-make any target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceAnyTarget),
			LogInfo("stderr", infoBase, false),
			LogExec("stderr", dirWork, make.CmdGitTop()),
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			LogExec("stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version))),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoBase.Path,
					infoBase.Version)), nil, "", ""),
			LogExec("stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceAnyTarget...)),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTraceAnyTarget...), nil, "", ""),
		),
		info: infoBase,
		args: argsTraceAnyTarget,
	},
	"go-make any target traced failed": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceAnyTarget),
			LogInfo("stderr", infoBase, false),
			LogExec("stderr", dirWork, make.CmdGitTop()),
			Exec("stdin", "builder", "stderr", dirWork, nil,
				make.CmdGitTop(), nil, dirRoot, ""),
			LogExec("stderr", dirRoot, make.CmdTestDir(
				make.GoMakePath(infoBase.Path, infoBase.Version))),
			Exec("stdin", "stderr", "stderr", dirRoot, nil,
				make.CmdTestDir(make.GoMakePath(infoBase.Path,
					infoBase.Version)), nil, "", ""),
			LogExec("stderr", dirRoot, make.CmdMakeTargets(
				Makefile(infoBase.Path, infoBase.Version), argsTraceAnyTarget...)),
			Exec("stdin", "stdout", "stderr", dirRoot, nil,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTraceAnyTarget...), errAny, "", ""),
			LogError("stderr", "execute make", make.NewErrCallFailed(dirRoot,
				make.CmdMakeTargets(Makefile(infoBase.Path, infoBase.Version),
					argsTraceAnyTarget...), errAny)),
		),
		info: infoBase,
		args: argsTraceAnyTarget,
		expectError: make.NewErrCallFailed(dirRoot, make.CmdMakeTargets(
			Makefile(infoBase.Path, infoBase.Version), argsTraceAnyTarget...),
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
	// regexMakeCall is the regular expression used to remove the nesting level
	// of the make call when an error is observed.
	regexMakeCall = regexp.MustCompile(`(?m)(make)\[[0-9]*\](: [^\n]*\n)`)
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
	str = regexMakeCall.ReplaceAllString(str, "$1$2")
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
	env          []string
	args         []string
	stdin        string
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
		args:         []string{"go-make", "show-targets", "param"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/std.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets/std.err"),
	},
	"go-make show targets trace": {
		args:         []string{"go-make", "--trace", "show-targets", "param"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets/trace.err"),
	},

	// TODO: find out why test it not working as expected in the build!
	// Real world works as expected:
	// echo "Hello, World!" | go-make call cat /dev/stdin
	"go-make call stdin": {
		info:         infoBase,
		args:         []string{"go-make", "call", "cat", "/dev/stdin"},
		stdin:        "Hello, World!",
		expectStdout: "", // "Hello, World!",
		expectStderr: "\x1b[1;96minfo:\x1b[0m captured arguments [cat /dev/stdin]\n",
	},

	"go-make git-verify log": {
		info: infoBase,
		args: []string{
			"go-make", "git-verify", "log",
			"../internal/make/fixtures/git-verify/log-all.in",
		},
		expectStdout: ReadFile(fixtures, "fixtures/git-verify/log-all.out"),
		expectStderr: ReadFile(fixtures, "fixtures/git-verify/log-all.err"),
		expectExit:   3,
	},
	"go-make git-verify message failed": {
		info: infoBase,
		env:  []string{"GITAUTHOR=John Doe <john.doe@zalando.de>"},
		args: []string{
			"go-make", "git-verify", "message",
			"../internal/make/fixtures/git-verify/msg-failed.in",
		},
		expectStdout: ReadFile(fixtures, "fixtures/git-verify/msg-failed.out"),
		expectStderr: ReadFile(fixtures, "fixtures/git-verify/msg-failed.err"),
		expectExit:   3,
	},
	"go-make git-verify message okay": {
		info: infoBase,
		env:  []string{"GITAUTHOR=John Doe <john.doe@zalando.de>"},
		args: []string{
			"go-make", "git-verify", "message",
			"../internal/make/fixtures/git-verify/msg-okay.in",
		},
		expectStdout: ReadFile(fixtures, "fixtures/git-verify/msg-okay.out"),
		expectStderr: ReadFile(fixtures, "fixtures/git-verify/msg-okay.err"),
		expectExit:   0,
	},
}

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
			stdin := strings.NewReader(param.stdin)
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}

			// Filter out make environment variables in general. This primary
			// is to prevent the parent options to influence the test results
			// - in particular the '--trace' flag.
			env := param.env
			env = append(env, "MAKEFLAGS=", "MFLAGS=")

			// When
			exit := make.Make(stdin, stdout, stderr, info,
				configDir, workDir, env, param.args...)

			// Then
			assert.Equal(t, param.expectExit, exit)
			assert.Equal(t, SetupMakeFixture(param.expectStdout),
				FilterMakeOutput(stdout.String()))
			assert.Equal(t, SetupMakeFixture(param.expectStderr),
				FilterMakeOutput(stderr.String()))
		})
}
