package main

import (
	"testing"

	"github.com/tkrop/go-testing/test"

	"github.com/tkrop/go-make/internal/make"
)

var testMainParams = map[string]test.MainParams{
	"config missing": {
		Args:     []string{"go-mock", "show-targets"},
		ExitCode: make.ExitConfigFailure,
	},
	"show-targets": {
		Args:     []string{"go-mock", "--config=config", "show-targets"},
		ExitCode: make.ExitSuccess,
	},
}

func TestMain(t *testing.T) {
	test.Map(t, testMainParams).Run(test.TestMain(main))
}
