//go:build !test

// Main entry point of the go-make command.
package main

import (
	"os"

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
)

// NewInfo returns the build information of a command or module with
// default values.
func NewInfo() *info.Info {
	return info.NewInfo(Path, Version, Revision, Build, Commit, true)
}

// main is the main entry point of the go-make command.
func main() {
	os.Exit(make.Make(NewInfo(),
		os.Stdout, os.Stderr, os.Args...))
}
