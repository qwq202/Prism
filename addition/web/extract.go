package web

import (
	adaptercommon "chat/adapter/common"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const extractPrompt = "You are a search keyword extraction task model. Extract the single most useful internet search query from the user's recent message context. When the user mentions relative time such as today, yesterday, tomorrow, this week, this month, recently, latest, or current, convert it into an explicit date or date range using the provided current date and time, and keep that time information in the query. Preserve the user's topic and important entities. Prefer a search-ready query, not a vague summary phrase. Return only one concise search query in plain text. Do not explain, do not add quotes, markdown, numbering, or extra commentary."

func normalizeExtractedQuery(query string) string {
	query = strings.TrimSpace(query)
	query = strings.Trim(query, "\"'`")
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.Join(strings.Fields(query), " ")
	return query
}

func ExtractSearchQuery(group, query string, cache *redis.Client) string {
	original := strings.TrimSpace(query)
	model := globals.GetTaskModel()

	if original == "" || model == "" {
		return original
	}

	message := []globals.Message{
		{
			Role:    globals.System,
			Content: fmt.Sprintf("%s Current date and time: %s.", extractPrompt, time.Now().Format("2006-01-02 15:04:05")),
		},
		{Role: globals.User, Content: original},
	}

	buffer := utils.NewBuffer(model, message, channel.ChargeInstance.GetCharge(model))
	_, err := channel.NewChatRequestWithCache(
		cache,
		buffer,
		group,
		adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
			Model:       model,
			Message:     message,
			MaxTokens:   utils.ToPtr(64),
			Temperature: utils.ToPtr[float32](0),
		}, buffer),
		func(data *globals.Chunk) error {
			buffer.WriteChunk(data)
			return nil
		},
	)
	if err != nil {
		globals.Warn(fmt.Sprintf("[web] failed to extract search query with task model %s: %s", model, err.Error()))
		return original
	}

	extracted := normalizeExtractedQuery(buffer.Read())
	if extracted == "" {
		return original
	}

	globals.Debug(fmt.Sprintf("[web] extracted search query: %s -> %s", utils.Extract(original, 40, "..."), utils.Extract(extracted, 40, "...")))
	return extracted
}
