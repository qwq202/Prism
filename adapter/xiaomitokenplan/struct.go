package xiaomitokenplan

import (
	factory "chat/adapter/common"
	"chat/globals"
	"strings"
)

const defaultEndpoint = "https://token-plan-cn.xiaomimimo.com/v1"
const mimoDefaultEndpoint = "https://api.xiaomimimo.com/v1"

type ChatInstance struct {
	Endpoint         string
	ApiKey           string
	ErrorPrefix      string
	defaultEndpoint  string
	isFirstReasoning bool
	isReasonOver     bool
	toolCalls        map[int]globals.ToolCall
}

func normalizeEndpoint(endpoint string, defaultEndpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}

	return strings.TrimRight(endpoint, "/")
}

func newChatInstance(endpoint, apiKey string, defaultEndpoint string, errorPrefix string) *ChatInstance {
	return &ChatInstance{
		Endpoint:         normalizeEndpoint(endpoint, defaultEndpoint),
		ApiKey:           apiKey,
		ErrorPrefix:      errorPrefix,
		defaultEndpoint:  defaultEndpoint,
		isFirstReasoning: true,
		toolCalls:        make(map[int]globals.ToolCall),
	}
}

func NewChatInstance(endpoint, apiKey string) *ChatInstance {
	return newChatInstance(endpoint, apiKey, defaultEndpoint, "xiaomi token plan")
}

func NewMiMoChatInstance(endpoint, apiKey string) *ChatInstance {
	return newChatInstance(endpoint, apiKey, mimoDefaultEndpoint, "xiaomi mimo")
}

func NewChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}

func NewMiMoChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewMiMoChatInstance(
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

func (c *ChatInstance) GetErrorPrefix() string {
	if strings.TrimSpace(c.ErrorPrefix) == "" {
		return "xiaomi token plan"
	}
	return c.ErrorPrefix
}

func (c *ChatInstance) usesOfficialEndpoint() bool {
	endpoint := strings.TrimSuffix(strings.TrimSpace(c.Endpoint), "/")
	return endpoint == c.defaultEndpoint ||
		endpoint == defaultEndpoint ||
		endpoint == mimoDefaultEndpoint ||
		endpoint == "https://api.xiaomimimo.com"
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
