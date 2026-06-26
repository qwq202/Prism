package adapter

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var visibleThinkBlockPattern = regexp.MustCompile(`(?s)<think>\s*.*?\s*</think>\s*`)

func isGeminiAdapterRequest(channelType string, model string) bool {
	return (channelType == globals.PalmChannelType ||
		channelType == globals.GeminiEnterpriseAgentPlatformChannelType) &&
		globals.IsGeminiModel(model)
}

func isAnthropicAdapterRequest(channelType string) bool {
	return channelType == globals.ClaudeChannelType ||
		channelType == globals.GLMCodingPlanCNChannelType ||
		channelType == globals.MiniMaxTokenPlanCNChannelType
}

func isDeepseekAdapterRequest(channelType string) bool {
	return channelType == globals.DeepseekChannelType
}

func isXiaomiMiMoAdapterRequest(channelType string) bool {
	return channelType == globals.XiaomiMiMoChannelType ||
		channelType == globals.XiaomiTokenPlanCNChannelType
}

func stripHiddenMetadata(messages []globals.Message, stripGemini bool, stripClaude bool) ([]globals.Message, bool) {
	sanitized := make([]globals.Message, len(messages))
	changed := false

	for idx, message := range messages {
		sanitized[idx] = message

		if stripGemini && message.GeminiHiddenMetadata != nil {
			sanitized[idx].GeminiHiddenMetadata = nil
			changed = true
		}

		if stripClaude && message.ClaudeHiddenMetadata != nil {
			sanitized[idx].ClaudeHiddenMetadata = nil
			changed = true
		}
	}

	if !changed {
		return messages, false
	}

	return sanitized, true
}

func stripInternalMessageMetadata(messages []globals.Message) ([]globals.Message, bool) {
	sanitized := make([]globals.Message, len(messages))
	changed := false

	for idx, message := range messages {
		sanitized[idx] = message
		if message.ContextCleared {
			sanitized[idx].ContextCleared = false
			changed = true
		}
		if message.Quota != 0 {
			sanitized[idx].Quota = 0
			changed = true
		}
		if message.Plan {
			sanitized[idx].Plan = false
			changed = true
		}
	}

	if !changed {
		return messages, false
	}

	return sanitized, true
}

func stripVisibleThinkingReplay(messages []globals.Message, allowReasoningReplay bool) ([]globals.Message, bool) {
	sanitized := make([]globals.Message, len(messages))
	changed := false

	for idx, message := range messages {
		sanitized[idx] = message

		if message.Role != globals.Assistant {
			continue
		}

		content := strings.TrimSpace(message.Content)
		if strings.Contains(content, "<think>") || strings.Contains(content, "</think>") {
			cleaned := strings.TrimSpace(visibleThinkBlockPattern.ReplaceAllString(message.Content, ""))
			if cleaned != message.Content {
				sanitized[idx].Content = cleaned
				changed = true
			}
		}

		if !allowReasoningReplay && message.ReasoningContent != nil {
			sanitized[idx].ReasoningContent = nil
			changed = true
		}
	}

	if !changed {
		return messages, false
	}

	return sanitized, true
}

func stripOrphanedToolCalls(messages []globals.Message) ([]globals.Message, bool) {
	sanitized := make([]globals.Message, 0, len(messages))
	changed := false

	for index := 0; index < len(messages); {
		message := messages[index]

		if message.Role == globals.Assistant && message.ToolCalls != nil && len(*message.ToolCalls) > 0 {
			expected := make(map[string]struct{}, len(*message.ToolCalls))
			for _, call := range *message.ToolCalls {
				if id := strings.TrimSpace(call.Id); id != "" {
					expected[id] = struct{}{}
				}
			}

			next := index + 1
			matched := make(map[string]struct{}, len(expected))
			for next < len(messages) && messages[next].Role == globals.Tool {
				if messages[next].ToolCallId != nil {
					if id := strings.TrimSpace(*messages[next].ToolCallId); id != "" {
						if _, ok := expected[id]; ok {
							matched[id] = struct{}{}
						}
					}
				}
				next++
			}

			if len(expected) > 0 && len(matched) == len(expected) {
				sanitized = append(sanitized, message)
				sanitized = append(sanitized, messages[index+1:next]...)
				index = next
				continue
			}

			callSummary := make([]string, 0, len(*message.ToolCalls))
			for _, call := range *message.ToolCalls {
				callSummary = append(callSummary, fmt.Sprintf("%s:%s", call.Id, call.Function.Name))
			}

			globals.Debug(fmt.Sprintf(
				"[adapter] stripping orphaned assistant tool calls at index=%d calls=%v content_len=%d matched=%d expected=%d",
				index,
				callSummary,
				len(strings.TrimSpace(message.Content)),
				len(matched),
				len(expected),
			))

			if strings.TrimSpace(message.Content) != "" || message.FunctionCall != nil ||
				message.ReasoningContent != nil || message.GeminiHiddenMetadata != nil ||
				message.ClaudeHiddenMetadata != nil {
				cleaned := message
				cleaned.ToolCalls = nil
				sanitized = append(sanitized, cleaned)
			}

			changed = true
			index = next
			continue
		}

		if message.Role == globals.Tool {
			changed = true
			globals.Debug(fmt.Sprintf(
				"[adapter] dropping orphaned tool message at index=%d tool_call_id=%s",
				index,
				utils.ToString(message.ToolCallId),
			))
			index++
			continue
		}

		sanitized = append(sanitized, message)
		index++
	}

	if !changed {
		return messages, false
	}

	return sanitized, true
}

func sanitizeChatMessagesForRequest(conf globals.ChannelConfig, props *adaptercommon.ChatProps) func() {
	if props == nil || len(props.Message) == 0 {
		return func() {}
	}

	originalModel := props.OriginalModel
	if originalModel == "" {
		originalModel = props.Model
	}

	reflectedModel := conf.GetModelReflect(originalModel)
	stripGemini := !isGeminiAdapterRequest(conf.GetType(), reflectedModel)
	stripClaude := !isAnthropicAdapterRequest(conf.GetType())
	allowReasoningReplay := isAnthropicAdapterRequest(conf.GetType()) ||
		isXiaomiMiMoAdapterRequest(conf.GetType()) ||
		(isDeepseekAdapterRequest(conf.GetType()) &&
			globals.IsDeepseekReasoningReplayModel(reflectedModel) &&
			!globals.IsDeepseekThinkingDisabled(props.Thinking))

	sanitized := props.Message
	changed := false

	next, internalChanged := stripInternalMessageMetadata(sanitized)
	sanitized = next
	changed = changed || internalChanged

	if stripGemini || stripClaude {
		next, metadataChanged := stripHiddenMetadata(sanitized, stripGemini, stripClaude)
		sanitized = next
		changed = changed || metadataChanged
	}

	next, reasoningChanged := stripVisibleThinkingReplay(sanitized, allowReasoningReplay)
	sanitized = next
	changed = changed || reasoningChanged

	next, toolChanged := stripOrphanedToolCalls(sanitized)
	sanitized = next
	changed = changed || toolChanged

	if !changed {
		return func() {}
	}

	original := props.Message
	props.Message = sanitized

	return func() {
		props.Message = original
	}
}

func isClientInterruptError(err error) bool {
	if err == nil {
		return false
	}

	content := err.Error()
	return content == "signal" ||
		strings.Contains(content, "signal") ||
		content == "interrupted" ||
		strings.Contains(content, "interrupted")
}

func isNonRetryableRequestError(err error) bool {
	if err == nil {
		return false
	}

	content := strings.ToLower(err.Error())
	return strings.Contains(content, "model_not_found") ||
		strings.Contains(content, "model not found") ||
		strings.Contains(content, "no available channel for model") ||
		strings.Contains(content, "invalid_request_error") ||
		strings.Contains(content, "invalid url")
}

func IsAvailableError(err error) bool {
	return err != nil && !isClientInterruptError(err)
}

func IsRetryableError(err error) bool {
	return IsAvailableError(err) && !isNonRetryableRequestError(err)
}

func IsSkipError(err error) bool {
	return err == nil || isClientInterruptError(err)
}

func isQPSOverLimit(model string, err error) bool {
	if strings.Contains(model, "spark-desk") {
		return strings.Contains(err.Error(), "AppIdQpsOverFlowError")
	}
	return false
}

func NewChatRequest(conf globals.ChannelConfig, props *adaptercommon.ChatProps, hook globals.Hook) error {
	restore := sanitizeChatMessagesForRequest(conf, props)
	defer restore()

	err := createChatRequest(conf, props, hook)

	retries := conf.GetRetry()
	props.Current++

	if IsRetryableError(err) {
		if isQPSOverLimit(props.OriginalModel, err) {
			// sleep for 0.5s to avoid qps limit

			globals.Info(fmt.Sprintf("qps limit for %s, sleep and retry (times: %d)", props.OriginalModel, props.Current))
			time.Sleep(500 * time.Millisecond)
			return NewChatRequest(conf, props, hook)
		}

		if props.Current < retries {
			content := strings.Replace(err.Error(), "\n", "", -1)
			globals.Warn(fmt.Sprintf("retrying chat request for %s (attempt %d/%d, error: %s)", props.OriginalModel, props.Current+1, retries, content))
			return NewChatRequest(conf, props, hook)
		}
	}

	return conf.ProcessError(err)
}

func NewVideoRequest(conf globals.ChannelConfig, props *adaptercommon.VideoProps, hook globals.Hook) error {
	err := createVideoRequest(conf, props, hook)

	retries := conf.GetRetry()
	props.Current++

	if IsRetryableError(err) {
		if isQPSOverLimit(props.OriginalModel, err) {
			// sleep for 0.5s to avoid qps limit

			globals.Info(fmt.Sprintf("qps limit for %s, sleep and retry (times: %d)", props.OriginalModel, props.Current))
			time.Sleep(500 * time.Millisecond)
			return NewVideoRequest(conf, props, hook)
		}

		if props.Current < retries {
			content := strings.Replace(err.Error(), "\n", "", -1)
			globals.Info(fmt.Sprintf("retrying error request for %s (attempt %d/%d, error: %s)", props.OriginalModel, props.Current+1, retries, content))
			return NewVideoRequest(conf, props, hook)
		}
	}

	return conf.ProcessError(err)
}

func ClearMessages(model string, messages []globals.Message) []globals.Message {
	if globals.IsVisionModel(model) || globals.IsConfiguredVisionModel(model) || globals.IsXAIModel(model) {
		return messages
	}

	return utils.Each[globals.Message](messages, func(message globals.Message) globals.Message {
		if message.Role != globals.User {
			return message
		}

		images := utils.ExtractBase64Images(message.Content)
		for _, image := range images {
			if len(image) <= 46 {
				continue
			}

			message.Content = strings.Replace(message.Content, image, utils.Extract(image, 46, " ..."), -1)
		}
		return message
	})
}
