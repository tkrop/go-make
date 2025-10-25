package cmd_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-testing/test"
)

const (
	DevNullFileOpenFailure = 0x4000
	ProcessReleaseFailure  = 0x8000
)

var ctx = context.Background()

type ExecParams struct {
	cmd          *cmd.Cmd
	stdin        string
	expectStdout string
	expectStderr string
	expectError  error
}

// setup sets up the parameters using given IO parameters.
func (p ExecParams) setup(e *cmd.CmdExecutor) (
	stdout *strings.Builder, stderr *strings.Builder,
	command *cmd.Cmd, executor *cmd.CmdExecutor,
) {
	out := &strings.Builder{}
	err := &strings.Builder{}

	// Configure command input/output.
	c := p.cmd.Copy().WithIO(nil, nil, nil).
		WithStdin(strings.NewReader(p.stdin)).
		WithStdout(out).WithStderr(err)

	// Configure executor failure.
	if p.cmd.IsMode(DevNullFileOpenFailure) {
		e := cmd.NewExecutor()
		test.NewAccessor(e).Set("devnull", "/dev/xxx")
		return out, err, c, e
	} else if p.cmd.IsMode(ProcessReleaseFailure) {
		e := cmd.NewExecutor()
		test.NewAccessor(e).Set("finish",
			func(_ cmd.Mode, _ *exec.Cmd) error {
				return assert.AnError
			})
		return out, err, c, e
	}

	return out, err, c, e
}

var execTestCases = map[string]ExecParams{
	"nil executor": {
		expectError: cmd.NewCmdError("nil executor", nil, nil),
	},
	"nil command": {
		expectError: cmd.NewCmdError("nil command", nil, nil),
	},

	"attached cat": {
		cmd:          cmd.New().WithArgs("cat", "-"),
		stdin:        "Hello, World!",
		expectStdout: "Hello, World!",
	},
	"attached echo stdout": {
		cmd:          cmd.New("echo", "Hello, World!"),
		expectStdout: "Hello, World!\n",
	},
	"attached bash stdout": {
		cmd:          cmd.New().WithArgs("bash"),
		stdin:        "echo Hello, World!\n",
		expectStdout: "Hello, World!\n",
	},
	"attached bash stderr": {
		cmd:          cmd.New().WithArgs("bash"),
		stdin:        "echo Hello, World! >&2\n",
		expectStderr: "Hello, World!\n",
	},
	"attached sleep": {
		cmd: cmd.NewExecutor().New("sleep", "0.01"),
	},
	"attached background sleep": {
		cmd: cmd.New("sleep", "30").WithMode(cmd.Background),
	},
	"attached command error": {
		cmd: cmd.New("_non-existing-command_"),
		expectError: cmd.New("_non-existing-command_").
			Error("starting process", &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},
	"attached background command error": {
		cmd: cmd.New("_non-existing-command_").WithMode(cmd.Background),
		expectError: cmd.New("_non-existing-command_").WithMode(cmd.Background).
			Error("starting process", &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},

	"detached echo stdout": {
		cmd: cmd.New("echo", "Hello, World!").WithMode(cmd.Detached),
	},
	"detached bash stdout": {
		cmd:   cmd.New("bash").WithMode(cmd.Detached),
		stdin: "echo Hello, World!\n",
	},
	"detached bash stderr": {
		cmd:   cmd.New("bash").WithMode(cmd.Detached),
		stdin: "echo Hello, World! >&2\n",
	},
	"detached sleep": {
		cmd: cmd.New("sleep", "0.01").WithMode(cmd.Detached),
	},
	"detached background sleep": {
		cmd: cmd.NewExecutor().New("sleep", "30").
			WithMode(cmd.Detached | cmd.Background),
	},
	"detached command error": {
		cmd: cmd.New("_non-existing-command_").WithMode(cmd.Detached),
		expectError: cmd.New("_non-existing-command_").WithMode(cmd.Detached).
			Error("starting process", &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},
	"detached devnull open failure": {
		cmd: cmd.New("echo", "Hello, World!").
			WithMode(cmd.Detached | DevNullFileOpenFailure),
		expectError: cmd.New("echo", "Hello, World!").
			WithMode(cmd.Detached|DevNullFileOpenFailure).
			Error("opening /dev/null", &os.PathError{
				Op: "open", Path: "/dev/xxx", Err: syscall.Errno(2),
			}),
	},
	"detached background command error": {
		cmd: cmd.New("_non-existing-command_").
			WithMode(cmd.Detached | cmd.Background),
		expectError: cmd.New("_non-existing-command_").
			WithMode(cmd.Detached|cmd.Background).
			Error("starting process", &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},
	"detached background release failure": {
		cmd: cmd.New("echo", "Hello, World!").
			WithMode(cmd.Detached | cmd.Background | ProcessReleaseFailure),
		expectError: cmd.New("echo", "Hello, World!").
			WithMode(cmd.Detached|cmd.Background|ProcessReleaseFailure).
			Error("releasing process", assert.AnError),
	},
}

func TestExec(t *testing.T) {
	test.Map(t, execTestCases).
		Filter("nil-executor", false).
		Run(func(t test.Test, param ExecParams) {
			// Given
			stdout, stderr, cmd, exec := param.setup(cmd.NewExecutor())

			// When
			err := exec.Exec(ctx, cmd)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}

func TestCmdExec(t *testing.T) {
	test.Map(t, execTestCases).
		Filter("nil-executor", false).
		Run(func(t test.Test, param ExecParams) {
			// Given
			stdout, stderr, cmd, exec := param.setup(nil)

			// When
			err := cmd.WithExecutor(exec).Exec(ctx)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}

func TestNilExec(t *testing.T) {
	test.Map(t, execTestCases).
		Filter("nil-executor", true).
		Run(func(t test.Test, param ExecParams) {
			// Given
			stdout, stderr, cmd, exec := param.setup(nil)

			// When
			err := exec.Exec(ctx, cmd)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}

type CmdErrorParams struct {
	message       string
	cmd           *cmd.Cmd
	cause         error
	expectedError string
}

var cmdErrorTestCases = map[string]CmdErrorParams{
	"nil": {
		message:       "nil",
		expectedError: "command - nil: <nil>",
	},
	"empty": {
		message:       "empty",
		cmd:           cmd.New(),
		expectedError: "command - empty [dir=., env=[], call=[]]: <nil>",
	},
	"values": {
		message: "test message",
		cmd: cmd.New("cmd", "arg1", "arg2").
			WithEnv("VAR1=value1", "VAR2=value2").
			WithWorkDir("/test/dir"),
		cause: assert.AnError,
		expectedError: "command - test message [dir=/test/dir, " +
			"env=[VAR1=value1 VAR2=value2], call=[cmd arg1 arg2]]: " +
			assert.AnError.Error(),
	},
}

func TestCmdError(t *testing.T) {
	test.Map(t, cmdErrorTestCases).
		Run(func(t test.Test, param CmdErrorParams) {
			// Given
			cmdErr := param.cmd.Error(param.message, param.cause)

			// When
			result := cmdErr.Error()

			// Then
			assert.Equal(t, param.expectedError, result)
		})
}

func TestCmdError_Unwrap(t *testing.T) {
	// Given
	unit := cmd.CmdError{
		Message: "test", Cause: assert.AnError,
	}

	// When
	err := unit.Unwrap()

	// Then
	assert.Equal(t, assert.AnError, err)
}

func TestCmdError_Is(t *testing.T) {
	// Given
	unit := cmd.CmdError{
		Message: "test", Cause: assert.AnError,
	}

	// When/Then
	assert.True(t, unit.Is(cmd.ErrCmd))
	assert.False(t, unit.Is(assert.AnError))
	assert.False(t, unit.Is(nil))
}
