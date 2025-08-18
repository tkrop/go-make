package cmd_test

import (
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

type ExecParams struct {
	mode         cmd.Mode
	args         []string
	env          []string
	stdin        string
	expectStdout string
	expectStderr string
	expectError  error
}

var testExecParams = map[string]ExecParams{
	"attached cat": {
		mode:         cmd.Attached,
		args:         []string{"cat", "-"},
		stdin:        "Hello, World!",
		expectStdout: "Hello, World!",
	},
	"attached echo stdout": {
		mode:         cmd.Attached,
		args:         []string{"echo", "Hello, World!"},
		expectStdout: "Hello, World!\n",
	},
	"attached bash stdout": {
		mode:         cmd.Attached,
		args:         []string{"bash"},
		stdin:        "echo Hello, World!\n",
		expectStdout: "Hello, World!\n",
	},
	"attached bash stderr": {
		mode:         cmd.Attached,
		args:         []string{"bash"},
		stdin:        "echo Hello, World! >&2\n",
		expectStderr: "Hello, World!\n",
	},
	"attached sleep": {
		mode: cmd.Attached,
		args: []string{"sleep", "0.01"},
	},
	"attached background sleep": {
		mode: cmd.Attached | cmd.Background,
		args: []string{"sleep", "30"},
	},
	"attached command error": {
		mode: cmd.Attached,
		args: []string{"_non-existing-command_"},
		expectError: cmd.NewCmdError("starting process",
			".", nil, []string{"_non-existing-command_"}, &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},
	"attached background command error": {
		mode: cmd.Attached | cmd.Background,
		args: []string{"_non-existing-command_"},
		expectError: cmd.NewCmdError("starting process",
			".", nil, []string{"_non-existing-command_"}, &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},

	"detached echo stdout": {
		mode: cmd.Detached,
		args: []string{"echo", "Hello, World!"},
	},
	"detached bash stdout": {
		mode:  cmd.Detached,
		args:  []string{"bash"},
		stdin: "echo Hello, World!\n",
	},
	"detached bash stderr": {
		mode:  cmd.Detached,
		args:  []string{"bash"},
		stdin: "echo Hello, World! >&2\n",
	},
	"detached sleep": {
		mode: cmd.Detached,
		args: []string{"sleep", "0.01"},
	},
	"detached background sleep": {
		mode: cmd.Detached | cmd.Background,
		args: []string{"sleep", "30"},
	},
	"detached command error": {
		mode: cmd.Detached,
		args: []string{"_non-existing-command_"},
		expectError: cmd.NewCmdError("starting process",
			".", nil, []string{"_non-existing-command_"}, &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},
	"detached devnull open failure": {
		mode: cmd.Detached | DevNullFileOpenFailure,
		args: []string{"echo", "Hello, World!"},
		expectError: cmd.NewCmdError("opening /dev/null",
			".", nil, []string{"echo", "Hello, World!"}, &os.PathError{
				Op: "open", Path: "/dev/xxx", Err: syscall.Errno(2),
			}),
	},
	"detached background command error": {
		mode: cmd.Detached | cmd.Background,
		args: []string{"_non-existing-command_"},
		expectError: cmd.NewCmdError("starting process",
			".", nil, []string{"_non-existing-command_"}, &exec.Error{
				Name: "_non-existing-command_", Err: exec.ErrNotFound,
			}),
	},
	"detached background release failure": {
		mode: cmd.Detached | cmd.Background | ProcessReleaseFailure,
		args: []string{"echo", "Hello, World!"},
		expectError: cmd.NewCmdError("releasing process",
			".", nil, []string{"echo", "Hello, World!"}, assert.AnError),
	},
}

func TestExec(t *testing.T) {
	test.Map(t, testExecParams).
		Run(func(t test.Test, param ExecParams) {
			// Given
			unit := cmd.NewExecutor()
			if param.mode&DevNullFileOpenFailure == DevNullFileOpenFailure {
				test.NewAccessor(unit).Set("devnull", "/dev/xxx")
			} else if param.mode&ProcessReleaseFailure == ProcessReleaseFailure {
				test.NewAccessor(unit).Set("finish",
					func(_ cmd.Mode, _ *exec.Cmd) error { return assert.AnError })
			}
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			stdin := strings.NewReader(param.stdin)

			// When
			err := unit.Exec(param.mode, stdin, stdout, stderr,
				".", param.env, param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}

type CmdErrorParams struct {
	message       string
	dir           string
	env           []string
	args          []string
	cause         error
	expectedError string
}

var testCmdErrorParams = map[string]CmdErrorParams{
	"nil": {
		message:       "nil",
		expectedError: "command - nil [dir=, env=[], call=[]]: <nil>",
	},
	"empty": {
		message:       "empty",
		env:           []string{},
		args:          []string{},
		expectedError: "command - empty [dir=, env=[], call=[]]: <nil>",
	},
	"values": {
		message: "test message",
		dir:     "/test/dir",
		env:     []string{"VAR1=value1", "VAR2=value2"},
		args:    []string{"cmd", "arg1", "arg2"},
		cause:   assert.AnError,
		expectedError: "command - test message [dir=/test/dir, " +
			"env=[VAR1=value1 VAR2=value2], call=[cmd arg1 arg2]]: " +
			assert.AnError.Error(),
	},
}

func TestCmdError_Error(t *testing.T) {
	test.Map(t, testCmdErrorParams).
		Run(func(t test.Test, param CmdErrorParams) {
			// Given
			cmdErr := cmd.NewCmdError(param.message, param.dir, param.env, param.args, param.cause)

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
