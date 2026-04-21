package adaptercommon

import (
	"chat/globals"
	"strings"
	"testing"
)

func TestCreateChatPropsInjectsCurrentDateTimeSystemMessage(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model: "grok-4.20-reasoning",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected injected system message, got %d messages", len(props.Message))
	}

	if props.Message[0].Role != globals.System {
		t.Fatalf("expected first message role %q, got %q", globals.System, props.Message[0].Role)
	}

	if !strings.HasPrefix(props.Message[0].Content, currentDateTimePromptPrefix) {
		t.Fatalf("expected current time prompt prefix, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsIncludesClientContextWhenProvided(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:         "gemini-3-flash-preview",
		ClientContext: "Operating system: macOS; Browser/App: Chrome; Device type: computer.",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if !strings.Contains(props.Message[0].Content, clientContextPromptPrefix) {
		t.Fatalf("expected client context prompt prefix, got %q", props.Message[0].Content)
	}

	if !strings.Contains(props.Message[0].Content, "Operating system: macOS") {
		t.Fatalf("expected client context to be preserved, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsPrefixesExistingSystemMessage(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model: "gemini-3-flash-preview",
		Message: []globals.Message{
			{Role: globals.System, Content: "You are helpful."},
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected message count to stay the same, got %d", len(props.Message))
	}

	if !strings.HasPrefix(props.Message[0].Content, currentDateTimePromptPrefix) {
		t.Fatalf("expected current time prompt prefix, got %q", props.Message[0].Content)
	}

	if !strings.Contains(props.Message[0].Content, "You are helpful.") {
		t.Fatalf("expected original system prompt to be preserved, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsAvoidsDuplicateCurrentDateTimeInjection(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model: "claude-3-7-sonnet",
		Message: []globals.Message{
			{Role: globals.System, Content: currentDateTimePromptPrefix + " 2026-04-20 23:30:00 (Asia/Shanghai)."},
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected message count to stay the same, got %d", len(props.Message))
	}

	if strings.Count(props.Message[0].Content, currentDateTimePromptPrefix) != 1 {
		t.Fatalf("expected prompt prefix to appear once, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsInjectsPersonalizationPreferences(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:             "claude-3-7-sonnet",
		CustomInstruction: "Address the user as Captain and keep the tone upbeat.",
		Message: []globals.Message{
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected injected system message, got %d", len(props.Message))
	}

	if !strings.Contains(props.Message[0].Content, personalizationPromptPrefix) {
		t.Fatalf("expected personalization prompt prefix, got %q", props.Message[0].Content)
	}

	if !strings.Contains(props.Message[0].Content, "Address the user as Captain") {
		t.Fatalf("expected personalization content to be preserved, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsAvoidsDuplicatePersonalizationInjection(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:             "claude-3-7-sonnet",
		CustomInstruction: "Use a direct tone.",
		Message: []globals.Message{
			{
				Role: globals.System,
				Content: currentDateTimePromptPrefix + " 2026-04-20 23:30:00 (Asia/Shanghai).\n\n" +
					personalizationPromptPrefix + "\nUse a direct tone.",
			},
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if strings.Count(props.Message[0].Content, personalizationPromptPrefix) != 1 {
		t.Fatalf("expected personalization prompt prefix to appear once, got %q", props.Message[0].Content)
	}
}
