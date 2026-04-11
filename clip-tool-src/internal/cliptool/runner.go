package cliptool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type ShellRunner interface {
	Run(ctx context.Context, command string, input string) (ExecutionResult, error)
}

type ExecutionResult struct {
	Stdout string
	Stderr string
}

type shellRunner struct{}

func NewShellRunner() ShellRunner {
	return shellRunner{}
}

func (shellRunner) Run(ctx context.Context, command string, input string) (ExecutionResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return ExecutionResult{}, fmt.Errorf("shell command is empty")
	}

	cmd := exec.CommandContext(ctx, shellCommand(command)[0], shellCommand(command)[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("open shell stdin: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return ExecutionResult{}, fmt.Errorf("start shell command: %w", err)
	}
	if err := writeAll(stdin, input); err != nil {
		return ExecutionResult{}, fmt.Errorf("write shell stdin: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ExecutionResult{}, ctx.Err()
		}
		return ExecutionResult{Stdout: stdout.String(), Stderr: stderr.String()}, err
	}

	return ExecutionResult{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func shellCommand(command string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", command}
	}
	return []string{"sh", "-c", command}
}
