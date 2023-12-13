package log_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tkrop/go-make/internal/info"
	"github.com/tkrop/go-make/internal/log"
	"github.com/tkrop/go-testing/test"
)

var (
	// log is the singleton logger for testing.
	logger = log.NewLogger()
	// infoDirty is an arbitrary dirty info for testing.
	infoDirty = info.NewInfo("", "", "", "", "", true)
	// Any error that can happen.
	errAny = errors.New("any error")
)

type InfoParams struct {
	info         *info.Info
	raw          bool
	expectString string
}

var testInfoParams = map[string]InfoParams{
	"dirty info": {
		info:         infoDirty,
		expectString: "info: " + infoDirty.String() + "\n",
	},
	"dirty info raw": {
		info:         infoDirty,
		raw:          true,
		expectString: infoDirty.String() + "\n",
	},
}

func TestInfo(t *testing.T) {
	test.Map(t, testInfoParams).
		Run(func(t test.Test, param InfoParams) {
			// Given
			writer := &strings.Builder{}

			// When
			logger.Info(writer, param.info, param.raw)

			// Then
			assert.Equal(t, param.expectString, writer.String())
		})
}

type ExecParams struct {
	dir          string
	args         []string
	expectString string
}

var testExecParams = map[string]ExecParams{
	"empty args": {
		expectString: "exec: []\n",
	},
	"single arg": {
		args:         []string{"arg"},
		expectString: "exec: arg []\n",
	},
	"multiple args": {
		args:         []string{"arg1", "arg2"},
		expectString: "exec: arg1 arg2 []\n",
	},

	"empty args with dir": {
		dir:          "dir",
		expectString: "exec: [dir]\n",
	},
	"single arg with dir": {
		dir:          "dir",
		args:         []string{"arg"},
		expectString: "exec: arg [dir]\n",
	},
	"multiple args with dir": {
		dir:          "dir",
		args:         []string{"arg1", "arg2"},
		expectString: "exec: arg1 arg2 [dir]\n",
	},
}

func TestExec(t *testing.T) {
	test.Map(t, testExecParams).
		Run(func(t test.Test, param ExecParams) {
			// Given
			writer := &strings.Builder{}

			// When
			logger.Exec(writer, param.dir, param.args...)

			// Then
			assert.Equal(t, param.expectString, writer.String())
		})
}

type CallParams struct {
	args         []string
	expectString string
}

var testCallParams = map[string]CallParams{
	"empty args": {
		expectString: "call:\n",
	},
	"single arg": {
		args:         []string{"arg"},
		expectString: "call: arg\n",
	},
	"multiple args": {
		args:         []string{"arg1", "arg2"},
		expectString: "call: arg1 arg2\n",
	},
}

func TestCall(t *testing.T) {
	test.Map(t, testCallParams).
		Run(func(t test.Test, param CallParams) {
			// Given
			writer := &strings.Builder{}

			// When
			logger.Call(writer, param.args...)

			// Then
			assert.Equal(t, param.expectString, writer.String())
		})
}

type ErrorParams struct {
	message      string
	error        error
	expectString string
}

var testErrorParams = map[string]ErrorParams{
	"empty message": {
		expectString: "error: <unknown>\n",
	},
	"non-empty message": {
		message:      "message",
		expectString: "error: message\n",
	},
	"empty message with error": {
		error:        errAny,
		expectString: fmt.Sprintf("error: %v\n", errAny),
	},
	"non-empty message with error": {
		message:      "message",
		error:        errAny,
		expectString: fmt.Sprintf("error: message: %v\n", errAny),
	},
}

func TestError(t *testing.T) {
	test.Map(t, testErrorParams).
		Run(func(t test.Test, param ErrorParams) {
			// Given
			writer := &strings.Builder{}

			// When
			logger.Error(writer, param.message, param.error)

			// Then
			assert.Equal(t, param.expectString, writer.String())
		})
}

type MessageParams struct {
	message      string
	expectString string
}

var testMessageParams = map[string]MessageParams{
	"empty message": {
		expectString: "\n",
	},
	"non-empty message": {
		message:      "message",
		expectString: "message\n",
	},
}

func TestMessage(t *testing.T) {
	test.Map(t, testMessageParams).
		Run(func(t test.Test, param MessageParams) {
			// Given
			writer := &strings.Builder{}

			// When
			logger.Message(writer, param.message)

			// Then
			assert.Equal(t, param.expectString, writer.String())
		})
}
