// Main entry point of the go-make command.
package main

import (
	"os"

	"github.com/tkrop/go-config/info"
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

// main is the main entry point of the go-make command.
func main() {
	os.Exit(make.Make(os.Stdin, os.Stdout, os.Stderr,
		info.New(Path, Version, Revision, Build, Commit, Dirty),
		make.GetEnvDefault(make.EnvGoMakeConfig, Config),
		".", nil, os.Args...))
}
