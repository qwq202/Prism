package responsescompat

import "chat/globals"

type InputMessageContent struct {
	Type     string  `json:"type"`
	Text     *string `json:"text,omitempty"`
	ImageURL *string `json:"image_url,omitempty"`
	Detail   *string `json:"detail,omitempty"`
}

type InputMessage struct {
	Role    string                `json:"role"`
	Content []InputMessageContent `json:"content"`
}

type FunctionCallOutputInput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type OutputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ReasoningSummaryContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type OutputItem struct {
	ID               string                    `json:"id,omitempty"`
	Type             string                    `json:"type"`
	Role             string                    `json:"role,omitempty"`
	Content          []OutputContent           `json:"content,omitempty"`
	Summary          []ReasoningSummaryContent `json:"summary,omitempty"`
	EncryptedContent string                    `json:"encrypted_content,omitempty"`
	Name             string                    `json:"name,omitempty"`
	Arguments        string                    `json:"arguments,omitempty"`
	CallID           string                    `json:"call_id,omitempty"`
}

type InputTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
	ImageTokens  int `json:"image_tokens,omitempty"`
}

type OutputTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type ResponseUsage struct {
	InputTokens         int                 `json:"input_tokens,omitempty"`
	InputTokensDetails  *InputTokensDetails `json:"input_tokens_details,omitempty"`
	OutputTokens        int                 `json:"output_tokens,omitempty"`
	OutputTokensDetails OutputTokensDetails `json:"output_tokens_details,omitempty"`
	TotalTokens         int                 `json:"total_tokens,omitempty"`
}

func (u *ResponseUsage) TokenUsage() *globals.TokenUsage {
	if u == nil {
		return nil
	}

	var promptDetails *globals.PromptTokensDetails
	if u.InputTokensDetails != nil && u.InputTokensDetails.CachedTokens > 0 {
		promptDetails = &globals.PromptTokensDetails{
			CachedTokens: u.InputTokensDetails.CachedTokens,
			ImageTokens:  u.InputTokensDetails.ImageTokens,
		}
	} else if u.InputTokensDetails != nil && u.InputTokensDetails.ImageTokens > 0 {
		promptDetails = &globals.PromptTokensDetails{
			ImageTokens: u.InputTokensDetails.ImageTokens,
		}
	}

	return globals.NormalizeTokenUsage(&globals.TokenUsage{
		PromptTokens:        u.InputTokens,
		CompletionTokens:    u.OutputTokens,
		TotalTokens:         u.TotalTokens,
		PromptTokensDetails: promptDetails,
		CompletionTokensDetails: globals.CompletionTokensDetails{
			ReasoningTokens: u.OutputTokensDetails.ReasoningTokens,
		},
	})
}
