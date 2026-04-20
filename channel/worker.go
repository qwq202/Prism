package channel

import (
	"chat/adapter"
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

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
			props.MaxRetries = utils.ToPtr(channel.GetRetry())
			if err = adapter.NewChatRequest(channel, props, hook); adapter.IsSkipError(err) {
				return err
			}

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

	data := buf.Read()
	toolCalls := buf.GetToolCalls()
	functionCall := buf.GetFunctionCall()
	hiddenMetadata := buf.GetGeminiHiddenMetadata()
	if data == "" && hiddenMetadata.IsEmpty() {
		return idx, false, nil
	}

	buffer.SetInputTokens(buf.CountInputToken())
	buffer.SetToolCalls(toolCalls)
	buffer.SetFunctionCall(functionCall)
	buffer.SetGeminiHiddenMetadata(hiddenMetadata)

	return idx, true, hook(&globals.Chunk{
		Content:              data,
		FunctionCall:         functionCall,
		ToolCall:             toolCalls,
		GeminiHiddenMetadata: hiddenMetadata,
	})
}

func StoreCache(cache *redis.Client, hash string, index int64, buffer *utils.Buffer) {
	key := fmt.Sprintf("chat-cache:%d:%s", index, hash)
	raw := utils.Marshal(buffer)
	expire := time.Duration(globals.CacheAcceptedExpire) * time.Second

	cache.Set(cache.Context(), key, raw, expire)
}

func NewChatRequestWithCache(cache *redis.Client, buffer *utils.Buffer, group string, props *adaptercommon.ChatProps, hook globals.Hook) (bool, error) {
	hash := utils.Md5Encrypt(utils.Marshal(props))

	if len(props.OriginalModel) == 0 {
		props.OriginalModel = props.Model
	}

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
