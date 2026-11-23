**Please use the [`go-yaml`][cli] CLI instead of Go code to demonstrate YAML
behavior, when possible.**

[cli]: https://github.com/yaml/go-yaml/blob/main/cmd/go-yaml/README.md

Read [Reporting with the CLI](https://github.com/yaml/go-yaml/blob/main/docs/reporting.md)
for installation and examples.
Include a runnable Go program if the behavior depends on Go types, struct tags,
or custom marshalers, or cannot be reproduced with the CLI.
For questions and proposals, show current and desired behavior where possible.

## Problem or question

<!-- Explain what you are trying to do and what goes wrong or should change. -->

## CLI reproduction

<!-- Paste minimal input and the exact command as fenced text.
Example:

```bash
printf '%s\n' 'a: b' | go-yaml -y
```

Include any configuration file contents and plugin build/selection flags.
If a CLI example does not apply, explain why and provide relevant code instead.
-->

## Actual output

<!-- Paste output and errors as text, preserving whitespace. -->

## Expected output

<!-- Show the expected result and explain why. -->

## Version and environment

<!-- Include:
- CLI version from go-yaml -h
- go version and operating system
- go version -m "$(command -v go-yaml)" for an installed CLI
- Checkout commit and local changes for a build made with make cli
-->
