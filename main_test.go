package main

import (
	"testing"

	"github.com/tkrop/go-testing/test"

	"github.com/tkrop/go-make/internal/make"
)

var testMainParams = map[string]test.MainParams{
	"config missing": {
		Args:     []string{"go-mock", "show-help"},
		ExitCode: make.ExitConfigFailure,
	},
	"show-help": {
		Args:     []string{"go-mock", "--config=config", "show-help"},
		ExitCode: make.ExitSuccess,
	},
	"show-targets": {
		Args:     []string{"go-mock", "--config=config", "show-targets"},
		ExitCode: make.ExitSuccess,
	},
}

func TestMain(t *testing.T) {
	test.Map(t, testMainParams).Run(test.Main(main))
}
