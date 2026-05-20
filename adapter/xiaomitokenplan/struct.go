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
	toolCalls        map[int]globals.ToolCall
	textToolCallSeq  int
	textToolBuffer   string
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
		toolCalls:        make(map[int]globals.ToolCall),
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

func (c *ChatInstance) usesOfficialEndpoint() bool {
	endpoint := strings.TrimSuffix(strings.TrimSpace(c.Endpoint), "/")
	return endpoint == defaultEndpoint
}

func (c *ChatInstance) GetHeader() map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
		"api-key":      c.GetApiKey(),
	}

	if !c.usesOfficialEndpoint() {
		headers["Authorization"] = "Bearer " + c.GetApiKey()
	}

	return headers
}
