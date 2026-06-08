package openai

import (
	factory "chat/adapter/common"
	"chat/globals"
	"fmt"
	"net/url"
	"strings"
)

type ChatInstance struct {
	Endpoint         string
	ApiKey           string
	ExtraHeaders     map[string]string
	ErrorPrefix      string
	isFirstReasoning bool
	isReasonOver     bool
}

func (c *ChatInstance) GetEndpoint() string {
	return c.Endpoint
}

func (c *ChatInstance) GetApiKey() string {
	return c.ApiKey
}

func (c *ChatInstance) GetHeader() map[string]string {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", c.GetApiKey()),
	}

	for key, value := range c.ExtraHeaders {
		headers[key] = value
	}

	return headers
}

func (c *ChatInstance) GetErrorPrefix() string {
	if strings.TrimSpace(c.ErrorPrefix) == "" {
		return "openai"
	}
	return c.ErrorPrefix
}

func (c *ChatInstance) GetAPIEndpoint(path string) string {
	endpoint := strings.TrimRight(strings.TrimSpace(c.GetEndpoint()), "/")
	path = strings.TrimLeft(path, "/")
	if strings.HasSuffix(endpoint, "/v1") {
		return fmt.Sprintf("%s/%s", endpoint, path)
	}
	return fmt.Sprintf("%s/v1/%s", endpoint, path)
}

func NewChatInstance(endpoint, apiKey string) *ChatInstance {
	return &ChatInstance{
		Endpoint:         strings.TrimRight(strings.TrimSpace(endpoint), "/"),
		ApiKey:           apiKey,
		ErrorPrefix:      "openai",
		isFirstReasoning: true,
	}
}

func NewChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}

func normalizeOpenRouterEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "https://openrouter.ai/api/v1"
	}

	parsed, err := url.Parse(endpoint)
	if err == nil && strings.EqualFold(parsed.Host, "openrouter.ai") && (parsed.Path == "" || parsed.Path == "/") {
		return "https://openrouter.ai/api/v1"
	}

	return endpoint
}

func NewOpenRouterChatInstance(endpoint, apiKey string) *ChatInstance {
	instance := NewChatInstance(normalizeOpenRouterEndpoint(endpoint), apiKey)
	instance.ErrorPrefix = "openrouter"
	instance.ExtraHeaders = map[string]string{
		"X-OpenRouter-Title": "Prism",
	}
	return instance
}

func NewOpenRouterChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewOpenRouterChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}

func normalizeSiliconFlowEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "https://api.siliconflow.cn/v1"
	}

	return endpoint
}

func NewSiliconFlowChatInstance(endpoint, apiKey string) *ChatInstance {
	instance := NewChatInstance(normalizeSiliconFlowEndpoint(endpoint), apiKey)
	instance.ErrorPrefix = "siliconflow"
	return instance
}

func NewSiliconFlowChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewSiliconFlowChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}
