package cliptool

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func renderLineDiff(original string, edited string) string {
	if original == edited {
		return "(no changes)"
	}

	dmp := diffmatchpatch.New()
	a, b, lines := dmp.DiffLinesToChars(original, edited)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)

	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	deleteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	insertStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	equalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var builder strings.Builder
	builder.WriteString(headerStyle.Render("--- original"))
	builder.WriteByte('\n')
	builder.WriteString(headerStyle.Render("+++ edited"))
	builder.WriteByte('\n')
	for _, diff := range diffs {
		prefix := "  "
		style := equalStyle
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			prefix = "- "
			style = deleteStyle
		case diffmatchpatch.DiffInsert:
			prefix = "+ "
			style = insertStyle
		}
		for _, line := range splitLines(diff.Text) {
			builder.WriteString(style.Render(prefix + displayDiffLine(line)))
			builder.WriteByte('\n')
		}
	}
	if original != "" && !strings.HasSuffix(original, "\n") {
		builder.WriteString(metaStyle.Render(`\ original has no trailing newline`))
		builder.WriteByte('\n')
	}
	if edited != "" && !strings.HasSuffix(edited, "\n") {
		builder.WriteString(metaStyle.Render(`\ edited has no trailing newline`))
		builder.WriteByte('\n')
	}
	return strings.TrimRight(builder.String(), "\n")
}

func displayDiffLine(line string) string {
	if line == "" {
		return "␤"
	}
	return line
}

func splitLines(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.SplitAfter(value, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		lines = append(lines, strings.TrimSuffix(part, "\n"))
	}
	return lines
}
