package cliptool

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShellRunnerRun(t *testing.T) {
	runner := NewShellRunner()
	result, err := runner.Run(context.Background(), "tr '[:lower:]' '[:upper:]'", "hello")
	if err != nil {
		t.Fatalf("runner.Run returned error: %v", err)
	}
	if result.Stdout != "HELLO" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestShellRunnerRunContextCancel(t *testing.T) {
	runner := NewShellRunner()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := runner.Run(ctx, "sleep 1", "")
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
