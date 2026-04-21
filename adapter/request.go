package adapter

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"fmt"
	"strings"
	"time"
)

func isGeminiAdapterRequest(channelType string, model string) bool {
	return channelType == globals.PalmChannelType && globals.IsGeminiModel(model)
}

func stripGeminiHiddenMetadata(messages []globals.Message) ([]globals.Message, bool) {
	sanitized := make([]globals.Message, len(messages))
	changed := false

	for idx, message := range messages {
		sanitized[idx] = message

		if message.GeminiHiddenMetadata == nil {
			continue
		}

		sanitized[idx].GeminiHiddenMetadata = nil
		changed = true
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
	if isGeminiAdapterRequest(conf.GetType(), reflectedModel) {
		return func() {}
	}

	sanitized, changed := stripGeminiHiddenMetadata(props.Message)
	if !changed {
		return func() {}
	}

	original := props.Message
	props.Message = sanitized

	return func() {
		props.Message = original
	}
}

func IsAvailableError(err error) bool {
	return err != nil && (err.Error() != "signal" && !strings.Contains(err.Error(), "signal"))
}

func IsSkipError(err error) bool {
	return err == nil || (err.Error() == "signal" || strings.Contains(err.Error(), "signal"))
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

	if IsAvailableError(err) {
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

	if IsAvailableError(err) {
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
