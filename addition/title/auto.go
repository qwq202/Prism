package title

import (
	adaptercommon "chat/adapter/common"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"strings"
	"unicode/utf8"

	"github.com/go-redis/redis/v8"
)

const defaultEmojiPrefix = "💬"

const autoTitlePrompt = "You are a conversation title generator. Create one short conversation title in the same language as the user's conversation. The title must contain at least one emoji, either at the beginning or the end. Return only the final title text. Do not use quotes, markdown, numbering, explanations, or line breaks."

func normalizeTitle(raw string) string {
	title := strings.TrimSpace(raw)
	title = strings.Trim(title, "\"'`")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Join(strings.Fields(title), " ")

	if title == "" {
		return ""
	}

	if !containsEmoji(title) {
		title = defaultEmojiPrefix + title
	}

	if utf8.RuneCountInString(title) > 24 {
		runes := []rune(title)
		title = string(runes[:24])
	}

	return strings.TrimSpace(title)
}

func containsEmoji(value string) bool {
	for _, r := range value {
		switch {
		case r >= 0x1F300 && r <= 0x1FAFF:
			return true
		case r >= 0x2600 && r <= 0x27BF:
			return true
		case r >= 0x2300 && r <= 0x23FF:
			return true
		case r >= 0x1F100 && r <= 0x1F1FF:
			return true
		}
	}

	return false
}

func buildTitleContext(messages []globals.Message) string {
	var builder strings.Builder

	for _, message := range messages {
		if message.Role != globals.User && message.Role != globals.Assistant {
			continue
		}

		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		content = utils.Extract(content, 240, "...")
		builder.WriteString(message.Role)
		builder.WriteString(": ")
		builder.WriteString(content)
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func GenerateConversationTitle(group string, messages []globals.Message, cache *redis.Client) string {
	model := strings.TrimSpace(globals.TaskModel)
	if model == "" {
		return ""
	}

	context := buildTitleContext(messages)
	if context == "" {
		return ""
	}

	request := []globals.Message{
		{Role: globals.System, Content: autoTitlePrompt},
		{Role: globals.User, Content: context},
	}

	buffer := utils.NewBuffer(model, request, channel.ChargeInstance.GetCharge(model))
	_, err := channel.NewChatRequestWithCache(
		cache,
		buffer,
		group,
		adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
			Model:       model,
			Message:     request,
			MaxTokens:   utils.ToPtr(48),
			Temperature: utils.ToPtr[float32](0.2),
		}, buffer),
		func(data *globals.Chunk) error {
			buffer.WriteChunk(data)
			return nil
		},
	)
	if err != nil {
		globals.Warn("[title] failed to generate conversation title with task model " + model + ": " + err.Error())
		return ""
	}

	title := normalizeTitle(buffer.Read())
	if title == "" {
		return ""
	}

	globals.Debug("[title] generated conversation title: " + title)
	return title
}
