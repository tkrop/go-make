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

[build-badge]: https://github.com/tkrop/go-make/actions/workflows/build.yaml/badge.svg
[build-link]: https://github.com/tkrop/go-make/actions/workflows/build.yaml

[coveralls-badge]: https://coveralls.io/repos/github/tkrop/go-make/badge.svg?branch=main
[coveralls-link]: https://coveralls.io/github/tkrop/go-make?branch=main

[coverage-badge]: https://app.codacy.com/project/badge/Coverage/b2bb898346ae4bb4be6414cd6dfe4932
[coverage-link]: https://app.codacy.com/gh/tkrop/go-make/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_coverage

[quality-badge]: https://app.codacy.com/project/badge/Grade/b2bb898346ae4bb4be6414cd6dfe4932
[quality-link]: https://app.codacy.com/gh/tkrop/go-make/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade

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

Goal of [`go-make`][go-make] is to provide a simple, versioned build and test
environment for "common [`go`][go]-projects" to make standard development tasks
easy (see also [Standard `go`-project](#standard-go-project) for details). To
accomplish this goal [`go-make`][go-make] provides default targets, tools, and
configs for testing, linting, building, installing, updating, running, and
releasing libraries, binaries, and container images.

[`go-make`][go-make] can be either run as command line tool or hooked into an
existing project via a minimal [`Makefile`](config/Makefile). Technically
[`go-make`][go-make] is a thin versioning wrapper for a very generic
[`Makefile`](config/Makefile.base) and configs supporting different tools and
unix platforms, i.e. Linux & MacOS (Darwin):

* [`gomock`][gomock] - go generating mocks.
* [`codacy`][codacy] - for code quality documentation.
* [`golangci-lint`][golangci] - for pre-commit linting.
* [`zally`][zally] - for pre-commit API linting.
* [`gitleaks`][gitleaks] - for sensitive data scanning.
* [`grype`][grype] - for security scanning.
* [`syft`][syft] - for material listing.

The [`go-make`][go-make] wrapper provides the necessary version control for the
[`Makefile`](config/Makefile.base) and the [`config`](config) of the tools. The
tools are automatically installed or updated when needed in the configured (or
latest) available version using a default or custom config file. All config
files can be installed and customized (see
[Setup and customization](MANUAL.md#setup-and-customization)).

**Note:** For many tools [`go-make`][go-make] accepts the risk that using the
latest versions of tools, e.g. for linting, may break the build to allow
continuous upgrading of dependencies by default. For tools were this is not
desireable, e.g. for [`revive`][revive] and [`golangci-lint`][golangci] the
default import is version. Other tools can be versioned if needed (see
[manual](MANUAL.md) for more information).

[go-make]: <https://github.com/tkrop/gomake>
[gomock]: <https://github.com/uber/mock>
[golangci]: <https://github.com/golangci/golangci-lint>
[revive]: <https://github.com/mgechev/revive>
[codacy]: <https://www.codacy.com/>
[zally]: <http://opensource.zalando.com/zally>
[gitleaks]: <https://github.com/gitleaks/gitleaks>
[grype]: <https://github.com/anchore/grype>
[syft]: <https://github.com/anchore/syft>


## Installation

To install [`go-make`][go-make] simply use [`go` install][go-install] command
(or any other means, e.g. [`curl`][curl] to obtain a released binary):

```bash
go install github.com/tkrop/go-make@latest
```

The scripts and configs are automatically checked out in the version matching
the wrapper. [`go-make`][go-make] has the following dependencies, that must be
satisfied by the runtime environment, e.g. using [`ubuntu-20.04`][ubuntu-20.04],
[`ubuntu-22.04`][ubuntu-22.04], [`ubunut-24.04`][ubuntu-24.04], or
[`MacOSX`][mac-osx]:

* [GNU `make`][make] (^4.1).
* [GNU `bash`][bash] (^5.0).
* [GNU `coreutils`][core] (^8.30)
* [GNU `findutils`][find] (^4.7)
* [GNU `awk`][awk] (^5.0).
* [GNU `sed`][sed] (^4.7)
* [`curl`][curl] (^7)

**Note:** Since [`MacOSX`][mac-osx] comes with heavily outdated GNU tools,
[`go-make`][go-make] is setting up its necessary environment using the
[`brew`][brew] package manager only requiring a minimal pre-condition of
[`go`][go] and [GNU `make`] that is usually satisfied by the standard
installation.

[ubuntu-20.04]: <https://releases.ubuntu.com/focal/>
[ubuntu-22.04]: <https://releases.ubuntu.com/jammy/>
[ubuntu-24.04]: <https://releases.ubuntu.com/noble/>
[mac-osx]: <https://support.apple.com/en-gb/mac>
[go-install]: <https://go.dev/doc/tutorial/compile-install>
[brew]: <https://brew.sh/>
[curl]: <https://curl.se/>
[make]: <https://www.gnu.org/software/make/>
[bash]: <https://www.gnu.org/software/bash/>
[core]: <https://www.gnu.org/software/coreutils/>
[find]: <https://www.gnu.org/software/findutils/>
[awk]: <https://www.gnu.org/software/awk/>
[sed]: <https://www.gnu.org/software/sed/>


## Example usage

After installing [`go-make`][go-make], all provided targets can executed by
simply calling `go-make <target>` in the project repository on the command
line, in another `Makefile`, in a github action, or any other delivery pipeline
config script:

```bash
go-make all        # execute a whole build pipeline depending on the project.
go-make test lint  # execute only test 'test' and 'lint' steps of a pipeline.
go-make image      # execute minimal steps to create all container images.
```

For further examples see [`go-make` manual](MANUAL.md).

**Note:** Many [`go-make`][go-make] targets can be customized via environment
variables, that by default are defined via [`Makefile.vars`](Makefiles.vars)
(see also [Modifying variables](Manual.md#modifying-variables)).


## Makefile integration

If you like to integrate [`go-make`][go-make] into another `Makefile` you may
find the [`Makefile`](config/Makefile.base) provided in the [config](config)
helpful that automatically installs [`go-make`][go-make] creates a set of phony
targets to allow auto-completion and delegates the execution (see also
[`Makefile`](config/Makefile)).

The default [`Makefile`](config/Makefile) can also be installed to a project
from the [config](config) via `go-make init-make` to boot strap a project.
Other available [config](config) files can be installed one by one using
`go-make init/<file>`.


## Shell integration

To set up command completion for [`go-make`][go-make], add the following
snippet to your [`.bashrc`][bashrc].

```bash
source <(go-make --completion=bash)
```

[bashrc]: <https://www.gnu.org/software/bash/manual/bash.html>


## Makefile development

To extend the `Makefile`, you develop own receipts in a custom file called
[`Makefile.ext`](Makefile.ext) that is included automatically. If you want to
extend original receipts, you can use `make install-make!` to automatically
replace the wrapper [`Makefile`](config/Makefile) against the original
[`Makefile.base`](config/Makefile.base) and adding a local
[`MANUAL.md`](MANUAL.md) to your project.


## Standard `go`-Project

The [`Makefile.base`](config/Makefile.base) provided in this project is based
on a standard [`go`][go]-project setting some limitations. The standard
[`go`][go]-project is defined to meet Zalando in-house requirements, but is
general enough to be useful in open source projects too. It adheres to the
following conventions:

1. All commands (services, jobs) are provided by a `main.go` placed as usual
   under `cmd` using the pattern `cmd/<name>/main.go` or in the root folder. In
   the latter case the project name is used as command name.

2. All source code files and package names following the [`go`][go]-standard
   only consist of lower case ASCII letters, hyphens (`-`), underscores (`_`),
   and dots (`.`). Files are ending with `.go` to be eligible.

3. Modules are placed in any sub-path of the repository, e.g. in `pkg`, `app`,
   `internal` are commonly used patterns, except for `build` and `run`. These
   are used by [`go-make`][go-make] as temporary folders to build commands and
   run commands and are cleaned up regularly.

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
   matching the command or without suffix in common container.

All targets in the [Makefile](config/Makefile.base) are designated to set up
the [`go`][go]-project automatically, installing the necessary tools - except
for the golang compiler and build environment -, and triggering the required
targets as necessary.

[go]: <https://go.dev/>


## Trouble Shooting

If we have published a non-working version of [`go-make`][go-make] and your
project is not able to build, test, run, etc, the quickest way to reset the
project [Makefile](config/Makefile) working [`go-make`][go-make] version is to
run:

```bash
go install github.com/tkrop/go-make@latest; go-make update;
```

If the latest version is not fixed yet, you can also try to move backward
finding the last working [tagged version](tags).


## Terms of usage

This software is open source as is under the MIT license. If you start using
the software, please give it a star, so that I know to be more careful to keep
changes non-breaking.


## Building

The project is using itself for building as a proof of concept. So either run
`make all` or `go-make all`. As fall back it is always possible to directly use
the core [Makefile](Makefile.base) calling:

```bash
make -f config/Makefile.base <target>...
```

You can also test the local build [`go-make`][go-make] application with the
local config. The project compiles itself to use the local config by default.


## Contributing

If you like to contribute, please create an issue and/or pull request with a
proper description of your proposal or contribution. I will review it and
provide feedback on it as soon as possible.
