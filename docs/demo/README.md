# README CLI demo

[cli.yaml](cli.yaml) generates `cli.gif` with
[YAMLCast](https://github.com/ingydotnet/yamlcast).
The cast copies [sample.yaml](sample.yaml) into an ignored working directory
and runs the real `go-yaml` binary built from this checkout.
The recording uses YAMLCast revision
`0dfec58fc184e97b21239d0b54dc222bd8311cf1` with YAMLScript 0.2.32 and VHS 0.11.0.
Setup is hidden; the recorded commands match the root README.

## Regenerate

Run from the go-yaml repository root.
You need Bash, Git, GNU Make, curl, FFmpeg, and a working Chromium browser.
YAMLCast uses Makes to install its Go, YAMLScript, VHS, and ttyd dependencies
under its own `.cache/` directory.
The first recording may also download a browser.

Build the CLI, install the pinned recording tools, and render the image:

```bash
make update
```

`make update` rebuilds the CLI and regenerates `docs/demo/cli.gif` each time.
The root README displays this file as an inline animation.
To install the recording tools without rendering:

```bash
make yamlcast
```

The Makefile downloads YAMLCast into `.cache/local/yamlcast-<revision>`.
Its dependencies live under that directory's `.cache/`.
It reuses this project's pinned Makes implementation.
Ordinary build and test targets do not install the recording tools.
The pinned YAMLCast revision uses filesystem helpers from YAMLScript 0.2.32.
The installation rule selects that runtime automatically.

The recording command uses `/tmp` for temporary files.
Makes normally uses a project-local temporary directory, but in a deeply
nested worktree that makes Chrome's Unix socket path exceed the system limit.
Installed tools remain under `.cache/`.

Rendering needs permission to execute Chromium and start a local ttyd server.
If your agent sandbox blocks browser execution, run `make update` from your
regular terminal.
Rendering failures cause `make update` to fail.

The GIF settings are 1200 by 700 pixels, with 22-pixel text and a dark
background.
Keep it below 5 MiB and check that every command and result is readable.
The root README includes descriptive image text and equivalent commands so
readers do not need to watch the animation.
