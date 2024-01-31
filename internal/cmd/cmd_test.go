package cmd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tkrop/go-make/internal/cmd"
	"github.com/tkrop/go-testing/test"
)

type ExecutorParams struct {
	env          []string
	args         []string
	expectStdout string
	expectStderr string
	expectError  error
}

var testExecutorParams = map[string]ExecutorParams{
	"ls": {
		args:         []string{"ls"},
		expectStdout: "cmd.go\ncmd_test.go\n",
	},
}

func TestExecutor(t *testing.T) {
	test.Map(t, testExecutorParams).
		Run(func(t test.Test, param ExecutorParams) {
			// Given
			exec := cmd.NewExecutor()
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}

			// When
			err := exec.Exec(stdout, stderr, ".", param.env, param.args...)

			// Then
			assert.Equal(t, param.expectError, err)
			assert.Equal(t, param.expectStdout, stdout.String())
			assert.Equal(t, param.expectStderr, stderr.String())
		})
}
