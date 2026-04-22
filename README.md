# CLI Tools

This repository is a small collection of custom command-line tools.

## Tools

- `remote`: opens the current Git repository's remote URL.
- `clip-tool`: a clipboard-driven TUI for previewing, editing, and applying text-processing commands.

## Layout

- `remote-src/`: source code for `remote`.
- `clip-tool-src/`: source code for `clip-tool`.
- `bin/`: symlinks to all tool entrypoints.

## Usage

```bash
./bin/remote
./bin/clip-tool
```

## Build

To build all tools and create symlinks:

```bash
./build.sh
```

To rebuild a single tool:

```bash
./clip-tool-src/build.sh  # rebuilds clip-tool only
```

## Dependencies

- `remote`: requires `git` and `gum`.
- `clip-tool`: requires Go to build, plus standard clipboard utilities on the target platform.
  - macOS: `pbcopy` / `pbpaste`
  - Linux: `wl-copy` / `wl-paste`, or `xclip`, or `xsel`
  - Windows: `clip` / `Get-Clipboard`
- For `clip-tool` editing mode, set `$VISUAL` or `$EDITOR` for the best experience.

## Convention

Each tool keeps a stable entrypoint in the `bin/` directory. Larger tools may keep their source code in their own subdirectory.
