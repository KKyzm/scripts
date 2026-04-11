package cliptool

import "testing"

func TestParseCommandsJSONMap(t *testing.T) {
	commands, err := parseCommands("commands.json", []byte(`{"upper":"tr '[:lower:]' '[:upper:]'","sort":"sort"}`))
	if err != nil {
		t.Fatalf("parseCommands returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if commands[0].Name != "sort" || commands[1].Name != "upper" {
		t.Fatalf("unexpected commands order: %#v", commands)
	}
}

func TestParseCommandsYAMLNested(t *testing.T) {
	commands, err := parseCommands("commands.yaml", []byte("commands:\n  lower: tr '[:upper:]' '[:lower:]'\n  keep: cat\n"))
	if err != nil {
		t.Fatalf("parseCommands returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if commands[0].Name != "keep" || commands[1].Name != "lower" {
		t.Fatalf("unexpected commands order: %#v", commands)
	}
}

func TestDefaultCommandsIdentityFirst(t *testing.T) {
	commands := defaultCommands()
	if len(commands) == 0 {
		t.Fatal("expected built-in commands")
	}
	if commands[0].Name != "identity" {
		t.Fatalf("expected identity to be first, got %q", commands[0].Name)
	}
}
