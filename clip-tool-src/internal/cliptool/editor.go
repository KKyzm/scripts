package cliptool

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func openEditorCmd(key string, original string, initial string) tea.Cmd {
	file, err := os.CreateTemp("", "clip-tool-*.txt")
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{Key: key, Err: fmt.Errorf("create temp file: %w", err)} }
	}
	path := file.Name()
	if _, err := file.WriteString(initial); err != nil {
		file.Close()
		os.Remove(path)
		return func() tea.Msg { return editorFinishedMsg{Key: key, Err: fmt.Errorf("write temp file: %w", err)} }
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return func() tea.Msg { return editorFinishedMsg{Key: key, Err: fmt.Errorf("close temp file: %w", err)} }
	}

	command, err := buildEditorProcess(path)
	if err != nil {
		os.Remove(path)
		return func() tea.Msg { return editorFinishedMsg{Key: key, Err: err} }
	}

	return tea.ExecProcess(command, func(execErr error) tea.Msg {
		defer os.Remove(path)
		if execErr != nil {
			return editorFinishedMsg{Key: key, Err: fmt.Errorf("editor exited with error: %w", execErr)}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return editorFinishedMsg{Key: key, Err: fmt.Errorf("read edited file: %w", err)}
		}
		return editorFinishedMsg{Key: key, Original: original, Edited: string(data)}
	})
}

func buildEditorProcess(path string) (*exec.Cmd, error) {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		for _, candidate := range []string{"vi", "vim", "nvim"} {
			if _, err := exec.LookPath(candidate); err == nil {
				editor = candidate
				break
			}
		}
	}
	if editor == "" {
		return nil, fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}

	command := fmt.Sprintf("%s %s", editor, shellQuote(path))
	args := shellCommand(command)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, `"`, `\"`))
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
