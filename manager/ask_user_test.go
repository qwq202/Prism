package manager

import (
	"chat/globals"
	"chat/manager/askuser"
	"chat/manager/conversation"
	"encoding/json"
	"testing"
)

func pendingAskUserConversation() (*conversation.Conversation, globals.ToolCall) {
	instance := conversation.NewAnonymousConversation()
	call := globals.ToolCall{
		Id:   "call_ask_1",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name: askuser.ToolName,
			Arguments: `{
				"questions":[{
					"id":"scope",
					"header":"Scope",
					"question":"Choose scope?",
					"type":"single",
					"options":[
						{"label":"Minimal","description":"Core only"},
						{"label":"Complete","description":"Everything"}
					]
				}]
			}`,
		},
	}
	calls := globals.ToolCalls{call}
	instance.AddMessage(globals.Message{
		Role:      globals.Assistant,
		ToolCalls: &calls,
	})
	return instance, call
}

func TestBuildAskUserAnswerMessageContinuesPendingToolCall(t *testing.T) {
	instance, call := pendingAskUserConversation()
	form := &conversation.FormMessage{
		ToolCallID: call.Id,
		ToolResult: json.RawMessage(`{
			"type":"ask_user_answer",
			"answers":{
				"scope":{"type":"single","value":"Complete","custom":false,"skipped":false}
			}
		}`),
	}

	message, err := buildAskUserAnswerMessage(instance, form)
	if err != nil {
		t.Fatalf("build answer message: %v", err)
	}
	if message.Role != globals.Tool || message.ToolCallId == nil || *message.ToolCallId != call.Id {
		t.Fatalf("unexpected tool answer message: %#v", message)
	}
	if !hasPendingAskUserCall(instance) {
		t.Fatalf("expected pending call to remain until tool message is persisted")
	}
}

func TestBuildAskUserAnswerMessageRejectsStaleCall(t *testing.T) {
	instance, call := pendingAskUserConversation()
	instance.AddMessage(globals.Message{Role: globals.User, Content: "new turn"})

	_, err := buildAskUserAnswerMessage(instance, &conversation.FormMessage{
		ToolCallID: call.Id,
		ToolResult: json.RawMessage(`{"type":"ask_user_answer","answers":{}}`),
	})
	if err == nil {
		t.Fatalf("expected stale answer to be rejected")
	}
}

func TestBuildAvailableToolDefinitionsIncludesAskUserByDefault(t *testing.T) {
	tools := buildAvailableToolDefinitions(true, false, false, false)
	if tools == nil || len(*tools) != 1 {
		t.Fatalf("expected one built-in tool, got %#v", tools)
	}
	if (*tools)[0].Function.Name != askuser.ToolName {
		t.Fatalf("expected ask_user tool, got %#v", (*tools)[0])
	}

	if tools := buildAvailableToolDefinitions(false, false, false, false); tools != nil {
		t.Fatalf("expected native API path to omit interactive tool, got %#v", tools)
	}
}
