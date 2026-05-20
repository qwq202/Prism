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

func TestCreateChatPropsInjectsLearningModeGuidance(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:        "gpt-5.4",
		LearningMode: true,
		Message: []globals.Message{
			{Role: globals.User, Content: "What is the answer?"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected injected system message, got %d messages", len(props.Message))
	}

	content := props.Message[0].Content
	if !strings.Contains(content, learningModePromptPrefix) {
		t.Fatalf("expected learning mode prompt prefix, got %q", content)
	}
	if !strings.Contains(content, "Do not give the final answer immediately") {
		t.Fatalf("expected guided learning instruction, got %q", content)
	}
	if !strings.Contains(content, "questions, hints, checkpoints") {
		t.Fatalf("expected Socratic guidance, got %q", content)
	}
}

func TestCreateChatPropsAvoidsDuplicateLearningModeGuidance(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:        "gpt-5.4",
		LearningMode: true,
		Message: []globals.Message{
			{
				Role: globals.System,
				Content: currentDateTimePromptPrefix + " 2026-04-20 23:30:00 (Asia/Shanghai).\n\n" +
					learningModePromptPrefix + "\n- Learning mode is enabled for this turn.",
			},
			{Role: globals.User, Content: "Help me learn."},
		},
	}, nil)

	if strings.Count(props.Message[0].Content, learningModePromptPrefix) != 1 {
		t.Fatalf("expected learning mode prompt prefix to appear once, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsInjectsMemoryCapabilityState(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:                "grok-4-1-fast-reasoning",
		MemoryEnabled:        false,
		MemoryHistoryEnabled: false,
		Message: []globals.Message{
			{Role: globals.User, Content: "Can you see my memories?"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected injected system message, got %d messages", len(props.Message))
	}

	content := props.Message[0].Content
	if !strings.Contains(content, memoryCapabilityPromptPrefix) {
		t.Fatalf("expected memory capability prompt prefix, got %q", content)
	}

	if !strings.Contains(content, "Saved user memories: disabled.") {
		t.Fatalf("expected saved memory state to be disabled, got %q", content)
	}

	if !strings.Contains(content, "Cross-conversation recent chat references: disabled.") {
		t.Fatalf("expected recent chat state to be disabled, got %q", content)
	}

	if !strings.Contains(content, "current conversation context") {
		t.Fatalf("expected current conversation clarification, got %q", content)
	}

	if !strings.Contains(content, "you may use them to answer questions about the user's long-term preferences") {
		t.Fatalf("expected saved memory usage guidance, got %q", content)
	}

	if !strings.Contains(content, "Do not claim that you cannot access saved memories") {
		t.Fatalf("expected no-false-denial guidance, got %q", content)
	}
}

func TestCreateChatPropsInjectsCurrentModelReference(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model: "grok-4-1-fast-reasoning",
		Message: []globals.Message{
			{Role: globals.User, Content: "Which model am I using?"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected injected system message, got %d messages", len(props.Message))
	}

	content := props.Message[0].Content
	if !strings.Contains(content, currentModelPromptPrefix) {
		t.Fatalf("expected current model prompt prefix, got %q", content)
	}

	if !strings.Contains(content, "The user is currently chatting with model: grok-4-1-fast-reasoning.") {
		t.Fatalf("expected current model name to be injected, got %q", content)
	}

	if !strings.Contains(content, "authoritative current-turn model identity") {
		t.Fatalf("expected authoritative current model guidance, got %q", content)
	}

	if !strings.Contains(content, "treat them as stale outputs from an earlier model selection") {
		t.Fatalf("expected stale identity guidance, got %q", content)
	}

	if !strings.Contains(content, "Do not manually emit <think> or </think> tags") {
		t.Fatalf("expected manual think-tag ban, got %q", content)
	}
}

func TestCreateChatPropsUpdatesExistingCurrentModelReference(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model: "deepseek-v4-flash",
		Message: []globals.Message{
			{
				Role: globals.System,
				Content: currentDateTimePromptPrefix + " 2026-04-20 23:30:00 (Asia/Shanghai).\n\n" +
					currentModelPromptPrefix + "\n- The user is currently chatting with model: grok-4-1-fast-reasoning.",
			},
			{Role: globals.User, Content: "Which model am I using now?"},
		},
	}, nil)

	content := props.Message[0].Content
	if strings.Count(content, currentModelPromptPrefix) != 1 {
		t.Fatalf("expected current model prompt prefix to appear once, got %q", content)
	}

	if strings.Contains(content, "grok-4-1-fast-reasoning") {
		t.Fatalf("expected previous model reference to be replaced, got %q", content)
	}

	if !strings.Contains(content, "The user is currently chatting with model: deepseek-v4-flash.") {
		t.Fatalf("expected updated current model reference, got %q", content)
	}

	if !strings.Contains(content, "answer only with the current model above") {
		t.Fatalf("expected current-model-only guidance, got %q", content)
	}
}

func TestCreateChatPropsAvoidsDuplicateMemoryCapabilityInjection(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:                "grok-4-1-fast-reasoning",
		MemoryEnabled:        true,
		MemoryHistoryEnabled: false,
		Message: []globals.Message{
			{
				Role: globals.System,
				Content: currentDateTimePromptPrefix + " 2026-04-20 23:30:00 (Asia/Shanghai).\n\n" +
					memoryCapabilityPromptPrefix + "\n- Saved user memories: enabled.",
			},
			{Role: globals.User, Content: "hello"},
		},
	}, nil)

	if strings.Count(props.Message[0].Content, memoryCapabilityPromptPrefix) != 1 {
		t.Fatalf("expected memory capability prompt prefix to appear once, got %q", props.Message[0].Content)
	}
}

func TestCreateChatPropsInjectsReferenceSectionsAlongsideMemoryCapabilityState(t *testing.T) {
	props := CreateChatProps(&ChatProps{
		Model:                "deepseek-v4-flash",
		MemoryEnabled:        true,
		MemoryHistoryEnabled: true,
		MemoryPrompt:         "Remember that the user likes Genshin Impact.",
		RecentChatsPrompt:    "Conversation 9 was about game preferences.",
		Message: []globals.Message{
			{Role: globals.User, Content: "What do I like?"},
		},
	}, nil)

	if len(props.Message) != 2 {
		t.Fatalf("expected injected system message, got %d messages", len(props.Message))
	}

	content := props.Message[0].Content
	if !strings.Contains(content, memoryCapabilityPromptPrefix) {
		t.Fatalf("expected memory capability prompt prefix, got %q", content)
	}

	if !strings.Contains(content, "- Saved user memories: enabled.") {
		t.Fatalf("expected enabled saved memory state, got %q", content)
	}

	if !strings.Contains(content, memoryPromptPrefix+"\nRemember that the user likes Genshin Impact.") {
		t.Fatalf("expected saved memory reference section to be injected, got %q", content)
	}

	if !strings.Contains(content, recentChatsPromptPrefix+"\nConversation 9 was about game preferences.") {
		t.Fatalf("expected recent chats reference section to be injected, got %q", content)
	}
}
