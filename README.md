# CLI Tools

This repository is a small collection of custom command-line tools.

## Tools

- `remote`: opens the current Git repository's remote URL.
- `clip-tool`: a clipboard-driven TUI for previewing, editing, and applying text-processing commands.

## Layout

- `remote`: standalone shell script at the repository root.
- `clip-tool-src/`: source code for `clip-tool`.
- `clip-tool`: symlink to the built binary at `clip-tool-src/bin/clip-tool`.

## Usage

```bash
./remote
./clip-tool
```

## Build

To rebuild `clip-tool`:

```bash
./clip-tool-src/build.sh
```

This rebuilds the binary and refreshes the root-level symlink.

## Dependencies

- `remote`: requires `git` and `gum`.
- `clip-tool`: requires Go to build, plus standard clipboard utilities on the target platform.
  - macOS: `pbcopy` / `pbpaste`
  - Linux: `wl-copy` / `wl-paste`, or `xclip`, or `xsel`
  - Windows: `clip` / `Get-Clipboard`
- For `clip-tool` editing mode, set `$VISUAL` or `$EDITOR` for the best experience.

## Convention

Each tool keeps a stable entrypoint at the repository root. Larger tools may keep their source code in their own subdirectory.
