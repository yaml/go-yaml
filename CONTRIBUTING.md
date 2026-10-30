Contributing to go-yaml
=======================

Thank you for your interest in contributing to go-yaml!

This document provides guidelines and instructions for contributing to this
project.


## Code of Conduct

By participating in this project, you agree to follow our Code of Conduct.

We expect all contributors to:

- Be respectful and inclusive
- Use welcoming and inclusive language
- Be collaborative and constructive
- Focus on what is best for both the Go and YAML communities


## How to Contribute


### Reporting Issues

Before submitting an issue, please:

- Check if the issue already exists in our issue tracker
- Use a clear and descriptive title
- Provide detailed steps to reproduce the issue
- Include relevant code samples and error messages
- Specify your Go version and operating system
- Use the `go-yaml` CLI tool described below


### Using the `go-yaml` CLI Tool

This tool can be used to inspect both the internal stages and final results of
YAML processing with the go-yaml library.
It should be used when reporting most bugs.

The `go-yaml` CLI tool uses the `go.yaml.in/yaml/v4` library to decode and
encode YAML.
Decoding YAML is a multi-stage process that involves tokens, events, and nodes.
The `go-yaml` CLI tool lets you see all of these intermediate stages of the
decoding process.
This is crucial for understanding what go-yaml is doing internally.

The `go-yaml` CLI tool can be built with the `make go-yaml` command or installed
with the `go install go.yaml.in/yaml/v4/cmd/go-yaml@latest` command.

You can learn about all of its options with the `go-yaml -h` command.

Here is an example of using it on a small piece of YAML:

```bash
./go-yaml -t <<< '
foo: &a1 bar
*a1: baz
```


### Coding Conventions

- Follow standard Go coding conventions
- Use `make fmt` to format your code
- Write descriptive comments for non-obvious code
- Add tests for your work
- Keep line length to 80 characters
- Use meaningful variable and function names
- Start doc and comment sentences on a new line
- Test your changes with the `go-yaml` CLI tool when working on parsing logic


### Commit Conventions

- No merge commits
- Commit subject line should:
  - Start with a capital letter
  - Not end with a period
  - Be no more than 50 characters


### Pull Requests

1. Fork the repository
1. Create a new branch for your changes
1. Make your changes following our coding conventions
   - If you are not sure about the coding conventions, please ask
   - Look at the existing code for examples
1. Write clear commit messages
1. Update tests and documentation
1. Submit a pull request


### Testing

- Ensure all tests pass with `make test`
- Add new tests for new functionality
- Update existing tests when modifying functionality

### Downstream Integration Testing

Run `make test-integration` to assess downstream upgrades against this checkout,
including uncommitted library changes.
Select one project with `make test-integration-yq`, or several with
`make test-integration INTEGRATION-PROJECT=yq,dasel`.
On a terminal, the default display scrolls the current check's last 10 output
lines in place and updates its elapsed time while it runs.
Each completed check leaves a colored status line with its runtime and exit
code.
Each project's run is enclosed in start and finish banners.
Every line of the finish block starts with `> `, including the overall status,
so `grep '^> '` extracts a condensed report from captured output.
Set `NO_COLOR=1` to disable colors.
Redirected output uses plain start and completion lines without cursor controls.
Use `v=1` to stream full output and request verbose Go tests.
Full output is always preserved in the log files.
Integration tests run separately from `make test`, with one CI job per enabled
project.
In GitHub Actions, manually run the **Integration Test** workflow and select the
branch to test.

`testdata/integration/projects.yaml` is the project catalog and report source.
Each top-level key identifies a project, with `repo`, `vref`, and informational
metadata such as `star` and `time` beneath it.
Set `skip: true` on a project to exclude it from default runs and the CI matrix.
Explicit selection with `INTEGRATION-PROJECT` or `make test-integration-<project>`
runs the project even when `skip: true` is set.
It remains visible in listings and the generated report.
Run `make integration-list` to list all projects and their enabled/skipped state.
Run `make integration-report` to regenerate `go-yaml-v4-projects.md`.
Makes supplies the Go toolchain for these commands.

Each run resolves the configured upstream revision and prints its commit SHA.
Moving revisions are fetched each time; full SHAs reuse cached source.
To reproduce a run:

```bash
make test-integration-yq INTEGRATION-REF=<full-commit-sha>
```

`INTEGRATION-REF` requires one selected project and also accepts upstream branches
or tags.
`INTEGRATION-CONFIG` overrides the catalog path.
`INTEGRATION-TIMEOUT` defaults to `30m` per baseline/candidate phase.
The default Go version comes from Makes; use `GO-VERSION` when an upstream
revision needs a newer toolchain.

Managed sources and logs live under `.cache/integration/`.
The runner creates disposable, independent baseline and candidate clones at the
same commit, leaving `repos/` untouched.
The baseline uses the downstream's existing dependency.
A temporary Go workspace overrides that dependency with this checkout for the
candidate, and the runner verifies module resolution.
Both phases run even if the baseline fails.

Project scripts live at `util/integration/<project>.bash`.
Each accepts `prepare` or `test`, followed by its disposable source directory.
The runner supplies `INTEGRATION_ROOT`, `INTEGRATION_PROJECT`,
`INTEGRATION_PHASE`, `INTEGRATION_LOG_DIR`, `INTEGRATION_RESULTS`, and
`INTEGRATION_COMPLETION`.
Scripts source `common.bash`, use `check` for named commands, and call `finish`
after reporting every check.
Commands run in strict Bash shells, and failures do not stop later checks.

Scripts own upstream setup, tool provisioning, environment settings, and test
commands.
The shared `makes` helper installs a selected module's target and imports its
exported tool environment.
For example, `makes node.mk NODE` supplies Node and npm.
An empty module selects a project rule, such as
`makes '' GOLANGCI-LINT`.
Call provisioning from both actions when its exported environment is needed.
Tool installation failures are errors.
Use `check-if` only for unavailable host capabilities, such as a Docker daemon.
Use `pending` to report CI validation whose local setup is not yet implemented;
these entries remain visible as `setup-needed` and do not fail the run.
The scripts retain the upstream validation inventory, including unfinished
container, VM, security, frontend, and documentation checks.
They do not invoke publication or deployment workflows.

Each run prints a results directory containing `results.json`, `summary.md`,
and per-project baseline/candidate logs and named check results.
JSON and Markdown include check runtimes and exit codes; checks that did not
execute show no exit code.
CI also includes the summary in the job report and saves those files as an
artifact.
Results distinguish:

- `upgrade-blocked`: a named check passes its baseline and fails the candidate.
- `baseline-failed`: existing-dependency checks already fail, so those checks
  cannot establish upgrade compatibility.
- `setup-error` or `candidate-failed`: setup or execution needs investigation.
- `partial`: executed checks pass, with skipped host capability checks or
  unfinished CI setup.
- `no-coverage`: no checks executed.
- `skipped`: excluded by the catalog.

Executed failures and setup errors return a nonzero exit status.
Capability skips and pending CI setup do not fail the aggregate, but remain
visible.
A zero-coverage selection is explicitly reported and does not establish
compatibility.
Passing available checks alone does not imply all upstream CI is covered.

Reviewed yq test accommodations remain in `testdata/integration/yq/*.patch`.
The yq script applies them equally to both disposable clones.
The current patch corrects a stale malformed-JSON error expectation that also
fails with the existing dependency.
It does not accommodate cross-document alias behavior; those failures remain
upgrade blockers.

Run `make test-integration-runner` and `make test-shell` when changing the
integration harness or project scripts.


## Development Process

- This project makes use of a GNU makefile (`GNUmakefile`) for many dev tasks
- The makefile doesn't use your locally installed Go commands; it auto-installs
  them, so that all results are deterministic
- Fork and clone the repository
- Make your changes
- Run tests, linters and formatters
  - `make fmt`
  - `make tidy`
  - `make lint`
  - `make lint-yaml`
  - `make test`
  - You can use `make check` to run all of the above
- Submit a [Pull Request](https://github.com/yaml/go-yaml/pulls)


### Using Your Own Go with the Makefile

We ask that you always test with the makefile installed Go before committing,
since it is deterministic and uses the exact same flow as the go-yaml CI.

We also realize that many Go devs need to run their locally installed Go
commands for their development environment, and might want to use them with
the go-yaml makefile.

If you need to use your own Go utils with the makefile, set `GO_YAML_PATH` to
the directory(s) containing them (either by exporting it or passing it to
`make`).

Something like this:

```bash
export GO_YAML_PATH=$(dirname "$(command -v go)")
make test
# or
make test GO_YAML_PATH=$(dirname "$(command -v go)")
```

**Note:** `GO-VERSION` and `GO_YAML_PATH` are mutually exclusive.
When `GO_YAML_PATH` is set, the makefile uses your own Go environment and
ignores any `GO-VERSION` setting.


### Using the Makefile Environment as a Shell

Sometimes you might want to run your own shell commands using the same binaries
that the makefile installs.

To get a subshell with this environment, run one of:

```bash
make shell
make bash
make zsh
make shell GO-VERSION=1.23.4
```



## Makefile Targets

The repository's makefile (`GNUmakefile`) provides a number of useful targets:

- `make test` runs all tests including yaml-test-suite tests
- `make test-unit` runs just the unit tests
- `make test-internal` runs just the internal tests
- `make test-yts` runs just the yaml-test-suite tests
- `make test v=1 count=3` runs the tests with options
- `make test GO-VERSION=1.23.4` runs the tests with a specific Go version
- `make test GO_YAML_PATH=/path/to/go/bin` uses your own Go installation
- `make shell` opens a shell with the project's dependencies set up
- `make shell GO-VERSION=1.23.4` opens a shell with a specific Go version
- `make fmt` runs `golangci-lint fmt ./...`
- `make lint` runs `golangci-lint run`
- `make lint-yaml` runs a pinned yamllint using uv from Makes
- `make tidy` runs `go mod tidy`
- `make distclean` cleans the project completely


## Getting Help

If you need help, you can:
- Open an [issue](https://github.com/yaml/go-yaml/issues) with your question
- Start a [discussion](https://github.com/yaml/go-yaml/discussions)
- Read through our [documentation](https://pkg.go.dev/go.yaml.in/yaml/v4)
- Check the [migration guide](docs/v3-to-v4-migration.md) if upgrading from v3
- Join our [Slack channel](https://cloud-native.slack.com/archives/C08PPAT8PS7)


## We are a Work in Progress

This project is very much a team effort.
We are just getting things rolling and trying to get the foundations in place.
There are lots of opinions and ideas about how to do things, even within the
core team.

Once our process is more mature, we will likely change the rules here.
We'll make the new rules as a team.
For now, please stick to the rules as they are.

This project is focused on serving the needs of both the Go and YAML
communities.
Sometimes those needs can be in conflict, but we'll try to find common ground.


## Thank You

Thank you for contributing to go-yaml!
