package make_test

import (
	"context"
	"embed"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/tkrop/go-config/info"
	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-make/internal/log"
	. "github.com/tkrop/go-make/internal/make"
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

var ctx, _ = context.WithCancel(context.Background())

//go:embed fixtures/*
var fixtures embed.FS

// ReadFile reads a file from the embedded filesystem and returns its content
// as a string.
func ReadFile(fs embed.FS, name string) string {
	out, err := fs.ReadFile(name)
	if err == nil && out != nil {
		return string(out)
	} else if err != nil {
		panic(err)
	}
	panic("no output")
}

// WriteFile writes a string value to a file with the given name and
// permissions.
func WriteFile(name string, perm os.FileMode, value string) {
	err := os.WriteFile(name, []byte(value), perm)
	if err != nil {
		panic(err)
	}
}

var (
	// dirWork contains an arbitrary working directory (uses current).
	dirWork = "."
	// dirRoot contains an arbitrary absolute test directory (use current).
	dirRoot = filepath.Dir(filepath.Dir(AbsPath(dirWork)))
	// envMakeMock contains the environment variables for the targets files.
	envMakeMock = []string{
		"FILE_TARGETS=" + filepath.Join(dirRoot,
			"internal", "make", "fixtures", "targets", "std.out"),
		"FILE_TARGETS_MAKE=" + filepath.Join(dirRoot,
			"internal", "make", "fixtures", "targets", "make-std.out"),
		"FILE_TARGETS_GOMAKE=" + filepath.Join(dirRoot,
			"internal", "make", "fixtures", "targets", "go-make-std.out"),
	}

	// infoBase with version and revision.
	infoBase = info.New(goMakePath, "v0.0.25",
		"ba4ff068e795443f256caa06180d976a0fb244e9",
		"2024-01-09T13:02:46+01:00",
		"2024-01-10T16:22:54+01:00", "true")

	infoNew = info.New(goMakePath, "latest",
		"ba4ff068e795443f256caa06180d976a0fb244e9",
		"2023-11-14T13:02:46+01:00",
		"2023-11-10T16:22:54+01:00", "false")

	goMakeInfoBase = GoMakePath(infoBase.Path, infoBase.Version)
	goMakeInfoNew  = GoMakePath(infoNew.Path, infoNew.Version)

	makeInfoBase = MakefilePath(infoBase.Path, infoBase.Version)
	makeInfoNew  = MakefilePath(infoNew.Path, infoNew.Version)

	argsVersion           = []string{"go-make", "--version"}
	argsTraceVersion      = []string{"go-make", "--trace", "--version"}
	argsBash              = []string{"go-make", "--completion=bash"}
	argsBashTrace         = []string{"go-make", "--trace", "--completion=bash"}
	argsZsh               = []string{"go-make", "--completion=zsh"}
	argsZshTrace          = []string{"go-make", "--trace", "--completion=zsh"}
	argsShowTargets       = []string{"go-make", "show-targets"}
	argsShowTargetsMake   = []string{"go-make", "show-targets-make"}
	argsShowTargetsGoMake = []string{"go-make", "show-targets-go-make"}
	argsShowTargetsParam  = []string{"go-make", "show-targets", "param"}
	argsShowTargetsCustom = []string{"go-make", "--config=custom", "show-targets"}
	argsShowTargetsLatest = []string{"go-make", "--config=latest", "show-targets"}
	argsTraceAnyTarget    = []string{"go-make", "--trace", "target"}
)

// MakefilePath returns the path to the Makefile for the given path and version.
func MakefilePath(path string, version string) string {
	return filepath.Join(GoMakePath(path, version), Makefile)
}

// NewReader creates a new reader with the given id.
func NewReader(id string) io.Reader {
	return strings.NewReader(id)
}

// NewWriter creates a new writer with the given id.
func NewWriter(id string) io.Writer {
	builder := &strings.Builder{}
	builder.WriteString(id)
	return builder
}

// GoMakeSetup sets up a new go-make test with mocks.
func GoMakeSetup(
	t test.Test, param MakeParams,
) (*GoMake, *mock.Mocks) {
	mocks := mock.NewMocks(t).
		SetArg("nil", nil).
		SetArg("stdin", NewReader("stdin")).
		SetArg("stdout", NewWriter("stdout")).
		SetArg("stderr", NewWriter("stderr")).
		SetArg("builder", &strings.Builder{}).
		Expect(param.mockSetup)

	gm := NewGoMake(
		mocks.GetArg("stdin").(io.Reader),
		mocks.GetArg("stdout").(io.Writer),
		mocks.GetArg("stderr").(io.Writer),
		// Filling the test coverage gap of returning the default.
		param.info, GetEnvDefault(EnvGoMakeConfig, ""),
		".", param.env...,
	)

	gm.Executor = mock.Get(mocks, NewMockExecutor)
	gm.Logger = mock.Get(mocks, NewMockLogger)

	return gm, mocks
}

func ToAny(args ...any) []any {
	return args
}

func Cast[T any](value any) T {
	if value == nil {
		return *new(T)
	}
	return value.(T)
}

func Exec( //revive:disable-line:argument-limit
	c *cmd.Cmd, stdin, stdout, stderr string,
	sout, serr string, err error,
) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		if err != nil {
			err = c.Error("any", err)
		}
		c.WithStdin(Cast[io.Reader](mocks.GetArg(stdin))).
			WithStdout(Cast[io.Writer](mocks.GetArg(stdout))).
			WithStderr(Cast[io.Writer](mocks.GetArg(stderr)))
		return mock.Get(mocks, NewMockExecutor).EXPECT().
			Exec(gomock.AssignableToTypeOf(ctx), c).
			DoAndReturn(mocks.Call(cmd.Executor.Exec,
				func(args ...any) []any {
					cmd := args[1].(*cmd.Cmd)
					if _, err := cmd.Stdout.Write([]byte(sout)); err != nil {
						assert.Fail(mocks.Ctrl.T, "failed to write to stdout", err)
					}
					if _, err := cmd.Stderr.Write([]byte(serr)); err != nil {
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

func LogExec(writer string, cmd *cmd.Cmd) mock.SetupFunc {
	return func(mocks *mock.Mocks) any {
		return mock.Get(mocks, NewMockLogger).EXPECT().
			Exec(mocks.GetArg(writer), cmd.Dir, ToAny(cmd.Args)...).
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

var makeMockTestCases = map[string]MakeParams{
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
			LogMessage("stdout", CompleteBash),
		),
		info: infoBase,
		args: argsBash,
	},
	"go-make completion zsh": {
		mockSetup: mock.Chain(
			LogMessage("stdout", CompleteZsh),
		),
		info: infoBase,
		args: argsZsh,
	},

	"go-make show targets": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoBase, argsShowTargets[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		args: argsShowTargets,
	},
	"go-make show targets with file": {
		mockSetup: mock.Chain(
			LogMessage("stdout", ReadFile(fixtures, "fixtures/targets/std.out")),
			Exec(CmdGitTop(dirWork, envMakeMock...),
				"nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot, envMakeMock...),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoBase, argsShowTargets[1:], dirRoot,
				envMakeMock...).WithMode(cmd.Detached|cmd.Background),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		env:  envMakeMock,
		args: argsShowTargets,
	},
	"go-make show targets make with file": {
		mockSetup: mock.Chain(
			LogMessage("stdout",
				ReadFile(fixtures, "fixtures/targets/make-std.out")),
			Exec(CmdGitTop(dirWork, envMakeMock...),
				"nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot, envMakeMock...),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoBase, argsShowTargetsMake[1:], dirRoot,
				envMakeMock...).WithMode(cmd.Detached|cmd.Background),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		env:  envMakeMock,
		args: argsShowTargetsMake,
	},
	"go-make show targets go-make with file": {
		mockSetup: mock.Chain(
			LogMessage("stdout",
				ReadFile(fixtures, "fixtures/targets/go-make-std.out")),
			Exec(CmdGitTop(dirWork, envMakeMock...),
				"nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot, envMakeMock...),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoBase, argsShowTargetsGoMake[1:], dirRoot,
				envMakeMock...).WithMode(cmd.Detached|cmd.Background),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		env:  envMakeMock,
		args: argsShowTargetsGoMake,
	},
	"go-make show targets with param": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoBase, argsShowTargetsParam[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		args: argsShowTargetsParam,
	},
	"go-make show targets install": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoNew, dirRoot),
				"nil", "stderr", "stderr", "", "", assert.AnError),
			Exec(CmdGoInstall(infoNew.Path, infoNew.Version, dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoNew, argsShowTargets[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoNew,
		args: argsShowTargets,
	},
	"go-make show targets config custom": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(AbsPath("custom"), dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(filepath.Join(AbsPath("custom"), Makefile),
				argsShowTargets[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		args: argsShowTargetsCustom,
	},
	"go-make show targets config version latest": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(AbsPath("latest"), dirRoot),
				"nil", "stderr", "stderr", "", "", assert.AnError),
			Exec(CmdTestDir(GoMakePath(infoBase.Path, "latest"), dirRoot),
				"nil", "stderr", "stderr", "", "", assert.AnError),
			Exec(CmdGoInstall(infoBase.Path, "latest", dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(MakefilePath(infoBase.Path, "latest"),
				argsShowTargets[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		args: argsShowTargetsLatest,
	},

	"go-make show targets install failed": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoNew, dirRoot),
				"nil", "stderr", "stderr", "", "", assert.AnError),
			Exec(CmdGoInstall(infoNew.Path, infoNew.Version, dirRoot),
				"nil", "stderr", "stderr", "", "", assert.AnError),
			LogError("stderr", "ensure config", NewErrNotFound(
				infoNew.Path, infoNew.Version, NewErrCallFailed(
					CmdGoInstall(infoNew.Path, infoNew.Version, dirRoot),
					assert.AnError))),
		),
		info: infoNew,
		args: argsShowTargets,
		expectError: NewErrNotFound(
			infoNew.Path, infoNew.Version, NewErrCallFailed(
				CmdGoInstall(infoNew.Path, infoNew.Version, dirRoot),
				assert.AnError)),
		expectExit: ExitConfigFailure,
	},
	"go-make show targets failed": {
		mockSetup: mock.Chain(
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			Exec(CmdMakeTargets(makeInfoBase, argsShowTargets[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", assert.AnError),
			LogError("stderr", "execute make", NewErrCallFailed(CmdMakeTargets(
				makeInfoBase, argsShowTargets[1:], dirRoot), assert.AnError)),
		),
		info: infoBase,
		args: argsShowTargets,
		expectError: NewErrCallFailed(CmdMakeTargets(makeInfoBase,
			argsShowTargets[1:], dirRoot), assert.AnError),
		expectExit: ExitTargetFailure,
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
			LogCall("stderr", argsBashTrace),
			LogInfo("stderr", infoBase, false),
			LogMessage("stdout", CompleteBash),
		),
		info: infoBase,
		args: argsBashTrace,
	},
	"go-make completion zsh traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsZshTrace),
			LogInfo("stderr", infoBase, false),
			LogMessage("stdout", CompleteZsh),
		),
		info: infoBase,
		args: argsZshTrace,
	},
	"go-make any target traced": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceAnyTarget),
			LogInfo("stderr", infoBase, false),
			LogExec("stderr", CmdGitTop(dirWork)),
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			LogExec("stderr", CmdTestDir(goMakeInfoBase, dirRoot)),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			LogExec("stderr", CmdMakeTargets(makeInfoBase,
				argsTraceAnyTarget[1:], dirRoot)),
			Exec(CmdMakeTargets(makeInfoBase, argsTraceAnyTarget[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", nil),
		),
		info: infoBase,
		args: argsTraceAnyTarget,
	},
	"go-make any target traced failed": {
		mockSetup: mock.Chain(
			LogCall("stderr", argsTraceAnyTarget),
			LogInfo("stderr", infoBase, false),
			LogExec("stderr", CmdGitTop(dirWork)),
			Exec(CmdGitTop(dirWork), "nil", "builder", "stderr", dirRoot, "", nil),
			LogExec("stderr", CmdTestDir(goMakeInfoBase, dirRoot)),
			Exec(CmdTestDir(goMakeInfoBase, dirRoot),
				"nil", "stderr", "stderr", "", "", nil),
			LogExec("stderr", CmdMakeTargets(makeInfoBase,
				argsTraceAnyTarget[1:], dirRoot)),
			Exec(CmdMakeTargets(makeInfoBase, argsTraceAnyTarget[1:], dirRoot),
				"stdin", "stdout", "stderr", "", "", assert.AnError),
			LogError("stderr", "execute make", NewErrCallFailed(CmdMakeTargets(
				makeInfoBase, argsTraceAnyTarget[1:], dirRoot), assert.AnError)),
		),
		info: infoBase,
		args: argsTraceAnyTarget,
		expectError: NewErrCallFailed(CmdMakeTargets(makeInfoBase,
			argsTraceAnyTarget[1:], dirRoot), assert.AnError),
		expectExit: ExitTargetFailure,
	},
}

func TestMakeMock(t *testing.T) {
	test.Map(t, makeMockTestCases).
		Run(func(t test.Test, param MakeParams) {
			// Given
			gm, _ := GoMakeSetup(t, param)

			// When
			exit, err := gm.Make(param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectExit, exit)
		})
}

var (
	// dirConfig contains the config directory of go-make.
	dirConfig = AbsPath(filepath.Join("..", "..", "config"))
	// dirFixtures contains the fixtures directory for go-make tests.
	dirFixtures = AbsPath(filepath.Join(dirRoot, "internal", "make", "fixtures"))
	// dirCache contains the temporary cache for targets working directory.
	dirCache = filepath.Join(AbsPath(GetEnvDefault("TMPDIR", "/tmp")),
		"go-make-"+os.Getenv("USER"))

	// regexTargets is used to match the ${dir} variable in the environment
	// values to replace it with the actual test directory.
	regexTargets = regexp.MustCompile(`(?m)\${dir}`)
	// regexMakeCall is used to remove the nesting level of the make call when
	// an error is observed (obsoleted by `--no-print-directory` flag).
	regexMakeCall = regexp.MustCompile(`(?m)(make)\[[0-9]*\](: [^\n]*\n)`)
	// regexMakeTimestamp is used to remove the constantly changing timestamp
	// from the make output.
	regexMakeTimestamp = regexp.MustCompile(
		`([0-9]{4})(-[0-9]{2}){2} ([0-9]{2}:){2}[0-9]{2}(.[0-9]{3})? `)
	// regexGoMakeWarning is used to remove the `go-make` version mismatch
	// warning that happens after bumping the version.
	regexGoMakeWarning = regexp.MustCompile(`(?m).*warning:.*go-make version.*\n`)
	// regexGoMakeDebug is used to remove all `go-make` platform dependent
	// debug information.
	regexGoMakeDebug = regexp.MustCompile(`(?m).*debug:.*\n`)
	// regexGoBinPath is used to match the `GOBIN`-path in the output and to
	// replace it against a common prefix.
	regexGoBinPath = regexp.MustCompile(`(?m)(^` + AbsPath(os.Getenv("GOBIN")) + `)`)
	// regexGoMakeTemp is used to remove the `go-make` root specific path
	// information.
	regexGoMakeRoot = regexp.MustCompile(`(?m)` + dirRoot)
	// regexGoMakeCache is used to remove the `go-make` cache specific path
	// information.
	regexGoMakeCache = regexp.MustCompile(`(?m)` + dirCache)
	// regexMakeTrace is used to match the make trace output and to remove the
	// changing line numbers to resiliently match output when `go-make` targets
	// are moved around.
	regexMakeTrace = regexp.MustCompile(
		`(?m)(/root/go-make/config/Makefile.base:)[0-9]+:`)
	// regexMakeTarget is used to match the make trace output to remove the
	// target update message that changes between make 4.3 and make 4.4.
	regexMakeTarget = regexp.MustCompile(
		`(?m)(/root/go-make/config/Makefile.base:).*('.*').*`)
	// regexMakeUpdate is used to match the make trace output that spuriously
	// appears in the output when running `test-self`. In reality this output
	// is: `config/Makefile.base:771: warning: undefined variable 'dir'`, but
	// it is unclear why it appears in the output.
	regexMakeUpdate = regexp.MustCompile(
		`(?m)/root/go-make/config/Makefile.base: update target 'dir'\n`)
	// regexMakeOptions is used to match the `go-make` options output to remove
	// the options that have added between make 4.3 and make 4.4.
	regexMakeOptions = regexp.MustCompile(`(?m)(^--(shuffle|jobserver-style)=?)\n`)

	// phFixtures replaces the placeholders in the test fixture with the
	// values provided to the replacer.
	phFixtures = strings.NewReplacer(
		"{{GOVERSION}}", runtime.Version()[2:],
		"{{PLATFORM}}", runtime.GOOS+"/"+runtime.GOARCH,
		"{{COMPILER}}", runtime.Compiler)
)

// EnvPrepare copies the environment variables and replaces the variable
// `${dir}` with the given directory. It also appends empty make flags to the
// end of the slice to ensure that parent options influence the test results
// - in particular the '--trace' flag.
func EnvPrepare(env []string, dir string) []string {
	result := make([]string, 0, len(env)+4)
	result = append(result, "FILE_TARGETS=${dir}/targets")

	for _, value := range env {
		result = append(result, regexTargets.ReplaceAllString(value, dir))
	}

	return append(result, "MAKEFLAGS=", "MFLAGS=", "GOMAKE_MODE=no-config")
}

// CreateFilter returns a function that filters the output of `go-make`
// commands to create an environment agnostic output. It removes the
// timestamps, the `go-make` version warnings, the debug information, the
// cache and root paths, and replaces the `GOBIN` path with a common prefix.
// It also removes the make trace output and the target update messages.
// The function takes a regular expression to match the `go-make` test path
// and replaces it with a common test path.
func CreateFilter(regexGoMakeTest *regexp.Regexp) func(string) string {
	return func(str string) string {
		str = regexMakeCall.ReplaceAllString(str, "$1$2")
		str = regexMakeTimestamp.ReplaceAllString(str, "")
		str = regexGoMakeWarning.ReplaceAllString(str, "")
		str = regexGoMakeDebug.ReplaceAllString(str, "")
		str = regexGoMakeCache.ReplaceAllString(str, "/tmp/go-make")
		str = regexGoMakeRoot.ReplaceAllString(str, "/root/go-make")
		str = regexGoMakeTest.ReplaceAllString(str, "/test/go-make")
		str = regexGoBinPath.ReplaceAllString(str, "go/bin")
		str = regexMakeTrace.ReplaceAllString(str, "$1")
		str = regexMakeTarget.ReplaceAllString(str, "$1 update target $2")
		str = regexMakeUpdate.ReplaceAllString(str, "")
		str = regexMakeOptions.ReplaceAllString(str, "")
		return str
	}
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

var makeExecTestCases = map[string]MakeExecParams{
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
		expectStdout: ReadFile(fixtures, "fixtures/completion/bash.out"),
	},
	"go-make bash trace": {
		info:         infoBase,
		args:         []string{"go-make", "--trace", "--completion=bash"},
		expectStdout: ReadFile(fixtures, "fixtures/completion/bash.out"),
		expectStderr: ReadFile(fixtures, "fixtures/completion/bash.err"),
	},

	"go-make zsh": {
		info:         infoBase,
		args:         []string{"go-make", "--completion=zsh"},
		expectStdout: ReadFile(fixtures, "fixtures/completion/zsh.out"),
	},
	"go-make zsh trace": {
		info:         infoBase,
		args:         []string{"go-make", "--trace", "--completion=zsh"},
		expectStdout: ReadFile(fixtures, "fixtures/completion/zsh.out"),
		expectStderr: ReadFile(fixtures, "fixtures/completion/zsh.err"),
	},

	"go-make show targets": {
		info:         infoBase,
		env:          []string{"FILE_TARGETS=${dir}/targets~"},
		args:         []string{"go-make", "show-targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/std.out"),
	},
	"go-make show targets trace": {
		env:          []string{"FILE_TARGETS=${dir}/targets"},
		args:         []string{"go-make", "--trace", "show-targets"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets/trace.err"),
	},
	"go-make show targets make": {
		info:         infoBase,
		env:          []string{"FILE_TARGETS_MAKE=${dir}/targets.make~"},
		args:         []string{"go-make", "show-targets-make"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/make-std.out"),
	},
	"go-make show targets make trace": {
		env:          []string{"FILE_TARGETS_MAKE=${dir}/targets.make"},
		args:         []string{"go-make", "--trace", "show-targets-make"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/make-trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets/make-trace.err"),
	},
	"go-make show targets go-make": {
		info:         infoBase,
		env:          []string{"FILE_TARGETS_GOMAKE=${dir}/targets.go-make~"},
		args:         []string{"go-make", "show-targets-go-make"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/go-make-std.out"),
	},
	"go-make show targets go-make trace": {
		env:          []string{"FILE_TARGETS_GOMAKE=${dir}/targets.go-make"},
		args:         []string{"go-make", "--trace", "show-targets-go-make"},
		expectStdout: ReadFile(fixtures, "fixtures/targets/go-make-trace.out"),
		expectStderr: ReadFile(fixtures, "fixtures/targets/go-make-trace.err"),
	},

	"go-make call stdin": {
		info:         infoBase,
		args:         []string{"go-make", "call", "cat"},
		stdin:        "Hello, World!",
		expectStdout: "Hello, World!",
		expectStderr: ReadFile(fixtures, "fixtures/cat.err"),
	},

	"go-make git-verify log": {
		info: infoBase,
		args: []string{
			"go-make", "git-verify", "log",
			filepath.Join(dirFixtures, "git-verify", "log-all.in"),
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
			filepath.Join(dirFixtures, "git-verify", "msg-failed.in"),
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
			filepath.Join(dirFixtures, "git-verify", "msg-okay.in"),
		},
		expectStdout: ReadFile(fixtures, "fixtures/git-verify/msg-okay.out"),
		expectStderr: ReadFile(fixtures, "fixtures/git-verify/msg-okay.err"),
		expectExit:   0,
	},
}

func TestMakeExec(t *testing.T) {
	// Ensure test environment is setup.
	dirTest := AbsPath(t.TempDir())

	// Filter that also match the temporary test directory path (`/private` is
	// prefix visible in MacOS builds).
	filter := CreateFilter(regexp.MustCompile(`(?m)(/private)?` + dirTest))
	replace := phFixtures.Replace

	// Cleanup cache directory entries.
	dirCache := AbsPath(filepath.Join(dirCache, dirTest, ".."))

	cmd := exec.Command("git", "init", dirTest)
	assert.NoError(t, cmd.Run())
	cmd = exec.Command("mkdir", "--parents", dirCache)
	assert.NoError(t, cmd.Run())

	WriteFile(filepath.Join(dirTest, "targets~"), os.FileMode(0o644),
		ReadFile(fixtures, "fixtures/targets/std.out"))
	WriteFile(filepath.Join(dirTest, "targets.make~"), os.FileMode(0o644),
		ReadFile(fixtures, "fixtures/targets/make-std.out"))
	WriteFile(filepath.Join(dirTest, "targets.go-make~"), os.FileMode(0o644),
		ReadFile(fixtures, "fixtures/targets/go-make-std.out"))

	test.Map(t, makeExecTestCases).
		Run(func(t test.Test, param MakeExecParams) {
			// Given
			info := infoBase
			stdin := strings.NewReader(param.stdin)
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			env := EnvPrepare(param.env, dirTest)

			// When
			exit := Make(stdin, stdout, stderr, info,
				dirConfig, dirTest, env, param.args...)

			// Then
			assert.Equal(t, param.expectExit, exit)
			assert.Equal(t, replace(param.expectStdout), filter(stdout.String()))
			assert.Equal(t, replace(param.expectStderr), filter(stderr.String()))
		}).
		Cleanup(func() {
			assert.NoError(t, os.RemoveAll(dirCache))
		})
}

type handleSignalParams struct {
	signal        syscall.Signal
	expectAborted bool
}

var handleSignalTestCases = map[string]handleSignalParams{
	"signal abrt": {
		signal:        syscall.SIGABRT,
		expectAborted: true,
	},
	"signal term": {
		signal:        syscall.SIGTERM,
		expectAborted: false,
	},
}

func TestHandleSignal(t *testing.T) {
	test.Map(t, handleSignalTestCases).
		Run(func(t test.Test, param handleSignalParams) {
			// Given
			gm := &GoMake{}
			var cancelled atomic.Bool
			cancel := func() { cancelled.Store(true) }

			// When
			gm.HandleSignal(cancel, param.signal)

			// Then
			assert.Equal(t, param.expectAborted, gm.Aborted.Load())
			assert.True(t, cancelled.Load())
		})
}
