package cliptool

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

const (
	previewInputLimit  = 10 * 1024
	previewOutputLimit = 40 * 1024
	previewDebounce    = 80 * time.Millisecond
)

type inputMode int

const (
	inputInactive inputMode = iota
	inputSearch
	inputCustom
)

type actionMode int

const (
	actionClipboard actionMode = iota
	actionStdout
)

type editedViewMode int

const (
	editedViewDiff editedViewMode = iota
	editedViewFull
)

type entry struct {
	Label   string
	Command string
	Custom  bool
}

type previewTriggerMsg struct{ Seq int }

type previewResultMsg struct {
	Seq     int
	Output  string
	Error   string
	Failed  bool
	Command string
}

type execResultMsg struct {
	Mode   actionMode
	Output string
	Err    error
}

type editPreparedMsg struct {
	Key      string
	Original string
	Err      error
}

type editorFinishedMsg struct {
	Key      string
	Original string
	Edited   string
	Err      error
}

type editedContent struct {
	Original string
	Edited   string
}

type Model struct {
	commands       []Command
	filtered       []entry
	selected       int
	inputMode      inputMode
	search         textinput.Model
	preview        viewport.Model
	spinner        spinner.Model
	clipboard      ClipboardProvider
	runner         ShellRunner
	clipboardText  string
	previewInput   string
	previewSeq     int
	previewCancel  context.CancelFunc
	execCancel     context.CancelFunc
	previewLoading bool
	executing      bool
	previewFailed  bool
	previewBody    string
	previewCommand string
	edited         map[string]editedContent
	editedViewMode editedViewMode
	status         string
	warning        string
	width          int
	height         int
	result         Result
	userAborted    bool
}

func NewModel(commands []Command, clipboardText string, warning string, clipboard ClipboardProvider, runner ShellRunner) *Model {
	search := textinput.New()
	search.Placeholder = "Search alias or type custom shell command"
	search.Prompt = "/ "
	search.CharLimit = 512

	preview := viewport.New(0, 0)
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := &Model{
		commands:       commands,
		inputMode:      inputInactive,
		search:         search,
		preview:        preview,
		spinner:        sp,
		clipboard:      clipboard,
		runner:         runner,
		clipboardText:  clipboardText,
		previewInput:   truncate(clipboardText, previewInputLimit),
		edited:         make(map[string]editedContent),
		editedViewMode: editedViewDiff,
		warning:        warning,
	}
	m.setInputMode(inputInactive)
	m.rebuildEntries()
	if first := m.selectedEntry(); first != nil {
		m.previewCommand = first.Command
	}
	return m
}

func (m *Model) Init() tea.Cmd {
	return m.schedulePreview(0)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.executing {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if tick, ok := msg.(spinner.TickMsg); ok {
			_ = tick
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case previewTriggerMsg:
		if msg.Seq != m.previewSeq {
			return m, nil
		}
		selected := m.selectedEntry()
		if selected == nil {
			return m, nil
		}
		if m.previewCancel != nil {
			m.previewCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.previewCancel = cancel
		m.previewLoading = true
		m.previewFailed = false
		m.previewBody = "Generating preview..."
		m.previewCommand = selected.Command
		m.syncPreviewViewport()
		return m, runPreviewCmd(ctx, m.runner, msg.Seq, selected.Command, m.effectivePreviewInput())
	case previewResultMsg:
		if msg.Seq != m.previewSeq {
			return m, nil
		}
		m.previewLoading = false
		m.previewFailed = msg.Failed
		m.previewCommand = msg.Command
		if msg.Failed {
			m.previewBody = msg.Error
			m.status = "Preview failed"
		} else {
			m.previewBody = msg.Output
			if m.status == "Preview failed" {
				m.status = ""
			}
		}
		m.syncPreviewViewport()
		return m, nil
	case editPreparedMsg:
		m.executing = false
		m.execCancel = nil
		if msg.Err != nil {
			m.status = fmt.Sprintf("Edit preparation failed: %v", msg.Err)
			return m, nil
		}
		m.status = ""
		return m, openEditorCmd(msg.Key, msg.Original, msg.Original)
	case editorFinishedMsg:
		if msg.Err != nil {
			m.status = fmt.Sprintf("Edit failed: %v", msg.Err)
			return m, nil
		}
		if msg.Edited == msg.Original {
			delete(m.edited, msg.Key)
			m.status = ""
			return m, m.schedulePreview(0)
		}
		m.edited[msg.Key] = editedContent{Original: msg.Original, Edited: msg.Edited}
		m.editedViewMode = editedViewDiff
		m.status = ""
		return m, m.schedulePreview(0)
	case execResultMsg:
		m.executing = false
		m.execCancel = nil
		if msg.Err != nil {
			m.status = fmt.Sprintf("Execution failed: %v", msg.Err)
			return m, nil
		}
		if msg.Mode == actionStdout {
			m.result.Stdout = msg.Output
			return m, tea.Quit
		}
		m.result.Notice = "Copied processed text to clipboard."
		return m, tea.Quit
	}

	if m.inputMode != inputInactive {
		before := m.search.Value()
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		if after := m.search.Value(); after != before {
			m.rebuildEntries()
			return m, tea.Batch(cmd, m.schedulePreview(previewDebounce))
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.executing {
		switch msg.String() {
		case "ctrl+c":
			if m.execCancel != nil {
				m.execCancel()
			}
			m.status = "Execution canceled"
			m.executing = false
			return m, nil
		}
		return m, nil
	}

	if m.inputMode != inputInactive {
		switch msg.String() {
		case "esc":
			m.setInputMode(inputInactive)
			m.rebuildEntries()
			return m, m.schedulePreview(previewDebounce)
		case "enter":
			return m.startExecution(actionClipboard)
		case "tab":
			if m.inputMode == inputSearch {
				return m.toggleEditedView()
			}
		case "up", "k":
			if m.inputMode == inputSearch {
				m.moveSelection(-1)
				return m, m.schedulePreview(previewDebounce)
			}
		case "down", "j":
			if m.inputMode == inputSearch {
				m.moveSelection(1)
				return m, m.schedulePreview(previewDebounce)
			}
		}
		before := m.search.Value()
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		if m.search.Value() != before {
			m.rebuildEntries()
			return m, tea.Batch(cmd, m.schedulePreview(previewDebounce))
		}
		return m, cmd
	} else {
		switch msg.String() {
		case "ctrl+c", "esc":
			m.userAborted = true
			if m.previewCancel != nil {
				m.previewCancel()
			}
			return m, tea.Quit
		case "up", "k":
			m.moveSelection(-1)
			return m, m.schedulePreview(previewDebounce)
		case "down", "j":
			m.moveSelection(1)
			return m, m.schedulePreview(previewDebounce)
		case "enter", "s":
			return m.startExecution(actionClipboard)
		case "o":
			return m.startExecution(actionStdout)
		case "e":
			return m.startEditor()
		case "r":
			return m.resetEditedPreview()
		case "tab":
			return m.toggleEditedView()
		case "/":
			m.setInputMode(inputSearch)
			m.rebuildEntries()
			return m, m.schedulePreview(previewDebounce)
		case "!":
			m.setInputMode(inputCustom)
			m.rebuildEntries()
			return m, m.schedulePreview(previewDebounce)
		}
	}

	return m, nil
}

func (m *Model) startExecution(mode actionMode) (tea.Model, tea.Cmd) {
	selected := m.selectedEntry()
	if selected == nil || strings.TrimSpace(selected.Command) == "" {
		return m, nil
	}
	if edited, ok := m.currentEditedContent(); ok {
		if mode == actionStdout {
			m.result.Stdout = edited.Edited
			return m, tea.Quit
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.execCancel = cancel
		m.executing = true
		m.status = "Writing edited content to clipboard..."
		return m, tea.Batch(m.spinner.Tick, writeClipboardCmd(ctx, m.clipboard, edited.Edited))
	}
	if m.execCancel != nil {
		m.execCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.execCancel = cancel
	m.executing = true
	m.status = "Running full clipboard command..."
	return m, tea.Batch(m.spinner.Tick, runExecCmd(ctx, mode, m.runner, m.clipboard, selected.Command, m.effectiveInputText()))
}

func (m *Model) schedulePreview(delay time.Duration) tea.Cmd {
	m.previewSeq++
	seq := m.previewSeq
	selected := m.selectedEntry()
	if edited, ok := m.currentEditedContent(); ok {
		m.previewLoading = false
		m.previewFailed = false
		if selected != nil {
			m.previewCommand = selected.Command
		}
		m.previewBody = truncate(m.renderEditedPreview(edited), previewOutputLimit)
		m.syncPreviewViewport()
		return nil
	}
	if selected != nil && strings.TrimSpace(selected.Command) != "" {
		m.previewCommand = selected.Command
	} else {
		m.previewCommand = ""
		m.previewLoading = false
		m.previewFailed = false
		m.previewBody = m.emptyPreviewMessage()
		m.syncPreviewViewport()
		return nil
	}
	m.previewLoading = true
	m.previewBody = "Generating preview..."
	m.syncPreviewViewport()
	return func() tea.Msg {
		if delay > 0 {
			time.Sleep(delay)
		}
		return previewTriggerMsg{Seq: seq}
	}
}

func (m *Model) rebuildEntries() {
	query := strings.TrimSpace(m.search.Value())
	entries := make([]entry, 0, len(m.commands)+1)
	if m.inputMode == inputCustom {
		if query == "" {
			entries = append(entries, entry{Label: "Type a shell command", Command: "", Custom: true})
		} else {
			entries = append(entries, entry{Label: fmt.Sprintf("custom: %s", query), Command: query, Custom: true})
		}
	} else {
		for _, command := range m.commands {
			if query == "" || fuzzy.MatchNormalizedFold(query, command.Name) || fuzzy.MatchNormalizedFold(query, command.Shell) {
				entries = append(entries, entry{Label: command.Name, Command: command.Shell})
			}
		}
	}
	if len(entries) == 0 {
		entries = append(entries, entry{Label: "No matches", Command: ""})
	}
	m.filtered = entries
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if current := m.selectedEntry(); current != nil {
		m.previewCommand = current.Command
	}
}

func (m *Model) selectedEntry() *entry {
	if len(m.filtered) == 0 || m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	return &m.filtered[m.selected]
}

func (m *Model) moveSelection(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.selected = (m.selected + delta + len(m.filtered)) % len(m.filtered)
}

func (m *Model) resize() {
	if m.width == 0 || m.height == 0 {
		return
	}
	panelHeight := maxInt(12, m.height-4)
	rightWidth := maxInt(20, m.width*3/5-3)
	rightHeight := maxInt(4, panelHeight-6)
	m.preview.Width = rightWidth
	m.preview.Height = rightHeight
	m.syncPreviewViewport()
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	leftWidth := maxInt(24, m.width*2/5-2)
	dividerWidth := 3
	rightWidth := maxInt(32, m.width-leftWidth-dividerWidth)
	panelHeight := maxInt(12, m.height-4)

	left := lipgloss.NewStyle().Width(leftWidth).Height(panelHeight).Padding(0, 1).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.renderSearch(),
			"",
			m.renderList(leftWidth-2, panelHeight-3),
		),
	)

	divider := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("│\n", maxInt(1, panelHeight-1)) + "│")

	right := lipgloss.NewStyle().Width(rightWidth).Height(panelHeight).Padding(0, 1).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render("Command"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(m.previewCommand),
			"",
			lipgloss.NewStyle().Bold(true).Render(m.previewTitle()),
			m.preview.View(),
		),
	)

	status := m.status
	if m.warning != "" {
		status = m.warning
	}
	if m.executing {
		status = fmt.Sprintf("%s %s", m.spinner.View(), m.status)
	}

	header := lipgloss.NewStyle().Bold(true).Render("clip-tool")
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(m.helpText())
	sections := []string{header}
	if strings.TrimSpace(status) != "" {
		sections = append(sections, lipgloss.NewStyle().Foreground(statusColor(m)).Render(status))
	}
	sections = append(sections, body, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *Model) renderSearch() string {
	label := "Commands"
	switch m.inputMode {
	case inputSearch:
		label = "Search (active)"
	case inputCustom:
		label = "Custom Command (active)"
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(label),
		m.search.View(),
	)
}

func (m *Model) renderList(width int, height int) string {
	var lines []string
	for i, item := range m.filtered {
		label := item.Label
		if item.Custom {
			label = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(label)
		}
		content := label
		if m.hasEditedEntry(item) {
			content = lipgloss.JoinHorizontal(lipgloss.Left,
				label,
				lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(" (*)"),
			)
		}
		line := lipgloss.NewStyle().Width(width).Render(content)
		if i == m.selected {
			line = lipgloss.NewStyle().Background(lipgloss.Color("62")).Width(width).Render(content)
		}
		lines = append(lines, line)
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", maxInt(0, width)))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) syncPreviewViewport() {
	body := m.previewBody
	if body == "" {
		body = "No preview available."
	}
	style := lipgloss.NewStyle()
	if m.previewFailed {
		style = style.Foreground(lipgloss.Color("204"))
	}
	m.preview.SetContent(style.Render(body))
}

func (m *Model) helpText() string {
	if m.executing {
		return "ctrl+c cancel execution"
	}
	if m.inputMode == inputSearch {
		return "type to search • ↑/↓ choose • enter copy • tab toggle edited view • esc back"
	}
	if m.inputMode == inputCustom {
		return "type custom shell command • enter copy to clipboard • esc back"
	}
	return "↑/↓ or j/k navigate • enter/s copy • o stdout • e edit • r reset • tab toggle edited view • / search • ! custom • esc quit"
}

func (m *Model) startEditor() (tea.Model, tea.Cmd) {
	selected := m.selectedEntry()
	if selected == nil || strings.TrimSpace(selected.Command) == "" {
		return m, nil
	}
	if edited, ok := m.currentEditedContent(); ok {
		return m, openEditorCmd(m.currentEntryKey(), edited.Original, edited.Edited)
	}
	if m.execCancel != nil {
		m.execCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.execCancel = cancel
	m.executing = true
	m.status = "Preparing full output for editor..."
	return m, tea.Batch(m.spinner.Tick, runFullCommandCmd(ctx, m.currentEntryKey(), m.runner, selected.Command, m.effectiveInputText()))
}

func (m *Model) resetEditedPreview() (tea.Model, tea.Cmd) {
	key := m.currentEntryKey()
	if key == "" {
		return m, nil
	}
	if _, ok := m.edited[key]; !ok {
		return m, nil
	}
	delete(m.edited, key)
	m.status = ""
	return m, m.schedulePreview(0)
}

func (m *Model) toggleEditedView() (tea.Model, tea.Cmd) {
	if _, ok := m.currentEditedContent(); !ok {
		return m, nil
	}
	if m.editedViewMode == editedViewDiff {
		m.editedViewMode = editedViewFull
	} else {
		m.editedViewMode = editedViewDiff
	}
	return m, m.schedulePreview(0)
}

func (m *Model) currentEditedContent() (editedContent, bool) {
	key := m.currentEntryKey()
	if key == "" {
		return editedContent{}, false
	}
	edited, ok := m.edited[key]
	return edited, ok
}

func (m *Model) hasEditedEntry(item entry) bool {
	key := m.entryKey(item)
	if key == "" {
		return false
	}
	_, ok := m.edited[key]
	return ok
}

func (m *Model) effectiveInputText() string {
	if edited, ok := m.edited[m.identityEntryKey()]; ok {
		return edited.Edited
	}
	return m.clipboardText
}

func (m *Model) effectivePreviewInput() string {
	return truncate(m.effectiveInputText(), previewInputLimit)
}

func (m *Model) identityEntryKey() string {
	for _, command := range m.commands {
		if command.Name == "identity" {
			return "builtin:" + command.Name + "\x00" + command.Shell
		}
	}
	return "builtin:identity\x00cat"
}

func (m *Model) currentEntryKey() string {
	selected := m.selectedEntry()
	if selected == nil {
		return ""
	}
	return m.entryKey(*selected)
}

func (m *Model) entryKey(item entry) string {
	if strings.TrimSpace(item.Command) == "" {
		return ""
	}
	if item.Custom {
		return "custom:" + item.Command
	}
	return "builtin:" + item.Label + "\x00" + item.Command
}

func (m *Model) renderEditedPreview(edited editedContent) string {
	if m.editedViewMode == editedViewFull {
		return edited.Edited
	}
	return renderLineDiff(edited.Original, edited.Edited)
}

func (m *Model) previewTitle() string {
	if _, ok := m.currentEditedContent(); ok {
		if m.editedViewMode == editedViewFull {
			return "Preview (edited full)"
		}
		return "Preview (edited diff)"
	}
	return "Preview"
}

func runPreviewCmd(ctx context.Context, runner ShellRunner, seq int, command string, input string) tea.Cmd {
	return func() tea.Msg {
		result, err := runner.Run(ctx, command, input)
		if err != nil {
			return previewResultMsg{
				Seq:     seq,
				Failed:  true,
				Error:   truncate(strings.TrimSpace(coalesce(result.Stderr, err.Error())), previewOutputLimit),
				Command: command,
			}
		}
		output := strings.TrimRight(result.Stdout, "\n")
		if output == "" {
			output = "(empty output)"
		}
		return previewResultMsg{Seq: seq, Output: truncate(output, previewOutputLimit), Command: command}
	}
}

func runExecCmd(ctx context.Context, mode actionMode, runner ShellRunner, clipboard ClipboardProvider, command string, input string) tea.Cmd {
	return func() tea.Msg {
		result, err := runner.Run(ctx, command, input)
		if err != nil {
			if result.Stderr != "" {
				return execResultMsg{Mode: mode, Err: fmt.Errorf("%s", strings.TrimSpace(result.Stderr))}
			}
			return execResultMsg{Mode: mode, Err: err}
		}
		if mode == actionClipboard {
			if err := clipboard.WriteText(ctx, result.Stdout); err != nil {
				return execResultMsg{Mode: mode, Err: err}
			}
		}
		return execResultMsg{Mode: mode, Output: result.Stdout}
	}
}

func writeClipboardCmd(ctx context.Context, clipboard ClipboardProvider, output string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteText(ctx, output); err != nil {
			return execResultMsg{Mode: actionClipboard, Err: err}
		}
		return execResultMsg{Mode: actionClipboard, Output: output}
	}
}

func runFullCommandCmd(ctx context.Context, key string, runner ShellRunner, command string, input string) tea.Cmd {
	return func() tea.Msg {
		result, err := runner.Run(ctx, command, input)
		if err != nil {
			if result.Stderr != "" {
				return editPreparedMsg{Key: key, Err: fmt.Errorf("%s", strings.TrimSpace(result.Stderr))}
			}
			return editPreparedMsg{Key: key, Err: err}
		}
		return editPreparedMsg{Key: key, Original: result.Stdout}
	}
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "\n\n[truncated]"
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func statusColor(m *Model) lipgloss.Color {
	if m.warning != "" || m.previewFailed {
		return lipgloss.Color("204")
	}
	return lipgloss.Color("70")
}

func (m *Model) setInputMode(mode inputMode) {
	m.inputMode = mode
	m.search.SetValue("")
	switch mode {
	case inputSearch:
		m.search.Prompt = "/ "
		m.search.Placeholder = "Search command aliases"
		m.search.Focus()
	case inputCustom:
		m.search.Prompt = "! "
		m.search.Placeholder = "Type a shell command"
		m.search.Focus()
	default:
		m.search.Prompt = "/ "
		m.search.Placeholder = "Press / to search or ! to run a custom command"
		m.search.Blur()
	}
}

func (m *Model) emptyPreviewMessage() string {
	if m.inputMode == inputCustom {
		return "Type a custom shell command to preview its output."
	}
	return "No preview available."
}
