package cliptool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type ClipboardProvider interface {
	ReadText(ctx context.Context) (string, error)
	WriteText(ctx context.Context, text string) error
}

type shellClipboard struct{}

func NewShellClipboard() ClipboardProvider {
	return shellClipboard{}
}

func (shellClipboard) ReadText(ctx context.Context) (string, error) {
	cmdArgs, err := readClipboardCommand()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("read clipboard: %s", strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("read clipboard: %w", err)
	}

	text := stdout.String()
	if text == "" {
		return "", fmt.Errorf("clipboard is empty or does not contain plain text")
	}
	return text, nil
}

func (shellClipboard) WriteText(ctx context.Context, text string) error {
	cmdArgs, err := writeClipboardCommand()
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open clipboard stdin: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start clipboard writer: %w", err)
	}
	if err := writeAll(stdin, text); err != nil {
		return fmt.Errorf("write clipboard stdin: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("write clipboard: %s", strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("write clipboard: %w", err)
	}

	return nil
}

func readClipboardCommand() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		return []string{"pbpaste"}, nil
	case "linux":
		for _, candidate := range [][]string{{"wl-paste", "--no-newline"}, {"xclip", "-selection", "clipboard", "-out"}, {"xsel", "--clipboard", "--output"}} {
			if _, err := exec.LookPath(candidate[0]); err == nil {
				return candidate, nil
			}
		}
		return nil, fmt.Errorf("no supported clipboard reader found (expected one of wl-paste, xclip, xsel)")
	case "windows":
		return []string{"powershell", "-NoProfile", "-Command", "Get-Clipboard"}, nil
	default:
		return nil, fmt.Errorf("unsupported platform %q", runtime.GOOS)
	}
}

func writeClipboardCommand() ([]string, error) {
	switch runtime.GOOS {
	case "darwin":
		return []string{"pbcopy"}, nil
	case "linux":
		for _, candidate := range [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard", "-in"}, {"xsel", "--clipboard", "--input"}} {
			if _, err := exec.LookPath(candidate[0]); err == nil {
				return candidate, nil
			}
		}
		return nil, fmt.Errorf("no supported clipboard writer found (expected one of wl-copy, xclip, xsel)")
	case "windows":
		return []string{"cmd", "/c", "clip"}, nil
	default:
		return nil, fmt.Errorf("unsupported platform %q", runtime.GOOS)
	}
}
