package xiaomitokenplan

import (
	factory "chat/adapter/common"
	"chat/globals"
	"strings"
)

const defaultEndpoint = "https://token-plan-cn.xiaomimimo.com/v1"

type ChatInstance struct {
	Endpoint         string
	ApiKey           string
	isFirstReasoning bool
	isReasonOver     bool
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	return strings.TrimRight(endpoint, "/")
}

func NewChatInstance(endpoint, apiKey string) *ChatInstance {
	return &ChatInstance{
		Endpoint:         normalizeEndpoint(endpoint),
		ApiKey:           apiKey,
		isFirstReasoning: true,
	}
}

func NewChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}

func (c *ChatInstance) GetEndpoint() string {
	return c.Endpoint
}

func (c *ChatInstance) GetApiKey() string {
	return c.ApiKey
}

func (c *ChatInstance) GetHeader() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
		"api-key":      c.GetApiKey(),
	}
}
