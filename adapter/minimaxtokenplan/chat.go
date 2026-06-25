package minimaxtokenplan

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	defaultTokens    = 2500
	anthropicVersion = "2023-06-01"
)

var thinkTagPattern = regexp.MustCompile(`(?s)<think>\s*.*?\s*</think>`)

func (c *ChatInstance) resetStreamState() {
	c.blockTypes = make(map[int]string)
	c.thinking = make(map[int]*strings.Builder)
	c.signatures = make(map[int]string)
	c.toolInputs = make(map[int]*strings.Builder)
	c.toolMeta = make(map[int]globals.ToolCall)
	c.thinkingOpen = false
}

func (c *ChatInstance) GetChatEndpoint() string {
	return fmt.Sprintf("%s/v1/messages", c.GetEndpoint())
}

func (c *ChatInstance) GetChatHeaders() map[string]string {
	return map[string]string{
		"content-type":      "application/json",
		"anthropic-version": anthropicVersion,
		"x-api-key":         c.GetApiKey(),
	}
}

func (c *ChatInstance) GetTokens(props *adaptercommon.ChatProps) int {
	if props.MaxTokens == nil || *props.MaxTokens <= 0 {
		return defaultTokens
	}

	return *props.MaxTokens
}

func stripThinkingMarkup(content string) string {
	content = thinkTagPattern.ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}

func parseToolInput(arguments string) interface{} {
	arguments = strings.TrimSpace(arguments)
	if arguments == "" {
		return map[string]interface{}{}
	}

	if parsed, err := utils.UnmarshalString[interface{}](arguments); err == nil {
		return parsed
	}

	return map[string]interface{}{
		"input": arguments,
	}
}

func encodeJSON(value interface{}) string {
	if value == nil {
		return ""
	}

	raw := strings.TrimSpace(utils.Marshal(value))
	if raw == "" || raw == "null" {
		return ""
	}

	return raw
}

func formatReasoningContent(reasoning string, content string) string {
	reasoning = strings.TrimSpace(reasoning)
	content = strings.TrimSpace(content)

	if reasoning == "" {
		return content
	}

	if content == "" {
		return fmt.Sprintf("<think>\n%s\n</think>", reasoning)
	}

	return fmt.Sprintf("<think>\n%s\n</think>\n\n%s", reasoning, content)
}

func getThinkingBlocks(message globals.Message) []ContentBlock {
	if message.ClaudeHiddenMetadata != nil && !message.ClaudeHiddenMetadata.IsEmpty() {
		blocks := make([]ContentBlock, 0, len(message.ClaudeHiddenMetadata.ThinkingBlocks))
		for _, block := range message.ClaudeHiddenMetadata.ThinkingBlocks {
			thinking := strings.TrimSpace(block.Thinking)
			signature := strings.TrimSpace(block.Signature)
			if thinking == "" && signature == "" {
				continue
			}

			item := ContentBlock{
				Type: "thinking",
			}
			if thinking != "" {
				item.Thinking = &thinking
			}
			if signature != "" {
				item.Signature = &signature
			}
			blocks = append(blocks, item)
		}

		if len(blocks) > 0 {
			return blocks
		}
	}

	if message.ReasoningContent == nil || strings.TrimSpace(*message.ReasoningContent) == "" {
		return nil
	}

	thinking := strings.TrimSpace(*message.ReasoningContent)
	return []ContentBlock{
		{
			Type:     "thinking",
			Thinking: &thinking,
		},
	}
}

func (c *ChatInstance) getTextBlocks(props *adaptercommon.ChatProps, message globals.Message) []ContentBlock {
	content := strings.TrimSpace(message.Content)
	if message.Role == globals.Assistant && message.ReasoningContent != nil {
		content = stripThinkingMarkup(content)
	}

	if message.Role == globals.User && globals.IsVisionModel(props.Model) {
		text, urls := utils.ExtractImages(content, true)
		blocks := make([]ContentBlock, 0, len(urls)+1)

		if strings.TrimSpace(text) != "" {
			trimmed := strings.TrimSpace(text)
			blocks = append(blocks, ContentBlock{
				Type: "text",
				Text: &trimmed,
			})
		}

		for _, url := range urls {
			obj, err := utils.NewImage(url)
			if props.Buffer != nil {
				props.Buffer.AddImage(obj)
			}
			if err != nil {
				globals.Info(fmt.Sprintf("cannot process image: %s (source: %s)", err.Error(), utils.Extract(url, 24, "...")))
			}

			image, err := getInlineBase64Image(url)
			if err != nil {
				globals.Info(fmt.Sprintf("cannot normalize image input: %s (source: %s)", err.Error(), utils.Extract(url, 24, "...")))
				continue
			}

			blocks = append(blocks, ContentBlock{
				Type: "image",
				Source: &MessageImage{
					Type:      "base64",
					MediaType: image.MIMEType,
					Data:      image.RawBase64,
				},
			})
		}

		return blocks
	}

	if content == "" {
		return nil
	}

	return []ContentBlock{
		{
			Type: "text",
			Text: &content,
		},
	}
}

func getInlineBase64Image(url string) (*adaptercommon.NormalizedImageInput, error) {
	return adaptercommon.NormalizeImageForCapability(url, adaptercommon.InlineBase64ImageInputCapability)
}

func getToolUseBlocks(toolCalls *globals.ToolCalls) []ContentBlock {
	if toolCalls == nil || len(*toolCalls) == 0 {
		return nil
	}

	blocks := make([]ContentBlock, 0, len(*toolCalls))
	for _, call := range *toolCalls {
		id := call.Id
		name := call.Function.Name
		input := parseToolInput(call.Function.Arguments)

		blocks = append(blocks, ContentBlock{
			Type:  "tool_use",
			ID:    &id,
			Name:  &name,
			Input: input,
		})
	}

	return blocks
}

func toolResultMessage(message globals.Message) *Message {
	if message.ToolCallId == nil || strings.TrimSpace(*message.ToolCallId) == "" {
		return nil
	}

	toolUseID := strings.TrimSpace(*message.ToolCallId)
	content := strings.TrimSpace(message.Content)
	return &Message{
		Role: "user",
		Content: []ContentBlock{
			{
				Type:      "tool_result",
				ToolUseID: &toolUseID,
				Content:   content,
			},
		},
	}
}

func (c *ChatInstance) GetMessages(props *adaptercommon.ChatProps) []Message {
	messages := make([]Message, 0, len(props.Message))
	for _, message := range props.Message {
		if message.Role == globals.System {
			continue
		}

		if message.Role == globals.Tool {
			if item := toolResultMessage(message); item != nil {
				messages = append(messages, *item)
			}
			continue
		}

		role := message.Role
		if role != globals.User && role != globals.Assistant {
			role = globals.User
		}

		blocks := make([]ContentBlock, 0)
		if role == globals.Assistant {
			blocks = append(blocks, getThinkingBlocks(message)...)
		}
		blocks = append(blocks, c.getTextBlocks(props, message)...)
		if role == globals.Assistant {
			blocks = append(blocks, getToolUseBlocks(message.ToolCalls)...)
		}

		if len(blocks) == 0 {
			continue
		}

		messages = append(messages, Message{
			Role:    role,
			Content: blocks,
		})
	}

	return messages
}

func (c *ChatInstance) GetSystemPrompt(props *adaptercommon.ChatProps) (prompt string) {
	for _, message := range props.Message {
		if message.Role == globals.System {
			prompt += message.Content
		}
	}
	return
}

func getTools(tools *globals.FunctionTools) []ToolDefinition {
	if tools == nil || len(*tools) == 0 {
		return nil
	}

	result := make([]ToolDefinition, 0, len(*tools))
	for _, tool := range *tools {
		result = append(result, ToolDefinition{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}
	return result
}

func normalizeToolChoice(choice *interface{}) *ToolChoice {
	if choice == nil || *choice == nil {
		return nil
	}

	switch v := (*choice).(type) {
	case string:
		value := strings.TrimSpace(v)
		if value != "" {
			return &ToolChoice{Type: value}
		}
	case map[string]interface{}:
		rawType, _ := v["type"].(string)
		rawType = strings.TrimSpace(rawType)

		if rawType == "function" {
			if function, ok := v["function"].(map[string]interface{}); ok {
				if name, ok := function["name"].(string); ok && strings.TrimSpace(name) != "" {
					name = strings.TrimSpace(name)
					return &ToolChoice{Type: "tool", Name: &name}
				}
			}
		}

		if rawType != "" {
			result := &ToolChoice{Type: rawType}
			if name, ok := v["name"].(string); ok && strings.TrimSpace(name) != "" {
				trimmed := strings.TrimSpace(name)
				result.Name = &trimmed
			}
			return result
		}
	}

	return nil
}

func (c *ChatInstance) GetChatBody(props *adaptercommon.ChatProps, stream bool) *ChatBody {
	return &ChatBody{
		Messages:    c.GetMessages(props),
		MaxTokens:   c.GetTokens(props),
		Model:       props.Model,
		System:      c.GetSystemPrompt(props),
		Stream:      stream,
		Temperature: props.Temperature,
		TopP:        props.TopP,
		TopK:        props.TopK,
		StopSeqs:    props.Stop,
		Tools:       getTools(props.Tools),
		ToolChoice:  normalizeToolChoice(props.ToolChoice),
		Thinking:    props.Thinking,
	}
}

func processChatErrorResponse(data string) *ChatErrorResponse {
	return utils.UnmarshalForm[ChatErrorResponse](data)
}

func processChatResponse(data string) *ChatResponse {
	return utils.UnmarshalForm[ChatResponse](data)
}

func processStreamEvent(data string) *StreamEvent {
	return utils.UnmarshalForm[StreamEvent](data)
}

func parseEvent(raw string) (string, string) {
	lines := strings.Split(raw, "\n")
	var eventType string
	dataLines := make([]string, 0)

	for _, line := range lines {
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	return eventType, strings.Join(dataLines, "\n")
}

func appendJSONString(builder *strings.Builder, data string) {
	if builder == nil || data == "" {
		return
	}

	builder.WriteString(data)
}

func collectResponse(form *ChatResponse) *globals.Chunk {
	if form == nil {
		return &globals.Chunk{}
	}

	content := form.Content
	texts := make([]string, 0)
	reasoningBlocks := make([]globals.ClaudeThinkingBlock, 0)
	reasoningTexts := make([]string, 0)
	toolCalls := make(globals.ToolCalls, 0)

	for idx, block := range content {
		switch block.Type {
		case "text":
			if block.Text != nil && strings.TrimSpace(*block.Text) != "" {
				texts = append(texts, *block.Text)
			}
		case "thinking":
			thinking := ""
			signature := ""
			if block.Thinking != nil {
				thinking = *block.Thinking
				if strings.TrimSpace(thinking) != "" {
					reasoningTexts = append(reasoningTexts, thinking)
				}
			}
			if block.Signature != nil {
				signature = *block.Signature
			}
			reasoningBlocks = append(reasoningBlocks, globals.ClaudeThinkingBlock{
				Thinking:  thinking,
				Signature: signature,
			})
		case "tool_use":
			id := ""
			name := ""
			if block.ID != nil {
				id = *block.ID
			}
			if block.Name != nil {
				name = *block.Name
			}
			toolCalls = append(toolCalls, toToolCall(id, name, encodeJSON(block.Input), idx))
		}
	}

	var toolCallPtr *globals.ToolCalls
	if len(toolCalls) > 0 {
		toolCallPtr = &toolCalls
	}

	reasoning := strings.TrimSpace(strings.Join(reasoningTexts, "\n\n"))
	var reasoningPtr *string
	if reasoning != "" {
		reasoningPtr = &reasoning
	}

	var hidden *globals.ClaudeHiddenMetadata
	if len(reasoningBlocks) > 0 {
		hidden = (&globals.ClaudeHiddenMetadata{
			ThinkingBlocks: reasoningBlocks,
		}).Normalized(globals.ClaudeThinkingBlockLimit)
	}

	return &globals.Chunk{
		Content:              formatReasoningContent(reasoning, strings.TrimSpace(strings.Join(texts, "\n\n"))),
		ToolCall:             toolCallPtr,
		ReasoningContent:     reasoningPtr,
		ClaudeHiddenMetadata: hidden,
		Usage:                form.Usage.TokenUsage(),
	}
}

func (c *ChatInstance) thinkingChunk(index int, closing bool) *globals.Chunk {
	builder, ok := c.thinking[index]
	if !ok {
		return &globals.Chunk{Content: ""}
	}

	thinking := strings.TrimSpace(builder.String())
	var reasoning *string
	if thinking != "" {
		reasoning = &thinking
	}

	var hidden *globals.ClaudeHiddenMetadata
	signature := strings.TrimSpace(c.signatures[index])
	if thinking != "" || signature != "" {
		hidden = (&globals.ClaudeHiddenMetadata{
			ThinkingBlocks: []globals.ClaudeThinkingBlock{
				{
					Thinking:  thinking,
					Signature: signature,
				},
			},
		}).Normalized(globals.ClaudeThinkingBlockLimit)
	}

	content := ""
	if closing && c.thinkingOpen {
		content = "\n</think>\n\n"
		c.thinkingOpen = false
	}

	return &globals.Chunk{
		Content:              content,
		ReasoningContent:     reasoning,
		ClaudeHiddenMetadata: hidden,
	}
}

func (c *ChatInstance) toolChunk(index int, partial string) *globals.Chunk {
	meta, ok := c.toolMeta[index]
	if !ok {
		return &globals.Chunk{Content: ""}
	}

	chunk := meta
	if partial != "" {
		chunk.Function.Arguments = partial
	} else if builder, ok := c.toolInputs[index]; ok {
		chunk.Function.Arguments = builder.String()
	}

	tools := globals.ToolCalls{chunk}
	return &globals.Chunk{
		ToolCall: &tools,
	}
}

func (c *ChatInstance) ProcessLine(data string) (*globals.Chunk, error) {
	eventType, payload := parseEvent(data)
	if payload == "" {
		if form := processChatErrorResponse(data); form != nil {
			return &globals.Chunk{Content: ""}, fmt.Errorf("minimax token plan error: %s (type: %s)", form.Error.Message, form.Error.Type)
		}
		return &globals.Chunk{Content: ""}, nil
	}

	if form := processChatErrorResponse(payload); form != nil && form.Error.Message != "" {
		return &globals.Chunk{Content: ""}, fmt.Errorf("minimax token plan error: %s (type: %s)", form.Error.Message, form.Error.Type)
	}

	if eventType == "" {
		if form := processChatResponse(payload); form != nil && len(form.Content) > 0 {
			return collectResponse(form), nil
		}
		return &globals.Chunk{Content: ""}, nil
	}

	form := processStreamEvent(payload)
	if form == nil {
		return &globals.Chunk{Content: ""}, nil
	}
	if form.Error != nil && form.Error.Message != "" {
		return &globals.Chunk{Content: ""}, fmt.Errorf("minimax token plan error: %s (type: %s)", form.Error.Message, form.Error.Type)
	}

	switch eventType {
	case "message_start":
		if form.Message != nil && form.Message.Usage != nil {
			return &globals.Chunk{Usage: form.Message.Usage.TokenUsage()}, nil
		}
	case "message_delta":
		if form.Usage != nil {
			return &globals.Chunk{Usage: form.Usage.TokenUsage()}, nil
		}
	case "content_block_start":
		if form.ContentBlock == nil {
			return &globals.Chunk{Content: ""}, nil
		}

		index := form.Index
		c.blockTypes[index] = form.ContentBlock.Type
		switch form.ContentBlock.Type {
		case "thinking":
			builder := &strings.Builder{}
			if form.ContentBlock.Thinking != nil {
				builder.WriteString(*form.ContentBlock.Thinking)
			}
			c.thinking[index] = builder
			if form.ContentBlock.Signature != nil {
				c.signatures[index] = strings.TrimSpace(*form.ContentBlock.Signature)
			}
		case "tool_use":
			builder := &strings.Builder{}
			initialInput := encodeJSON(form.ContentBlock.Input)
			builder.WriteString(initialInput)
			c.toolInputs[index] = builder

			id := ""
			name := ""
			if form.ContentBlock.ID != nil {
				id = *form.ContentBlock.ID
			}
			if form.ContentBlock.Name != nil {
				name = *form.ContentBlock.Name
			}
			c.toolMeta[index] = toToolCall(id, name, initialInput, index)
			if initialInput != "" || id != "" || name != "" {
				return c.toolChunk(index, initialInput), nil
			}
		}
		return &globals.Chunk{Content: ""}, nil
	case "content_block_delta":
		if form.Delta == nil {
			return &globals.Chunk{Content: ""}, nil
		}

		index := form.Index
		switch c.blockTypes[index] {
		case "text":
			if form.Delta.Text != nil {
				return &globals.Chunk{Content: *form.Delta.Text}, nil
			}
		case "thinking":
			if form.Delta.Thinking != nil {
				builder := c.thinking[index]
				if builder == nil {
					builder = &strings.Builder{}
					c.thinking[index] = builder
				}
				builder.WriteString(*form.Delta.Thinking)
				if !c.thinkingOpen {
					c.thinkingOpen = true
					return &globals.Chunk{Content: "<think>\n" + *form.Delta.Thinking}, nil
				}
				return &globals.Chunk{Content: *form.Delta.Thinking}, nil
			}
			if form.Delta.Signature != nil {
				c.signatures[index] = strings.TrimSpace(*form.Delta.Signature)
			}
		case "tool_use":
			if form.Delta.PartialJSON != nil {
				builder := c.toolInputs[index]
				if builder == nil {
					builder = &strings.Builder{}
					c.toolInputs[index] = builder
				}
				appendJSONString(builder, *form.Delta.PartialJSON)
				return c.toolChunk(index, *form.Delta.PartialJSON), nil
			}
		}
	case "content_block_stop":
		index := form.Index
		blockType := c.blockTypes[index]
		delete(c.blockTypes, index)

		switch blockType {
		case "thinking":
			chunk := c.thinkingChunk(index, true)
			delete(c.thinking, index)
			delete(c.signatures, index)
			return chunk, nil
		case "tool_use":
			delete(c.toolInputs, index)
			delete(c.toolMeta, index)
			return &globals.Chunk{Content: ""}, nil
		}
	case "message_stop":
		if c.thinkingOpen {
			c.thinkingOpen = false
			return &globals.Chunk{Content: "\n</think>"}, nil
		}
	}

	return &globals.Chunk{Content: ""}, nil
}

func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, hook globals.Hook) error {
	c.resetStreamState()

	err := utils.EventScanner(&utils.EventScannerProps{
		Method:  "POST",
		Uri:     c.GetChatEndpoint(),
		Headers: c.GetChatHeaders(),
		Body:    c.GetChatBody(props, true),
		Callback: func(data string) error {
			partial, err := c.ProcessLine(data)
			if err != nil {
				return err
			}

			return hook(partial)
		},
		FullSSE: true,
	}, props.Proxy)

	if err != nil {
		if form := processChatErrorResponse(err.Body); form != nil {
			if form.Error.Type == "" && form.Error.Message == "" {
				return errors.New(utils.ToMarkdownCode("json", err.Body))
			}

			return errors.New(fmt.Sprintf("%s (type: %s)", form.Error.Message, form.Error.Type))
		}
		return fmt.Errorf("%s\n%s", err.Error, errors.New(utils.ToMarkdownCode("json", err.Body)))
	}

	return nil
}
