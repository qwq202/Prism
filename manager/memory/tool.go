package memory

import (
	"chat/auth"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

var toolCallChannelTypes = map[string]struct{}{
	globals.OpenAIChannelType:                        {},
	globals.OpenAIResponsesChannelType:               {},
	globals.AzureOpenAIChannelType:                   {},
	globals.ClaudeChannelType:                        {},
	globals.GLMCodingPlanCNChannelType:               {},
	globals.MiniMaxTokenPlanCNChannelType:            {},
	globals.XiaomiMiMoChannelType:                    {},
	globals.XiaomiTokenPlanCNChannelType:             {},
	globals.PalmChannelType:                          {},
	globals.GeminiEnterpriseAgentPlatformChannelType: {},
	globals.DeepseekChannelType:                      {},
	globals.XAIChannelType:                           {},
}

var writableToolChannelTypes = toolCallChannelTypes

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

func canUseChannelTools(model, group string, allowed map[string]struct{}) bool {
	ticker := channel.ConduitInstance.GetTicker(model, group)
	if ticker == nil || ticker.IsEmpty() {
		return false
	}

	for _, item := range ticker.Sequence {
		if item == nil {
			continue
		}

		if _, ok := allowed[item.Type]; !ok {
			return false
		}
	}

	return true
}

func CanUseToolCalls(model, group string) bool {
	return canUseChannelTools(model, group, toolCallChannelTypes)
}

func CanUseWritableTools(model, group string) bool {
	return canUseChannelTools(model, group, writableToolChannelTypes)
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

func summarizeToolArguments(arguments string) string {
	arguments = strings.TrimSpace(arguments)
	if len(arguments) <= 240 {
		return arguments
	}

	return arguments[:240] + "..."
}

func summarizeQuotedArguments(arguments string) string {
	quoted := fmt.Sprintf("%q", arguments)
	if len(quoted) <= 320 {
		return quoted
	}

	return quoted[:320] + "..."
}

func summarizeArgumentBytes(arguments string) string {
	bytes := []byte(arguments)
	limit := len(bytes)
	if limit > 96 {
		limit = 96
	}

	hexText := hex.EncodeToString(bytes[:limit])
	if len(bytes) > limit {
		return hexText + "..."
	}

	return hexText
}

func summarizeParsedToolMap(arguments string) string {
	var raw map[string]any
	if err := json.Unmarshal([]byte(arguments), &raw); err != nil {
		return "unmarshal_map_error=" + err.Error()
	}

	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	action, _ := raw["action"].(string)
	category, _ := raw["category"].(string)
	content, _ := raw["content"].(string)
	reason, _ := raw["reason"].(string)

	memoryIDSummary := "missing"
	if value, ok := raw["memory_id"]; ok {
		memoryIDSummary = fmt.Sprintf("%T:%v", value, value)
	}

	return fmt.Sprintf(
		"keys=%v action=%q category=%q content_len=%d reason_len=%d memory_id=%s",
		keys,
		action,
		category,
		len(strings.TrimSpace(content)),
		len(strings.TrimSpace(reason)),
		memoryIDSummary,
	)
}

func logToolArgumentDiagnostics(callID, arguments string) {
	trimmed := strings.TrimSpace(arguments)
	globals.Debug(fmt.Sprintf(
		"[memory] tool call %s diagnostics len=%d trimmed_len=%d utf8_valid=%v json_valid=%v quoted=%s bytes=%s",
		callID,
		len(arguments),
		len(trimmed),
		utf8.ValidString(arguments),
		json.Valid([]byte(trimmed)),
		summarizeQuotedArguments(arguments),
		summarizeArgumentBytes(arguments),
	))

	globals.Debug(fmt.Sprintf(
		"[memory] tool call %s map diagnostics %s",
		callID,
		summarizeParsedToolMap(trimmed),
	))
}

func parseToolInput(arguments string) (ToolInput, error) {
	return utils.UnmarshalString[ToolInput](arguments)
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

	globals.Debug(fmt.Sprintf(
		"[memory] received tool call %s args=%s",
		call.Id,
		summarizeToolArguments(call.Function.Arguments),
	))
	logToolArgumentDiagnostics(call.Id, call.Function.Arguments)

	input, err := parseToolInput(call.Function.Arguments)
	if err != nil {
		globals.Warn(fmt.Sprintf(
			"[memory] invalid tool arguments for call %s: %s (raw=%s parsed=%s)",
			call.Id,
			err.Error(),
			summarizeToolArguments(call.Function.Arguments),
			summarizeParsedToolMap(strings.TrimSpace(call.Function.Arguments)),
		))
		result.Error = "invalid tool arguments"
		return toolResultMessage(call.Id, result)
	}

	input.Action = strings.TrimSpace(strings.ToLower(input.Action))
	input.Content = strings.TrimSpace(input.Content)
	input.Category = strings.TrimSpace(input.Category)
	input.Reason = strings.TrimSpace(input.Reason)
	result.Action = input.Action

	globals.Debug(fmt.Sprintf(
		"[memory] parsed tool call %s action=%s memory_id=%v category=%s content_len=%d reason_len=%d",
		call.Id,
		input.Action,
		input.MemoryID != nil,
		input.Category,
		len(input.Content),
		len(input.Reason),
	))

	if strings.TrimSpace(call.Function.Arguments) != "" &&
		input.Action == "" &&
		input.MemoryID == nil &&
		input.Category == "" &&
		input.Content == "" &&
		input.Reason == "" {
		globals.Warn(fmt.Sprintf(
			"[memory] suspicious empty tool parse for call %s raw=%s parsed=%s",
			call.Id,
			summarizeToolArguments(call.Function.Arguments),
			summarizeParsedToolMap(strings.TrimSpace(call.Function.Arguments)),
		))
	}

	if input.Reason == "" {
		globals.Warn(fmt.Sprintf(
			"[memory] missing reason for call %s after parsing (raw=%s parsed=%s)",
			call.Id,
			summarizeToolArguments(call.Function.Arguments),
			summarizeParsedToolMap(strings.TrimSpace(call.Function.Arguments)),
		))
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
			globals.Debug(fmt.Sprintf("[memory] duplicate memory hit for call %s memory_id=%d", call.Id, duplicate.ID))
			result.Status = "success"
			result.MemoryID = &duplicate.ID
			result.Message = "memory already exists"
			return toolResultMessage(call.Id, result)
		}

		record, err := Create(db, userID, input.Content, SourceToolAuto, input.Category)
		if err != nil {
			globals.Warn(fmt.Sprintf("[memory] create failed for call %s: %s", call.Id, err.Error()))
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}

		globals.Debug(fmt.Sprintf("[memory] created memory %d for call %s", record.ID, call.Id))
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
			globals.Warn(fmt.Sprintf("[memory] update failed for call %s memory_id=%d: %s", call.Id, *input.MemoryID, err.Error()))
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}

		globals.Debug(fmt.Sprintf("[memory] updated memory %d for call %s", record.ID, call.Id))
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
				globals.Warn(fmt.Sprintf("[memory] delete target missing for call %s memory_id=%d", call.Id, *input.MemoryID))
				result.Error = "memory not found"
				return toolResultMessage(call.Id, result)
			}
			globals.Warn(fmt.Sprintf("[memory] lookup before delete failed for call %s memory_id=%d: %s", call.Id, *input.MemoryID, err.Error()))
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}
		if record.Pinned {
			globals.Warn(fmt.Sprintf("[memory] refused deleting pinned memory %d for call %s", record.ID, call.Id))
			result.Error = "pinned memory cannot be deleted"
			return toolResultMessage(call.Id, result)
		}

		if err := Delete(db, userID, *input.MemoryID); err != nil {
			globals.Warn(fmt.Sprintf("[memory] delete failed for call %s memory_id=%d: %s", call.Id, *input.MemoryID, err.Error()))
			result.Error = err.Error()
			return toolResultMessage(call.Id, result)
		}

		globals.Debug(fmt.Sprintf("[memory] deleted memory %d for call %s", *input.MemoryID, call.Id))
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
