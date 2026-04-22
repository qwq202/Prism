package claude

import (
	factory "chat/adapter/common"
	"chat/globals"
	"strings"
)

type ChatInstance struct {
	Endpoint     string
	ApiKey       string
	blockTypes   map[int]string
	thinking     map[int]*strings.Builder
	signatures   map[int]string
	toolInputs   map[int]*strings.Builder
	toolMeta     map[int]globals.ToolCall
	thinkingOpen bool
}

func NewChatInstance(endpoint, apiKey string) *ChatInstance {
	return &ChatInstance{
		Endpoint:   endpoint,
		ApiKey:     apiKey,
		blockTypes: make(map[int]string),
		thinking:   make(map[int]*strings.Builder),
		signatures: make(map[int]string),
		toolInputs: make(map[int]*strings.Builder),
		toolMeta:   make(map[int]globals.ToolCall),
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
