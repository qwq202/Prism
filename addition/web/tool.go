package web

import (
	"chat/globals"
	"chat/utils"
	"strings"
)

const SearchToolName = "web_search"

type SearchToolInput struct {
	Query string `json:"query"`
}

type SearchToolResult struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Query   string `json:"query,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

func BuildToolDefinition() *globals.FunctionTools {
	required := []string{"query"}

	tools := globals.FunctionTools{
		{
			Type: "function",
			Function: globals.ToolFunction{
				Name:        SearchToolName,
				Description: "Search the live web for current, recent, or source-backed information. Use this when the answer may depend on up-to-date facts, news, prices, documentation, schedules, or external sources.",
				Parameters: globals.ToolParameters{
					Type: "object",
					Properties: globals.ToolProperties{
						"query": {
							"type":        "string",
							"description": "A concise search query containing the important entities and any relevant date or time range from the user's request.",
						},
					},
					Required: &required,
				},
			},
		},
	}

	return &tools
}

func searchToolResultMessage(callID string, result SearchToolResult) globals.Message {
	return globals.Message{
		Role:       globals.Tool,
		Content:    utils.Marshal(result),
		ToolCallId: utils.ToPtr(callID),
	}
}

func ExecuteToolCall(call globals.ToolCall) globals.Message {
	result := SearchToolResult{
		Status: "error",
		Action: call.Function.Name,
	}

	if call.Function.Name != SearchToolName {
		result.Error = "unsupported tool"
		return searchToolResultMessage(call.Id, result)
	}

	input, err := utils.UnmarshalString[SearchToolInput](call.Function.Arguments)
	if err != nil {
		result.Error = "invalid tool arguments"
		return searchToolResultMessage(call.Id, result)
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		result.Error = "query is required"
		return searchToolResultMessage(call.Id, result)
	}

	result.Query = query
	content, err := GenerateSearchResult(query)
	if err != nil {
		result.Error = err.Error()
		result.Content = content
		return searchToolResultMessage(call.Id, result)
	}

	result.Status = "success"
	result.Content = content
	return searchToolResultMessage(call.Id, result)
}
