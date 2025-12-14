# Copilot / AI generation standards

Apply these rules to all AI-assisted Go code in this repository. They override
generic Go conventions when different.


## General rules

* Prioritize the rules in this document over existing and applied code style or
  otherwise defined conventions.


## Formatting

* Limit lines to 80 characters:
  * prefer breaking after opening brackets, commas, and operators.
  * Keep directly succeeding brackets / parentheses always on the same line.
* Split long function calls, declarations, composite literals across lines for
  clarity.
* Use `github.com/tkrop/go-make format` for consistent formatting:
  * Use `goimports` to manage import order automatically.
  * Always use `gofmt` to format code.
* Add blank lines to separate logical groups of code.
* Add short, concise doc comments for all exported functions, types, and
  constants.
* Be sparse with inline comments and let the code speak.


## Naming & documentation

* Write self-documenting code with concise, clear names.
* Keep short-lived variable names short.
* Avoid meaningless prefixes or suffixes.
* Follow standard Go conventions unless superseded here.
* Document exported types, functions, methods, and packages.
* Write comments in English by default; translate only upon user request.
* Avoid using emoji in code and comments.
* Avoid stuttering (e.g. avoid http.HTTPServer, prefer http.Server).
* Name interfaces with `-er` suffix when possible (e.g., Reader, Writer,
  Formatter).
* Single-method interfaces should be named after the method (e.g.,
  Read → Reader).


## Project layout

* Follow standard `go`-project layout conventions.
* Keep main in sub-packages of the `cmd` directory. The directory name is
  determining the command name.
* Put reusable packages at following locations:
  * `./<name>` (top-level-packages) — for formally exported packages of a
    library.
  * `./app` — for components following the micros-service design pattern.
    Standard sub-packages are:
    * `./app/config` — for the configuration structures and methods.
    * `./app/model` — for domain models containing their `struct`s and
      `method`s.
    * `./app/gateway` — for the clients to access other services.
    * `./app/repository` — for the clients to access other databases.
    * `./app/resource` — for the implementation of micro-service resources,
      i.e. web handlers and web hooks.
    * `./app/service` — for the implementation of the service logic.
  * `./internal` — for private packages outside of the micro-service context.
  * `./pkg` may be used for components following the hexagonal architecture
* Group related functionality into packages.
* Avoid circular dependencies.


## Patterns & Design

* Write simple, clear, and idiomatic `go`-code.
* Favor clarity and simplicity over cleverness.
* Use Go modules for dependency management.
* Use interfaces for abstractions and to enable mocking.
* Keep functions and types small and focused.
* Keep the happy path left-aligned (minimize indentation).
* Prefer composition over inheritance.
* Make best use of fluent interface design patterns.
* Follow the principle of least surprise.
* Follow clean-code principles:
  * Single Responsibility.
  * Don't repeat yourself.
  * Don't re-invent the wheel (use standard library), e.g.:
    * `strings.Builder` for string concatenation.
    * `filepath.Join` for path construction
* Use `for` with `range` when applicable.
* Return early to reduce nesting.
* Make the zero value useful.


## Performance

* Use appropriate efficient algorithms and data structures.
* Minimize unnecessary memory allocations, especially in hot paths.
* Reuse objects when possible (consider sync.Pool).
* Preallocate slices when size is known.
* Avoid unnecessary string conversions.


## Concurrency

* Use `go`-routines for concurrent tasks.
* Use `channel`s for coordination and communication.
* Avoid shared mutable state where possible.


## Types

* Prefer generics and type values over unconstrained types.
* Prefer `struct`s for composite or complex data.
* Use type aliases for clarity when beneficial.
* Define custom types for domain concepts.
* For unconstrained types use `any` instead of `interface{}`.
* Pointers vs Values
  * Consider the zero value when choosing pointer vs value receivers;
    usually prefer pointer receivers.
  * Use pointers for large `struct`s or when you need to modify the receiver.
  * Use value receivers for small `struct`s and when immutability is desired.
  * Use pointer parameters when you need to modify the argument or for
    large struct.
  * Use value parameters for small `struct`s and when you want to prevent
    modification.
* Be consistent within a type's method set.


## Errors

* Don't log and return errors (choose one).
* Handle errors immediately after function call.
* Wrap errors with meaningful context information using `fmt.Errorf(...%w...)`.
* Provide exported helper functions to wrap and create errors.
* Create custom error types when you need to check for specific errors.
* Consider using structured errors for better debugging.
* Keep error messages lowercase and don't end with punctuation.
* Place error returns as the last return value.
* Name error variables `err`.


## Testing

All tests should be based on `github.com/tkrop/go-testing/{test,mocks,gock}`
and `github.com/stretcher/testify/{assert,require}` using the following
patterns:

* Test are always written into `<file>_test.go` matching the origin
  `<file>.go`.
* Test order should always follow the order of functions and types in
  `<file>.go`.
* Test should prefer table-driven format, especially for variations of inputs.
  * Non-table-driven test should only be refactored into table-driven test
    when this allows to join multiple none-table-driven tests.
* Test should use external components, if the provided output reliably
  supports the test cases for 100% test coverage.
  * If external components are unreliable or do not provide the require output,
    the test should fall back to mocking the components.
  * Mocks should always be based on `go.uber.org/mock/{gomock.mockgen}`.
  * Mocks should be dynamically generated via `go:generate mockgen` using
    `go.uber.org/mock/mockgen` into `mock_<name>_test.go` files.
  * Mock setup should be based on `github.com/tkrop/go-testing/mock`.


### Parameter type definition

* Define an unexported top-level type struct named in camel case with suffix
  `Params` for each test parameter set.
  * Do not add a prefix `test` to the parameter set types.
* Use consistent ordering of parameters in set:
  * `setup` (mock.SetupFunc) — setup for mocks and runtime expectations.
    * Use generic `func Call<[Component]Func>(...) mock.SetupFunc` helper
      function making best use of the fluent interface.
    * Return the last mock call in `mock.SetupFunc` helper functions to
      support meaningful ordering of mock calls.
    * `Call<Func>` helper functions should be defined at the top of the file.
    * Use `mock.{Setup,Chain,Parallel,Sub,Detach}` to set up mock calls with
      ordering conditions.
    * Also follow the before pattern when not setting up an explicit helper
      function.
  * Input values are in general defined by structs (preferred) and primitives.
  * `call` (test.SetupFunc,mock.SetupFunc) — optional dynamic execution
     function.
  * `{expect|error}[<name>]` — expectations for outcomes:
    * All expectations must be prefixed by `expect` followed by the output
      variable name.
    * All error expectations must be prefixed by `error` followed by the
      output case.
    * If only a single outcome and a single error is defined the parameters
      should be named `expect` and `error`.
* Ensure that parameters are unexported if possible.
* Ensure that expectations provide the exact match and can be compared
  by a single `assert.Equal`.
* Ensure that expectations cover both, validity and outcome.


### Test case declaration

* Declare test cases in an unexported top-level `map[string]<Type>Params`
  variable.
* Suffix the test case map variable name with `TestCases`.
* Use short names referencing the input and/or scenario only (not outcomes).
* Names must only use lower-case letters, numbers, spaces, and hyphens.
  * Convert CamelCase type names to hyphen-separated words if necessary.
  * Prefer hyphen separated words over space separated words in test names.
* Create helpers functions to simplify complex test setups.
  * Mark helper functions with `t.Helper()`. if called from the test
    function.
  * Also prefer to setup helper functions for setting up mocks using the
    patterns supported by `github.com/tkrop/go-testing/mock`.
* Test cases should be ordered into blocks of similar tests cases that are
  separated by empty lines and comments.
  * Simple tests cases with less setup should be ordered before test cases
    with extensive setup.
* The test cases are supposed to aim for 100% code coverage:
  * Test cases should always cover happy paths as well as error paths.
  * Test cases for `any` should include following test cases in order:
    => nil, primitive (bool, int), string, slice, map, empty struct,
    random structs, pointer to primitive, pointer to struct, function,
    channel.
  * If the tested function is based on reflect, also aliases and private
    fields in structs must be tested.
  * Use `test.Cast` to cast `any` to specific types.
* Parameters setup of test cases should be supported by top level setup
  functions if the setup of a parameter gets to complicated.
* If test cases are to long they should also be separated by empty lines.
* Construct the actual expected objects and errors instead of aiming to
  split them up and compare them by properties.


### Test function

* Code should be self-explanatory and use top level support functions for
  setup and execution when helpful.
* Use `test.Map(t,cases).Run[Seq](t,param)` for parameterized tests:
  * Use `Run(t,param)` for test cases implicitly running in parallel —
    remove `t.Parallel()` when refactoring using this.
  * Use `RunSeq(t,param)` for test cases that cannot run in parallel —
    use this in refactorings when no `t.Parallel()` was defined.
  * Break the line before `Run[Seq](t,param)` to improve readability and
    consistency.
* Mark blocks with empty line and `// Given` (setup), `// When` (execution),
  `// Then` (validation) comments.
* Use `github.com/tkrop/go-testing/mock` to setup and configure mocks.
* Do not add extra comments for explanation - also not on the block markers.
* Use `github.com/stretchr/testify/require` to validate setup values.
* Use `github.com/stretchr/testify/{assert,require}` to validate setup values.
* Use minimal assertions needed to validate expected values and errors.
  * Always test against the actual instances using `assert.Equal` - espacially
    errors must be tested this way.
  * Prefer exact matches in assertions above partial or type validation.
* Use `assert.AnError` for generic error scenarios.
* Do never comments on assertions; testify is providing meaningful context
  information.


### Test execution

* Always test for race conditions using `-race`.
* When investigating test coverage please use the following stub commands:
  * `go test -coverprofile=build/test-agent.cover ...` for generating, and
  * `go tools -func=build/test-agent.cover` for accessing the coverage results.
  * Similar use `build/test-agent.cover.html` for accessing the html page.
