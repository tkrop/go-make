# Make framework

[![Build][build-badge]][build-link]
[![Coverage][coveralls-badge]][coveralls-link]
[![Coverage][coverage-badge]][coverage-link]
[![Quality][quality-badge]][quality-link]
[![Report][report-badge]][report-link]
[![FOSSA][fossa-badge]][fossa-link]
[![License][license-badge]][license-link]
[![Docs][docs-badge]][docs-link]
<!--
[![Libraries][libs-badge]][libs-link]
[![Security][security-badge]][security-link]
-->

[build-badge]: https://github.com/tkrop/go-make/actions/workflows/go.yaml/badge.svg
[build-link]: https://github.com/tkrop/go-make/actions/workflows/go.yaml

[coveralls-badge]: https://coveralls.io/repos/github/tkrop/go-make/badge.svg?branch=main
[coveralls-link]: https://coveralls.io/github/tkrop/go-make?branch=main

[coverage-badge]: https://app.codacy.com/project/badge/Coverage/b2bb898346ae4bb4be6414cd6dfe4932
[coverage-link]: https://www.codacy.com/gh/tkrop/go-make/dashboard?utm_source=github.com&utm_medium=referral&utm_content=tkrop/go-make&utm_campaign=Badge_Coverage

[quality-badge]: https://app.codacy.com/project/badge/Grade/b2bb898346ae4bb4be6414cd6dfe4932
[quality-link]: b2bb898346ae4bb4be6414cd6dfe4932https://app.codacy.com/gh/tkrop/go-make/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade

[report-badge]: https://goreportcard.com/badge/github.com/tkrop/go-make
[report-link]: https://goreportcard.com/report/github.com/tkrop/go-make

[fossa-badge]: https://app.fossa.com/api/projects/git%2Bgithub.com%2Ftkrop%2Fgo-make.svg?type=shield&issueType=license
[fossa-link]: https://app.fossa.com/projects/git%2Bgithub.com%2Ftkrop%2Fgo-make?ref=badge_shield&issueType=license

[license-badge]: https://img.shields.io/badge/License-MIT-yellow.svg
[license-link]: https://opensource.org/licenses/MIT

[docs-badge]: https://pkg.go.dev/badge/github.com/tkrop/go-make.svg
[docs-link]: https://pkg.go.dev/github.com/tkrop/go-make

<!--
[libs-badge]: https://img.shields.io/librariesio/release/github/tkrop/go-make
[libs-link]: https://libraries.io/github/tkrop/go-make

[security-badge]: https://snyk.io/test/github/tkrop/go-make/main/badge.svg
[security-link]: https://snyk.io/test/github/tkrop/go-make
-->

## Introduction

Goal of `go-make` is to provide a simple, versioned build environment for
standard [`go`][go]-projects (see [Standard `go`-project](#standard-go-project)
for details) providing default targets and tool configs for testing, linting,
building, installing, updating, running, and releasing libraries, commands, and
container images.

`go-make` can be either run as command line tool or hooked into an existing
project as minimal [`Makefile`](Makefile). Technically `go-make` is just a thin
wrapper around a very generic and extensible [`Makefile`](Makefile.base) that
is based on a standard [`go`][go]-project supporting different tools:

* [`gomock`][gomock] - go generating mocks.
* [`codacy`][codacy] - for code quality documentation.
* [`golangci-lint`][golangci] - for pre-commit linting.
* [`zally`][zally] - for pre-commit API linting.
* [`gitleaks`][gitleaks] - for sensitive data scanning.
* [`grype`][grype] - for security scanning.
* [`syft`][syft] - for material listing.

The thin wrapper provides the necessary version control for the `Makefile` and
the default config of integrated tools. It installs these tools automatically
when needed in the latest available version.

**Note:** We except the risk that using the latest versions of tools, e.g. for
linting, may break the build for the sake of constantly updating dependencies by
default. For tools where this is not desireable, the default import can be
changed to contain a version (see [manual](MANUAL.md] for more information).

**Warning:** `go-make` automatically installs a `pre-commit` hook overwriting
and deleting any pre-existing hook. The hook calls `go-make commit` to enforce
run unit testing and linting successfully before allowing to commit, i.e. the
goals `test-go`, `test-unit`, `lint-base` (or what code quality level is
defined as standard), and `lint-markdown`.


[gomock]: <https://github.com/uber/mock>
[golangci]: <https://github.com/golangci/golangci-lint>
[codacy]: <https://www.codacy.com/>
[zally]: <http://opensource.zalando.com/zally>
[gitleaks]: <https://github.com/gitleaks/gitleaks>
[grype]: <https://github.com/anchore/grype>
[syft]: <https://github.com/anchore/syft>


## Installation

To install `go-make` simply use the standard [`go` install][go-install]
command (or any other means, e.g. [`curl`][curl] to obtain a released binary):

```bash
go install github.com/tkrop/go-make@latest
```

The scripts and configs are automatically checked out in the version matching
the wrapper. `go-make` has the following dependencies, that must be satisfied
by the runtime environment, e.g. using [`ubuntu-20.04`][ubuntu-20.04] or
[`ubuntu-22.04`][ubuntu-22.04]:

* [GNU `make`][make] (^4.2).
* [GNU `bash`][bash] (^5.0).
* [GNU `coreutils`][core] (^8.30)
* [GNU `findutils`][find] (^4.7)
* [GNU `awk`][awk] (^5.0).
* [GNU `sed`][sed] (^4.7)
* [`curl`][curl] (^7)

[ubuntu-20.04]: <https://releases.ubuntu.com/focal/>
[ubuntu-22.04]: <https://releases.ubuntu.com/jammy/>
[go-install]: <https://go.dev/doc/tutorial/compile-install>
[curl]: <https://curl.se/>
[make]: <https://www.gnu.org/software/make/>
[bash]: <https://www.gnu.org/software/bash/>
[core]: <https://www.gnu.org/software/coreutils/>
[find]: <https://www.gnu.org/software/findutils/>
[awk]: <https://www.gnu.org/software/awk/>
[sed]: <https://www.gnu.org/software/sed/>


## Example usage

After installing `go-make` and in the build environment, you can run all targets
by simply calling `go-make <target>` on the command line, in another `Makefile`,
in a github action, or any other delivery pipeline config script:

```bash
go-make all        # execute a whole build pipeline depending on the project.
go-make test lint  # execute only test 'test' and 'lint' steps of a pipeline.
go-make image      # execute minimal steps to create all container images.
```

If you like to integrate `go-make` into another `Makefile` you may find the
following template helpful:

```Makefile
GOBIN ?= $(shell go env GOPATH)/bin
GOMAKE := github.com/tkrop/go-make@latest
TARGETS := $(shell command -v go-make >/dev/null || \
    go install $(GOMAKE) && go-make targets)

# Include standard targets from go-make providing group targets as well as
# single target targets. The group target is used to delegate the remaining
# request targets, while the single target can be used to define the
# precondition of custom target.
.PHONY: $(TARGETS) $(addprefix target/,$(TARGETS))
$(TARGETS):; $(GOBIN)/go-make $(MAKEFLAGS) $(MAKECMDGOALS);
$(addprefix target/,$(TARGETS)): target/%:
    $(GOBIN)/go-make $(MAKEFLAGS) $*;
```

For further examples see [`go-make` manual](MANUAL.md).

**Note:** To setup command completion for `go-make`add the following command to
your `.bashrc`.

```bash
source <(go-make --completion=bash)
```


## Standard `go`-Project

The [Makefile](Makefile) provided in this project is working under the
conventions of a standard [`go`][go]-project. The standard [`go`][go]-project
is defined to meet Zalando in-house requirements, but is general enough to be
useful in open source projects too. It adheres to the following conventions:

1. All commands (services, jobs) are provided by a `main.go` placed as usual
   under `cmd` using the pattern `cmd/<name>/main.go` or in the root folder. In
   the latter case the project name is used as command name.

2. All source code files and package names following the [`go`][go]-standard
   only consist of lower case ASCII letters, hyphens (`-`), underscores (`_`),
   and dots (`.`). Files are ending with `.go` to be eligible.

3. Modules are placed in any sub-path of the repository, e.g. in `pkg`, `app`,
   `internal` are commonly used patterns, except for `build` and `run`. These
   are used by `go-make` as temporary folders to build commands and run commands
   and are cleaned up regularly.

4. The build target provides build context values to set up global variables in
   `main` and `config` packages.

   * `Path` - the formal package path of the command build.
   * `Version` - the version as provided by the `VERSION`-file in project root
     or via the `(BUILD_)VERSION` environ variables.
   * `Revision` - the actual full commit hash (`git rev-parse HEAD`).
   * `Build` - the current timestamp of the build (`date --iso-8601=seconds`).
   * `Commit` - the timestamp of the actual commit timestamp
     (`git log -1 --date=iso8601-strict --format=%cd`).
   * `Dirty` - the information whether build repository had uncommitted changes.

5. All container image build files must start with a common prefix (default is
   `Dockerfile`). The image name is derived from the organization and repository
   names and can contain an optional suffix, i.e `<org>/<repo-name>(-<suffix>)`.

6. For running a command in a container image, make sure that the command is
   installed in the default execution directory of the container image - usually
   the root directory. The container image must either be generated with suffix
   matching the command or without suffix.

All targets in the [Makefile](Makefile) are designated to autonomously set up
setup the [`go`][go]-project, installing the necessary tools - except for the
golang compiler and build environment -, and triggering the precondition
targets as necessary.

[go]: <https://go.dev/>


## Terms of usage

This software is open source as is under the MIT license. If you start using
the software, please give it a star, so that I know to be more careful with
changes. If this project has more than 25 Stars, I will introduce semantic
versions for changes.


## Building

The project is using itself for building as a proof of concept.


## Contributing

If you like to contribute, please create an issue and/or pull request with a
proper description of your proposal or contribution. I will review it and
provide feedback on it.
