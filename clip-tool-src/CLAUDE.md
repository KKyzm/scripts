# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build clip-tool (creates binary at bin/clip-tool and updates root symlink)
./build.sh

# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/cliptool/...

# Run a single test
go test -run TestShellRunnerRun ./internal/cliptool/
```

## Architecture Overview

This is a Go TUI application built with the Bubble Tea framework (charmbracelet/bubbletea). The codebase follows standard Go project layout:

- `cmd/clip-tool/main.go` - Thin CLI entrypoint that delegates to `cliptool.Run()`
- `internal/cliptool/` - All application logic

### Key Components (internal/cliptool/)

| File | Purpose |
|------|---------|
| `app.go` | Entry point `Run()` function; orchestrates initialization and tea.Program |
| `model.go` | Bubble Tea Model with all state, Update/View logic, message handling |
| `config.go` | Command loading from XDG config paths; YAML/JSON parsing; default commands |
| `runner.go` | `ShellRunner` interface for executing shell commands with stdin |
| `clipboard.go` | `ClipboardProvider` interface; platform-specific clipboard operations |
| `diff.go` | Line-based diff rendering for edited content preview |
| `editor.go` | External editor integration ($VISUAL > $EDITOR > vi/vim/nvim) |

### Design Patterns

- **Interfaces for external dependencies**: `ClipboardProvider` and `ShellRunner` are interfaces, enabling unit testing without real clipboard/shell operations
- **Bubble Tea architecture**: The Model handles all state mutations via `Update()` and renders via `View()`. Commands are async operations that return messages
- **Preview truncation**: Commands run against truncated input (10KB) for responsiveness, but full input is used for actual execution/editing
- **Edited state tracking**: `identity` command edits become the "effective input" for downstream commands, enabling composable transform pipelines

### Config File Location

Commands are loaded from `$XDG_CONFIG_HOME/clip-tool/commands.{yaml|json}` (falls back to `~/.config/clip-tool/` if XDG_CONFIG_HOME is unset). Malformed configs fall back to built-in commands with a warning.
