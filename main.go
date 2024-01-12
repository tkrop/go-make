//go:build !test

// Main entry point of the go-make command.
package main

import (
	"os"
	"strconv"

	"github.com/tkrop/go-make/internal/info"
	"github.com/tkrop/go-make/internal/make"
)

var (
	// Path contains the package path.
	Path string
	// Version contains the custom version.
	Version string
	// Build contains the custom build time.
	Build string
	// Revision contains the custom revision.
	Revision string
	// Commit contains the custom commit time.
	Commit string
	// Dirty contains the custom dirty flag.
	Dirty string
	// Config contains the custom go-make config.
	Config string
)

// NewInfo returns the build information of a command or module with
// default values.
func NewInfo() *info.Info {
	dirty, _ := strconv.ParseBool(Dirty)
	return info.NewInfo(Path, Version, Revision, Build, Commit, dirty)
}

// GetEnvDefault returns the value of the environment variable with given key
// or the given default value, if the environment variable is not set.
func GetEnvDefault(key, value string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return value
}

// main is the main entry point of the go-make command.
func main() {
	os.Exit(make.Make(NewInfo(),
		GetEnvDefault("GOMAKE_CONFIG", Config),
		os.Stdout, os.Stderr, os.Args...))
}
