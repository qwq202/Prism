package manager

import (
	"chat/globals"
	"chat/manager/askuser"
	"chat/manager/conversation"
	"encoding/json"
	"errors"
	"strings"
)

func pendingAskUserCall(instance *conversation.Conversation) (globals.ToolCall, bool) {
	if instance == nil {
		return globals.ToolCall{}, false
	}

	messages := instance.GetMessage()
	if len(messages) == 0 {
		return globals.ToolCall{}, false
	}

	last := messages[len(messages)-1]
	if last.Role != globals.Assistant {
		return globals.ToolCall{}, false
	}

	return askuser.FirstToolCall(last.ToolCalls)
}

func hasPendingAskUserCall(instance *conversation.Conversation) bool {
	_, ok := pendingAskUserCall(instance)
	return ok
}

func buildAskUserAnswerMessage(instance *conversation.Conversation, form *conversation.FormMessage) (globals.Message, error) {
	if form == nil {
		return globals.Message{}, errors.New("missing tool result form")
	}

	call, ok := pendingAskUserCall(instance)
	if !ok {
		return globals.Message{}, errors.New("no pending ask_user request")
	}
	if strings.TrimSpace(form.ToolCallID) == "" || strings.TrimSpace(form.ToolCallID) != strings.TrimSpace(call.Id) {
		return globals.Message{}, errors.New("tool call id does not match the pending ask_user request")
	}
	if len(form.ToolResult) == 0 || string(form.ToolResult) == "null" {
		return globals.Message{}, errors.New("tool result is required")
	}

	return askuser.AnswerMessage(call, json.RawMessage(form.ToolResult))
}
