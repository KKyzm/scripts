package cliptool

import (
	"strings"
	"testing"
)

func TestRenderLineDiff(t *testing.T) {
	diff := renderLineDiff("alpha\nbeta\n", "alpha\ngamma\n")
	if !strings.Contains(diff, "--- original") || !strings.Contains(diff, "+++ edited") {
		t.Fatalf("expected headers in diff, got %q", diff)
	}
	if !strings.Contains(diff, "- beta") {
		t.Fatalf("expected removed line in diff, got %q", diff)
	}
	if !strings.Contains(diff, "+ gamma") {
		t.Fatalf("expected added line in diff, got %q", diff)
	}
}

func TestRenderLineDiffShowsBlankLineChanges(t *testing.T) {
	diff := renderLineDiff("alpha\n\n", "alpha\n")
	if !strings.Contains(diff, "- ␤") {
		t.Fatalf("expected blank line marker in diff, got %q", diff)
	}
}

func TestRenderLineDiffShowsTrailingNewlineMetadata(t *testing.T) {
	diff := renderLineDiff("alpha\nbeta", "alpha\nbeta\n")
	if !strings.Contains(diff, `\ original has no trailing newline`) {
		t.Fatalf("expected trailing newline metadata, got %q", diff)
	}
}
