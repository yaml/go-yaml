**Please show behavior changes with a `go-yaml` CLI example before and after
this patch.**
Read [Reporting with the CLI](https://github.com/yaml/go-yaml/blob/main/docs/reporting.md)
for examples and comparison instructions.
Build each revision with `make cli` and run `./go-yaml` to test that checkout.
For Go-specific behavior, include a runnable Go reproducer too.
Changes without YAML behavior do not need a CLI example.

**We strongly encourage tests in the existing YAML `testdata/` style.**
Turn the reproduction into a fixture case where possible.
See [Writing testdata cases][testdata-guide] for fixture examples and guidance,
and [Stage fixture format and focused tests][stage-tests] for internal tests.

[testdata-guide]: https://github.com/yaml/go-yaml/blob/main/CONTRIBUTING.md#testing
[stage-tests]: https://github.com/yaml/go-yaml/blob/main/internal/libyaml/README.md#test-layout

## What changes and why

<!-- Describe the problem, resulting behavior, and related issues. -->

## CLI input and command

<!-- Paste minimal input and the exact command as fenced text.
Include options, configuration contents, and plugin build/selection flags.
If the CLI cannot demonstrate the change, explain why and provide relevant
Go code instead.
-->

## Before

<!-- Identify the base revision and paste its output or error. -->

## After

<!-- Identify the patched revision and paste output for the same example.
Explain why the new result is correct.
-->

## Validation

<!-- List testdata/ cases added or updated and checks run, including results.
If you used a Go test instead, explain what the fixture harness cannot express.
-->
