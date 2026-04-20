package conversation

import (
	"chat/globals"
	"testing"
)

func TestSaveResponseSkipsMetadataOnlyAssistantReply(t *testing.T) {
	instance := NewAnonymousConversation()

	saved := instance.SaveResponse(nil, globals.Message{
		Content: "",
		GeminiHiddenMetadata: &globals.GeminiHiddenMetadata{
			ThoughtSignatures: []string{"sig-1"},
		},
	})

	if saved {
		t.Fatalf("expected metadata-only response not to be persisted")
	}

	if got := instance.GetMessageLength(); got != 0 {
		t.Fatalf("expected no messages to be persisted, got %d", got)
	}
}

func TestSaveResponsePersistsToolCallsWithoutText(t *testing.T) {
	instance := NewAnonymousConversation()
	calls := globals.ToolCalls{
		{
			Type: "function",
			Id:   "tool-call-1",
			Function: globals.ToolCallFunction{
				Name:      "lookup_weather",
				Arguments: "{\"city\":\"Shanghai\"}",
			},
		},
	}

	saved := instance.SaveResponse(nil, globals.Message{
		Role:      globals.User,
		Content:   "",
		ToolCalls: &calls,
	})

	if !saved {
		t.Fatalf("expected tool-call response to be persisted")
	}

	if got := instance.GetMessageLength(); got != 1 {
		t.Fatalf("expected one persisted message, got %d", got)
	}

	last := instance.GetLastMessage()
	if last.Role != globals.Assistant {
		t.Fatalf("expected role %q, got %q", globals.Assistant, last.Role)
	}

	if last.ToolCalls == nil || len(*last.ToolCalls) != 1 {
		t.Fatalf("expected one tool call in persisted message, got %#v", last.ToolCalls)
	}
}

func TestSaveResponsePersistsFunctionCallWithoutText(t *testing.T) {
	instance := NewAnonymousConversation()

	saved := instance.SaveResponse(nil, globals.Message{
		Content: "",
		FunctionCall: &globals.FunctionCall{
			Name:      "lookup_air_quality",
			Arguments: "{\"city\":\"Shanghai\"}",
		},
	})

	if !saved {
		t.Fatalf("expected function-call response to be persisted")
	}

	if got := instance.GetMessageLength(); got != 1 {
		t.Fatalf("expected one persisted message, got %d", got)
	}

	last := instance.GetLastMessage()
	if last.Role != globals.Assistant {
		t.Fatalf("expected role %q, got %q", globals.Assistant, last.Role)
	}

	if last.FunctionCall == nil || last.FunctionCall.Name != "lookup_air_quality" {
		t.Fatalf("expected function call payload to be preserved, got %#v", last.FunctionCall)
	}
}

func TestSaveResponsePersistsConversationModelOnAssistantReply(t *testing.T) {
	instance := NewAnonymousConversation()
	instance.SetModel("grok-4.20-reasoning")

	saved := instance.SaveResponse(nil, globals.Message{
		Content: "hello from grok",
	})

	if !saved {
		t.Fatalf("expected assistant response to be persisted")
	}

	last := instance.GetLastMessage()
	if last.Model != "grok-4.20-reasoning" {
		t.Fatalf("expected persisted model to be preserved, got %q", last.Model)
	}
}
