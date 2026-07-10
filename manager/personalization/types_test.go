package personalization

import (
	"strings"
	"testing"
)

func TestNormalizeSettingsAppliesDefaultsAndTrimsText(t *testing.T) {
	settings, err := normalizeSettings(Settings{
		PersonaNickname:   "  Quinn  ",
		PersonaOccupation: "  Engineer  ",
	})
	if err != nil {
		t.Fatalf("normalize settings: %v", err)
	}
	if settings.PersonaStyle != "default" || settings.PersonaEmoji != "default" {
		t.Fatalf("expected default choices, got %#v", settings)
	}
	if settings.PersonaNickname != "Quinn" || settings.PersonaOccupation != "Engineer" {
		t.Fatalf("expected trimmed text, got %#v", settings)
	}
}

func TestNormalizeSettingsRejectsInvalidChoiceAndOversizedText(t *testing.T) {
	if _, err := normalizeSettings(Settings{PersonaStyle: "unknown"}); err == nil {
		t.Fatal("expected invalid style to fail")
	}
	if _, err := normalizeSettings(Settings{
		PersonaCustomInstruction: strings.Repeat("界", maxCustomInstructionLength+1),
	}); err == nil {
		t.Fatal("expected oversized custom instruction to fail")
	}
}
