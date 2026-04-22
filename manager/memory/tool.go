package memory

import (
	"chat/auth"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"errors"
	"strings"
)

var writableToolChannelTypes = map[string]struct{}{
	globals.OpenAIChannelType:          {},
	globals.AzureOpenAIChannelType:     {},
	globals.ClaudeChannelType:          {},
	globals.GLMCodingPlanCNChannelType: {},
	globals.PalmChannelType:            {},
	globals.DeepseekChannelType:        {},
}

func BuildToolDefinition() *globals.FunctionTools {
	required := []string{"action", "reason"}

	tools := globals.FunctionTools{
		{
			Type: "function",
			Function: globals.ToolFunction{
				Name:        MemoryToolName,
				Description: "Create, edit, or delete concise long-term user memories when the user reveals a stable preference, profile detail, or standing instruction worth keeping.",
				Parameters: globals.ToolParameters{
					Type: "object",
					Properties: globals.ToolProperties{
						"action": {
							"type":        "string",
							"enum":        []string{"create", "edit", "delete"},
							"description": "The memory operation to perform.",
						},
						"memory_id": {
							"type":        "integer",
							"description": "Required for edit and delete.",
						},
						"content": {
							"type":        "string",
							"description": "The final concise memory content. Required for create and edit.",
						},
						"category": {
							"type":        "string",
							"description": "Optional memory category such as preference, profile, project, or constraint.",
						},
						"reason": {
							"type":        "string",
							"description": "Short reason explaining why this memory change is appropriate.",
						},
					},
					Required: &required,
				},
			},
		},
	}

	return &tools
}

func BuildAutoToolChoice() *interface{} {
	choice := interface{}("auto")
	return &choice
}

func CanUseWritableTools(model, group string) bool {
	ticker := channel.ConduitInstance.GetTicker(model, group)
	if ticker == nil || ticker.IsEmpty() {
		return false
	}

	for _, item := range ticker.Sequence {
		if item == nil {
			continue
		}

		if _, ok := writableToolChannelTypes[item.Type]; !ok {
			return false
		}
	}

	return true
}

func containsSensitiveContent(content string) bool {
	content = strings.ToLower(strings.TrimSpace(content))
	if content == "" {
		return false
	}

	patterns := []string{
		"api key",
		"apikey",
		"access token",
		"refresh token",
		"password",
		"passwd",
		"secret",
		"private key",
		"ssh-rsa",
		"sk-",
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}

	return false
}

func toolResultMessage(callID string, result ToolResult) globals.Message {
	return globals.Message{
		Role:       globals.Tool,
		Content:    utils.Marshal(result),
		ToolCallId: utils.ToPtr(callID),
	}
}

func executeToolCall(db *sql.DB, user *auth.User, call globals.ToolCall) globals.Message {
	result := ToolResult{
		Status: "error",
		Action: call.Function.Name,
	}

	if user == nil {
		result.Error = "user not found"
		return toolResultMessage(call.Id, result)
	}

	if call.Function.Name != MemoryToolName {
		result.Error = "unsupported tool"
		return toolResultMessage(call.Id, result)
	}

	var input ToolInput
	if _, err := utils.UnmarshalString[ToolInput](call.Function.Arguments); err != nil {
		result.Error = "invalid tool arguments"
		return toolResultMessage(call.Id, result)
	}

	input.Action = strings.TrimSpace(strings.ToLower(input.Action))
	input.Content = strings.TrimSpace(input.Content)
	input.Category = strings.TrimSpace(input.Category)
	input.Reason = strings.TrimSpace(input.Reason)
	result.Action = input.Action

	if input.Reason == "" {
		result.Error = "reason is required"
		return toolResultMessage(call.Id, result)
	}

	userID := user.GetID(db)
	switch input.Action {
	case "create":
		if input.Content == "" {
			result.Error = "content is required"
			return toolResultMessage(call.Id, result)
		}
		if containsSensitiveContent(input.Content) {
			result.Error = "sensitive content cannot be stored"
			return toolResultMessage(call.Id, result)
		}

		if duplicate, err := FindDuplicate(db, userID, input.Content); err == nil && duplicate != nil {
			result.Status = "success"
			result.MemoryID = &duplicate.ID
			result.Message = "memory already exists"
			return toolResultMessage(call.Id, result)
		}

		record, err := Create(db, userID, input.Content, SourceToolAuto, input.Category)
		if err != nil {
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}

		result.Status = "success"
		result.MemoryID = &record.ID
		result.Message = "memory created"
		return toolResultMessage(call.Id, result)
	case "edit":
		if input.MemoryID == nil {
			result.Error = "memory_id is required"
			return toolResultMessage(call.Id, result)
		}
		if input.Content == "" {
			result.Error = "content is required"
			return toolResultMessage(call.Id, result)
		}
		if containsSensitiveContent(input.Content) {
			result.Error = "sensitive content cannot be stored"
			return toolResultMessage(call.Id, result)
		}

		record, err := Update(db, userID, *input.MemoryID, input.Content, input.Category)
		if err != nil {
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}

		result.Status = "success"
		result.MemoryID = &record.ID
		result.Message = "memory updated"
		return toolResultMessage(call.Id, result)
	case "delete":
		if input.MemoryID == nil {
			result.Error = "memory_id is required"
			return toolResultMessage(call.Id, result)
		}

		record, err := FindByID(db, userID, *input.MemoryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				result.Error = "memory not found"
				return toolResultMessage(call.Id, result)
			}
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}
		if record.Pinned {
			result.Error = "pinned memory cannot be deleted"
			return toolResultMessage(call.Id, result)
		}

		if err := Delete(db, userID, *input.MemoryID); err != nil {
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}

		result.Status = "success"
		result.MemoryID = input.MemoryID
		result.Message = "memory deleted"
		return toolResultMessage(call.Id, result)
	default:
		result.Error = "unsupported action"
		return toolResultMessage(call.Id, result)
	}
}

func ExecuteToolCalls(db *sql.DB, user *auth.User, calls *globals.ToolCalls) []globals.Message {
	if calls == nil || len(*calls) == 0 {
		return nil
	}

	messages := make([]globals.Message, 0, len(*calls))
	for _, call := range *calls {
		messages = append(messages, executeToolCall(db, user, call))
	}

	return messages
}
