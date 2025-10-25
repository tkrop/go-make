package main

import (
	"testing"

	"github.com/tkrop/go-testing/test"

	"github.com/tkrop/go-make/internal/make"
)

var mainTestCases = map[string]test.MainParam{
	"config missing": {
		Args:     []string{"go-make", "show-help"},
		ExitCode: make.ExitConfigFailure,
	},
	"show-help": {
		Args:     []string{"go-make", "--config=config", "show-help"},
		ExitCode: make.ExitSuccess,
	},
	"show-targets": {
		Args:     []string{"go-make", "--config=config", "show-targets"},
		ExitCode: make.ExitSuccess,
	},
}

func TestMain(t *testing.T) {
	test.Map(t, mainTestCases).Run(test.Main(main))
}
