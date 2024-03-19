package cmd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-testing/test"
)

type ExecParams struct {
	args         []string
	env          []string
	stdin        string
	expectStdout string
	expectStderr string
	expectError  error
}

var testExecParams = map[string]ExecParams{
	"cat": {
		args:         []string{"cat", "-"},
		stdin:        "Hello, World!",
		expectStdout: "Hello, World!",
	},
	"echo": {
		args:         []string{"echo", "Hello, World!"},
		expectStdout: "Hello, World!\n",
	},
	"bash stdout": {
		args:         []string{"bash"},
		stdin:        "echo Hello, World!\n",
		expectStdout: "Hello, World!\n",
	},
	"bash stderr": {
		args:         []string{"bash"},
		stdin:        "echo Hello, World! > /dev/stderr\n",
		expectStderr: "Hello, World!\n",
	},
}

func TestExec(t *testing.T) {
	test.Map(t, testExecParams).
		Run(func(t test.Test, param ExecParams) {
			// Given
			exec := cmd.NewExecutor()
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			stdin := strings.NewReader(param.stdin)

			// When
			err := exec.Exec(stdin, stdout, stderr,
				".", param.env, param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}

// var testExecPtyParams = map[string]ExecParams{
// 	"bash-error": {
// 		args:  []string{"bash"},
// 		stdin: "echo Hello, World!",
// 		expectError: &fs.PathError{
// 			Op: "fork/exec", Path: "/usr/bin/bash",
// 			Err: errors.New("Setctty set but Ctty not valid in child"),
// 		},
// 	},
// 	"bash-interactive": {
// 		args:         []string{"bash"},
// 		stdin:        "echo Hello, World!\n\x03",
// 		expectStdout: "Hello, World!\n",
// 	},
// }

// func TestExecPty(t *testing.T) {
// 	test.Map(t, testExecPtyParams).
// 		Run(func(t test.Test, param ExecParams) {
// 			// Given
// 			exec := cmd.NewExecutor()

// 			dir := t.TempDir()
// 			stdin, err := os.OpenFile(filepath.Join(dir, "stdin"),
// 				os.O_CREATE+os.O_WRONLY, 0o644)
// 			assert.NoError(t, err)
// 			_, err = stdin.Write([]byte(param.stdin))
// 			assert.NoError(t, err)
// 			stdin, err = os.OpenFile(filepath.Join(dir, "stdin"),
// 				os.O_RDONLY, 0o644)
// 			assert.NoError(t, err)

// 			stdout, err := os.OpenFile(filepath.Join(dir, "stdout"),
// 				os.O_CREATE+os.O_WRONLY, 0o644)
// 			assert.NoError(t, err)
// 			stderr, err := os.OpenFile(filepath.Join(dir, "stderr"),
// 				os.O_CREATE+os.O_WRONLY, 0o644)
// 			assert.NoError(t, err)

// 			// When
// 			err = exec.ExecPty(stdin, stdout, stderr, ".", param.env, param.args...)

// 			// Then
// 			assert.Equal(t, param.expectError, err)
// 			// assert.Equal(t, param.expectStdout, stdout.String())
// 			// assert.Equal(t, param.expectStderr, stderr.String())
// 		})
// }
