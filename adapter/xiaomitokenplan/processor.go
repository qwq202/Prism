package xiaomitokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	textToolCallPattern      = regexp.MustCompile(`(?s)<tool_call>\s*<function=([A-Za-z0-9_.:-]+)>\s*(.*?)\s*</function>\s*</tool_call>`)
	textToolCallParamPattern = regexp.MustCompile(`(?s)<parameter=([A-Za-z0-9_.:-]+)>(.*?)</parameter>`)
	textToolCallGapPattern   = regexp.MustCompile(`[ \t]*\n[ \t]*\n[ \t]*`)
)

const maxTextToolBufferBytes = 64 * 1024

func formatMessages(props *adaptercommon.ChatProps) interface{} {
	return utils.Each[globals.Message, Message](props.Message, func(message globals.Message) Message {
		content := interface{}(message.Content)

		if message.Role == globals.User && globals.IsVisionModel(props.Model) {
			text, urls := utils.ExtractImages(message.Content, true)
			parts := make(MessageContents, 0, len(urls)+1)
			parts = append(parts, MessageContent{
				Type: "text",
				Text: &text,
			})

			for _, url := range urls {
				obj, err := utils.NewImage(url)
				if props.Buffer != nil {
					props.Buffer.AddImage(obj)
				}
				if err != nil {
					globals.Info(fmt.Sprintf("cannot process image: %s (source: %s)", err.Error(), utils.Extract(url, 24, "...")))
				}

				parts = append(parts, MessageContent{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: url,
					},
				})
			}

			content = parts
		}

		return Message{
			Role:             message.Role,
			Content:          content,
			Name:             message.Name,
			FunctionCall:     message.FunctionCall,
			ToolCallID:       message.ToolCallId,
			ToolCalls:        message.ToolCalls,
			ReasoningContent: message.ReasoningContent,
		}
	})
}

func processChatResponse(data string) *ChatStreamResponse {
	return utils.UnmarshalForm[ChatStreamResponse](data)
}

func processChatErrorResponse(data string) *ChatStreamErrorResponse {
	return utils.UnmarshalForm[ChatStreamErrorResponse](data)
}

func formatReasoningContent(reasoning *string, content string) string {
	if reasoning == nil || *reasoning == "" {
		return content
	}

	if content == "" {
		return fmt.Sprintf("<think>\n%s\n</think>", *reasoning)
	}

	return fmt.Sprintf("<think>\n%s\n</think>\n\n%s", *reasoning, content)
}

func normalizeTextToolName(name string) string {
	switch strings.TrimSpace(name) {
	case "webfetch":
		return "fetch_webpage"
	default:
		return strings.TrimSpace(name)
	}
}

func cleanExtractedText(content string) *string {
	cleaned := strings.TrimSpace(content)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func cleanTextToolVisibleText(content string) *string {
	cleaned := textToolCallGapPattern.ReplaceAllString(content, "\n")
	return cleanExtractedText(cleaned)
}

func parseTextToolCalls(content string, startIndex int) (*string, *globals.ToolCalls, int) {
	if !strings.Contains(content, "<tool_call>") {
		return &content, nil, startIndex
	}

	matches := textToolCallPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return &content, nil, startIndex
	}

	calls := make(globals.ToolCalls, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		name := normalizeTextToolName(html.UnescapeString(match[1]))
		if name == "" {
			continue
		}

		params := map[string]string{}
		for _, param := range textToolCallParamPattern.FindAllStringSubmatch(match[2], -1) {
			if len(param) < 3 {
				continue
			}

			key := strings.TrimSpace(html.UnescapeString(param[1]))
			if key == "" {
				continue
			}
			params[key] = strings.TrimSpace(html.UnescapeString(param[2]))
		}

		rawArguments, err := json.Marshal(params)
		if err != nil {
			continue
		}

		index := startIndex
		calls = append(calls, globals.ToolCall{
			Index: utils.ToPtr(index),
			Type:  "function",
			Id:    fmt.Sprintf("call_mimo_text_%d", index),
			Function: globals.ToolCallFunction{
				Name:      name,
				Arguments: string(rawArguments),
			},
		})
		startIndex++
	}

	cleaned := textToolCallPattern.ReplaceAllString(content, "")
	cleaned = textToolCallGapPattern.ReplaceAllString(cleaned, "\n")
	if len(calls) == 0 {
		return &content, nil, startIndex
	}

	return cleanExtractedText(cleaned), &calls, startIndex
}

func parseTextToolCallBlock(block string, startIndex int) (*globals.ToolCalls, int, bool) {
	_, calls, nextIndex := parseTextToolCalls(block, startIndex)
	return calls, nextIndex, calls != nil && len(*calls) > 0
}

func parseStreamingTextToolCalls(content string, pending string, startIndex int) (*string, *globals.ToolCalls, int, string) {
	if pending == "" && !strings.Contains(content, "<tool_call>") {
		return &content, nil, startIndex, ""
	}

	source := pending + content
	if len(source) > maxTextToolBufferBytes {
		return &source, nil, startIndex, ""
	}

	var visible strings.Builder
	var calls *globals.ToolCalls
	cursor := 0
	for {
		startRelative := strings.Index(source[cursor:], "<tool_call>")
		if startRelative < 0 {
			visible.WriteString(source[cursor:])
			break
		}

		start := cursor + startRelative
		visible.WriteString(source[cursor:start])

		endRelative := strings.Index(source[start:], "</tool_call>")
		if endRelative < 0 {
			return cleanTextToolVisibleText(visible.String()), calls, startIndex, source[start:]
		}

		end := start + endRelative + len("</tool_call>")
		block := source[start:end]
		blockCalls, nextIndex, ok := parseTextToolCallBlock(block, startIndex)
		if !ok {
			visible.WriteString(block)
		} else {
			calls = mergeToolCalls(calls, blockCalls)
			startIndex = nextIndex
		}
		cursor = end
	}

	return cleanTextToolVisibleText(visible.String()), calls, startIndex, ""
}

func mergeToolCalls(left *globals.ToolCalls, right *globals.ToolCalls) *globals.ToolCalls {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}

	merged := make(globals.ToolCalls, 0, len(*left)+len(*right))
	merged = append(merged, (*left)...)
	merged = append(merged, (*right)...)
	return &merged
}

func (c *ChatInstance) extractTextToolCalls(content *string) (*string, *globals.ToolCalls) {
	if content == nil {
		return nil, nil
	}

	cleaned, calls, nextIndex, pending := parseStreamingTextToolCalls(*content, c.textToolBuffer, c.textToolCallSeq)
	c.textToolCallSeq = nextIndex
	c.textToolBuffer = pending
	return cleaned, calls
}

func (c *ChatInstance) normalizeToolCalls(toolCalls *globals.ToolCalls) *globals.ToolCalls {
	if toolCalls == nil || len(*toolCalls) == 0 {
		return nil
	}

	normalized := make(globals.ToolCalls, 0, len(*toolCalls))
	for fallbackIndex, call := range *toolCalls {
		index := fallbackIndex
		if call.Index != nil {
			index = *call.Index
		} else {
			call.Index = utils.ToPtr(index)
		}

		previous := c.toolCalls[index]
		if call.Id == "" {
			call.Id = previous.Id
		}
		if call.Type == "" {
			call.Type = previous.Type
		}
		if call.Function.Name == "" {
			call.Function.Name = previous.Function.Name
		}

		c.toolCalls[index] = globals.ToolCall{
			Index: call.Index,
			Type:  call.Type,
			Id:    call.Id,
			Function: globals.ToolCallFunction{
				Name: call.Function.Name,
			},
		}

		normalized = append(normalized, call)
	}

	return &normalized
}

func (c *ChatInstance) getChoices(form *ChatStreamResponse) *globals.Chunk {
	if len(form.Choices) == 0 {
		return &globals.Chunk{Content: ""}
	}

	choice := form.Choices[0].Delta
	reasoning := choice.ReasoningContent
	toolCalls := c.normalizeToolCalls(choice.ToolCalls)
	cleanReasoning, reasoningTextToolCalls := c.extractTextToolCalls(reasoning)
	toolCalls = mergeToolCalls(toolCalls, reasoningTextToolCalls)

	contentPtr := utils.ToPtr(choice.Content)
	cleanContent, contentTextToolCalls := c.extractTextToolCalls(contentPtr)
	toolCalls = mergeToolCalls(toolCalls, contentTextToolCalls)
	content := ""
	if cleanContent != nil {
		content = *cleanContent
	}

	if !c.isFirstReasoning && !c.isReasonOver && cleanReasoning == nil {
		c.isReasonOver = true
		if content != "" {
			return &globals.Chunk{
				Content:      fmt.Sprintf("\n</think>\n\n%s", content),
				ToolCall:     toolCalls,
				FunctionCall: choice.FunctionCall,
			}
		}

		return &globals.Chunk{
			Content:      "\n</think>\n\n",
			ToolCall:     toolCalls,
			FunctionCall: choice.FunctionCall,
		}
	}

	if cleanReasoning != nil {
		if c.isFirstReasoning {
			c.isFirstReasoning = false
			content = fmt.Sprintf("<think>\n%s", *cleanReasoning)
		} else {
			content = *cleanReasoning
		}
	}

	return &globals.Chunk{
		Content:          content,
		ToolCall:         toolCalls,
		FunctionCall:     choice.FunctionCall,
		ReasoningContent: cleanReasoning,
	}
}

func (c *ChatInstance) ProcessLine(data string) (*globals.Chunk, error) {
	if form := processChatResponse(data); form != nil {
		return c.getChoices(form), nil
	}

	if form := processChatErrorResponse(data); form != nil {
		return &globals.Chunk{Content: ""}, errors.New(fmt.Sprintf("xiaomi token plan error: %s (type: %s)", form.Error.Message, form.Error.Type))
	}

	globals.Warn(fmt.Sprintf("xiaomi token plan error: cannot parse chat completion response: %s", data))
	return &globals.Chunk{Content: ""}, errors.New("parser error: cannot parse chat completion response")
}

func collectResponse(form ChatStreamResponse) (*globals.Chunk, error) {
	if len(form.Choices) == 0 {
		return nil, errors.New("xiaomi token plan error: no choices")
	}

	message := form.Choices[0].Delta
	var cleanReasoning *string
	var reasoningToolCalls *globals.ToolCalls
	nextIndex := 0
	if message.ReasoningContent != nil {
		cleanReasoning, reasoningToolCalls, nextIndex = parseTextToolCalls(*message.ReasoningContent, 0)
	}
	cleanContent, contentToolCalls, _ := parseTextToolCalls(message.Content, nextIndex)
	toolCalls := mergeToolCalls(message.ToolCalls, reasoningToolCalls)
	toolCalls = mergeToolCalls(toolCalls, contentToolCalls)

	content := ""
	if cleanContent != nil {
		content = *cleanContent
	}

	return &globals.Chunk{
		Content:          formatReasoningContent(cleanReasoning, content),
		ToolCall:         toolCalls,
		FunctionCall:     message.FunctionCall,
		ReasoningContent: cleanReasoning,
	}, nil
}
