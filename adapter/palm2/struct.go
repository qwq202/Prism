package palm2

import (
	factory "chat/adapter/common"
	"chat/globals"
)

type ChatInstance struct {
	Endpoint         string
	ApiKey           string
	VertexAIExpress  bool
	isFirstReasoning bool
	isReasonOver     bool
}

func (c *ChatInstance) GetApiKey() string {
	return c.ApiKey
}

func (c *ChatInstance) GetEndpoint() string {
	return c.Endpoint
}

func NewChatInstance(endpoint string, apiKey string) *ChatInstance {
	return &ChatInstance{
		Endpoint:         endpoint,
		ApiKey:           apiKey,
		isFirstReasoning: true,
	}
}

func NewVertexAIExpressChatInstance(endpoint string, apiKey string) *ChatInstance {
	return &ChatInstance{
		Endpoint:         endpoint,
		ApiKey:           apiKey,
		VertexAIExpress:  true,
		isFirstReasoning: true,
	}
}

func NewChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}

func NewVertexAIExpressChatInstanceFromConfig(conf globals.ChannelConfig) factory.Factory {
	return NewVertexAIExpressChatInstance(
		conf.GetEndpoint(),
		conf.GetRandomSecret(),
	)
}
