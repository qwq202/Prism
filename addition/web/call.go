package web

import (
	"chat/globals"
	"chat/manager/conversation"
	"chat/utils"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type Hook func(message []globals.Message, token int) (string, error)

func recentUserSearchContext(message []globals.Message, limit int) string {
	if limit <= 0 {
		limit = 1
	}

	items := make([]string, 0, limit)
	for i := len(message) - 1; i >= 0 && len(items) < limit; i-- {
		if message[i].Role != globals.User {
			continue
		}

		content := strings.TrimSpace(message[i].Content)
		if content == "" {
			continue
		}

		items = append(items, content)
	}

	if len(items) == 0 {
		if len(message) == 0 {
			return ""
		}
		return strings.TrimSpace(message[len(message)-1].Content)
	}

	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}

	if len(items) == 1 {
		return items[0]
	}

	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "Recent user messages:")
	for i, item := range items {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, item))
	}

	return strings.Join(lines, "\n")
}

func toWebSearchingMessage(message []globals.Message, group string, cache *redis.Client) []globals.Message {
	query := recentUserSearchContext(message, 3)
	extracted := ExtractSearchQuery(group, query, cache)
	data, _ := GenerateSearchResult(extracted)

	return utils.Insert(message, 0, globals.Message{
		Role: globals.System,
		Content: fmt.Sprintf("You will play the role of an AI Q&A assistant, where your knowledge base is not offline, but can be networked in real time, and you can provide real-time networked information with links to networked search sources."+
			"Current time: %s, Search query used: %s, Real-time internet search results: %s",
			time.Now().Format("2006-01-02 15:04:05"), extracted, data,
		),
	})
}

func ToChatSearched(instance *conversation.Conversation, restart bool, group string, cache *redis.Client) []globals.Message {
	segment := conversation.CopyMessage(instance.GetChatMessage(restart))

	if instance.IsEnableWeb() &&
		!globals.IsGeminiModel(instance.GetModel()) &&
		!globals.IsXAIModel(instance.GetModel()) &&
		!globals.IsOpenAIResponsesNativeWebModel(instance.GetModel()) {
		segment = toWebSearchingMessage(segment, group, cache)
	}

	return segment
}

func ToSearched(enable bool, model string, message []globals.Message, group string, cache *redis.Client) []globals.Message {
	if enable &&
		!globals.IsGeminiModel(model) &&
		!globals.IsXAIModel(model) &&
		!globals.IsOpenAIResponsesNativeWebModel(model) {
		return toWebSearchingMessage(message, group, cache)
	}

	return message
}
