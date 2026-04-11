package cliptool

import "testing"

func TestEffectiveInputTextUsesIdentityEditedContent(t *testing.T) {
	commands := defaultCommands()
	model := NewModel(commands, "original clipboard", "", nil, nil)

	if got := model.effectiveInputText(); got != "original clipboard" {
		t.Fatalf("expected original clipboard input, got %q", got)
	}

	model.edited[model.identityEntryKey()] = editedContent{
		Original: "original clipboard",
		Edited:   "edited identity output",
	}

	if got := model.effectiveInputText(); got != "edited identity output" {
		t.Fatalf("expected identity edited content to become effective input, got %q", got)
	}
	if got := model.effectivePreviewInput(); got != "edited identity output" {
		t.Fatalf("expected preview input to follow identity edited content, got %q", got)
	}
}
