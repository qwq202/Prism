package adapter

import (
	"chat/adapter/azure"
	"chat/adapter/claude"
	adaptercommon "chat/adapter/common"
	"chat/adapter/deepseek"
	"chat/adapter/minimaxtokenplan"
	"chat/adapter/openai"
	"chat/adapter/openairesponses"
	"chat/adapter/palm2"
	"chat/adapter/xai"
	"chat/adapter/xiaomitokenplan"
	"chat/globals"
	"fmt"
)

var channelFactories = map[string]adaptercommon.FactoryCreator{
	globals.OpenAIChannelType:             openai.NewChatInstanceFromConfig,
	globals.OpenAIResponsesChannelType:    openairesponses.NewChatInstanceFromConfig,
	globals.XAIChannelType:                xai.NewChatInstanceFromConfig,
	globals.AzureOpenAIChannelType:        azure.NewChatInstanceFromConfig,
	globals.ClaudeChannelType:             claude.NewChatInstanceFromConfig,
	globals.GLMCodingPlanCNChannelType:    claude.NewChatInstanceFromConfig, // anthropic-compatible
	globals.MiniMaxTokenPlanCNChannelType: minimaxtokenplan.NewChatInstanceFromConfig,
	globals.XiaomiTokenPlanCNChannelType:  xiaomitokenplan.NewChatInstanceFromConfig,
	globals.PalmChannelType:               palm2.NewChatInstanceFromConfig,
	globals.DeepseekChannelType:           deepseek.NewChatInstanceFromConfig,
}

func createChatRequest(conf globals.ChannelConfig, props *adaptercommon.ChatProps, hook globals.Hook) error {
	props.Model = conf.GetModelReflect(props.OriginalModel)
	props.Proxy = conf.GetProxy()

	factoryType := conf.GetType()
	props.ChannelType = factoryType
	if factory, ok := channelFactories[factoryType]; ok {
		return factory(conf).CreateStreamChatRequest(props, hook)
	}

	return fmt.Errorf("unknown channel type %s (channel #%d)", conf.GetType(), conf.GetId())
}

func createVideoRequest(conf globals.ChannelConfig, props *adaptercommon.VideoProps, hook globals.Hook) error {
	props.Model = conf.GetModelReflect(props.OriginalModel)
	props.Proxy = conf.GetProxy()

	factoryType := conf.GetType()
	if creator, ok := channelFactories[factoryType]; ok {
		inst := creator(conf)
		if v, ok := inst.(adaptercommon.VideoFactory); ok {
			return v.CreateVideoRequest(props, hook)
		}
		return fmt.Errorf("video request not supported by channel type %s (channel #%d)", conf.GetType(), conf.GetId())
	}

	return fmt.Errorf("unknown channel type %s (channel #%d)", conf.GetType(), conf.GetId())
}
