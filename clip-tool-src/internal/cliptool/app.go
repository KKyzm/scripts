package cliptool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var ErrUserAborted = errors.New("user aborted")

type Result struct {
	Stdout string
	Notice string
}

func Run(ctx context.Context) (Result, error) {
	clipboard := NewShellClipboard()
	runner := NewShellRunner()

	commands, warning, err := LoadCommands()
	if err != nil {
		return Result{}, err
	}

	text, err := clipboard.ReadText(ctx)
	if err != nil {
		return Result{}, err
	}
	if text == "" {
		return Result{}, fmt.Errorf("clipboard is empty or does not contain plain text")
	}

	model := NewModel(commands, text, warning, clipboard, runner)
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	finalModel, err := program.Run()
	if err != nil {
		return Result{}, err
	}

	m, ok := finalModel.(*Model)
	if !ok {
		return Result{}, fmt.Errorf("unexpected final model type %T", finalModel)
	}
	if m.userAborted {
		return Result{}, ErrUserAborted
	}
	return m.result, nil
}

func mustGetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func writeAll(stdin io.WriteCloser, input string) error {
	defer stdin.Close()
	_, err := io.WriteString(stdin, input)
	return err
}
