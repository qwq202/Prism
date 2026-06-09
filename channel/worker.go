package channel

import (
	"chat/adapter"
	adaptercommon "chat/adapter/common"
	"chat/connection"
	"chat/globals"
	"chat/utils"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

func channelStatKey(prefix string, channelId int) string {
	return fmt.Sprintf("nio:%s-%d-%s", prefix, channelId, time.Now().Format("2006-01-02"))
}

func incrChannelRequest(channelId int) {
	utils.IncrOnce(connection.Cache, channelStatKey("channel-req", channelId), time.Hour*24*7)
}

func incrChannelError(channelId int) {
	utils.IncrOnce(connection.Cache, channelStatKey("channel-err", channelId), time.Hour*24*7)
}

func cacheModelForChatProps(props *adaptercommon.ChatProps) string {
	if props == nil {
		return ""
	}

	if props.OriginalModel != "" {
		return props.OriginalModel
	}

	return props.Model
}

func stripHiddenMetadataForCache(messages []globals.Message, stripGemini bool, stripClaude bool) []globals.Message {
	if len(messages) == 0 {
		return messages
	}

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
		return messages
	}

	return sanitized
}

func cacheHashForChatProps(props *adaptercommon.ChatProps) string {
	if props == nil {
		return utils.Md5Encrypt("")
	}

	model := cacheModelForChatProps(props)
	if globals.IsGeminiModel(model) ||
		props.ChannelType == globals.ClaudeChannelType ||
		props.ChannelType == globals.GLMCodingPlanCNChannelType ||
		props.ChannelType == globals.MiniMaxTokenPlanCNChannelType {
		return utils.Md5Encrypt(marshalChatPropsForCache(props))
	}

	cloned := *props
	cloned.Message = stripHiddenMetadataForCache(props.Message, true, true)

	return utils.Md5Encrypt(marshalChatPropsForCache(&cloned))
}

type chatPropsCacheKey struct {
	*adaptercommon.ChatProps
	EnableWeb            bool   `json:"enable_web"`
	EnableWebSearch      bool   `json:"enable_web_search"`
	EnableURLContext     bool   `json:"enable_url_context"`
	EnableXSearch        bool   `json:"enable_x_search"`
	GeminiThinkingBudget *int   `json:"gemini_thinking_budget,omitempty"`
	ChannelType          string `json:"channel_type,omitempty"`
}

func marshalChatPropsForCache(props *adaptercommon.ChatProps) string {
	if props == nil {
		return ""
	}

	return utils.Marshal(chatPropsCacheKey{
		ChatProps:            props,
		EnableWeb:            props.EnableWeb,
		EnableWebSearch:      props.EnableWebSearch,
		EnableURLContext:     props.EnableURLContext,
		EnableXSearch:        props.EnableXSearch,
		GeminiThinkingBudget: props.GeminiThinkingBudget,
		ChannelType:          props.ChannelType,
	})
}

func hasToolCalls(toolCalls *globals.ToolCalls) bool {
	return toolCalls != nil && len(*toolCalls) > 0
}

func buildCacheChunk(cacheBuffer *utils.Buffer, liveBuffer *utils.Buffer) *globals.Chunk {
	if cacheBuffer == nil {
		return &globals.Chunk{}
	}

	if liveBuffer != nil {
		liveBuffer.SetInputTokens(cacheBuffer.CountInputToken())
	}

	return &globals.Chunk{
		Content:              cacheBuffer.Read(),
		FunctionCall:         cacheBuffer.GetFunctionCall(),
		ToolCall:             cacheBuffer.GetToolCalls(),
		ReasoningContent:     cacheBuffer.GetReasoningContent(),
		GeminiHiddenMetadata: cacheBuffer.GetGeminiHiddenMetadata(),
		ClaudeHiddenMetadata: cacheBuffer.GetClaudeHiddenMetadata(),
		Usage:                cacheBuffer.GetUsage(),
	}
}

func NewChatRequest(group string, props *adaptercommon.ChatProps, hook globals.Hook) error {
	ticker := ConduitInstance.GetTicker(props.OriginalModel, group)
	if ticker == nil || ticker.IsEmpty() {
		return fmt.Errorf("cannot find channel for model %s", props.OriginalModel)
	}

	var err error
	for !ticker.IsDone() {
		if channel := ticker.Next(); channel != nil {
			if props.Buffer != nil {
				props.Buffer.SetChannel(channel.GetId(), channel.GetName())
			}
			props.Current = 0
			props.MaxRetries = utils.ToPtr(channel.GetRetry())
			if err = adapter.NewChatRequest(channel, props, hook); adapter.IsSkipError(err) {
				incrChannelRequest(channel.GetId())
				return err
			}

			incrChannelError(channel.GetId())
			globals.Warn(fmt.Sprintf("[channel] caught error %s for model %s at channel %s", err.Error(), props.OriginalModel, channel.GetName()))
		}
	}

	globals.Info(fmt.Sprintf("[channel] channels are exhausted for model %s", props.OriginalModel))

	if err == nil {
		err = fmt.Errorf("channels are exhausted for model %s", props.OriginalModel)
	}

	return err
}

func PreflightCache(cache *redis.Client, model string, hash string, buffer *utils.Buffer, hook globals.Hook) (int64, bool, error) {
	if !utils.Contains(model, globals.CacheAcceptedModels) {
		return 0, false, nil
	}

	idx := utils.Intn64(globals.CacheAcceptedSize)
	key := fmt.Sprintf("chat-cache:%d:%s", idx, hash)

	raw, err := cache.Get(cache.Context(), key).Result()
	if err != nil {
		return idx, false, nil
	}

	buf, err := utils.UnmarshalString[utils.Buffer](raw)
	if err != nil {
		return idx, false, nil
	}

	chunk := buildCacheChunk(&buf, buffer)
	data := chunk.Content
	toolCalls := chunk.ToolCall
	functionCall := chunk.FunctionCall
	reasoningContent := chunk.ReasoningContent
	hiddenMetadata := chunk.GeminiHiddenMetadata
	claudeHiddenMetadata := chunk.ClaudeHiddenMetadata
	if data == "" &&
		!hasToolCalls(toolCalls) &&
		functionCall == nil &&
		reasoningContent == nil &&
		hiddenMetadata.IsEmpty() &&
		claudeHiddenMetadata.IsEmpty() {
		return idx, false, nil
	}

	return idx, true, hook(chunk)
}

func StoreCache(cache *redis.Client, hash string, index int64, buffer *utils.Buffer) {
	key := fmt.Sprintf("chat-cache:%d:%s", index, hash)
	raw := utils.Marshal(buffer)
	expire := time.Duration(globals.CacheAcceptedExpire) * time.Second

	cache.Set(cache.Context(), key, raw, expire)
}

func NewChatRequestWithCache(cache *redis.Client, buffer *utils.Buffer, group string, props *adaptercommon.ChatProps, hook globals.Hook) (bool, error) {
	if len(props.OriginalModel) == 0 {
		props.OriginalModel = props.Model
	}

	hash := cacheHashForChatProps(props)
	idx, hit, err := PreflightCache(cache, props.OriginalModel, hash, buffer, hook)
	if hit {
		return true, err
	}

	if err = NewChatRequest(group, props, hook); err != nil {
		return false, err
	}

	StoreCache(cache, hash, idx, buffer)
	return false, nil
}

func NewVideoRequestWithCache(_ *redis.Client, buffer *utils.Buffer, group string, props *adaptercommon.VideoProps, hook globals.Hook) (bool, error) {
	// TODO: Implement video request with cache

	if len(props.OriginalModel) == 0 {
		props.OriginalModel = props.Model
	}

	ticker := ConduitInstance.GetTicker(props.OriginalModel, group)
	if ticker == nil || ticker.IsEmpty() {
		return false, fmt.Errorf("cannot find channel for model %s", props.OriginalModel)
	}

	var err error
	var times int = 0
	for !ticker.IsDone() {
		if channel := ticker.Next(); channel != nil {
			times++
			props.Current = 0
			props.MaxRetries = utils.ToPtr(channel.GetRetry())
			if err = adapter.NewVideoRequest(channel, props, hook); adapter.IsSkipError(err) {
				globals.Debug(fmt.Sprintf(
					"[channel] calling video request success (channel: %s, user: %s, model: %s, reflected-model: %s, secret: %s)",
					channel.GetName(), props.User, props.OriginalModel, props.Model,
					utils.HideSecret(channel.GetCurrentSecretValue(), 16),
				))
				return false, err
			}

			globals.Warn(fmt.Sprintf(
				"[channel] caught error: %s (channel: %s, user: %s, model: %s, reflected-model: %s, secret: %s)",
				err.Error(), channel.GetName(), props.User, props.OriginalModel, props.Model,
				utils.HideSecret(channel.GetCurrentSecretValue(), 16),
			))
		}
	}

	if err == nil {
		err = fmt.Errorf("channels are all used up (model: %s)", props.OriginalModel)
	}

	if adapter.IsAvailableError(err) {
		globals.Info(fmt.Sprintf("[channel] request failed: %s (model: %s, user: %s, attempts: %d, all channels are used up)", err.Error(), props.OriginalModel, props.User, times))
	}

	return false, err
}
