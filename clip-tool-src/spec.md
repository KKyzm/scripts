# Design Spec: Clipboard TUI Processor (`clip-tool`)

## Overview
`clip-tool` is a terminal UI that lets users inspect, preview, edit, and execute text-processing commands against clipboard content. It acts as a bridge between the system clipboard and shell pipelines, with fast preview behavior, editable per-command results, and a workflow that keeps intermediate edits visible while switching between commands.

The implementation now lives under `clip-tool-src/`, builds its binary into `clip-tool-src/bin/clip-tool`, and exposes a root-level symlink `./clip-tool` for convenient invocation.

## Goals & Success Criteria
- **Interactive Preview:** Provide a split-pane TUI for browsing built-in and custom commands.
- **Fast First Paint:** Show the TUI immediately and avoid unnecessary startup debounce for the initial preview.
- **Large-Text Safety:** Use truncated preview input for responsiveness, while keeping full-text execution semantics for actual editing and final actions.
- **Editable Workflow:** Let users edit full command results in an external editor, persist those edits per command, and reuse them across navigation.
- **Composable Transform Pipeline:** Allow edited `identity` output to become the effective input for downstream commands.
- **Convenient Packaging:** Keep source code isolated under its own directory and provide a stable root-level executable entry via symlink.

## Non-Goals (Out of Scope)
- Clipboard history management.
- Rich text, images, files, or binary clipboard formats.
- Interactive shell subprocesses inside the TUI.
- Fully in-place text editing inside the TUI.

## Repository Layout
- `clip-tool-src/go.mod`: Go module root.
- `clip-tool-src/cmd/clip-tool/main.go`: CLI entrypoint.
- `clip-tool-src/internal/cliptool/`: Application logic.
- `clip-tool-src/bin/clip-tool`: Built binary output.
- `clip-tool-src/build.sh`: One-command build script that rebuilds the binary and refreshes the root symlink.
- `./clip-tool`: Symlink to `clip-tool-src/bin/clip-tool`.

## Architecture & Data Flow

### 1. Input Stage
- On startup, read the full clipboard text into memory.
- Derive a preview input by truncating the effective input to the first `10KB`.
- The effective input is normally the raw clipboard text.
- If the user edits the `identity` command, its edited full result becomes the new effective input for all downstream commands.

### 2. Config & Built-In Commands
- Load commands from config if present.
- Fall back to built-in commands if config is missing or malformed.
- Keep `identity` as the first built-in command in the left pane.
- Include common text transforms such as removing blank lines, trimming spaces, sorting, numbering lines, and case conversion.

### 3. Preview Stage
- Left pane selection drives preview generation.
- Preview normally runs the selected command against the truncated effective input.
- The initial preview is triggered immediately at startup.
- Subsequent selection/search changes are debounced to reduce unnecessary process churn.
- Stale preview processes are canceled before new ones begin.

### 4. Edit Stage
- Pressing `e` always prepares the **full** output for the currently selected command.
- That full output is opened in the user’s external editor.
- Editor selection order:
  - `$VISUAL`
  - `$EDITOR`
  - fallback to `vi`, `vim`, or `nvim`
- When the editor exits, the edited content is cached per command.
- Cached edited content survives navigation: switching away and back restores that command’s edited state.

### 5. Edited State & Derived Input
- Each command can have its own edited result.
- Commands with edited content are marked in the left pane using an orange `(*)`.
- If the current command has edited content:
  - preview can show either a diff or the full edited text.
  - final execution uses the edited content directly.
- If `identity` has edited content, that content is used as the effective upstream input for previewing or executing other commands.

### 6. Output Stage
- `Enter` / `s`: copy the final result to the clipboard.
- `o`: print the final result to stdout.
- If edited content exists for the selected command, use that edited content directly.
- Otherwise, execute the selected command against the full effective input.

## UI/UX Design

### Layout
- **Left Pane (~40%)**
  - Mode-aware input box at the top.
  - Command list below.
  - Edited commands show an orange `(*)` suffix.
- **Divider**
  - A single vertical separator line between panes.
- **Right Pane (~60%)**
  - Header with raw shell command.
  - Preview title indicating normal preview, edited diff, or edited full mode.
  - Scrollable preview body.

### Input Modes
- **Default mode**
  - Normal list navigation and actions.
- **Search mode** (`/`)
  - Typed characters filter command aliases.
  - `↑`/`↓` or `j`/`k` still navigate filtered results.
  - `Enter` executes the selected result.
- **Custom command mode** (`!`)
  - Input is treated as a shell command.
  - Empty custom command shows an instructional placeholder instead of running anything.

### Keybindings
- `↑`/`↓` or `j`/`k`: move between commands.
- `/`: enter search mode.
- `!`: enter custom command mode.
- `Enter` or `s`: execute and copy to clipboard.
- `o`: execute and write to stdout.
- `e`: edit the selected command’s **full** output in an external editor.
- `r`: reset the selected command’s edited content.
- `Tab`: toggle between edited diff view and edited full-text view.
- `Esc` or `Ctrl+C`: exit or leave input mode.

## Diff View Behavior
- Edited previews support two render modes:
  - **Diff mode**
  - **Edited full-text mode**
- Diff mode is line-based and colorized:
  - deleted lines in red
  - inserted lines in green
  - unchanged lines in dim gray
- Blank-line changes are shown explicitly using `␤` so they are visible in terminal output.
- Missing trailing newline conditions are rendered as metadata lines, similar to Git-style notices.

## Error Handling & Edge Cases
- **Empty / unsupported clipboard:** fail early with a clear message.
- **Malformed config:** fall back to built-ins and surface a warning state.
- **Missing command dependency:** display stderr in the preview pane rather than crashing.
- **Command execution failure:** keep the TUI alive and surface the error in status/preview.
- **Editor unavailable:** show a clear message asking the user to set `$VISUAL` or `$EDITOR`.
- **Empty custom command:** do not execute; show instructional preview instead.
- **Long-running execution:** show spinner and allow cancellation via `Ctrl+C`.

## Testability
- Keep clipboard operations behind `ClipboardProvider`.
- Keep shell execution behind `ShellRunner`.
- Unit test config parsing, diff edge cases, runner behavior, and effective-input logic.
- Explicitly cover:
  - `identity` remains first in built-ins.
  - blank-line diff rendering.
  - trailing newline metadata rendering.
  - edited `identity` content becoming the effective input for downstream commands.

## Current Implementation Notes
- The tool is implemented with the Bubble Tea ecosystem.
- Preview truncation is currently `10KB`.
- Edited command state is keyed per command, including custom commands.
- Root invocation is expected to happen via the symlinked `./clip-tool` binary after running `./clip-tool-src/build.sh`.
