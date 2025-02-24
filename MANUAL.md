# `go-make` Manual

[`go-make`][go-make] is a thin versioned wrapper around a very generic and
customizable [`Makefile.base`](config/Makefile.base), that is usually wrapped
by a short project [`Makefile`](Makefile). In this setup the commands `go-make`
and `make` can be used as synonyms. The only difference is that `go-make` can
be executed anywhere in a project tree, while `make` can only be executed at
the root of the project tree. However, this comes with the downside that paths
are still interpreted relative to the root of the project.

In the following we use just `make`. Please substitute with `go-make`, if you
have not installed the wrapper by initially calling `go-make init-make` on a
project.

[go-make]: https://github.com/tkrop/go-make


## Setup and customization

While [`make`][go-make] should work out-of-the-box by using sensitive defaults,
the latest default [`Makefile`](Makefile) and [`Makefile.vars`](Makefile.vars)
can be installed in a project directory through calling `make init-make`.
Similar default config files for tools can be installed using `make init/<file>`
to allow customization. All files can be updated by calling `make update-make`
or simpler `make update`.

**Warning:** `go-make` automatically installs `pre-commit` and `commit-msg`
[hooks][git-hooks] overwriting and deleting pre-existing hooks (see also
[Customizing Git - Git Hooks][git-hooks]). The `pre-commit` hook calls
`make commit` as an alias for executing  `test-go`, `test-unit`, `lint-<level>`,
and `lint-markdown` to enforce successful testing and linting. The `commit-msg`
hook calls `make git-verify message` for validating whether the commit message
is following the [conventional commit][convent-commit] best practice.

[git-hooks]: <https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks>
[convent-commit]: <https://www.conventionalcommits.org/en/v1.0.0/>

For more information on customization, please see documentation of the different
target groups:

* [Standard targets](#standard-targets)
* [Test targets](#test-targets)
* [Linter targets](#linter-targets)
* [Build targets](#install-targets)
* [Image targets](#image-targets)
* [Release targets](#run-targets)
* [Install targets](#install-targets)
* [Uninstall targets](#uninstall-targets)
* [Version targets](#version-targets)
* [Update targets](#update-targets)
* [Cleanup targets](#cleanup-targets)
* [Init targets](#init-targets) (usually no need to call)

**Note:** To see an overview of actual targets, use the shell auto-completion
or run `make show targets`. To get a short help on most important targets and
target families run `make help`. To have a look at the effective targets and
receipts use `make show all`.

To customize the behavior, there exist two extension points. One at the begin
and one the end of the [Makefile](config/Makefile.base) that can be used to set
up additional variables, definitions, and targets that modify the behavior and
receipts.

* The extension point at begin is called [Makefile.vars](Makefile.vars) and
  allows to modify the behavior of targets by customizing [variables][make-vars]
  and [functions][make-calls] (see [Modifying variables](#modifying-variables)
  and [Running commands](#running-commands) for more details and examples).
* The extension point at the end is called [Makefile.ext](Makefile.ext) and is
  optimal to define additional [targets][make-rules] and [rules][make-rules],
  with [prerequisite][make-prerequisite] and [receipts][make-receipts].

**Note:** To efficiently support custom [targets][make-rules] and customization
of [rules][make-rules] the [Makefile](config/Makefile.base) is extensively
using [double colon rules][make-double-colon] (`::`). This has the following
consequences:

1. The targets of [double colon rules][make-double-colon] is [phony][make-phony]
   and triggers a [rule][make-rules] execution even if the [target][make-rules]
   exists and is up-to-date.
2. The target of [double colon rules][make-double-colon] can appear in multiple
   [rule][make-rules] making it easy to define additional [rules][make-rules]
   with new [prerequisite][make-prerequisite] and  [receipts][make-receipts]
   for existing targets.

To account for this nature [double colon rules][make-double-colon] are usually
defined using deputy targets. Please read the great [GNU `make` manual][make]
for more information on how [`make`][make] works and interacts with its
execution environment.

[make-vars]: https://www.gnu.org/software/make/manual/html_node/Using-Variables.html
[make-calls]: https://www.gnu.org/software/make/manual/html_node/Call-Function.html
[make-rules]: https://www.gnu.org/software/make/manual/html_node/Rule-Introduction.html
[make-prerequisite]: https://www.gnu.org/software/make/manual/html_node/Prerequisite-Types.html
[make-receipts]: https://www.gnu.org/software/make/manual/html_node/Recipes.html
[make-double-colon]: https://www.gnu.org/software/make/manual/html_node/Double_002dColon.html
[make-phony]: https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
[make]: https://www.gnu.org/software/make/manual/html_node/index.html


### Modifying variables

While there exist sensible defaults for all configurations variables, some of
them might need to be adjusted. The following list provides an overview of the
most prominent ones

```Makefile
# Setup default test timeout (default: 10s).
TEST_TIMEOUT := 15s
# Setup when to push images (default: pulls [never, pulls, merges])
IMAGE_PUSH ?= never
# Setup specific go-make version.
GOMAKE := github.com/tkrop/go-make@latest
# Setup the activated commit hooks (default: pre-commit commit-msg).
GITHOOKS := pre-commit commit-msg
# Setup code quality level (default: base).
CODE_QUALITY := base


# Setup codacy integration (default: enabled [enabled, disabled]).
CODACY := enabled
# Customizing codacy server (default: https://codacy.bus.zalan.do).
CODACY_API_BASE_URL := https://api.codacy.com
# (default: false / true [cdp-pipeline])
#CODACY_CONTINUE := true

# Setup required targets before testing (default: <empty>).
TEST_DEPS := run-db
# Setup required targets before running commands (default: <empty>).
RUN_DEPS := run-db
# Setup required aws services for testing (comma separated, default: <empty>).
AWS_SERVICES := s3,sqs

# Setup custom delivery files scanned for updating go versions
# (default: delivery*.yaml|.github/workflows/*.yaml).
#DELIVERY := delivery.yaml

# Setup custom local build targets (default: init test lint build).
TARGETS_ALL := init delivery test lint build

# Custom linters applied to prepare next level (default: <empty>).
LINTERS_CUSTOM := nonamedreturns gochecknoinits tagliatelle
# Linters swithed off to complete next level (default: <empty>).
LINTERS_DISABLED :=
```

The following variables are your entry points for customizing the version
of supported tools or adding additional tools:

```Makefile
TOOLS_NPM := \
  markdownlint-cli
TOOLS_GO := \
  github.com/golangci/golangci-lint/cmd/golangci-lint \
  github.com/zalando/zally/cli/zally \
  golang.org/x/vuln/cmd/govulncheck \
  github.com/uudashr/gocognit/cmd/gocognit \
  github.com/fzipp/gocyclo/cmd/gocyclo \
  github.com/mgechev/revive@v1.2.3 \
  github.com/securego/gosec/v2/cmd/gosec \
  github.com/tsenart/deadcode \
  github.com/tsenart/vegeta \
  honnef.co/go/tools/cmd/staticcheck \
  github.com/zricethezav/gitleaks/v8 \
  github.com/icholy/gomajor \
  github.com/golang/mock/mockgen \
  github.com/tkrop/go-testing/cmd/mock \
  github.com/tkrop/go-make
TOOLS_SH := \
  github.com/anchore/syft \
  github.com/anchore/grype
```

While the above list of variables is non-exhaustive, all other variables are
not part of the official interface and may be changed. Still, you can lookup
customizable variables using `make show vars`;


### Running commands

To `run-*` commands as expected, you need to setup the environment variables
for your designated runtime by defining the custom functions for setting it up
via `run-setup`, `run-vars`, `run-vars-local`, `run-vars-image`, and
`run-setup-aws`, in [Makefile.vars](Makefile.vars).

While tests are supposed to run with global defaults and test specific config,
the setup of the `run-*` commands strongly depends on the commands execution
context and its purpose. Still, there are common patterns to setup credentials
and environment variables, that can easily derived from the following example:

```Makefile
# Setup definition specific variables.
AWSOPTS ?= --region=eu-central-1 --endpoint-url=http://localhost:4566
AWSBUCKET ?= cas-apidocs-test

# Defines a make-fragment to set up all run-targets (default: true)
run-setup = \
    cp app/service/jobs.yaml $(DIR_RUN)/jobs.yaml; \
    $(call run-token-create); \
    $(call run-token-link,default,token-type,token-secret)

# Define variables for all run-targets (called with empty and '-env' argument)
run-vars = \
    $(1) GODEBUG="gctrace=1" \
    $(1) GIN_MODE="debug" \
    $(1) CAS_LOG_LEVEL="debug" \
    $(1) CAS_AUTH_TOKENINFOURL="$(HOST_TOKENINFO)" \
    $(1) CAS_S3_BUCKET="$(AWSBUCKET)" \
    $(1) CAS_S3_SHARED="false" \
    $(1) CAS_SCAN_MODE="dynamic" \
    $(1) CAS_AUTH_TIMEOUT="10s"

# Define variables for local run-targets (called only with empty argument)
run-vars-local = \
    $(1) CAS_AUTH_CREDENTIALSDIR="$(DIR_CRED)"
# Define variables for image run-targets (called with empty and '-env' argument)
run-vars-image =
# Define a make-fragment to set up aws localstack (default: true).
run-setup-aws = \
  if ! aws $(AWSOPTS) s3 ls s3://$(AWSBUCKET) >/dev/null 2>&1; then \
    aws $(AWSOPTS) s3 mb s3://$(AWSBUCKET); \
  fi
```

To enable postgres database support you must add `run-db` to `TEST_DEPS` and
`RUN_DEPS` variables to [Makefile.vars](Makefile.vars).

You can also override the default setup via the `DB_HOST`, `DB_PORT`, `DB_NAME`,
`DB_USER`, and `DB_PASSWORD` variables, but this is optional.

**Note:** when running test against a DB you usually have to extend the default
`TEST_TIMEOUT` of 10s to a less aggressive value.

To enable AWS [LocalStack][localstack] you have to add `run-aws` to the default
`TEST_DEPS` and `RUN_DEPS` variables, as well as to add your comma separated
list of required aws services to the `AWS_SERVICES` variable. The empty default
value enables all services as standard.

```Makefile
# Setup required targets before testing (default: <empty>).
TEST_DEPS := run-aws
# Setup required targets before running commands (default: <empty>).
RUN_DEPS := run-aws
# Setup required aws services for testing (comma separated, default: <empty>).
AWS_SERVICES := s3,sns,sqs
```

**Note:** Currently, the [Makefile](config/Makefile.base) does not support all
command-line arguments since make swallows arguments starting with `-`. To
compensate this shortcoming the commands need to support setup via command
specific environment variables following the principles of the
[Twelf Factor App][12factor].

[12factor]: https://12factor.net/
[localstack]: https://docs.localstack.cloud/


## Standard targets

The [Makefile](config/Makefile.base) supports the following often used standard
targets.

```bash
make all       # short cut target to init, test, and build binaries locally
make all-clean # short cut target to clean, init, test, and build binaries
make commit    # short cut target to execute pre-commit lint and test steps
```

The short cut targets can be customized by setting up the variables `TARGETS_*`
(in upper letters), according to your preferences in `Makefile.vars` or in your
+environment.

Other less customizable commands are targets to build, install, delete, and
cleanup project resources:

```bash
make test       # short cut to execute default test targets
make lint       # short cut to execute default lint targets
make build      # creates binary files of commands
make clean      # removes all resource created during build
```

While these targets allow to execute the most important tasks out-of-the-box,
there exist a high number of specialized (sometimes project specific) commands
that provide more features with quicker response times for building, testing,
releasing, and executing of components.

**Note:** All targets automatically trigger their preconditions and install the
latest version of the required tools, if some are missing. To enforce the setup
of a new tool, you need to run `make init` explicitly.

The following targets are helpful to investigate the
[Makefile](config/Makefile.base):

```bash
make show-help    # shows a short help about main targets (families)
make show-vars    # shows all supported (customizable) variables (with defaults)
make show-targets # shows all supported targets (without prerequisites)
make show-raw     # shows the raw base makefile as implemented
make show-make    # shows the effective makefile as evaluated by make
```

The `show` target supports an additional argument to switch to show different
[Makefile](config/Makefile) details:

* `raw` shows the raw text of the actual used [Makefile](config/Makefile)
  version, which allows the details of the implementation.
* `vars` shows only the customizable variables supported by the actual
  used [Makefile](config/Makefile) version.
* `targets` shows the list of actual targets supported by the effective
  [Makefile](config/Makefile) (default).
* `database` shows the effective [Makefile](config/Makefile) stored in the
  database created by `make`.


### Test targets

Often it is more efficient or even necessary to execute the fine grained test
targets to complete a task.

```bash
make test        # short cut to execute default test targets
make test-all    # executes the complete tests suite
make test-unit   # executes only unit tests by setting the short flag
make test-self   # executes a self-test of the build scripts
make test-cover  # opens the test coverage report in the browser
make test-upload # uploads the test coverage files
make test-clean  # cleans up the test files
make test-build  # test conflicts in program names
make test-image  # test conflicts in container image names
make test-cdp    # test cdp-runtime execution locally to debug scripts
make test-go     # test go versions
```

In addition, it is possible to restrict test target execution to packages,
files and test cases as follows:

* For a single package use `make test-(unit|all) <package> ...`.
* For a single test file `make test[-(unit|all) <package>/<file>_test.go ...`.
* For a single test case `make test[-(unit|all) <package>/<test-name> ...`.

The default test target can be customized by defining the `TARGETS_TEST`
variable in `Makefile.vars`. Usually this is not necessary.


### Linter targets

The [Makefile](config/Makefile.base) supports different targets that help with
linting according to different quality levels, i.e. `min`,`base` (default),
`plus`, `max`, (and `all`) as well as automatically fixing the issues.

```bash
make lint          # short cut to execute default lint targets
make lint-min      # lints the go-code using a minimal config
make lint-base     # lints the go-code using a baseline config
make lint-plus     # lints the go-code using an advanced config
make lint-max      # lints the go-code using an expert config
make lint-all      # lints the go-code using an insane all-in config
make lint-codacy   # lints the go-code using codacy client side tools
make lint-markdown # lints the documentation using markdownlint
make lint-revive   # lints the go-code using the revive standalone linter
make lint-shell    # lints the sh-code using shellcheck to find issues
make lint-leaks    # lints committed code using gitleaks for leaked secrets
make lint-leaks?   # lints un-committed code using gitleaks for leaked secrets
make lint-vuln     # lints the go-code using govulncheck to find vulnerabilities
make lint-api      # lints the api specifications in '/zalando-apis'
```

The default target for `make lint` is determined by the selected `CODE_QUALITY`
level (`min`, `base`, `plus`, and `max`), and the `CODACY` setup (`enabled`,
`disabled`). The default setup is to run the targets `lint-base`, `lint-apis`,
`lint-markdown`, and `lint-codacy`. It can be further customized via changing
the `TARGETS_LINT` in `Makefile.vars` - if necessary.

The `lint-*` targets for `golangci-lint` allow some command line arguments:

1. The keyword `fix` to lint with auto fixing enabled (when supported),
2. The keyword `config` to shows the effective linter configuration,
3. The keyword `linters` to display the linters with description, or
4. `<linter>,...` comma separated list of linters to enable for a quick checks.

The default linter config is providing a golden path with different levels
out-of-the-box, i.e. a `min` for legacy code, `base` as standard for active
projects, and `plus` for experts and new projects, and `max` enabling all
but the conflicting disabled linters. Besides, there is an `all` level that
allows to experience the full linting capability.

Independent of the golden path this setting provides, the lint expert levels
can be customized in three ways.

1. The default way to customize linters is adding and removing linters for all
   levels by setting the `LINTERS_CUSTOM` and `LINTERS_DISABLED` variables
   providing a white space separated list of linters.
2. Less comfortable and a bit trickier is the approach to override the linter
   config variables `LINTERS_DISCOURAGED`, `LINTERS_DEFAULT`, `LINTERS_MINIMUM`,
   `LINTERS_BASELINE`, and `LINTERS_EXPERT`, to change the standards.
3. Last the linter configs can be changed via `.golangci.yaml`, as well as
   via `.codacy.yaml`, `.markdownlint.yaml`, and `revive.toml`.

However, customizing `.golangci.yaml` and other config files is currently not
advised, since the `Makefile` is designed to update and enforce a common
version of all configs on running `update-*` targets.


### Build targets

The build targets can build native as well as linux platform executables using
the default system architecture.

```bash
make build         # builds default executables (native)
make build-native  # builds native executables using system architecture
make build-linux   # builds linux executable using the default architecture
make build-image   # builds container image (alias for image-build)
```

The platform and architecture of the created executables can be customized via
`BUILDOS` and `BUILDARCH` environment variables.


### Image targets

Based on the convention that all binaries are installed in a single container
image, the [Makefile](config/Makefile.base) supports to create and push the
container image as required for a pipeline.

```bash
make image        # short cut for 'image-build'
make image-build  # build a container image after building the commands
make image-push   # pushes a container image after building it
```

The targets are checking silently whether there is an image at all, and whether
it should be build and pushed according to the pipeline setup. You can control
this behavior by setting `IMAGE_PUSH` to `never` or `test` to disable pushing
(and building) or enable it in addition for pull requests. Any other value will
ensure that images are only pushed for `main`-branch and local builds.


### Run targets

The [Makefile](config/Makefile.base) supports targets to startup a common DB
and a common AWS container image as well as to run the commands provided by the
repository.

```bash
make run-db       # runs a postgres container image to provide a DBMS
make run-aws      # runs a localstack container image to simulate AWS
make run-*        # runs the matched command using its before build binary
make run-go-*     # runs the matched command using 'go run'
make run-image-*  # runs the matched command in the container image
make run-clean    # kills and removes all running container images
make run-clean-*  # kills and removes the container image of the matched command
```

To run commands successfully the environment needs to be setup to run the
commands in its runtim. Please visit [Running commands](#running-commands) for
more details on how to do this.

**Note:** The DB (postgres) and AWS (localstack) containers can be used to
support any number of parallel applications, if they use different tables,
queues, and buckets. Developers are encouraged to continue with this approach
and only switch application ports and setups manually when necessary.


### Update targets

The [Makefile](config/Makefile.base) supports targets for common update tasks
for package versions, for build, test, and linter tools, and for configuration
files.

```bash
make update        # short cut for 'update-{go,deps,make}'
make update-all    # short cut to execute all update targets
make update-go     # updates the go to the given, current, or latest version
make update-deps   # updates the project dependencies to the latest version
make update-make   # updates the build environment to a given/latest version
make update-tools  # updates the project tools to a given/latest version
make update-*      # updates a specific tool to a given/latest version
```

Most `update-*` targets support a optional `<version>` argument to select a
specific version (also `latest`) as well as a `?`-suffix to check whether an
update is available instead of executing it directly.

* For `update(-deps)` a `<mode>` can be supplied to update dependencies to the
  latest `minor` (default), `major`, or `pre`-release version.
* For `update(-make)` also `current` can be supplied as version to update the
  the config files to the version determined by the currently executed
  `Makefile`.


### Cleanup targets

The [Makefile](config/Makefile.base) is designed to clean up everything it has
created by executing the following targets.

```bash
make clean         # short cut for clean-init, clean-build
make clean-all     # cleans up all resources, i.e. also tools installed
make clean-init    # cleans up all resources created by init targets
make clean-hooks   # cleans up all resources created by init-hooks targets
make clean-build   # cleans up all resources created by build targets
make clean-run     # cleans up all running container images
make clean-run-*   # cleans up matched running container image
```


### Install targets

The install targets installs the latest build version of a command in the
`${GOPATH}/bin` directory for global command line execution. Usually commands
used by the project are installed automatically.

```bash
make install      # installs all software created by this project
make install-all  # installs all software created by this project
make install-*    # installs the matched software command or service
```

If a command, service, job has not been build before, it is first build.

**Note:** Please use carefully, if your project uses common command names.


### Uninstall targets

The uninstall targets remove the latest installed command from `${GOPATH}/bin`.
A full uninstall of commands used by the project can also be triggered by
`clean-all`.

```bash
make uninstall      # uninstalls all software created by this project
make uninstall-all  # uninstalls all software created or used by this project
make uninstall-*    # uninstalls the matched software command or service
```

**Note:** Please use carefully, if your project uses common command names.


### Version targets

Finally, the [Makefile](config/Makefile.base) supports targets for bumping,
releasing, and publishing the provided packages as library.

```bash
make version-bump <version>  # bumps version to prepare a new release
make version-release         # creates the release tags in the repository
make version-publish         # publishes the version to the go-proxy
```

The `version-bump` target supports an optional `<version>` argument that allows
defining a version or increment the version using the semantic keywords `major`,
`minor`, and `patch` to signal the scope.


### Init targets

The [Makefile](config/Makefile.base) supports initialization targets usually
added as prerequisites for targets that require them. So there is really no
need to call them manually.


```bash
make init         # short cut for 'init-git init-hooks init-go init-make'
make init-go      # initializes go project files
make init-git     # initializes git resource revision control
make init-hooks   # initializes git hooks for pre-commit, etc
make init-codacy  # initializes tools for running the codacy targets
make init-code    # initializes code by generating mocks and kube-apis
make init-make    # initializes project by copying template files
make init-make!   # initializes Makefile to contain a copy of Makefile.base
```

The `init-make` targets support a `<version>` argument to install the config
files from a specific config version.


### Git targets

The [Makefile](config/Makefile.base) supports the following experimental
targets that are featuring some complex git command chains helping to setup
conventional commits, where `*` is a placeholder for the [conventional commit
types](#commit-types):

```bash
make git-list          # shows the git log as pretty printed list
make git-graph         # shows the git log as pretty printed graph
make git-clean [all]   # cleans up git history by removing merged branches
make git-reset [all]   # checks out default branch and cleans up git history
make git-create(-*)    # creates and pushes a branch with the current changes
make git-commit(-*)    # commits the current change set to the current branch
make git-fix(-*) [...] # pushes the latest changes to the previous commit
make git-push          # pushes the current branch to the upstream repository
make git-verify        # checks git log to follow commit conventions
```

The `git-create(-*)` targets support `<branch>` and a `<message...>` argument
list. The message can be a loose collection of words, that will be extended
by adding a conventional [commit type](#commit-types) and should contain an
issue reference. If not issue reference is provided the last issue increased
by one is used. Similar `git-commit(-*)` targets support a `<message...>`
argument is enriched, but reusing the previous issue type.

The `git-reset` and `git-clean` targets support an optional `all` argument
to define whether also pushed branches should be cleaned up instead of only
merged branches.

The `git-fix` target supports `(no-)edit` and `(no)-verify` arguments to enable
and disable commit verification and comment editing. By default, it is enabling
verification but disabling editing. The default behavior can be defined by
setting providing the `GITFIX` environment variable.

The `git-verify` target verifies that commit messages in git log entries are
following a common commit convention containing a [commit types](#commit-types),
provide a github issue references in the title, and are signed-off. While the
target allows validating a full git `logs`, it by default, only verifies the
commits added to the current branch in relation to the main branch. The message
is expected to look as follows:

```text
<type>[(<scope>)][!]': <subject> (<issue>)

[body]

[footer(s)]
Signed-of-by: <author-name> <<author-email>>
```


## Commit types

The [Makefile](config/Makefile.base) supports the following commit types as
described by the [GitHub Development Convention][github-commit].

|    Type       | Title         | Description                    |
|:-------------:|---------------|--------------------------------|
| ✨ `feat`     | Features      | Adds a new feature. |
| ⌛ `deprecate`| Deprecation   | Deprecates an existing feature. |
| ❌ `remove`   | Removal       | Removes an existing feature. |
| 🗑 `revert`   | Reverts       | Reverts a previous commit. |
| 🪲 `fix`      | Bug Fix       | Fixes a bug in a feature. |
| ♻️ `chore`     | Chores        | Regular update for maintenance. |
| 📚 `docs`     | Documentation | Updates documentation only. |
| 💎 `style`    | Style Change  | Changes the code style only.  |
| 🛠 `refactor` | Refactoring   | Improves code quality by refactoring. |
| 🚀 `perf`     | Performance   | Improves the performance of a feature. |
| 🚗 `test`     | Tests         | Adds a missing or corrects an existing test. |
| 📦 `build`    | Builds        | Changes the product delivery. |
| 🏗️ `ci`       | Integrations  | Improves the build process. |

[github-commit]: https://github.com/FlowingCode/DevelopmentConventions/blob/main/conventional-commits.md


## Compatibility

This [Makefile](config/Makefile.base) is making extensive use of GNU tools but
is supposed to be compatible to all recent Linux and MacOS versions. Since
MacOS has not update GNU tool versions since a decade, `go-make` is installing
and applying latest versions provided via `brew`, which solves most potential
issues.

Beside this, we `go-make` keeps in mind a small number dependency issues on
older Ubuntu `20.04` standard installation issues that are documented in the
following.


### `realpath` not supported

In MacOS we need to use `readlink -f` instead of `realpath`, since there may
not even be a simplified fallback of this command available. This is not the
preferred command in Linux, but for compatibility would be still acceptable.


### `gensub` in `awk` not supported

Some build systems are not providing a recent `gawk` version, but provide
`mawk`as alternative, which is know to miss support for `gensub`. We have
substituted the usage of `gensub` in some essential functions that would
else fail.
