package globals

import "encoding/json"

type Hook func(data *Chunk) error

const (
	GeminiThoughtSignatureLimit     = 32
	GeminiThoughtSignatureMaxBytes  = 4096
	ClaudeThinkingBlockLimit        = 32
	ClaudeThinkingTextMaxBytes      = 32768
	ClaudeThinkingSignatureMaxBytes = 4096
)

type GeminiHiddenMetadata struct {
	ThoughtSignatures []string `json:"thought_signatures,omitempty"`
}

type ClaudeThinkingBlock struct {
	Thinking  string `json:"thinking,omitempty"`
	Signature string `json:"signature,omitempty"`
}

type ClaudeHiddenMetadata struct {
	ThinkingBlocks []ClaudeThinkingBlock `json:"thinking_blocks,omitempty"`
}

func (m *GeminiHiddenMetadata) UnmarshalJSON(data []byte) error {
	type rawMetadata struct {
		ThoughtSignatures []string `json:"thought_signatures,omitempty"`
		ThoughtSignature  *string  `json:"thought_signature,omitempty"`
	}

	var raw rawMetadata
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	signatures := make([]string, 0, len(raw.ThoughtSignatures)+1)
	for _, signature := range raw.ThoughtSignatures {
		signatures = append(signatures, signature)
	}

	if raw.ThoughtSignature != nil && len(*raw.ThoughtSignature) > 0 {
		signatures = append(signatures, *raw.ThoughtSignature)
	}

	m.ThoughtSignatures = NormalizeGeminiThoughtSignatures(signatures, GeminiThoughtSignatureLimit)
	return nil
}

type Message struct {
	Role                 string                `json:"role"`
	Content              string                `json:"content"`
	Model                string                `json:"model,omitempty"`
	Name                 *string               `json:"name,omitempty"`
	FunctionCall         *FunctionCall         `json:"function_call,omitempty"`          // only `function` role
	ToolCallId           *string               `json:"tool_call_id,omitempty"`           // only `tool` role
	ToolCalls            *ToolCalls            `json:"tool_calls,omitempty"`             // only `assistant` role
	ReasoningContent     *string               `json:"reasoning_content,omitempty"`      // only for deepseek reasoner models
	GeminiHiddenMetadata *GeminiHiddenMetadata `json:"gemini_hidden_metadata,omitempty"` // hidden gemini metadata for replay
	ClaudeHiddenMetadata *ClaudeHiddenMetadata `json:"claude_hidden_metadata,omitempty"` // hidden claude thinking metadata for replay
}

type Chunk struct {
	Content              string                `json:"content"`
	ToolCall             *ToolCalls            `json:"tool_call,omitempty"`
	FunctionCall         *FunctionCall         `json:"function_call,omitempty"`
	ReasoningContent     *string               `json:"reasoning_content,omitempty"`
	GeminiHiddenMetadata *GeminiHiddenMetadata `json:"gemini_hidden_metadata,omitempty"` // hidden gemini metadata for replay
	ClaudeHiddenMetadata *ClaudeHiddenMetadata `json:"claude_hidden_metadata,omitempty"` // hidden claude thinking metadata for replay
}

type ChatSegmentToolCall struct {
	Id        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
	Result    string `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
	Status    string `json:"status"`
}

type ChatSegmentResponse struct {
	Conversation int64                `json:"conversation"`
	Quota        float32              `json:"quota"`
	Keyword      string               `json:"keyword"`
	Message      string               `json:"message"`
	Title        string               `json:"title,omitempty"`
	ToolCall     *ChatSegmentToolCall `json:"tool_call,omitempty"`
	End          bool                 `json:"end"`
	Plan         bool                 `json:"plan"`
}

type GenerationSegmentResponse struct {
	Quota   float32 `json:"quota"`
	Message string  `json:"message"`
	Hash    string  `json:"hash"`
	End     bool    `json:"end"`
	Error   string  `json:"error"`
}

type ListModels struct {
	Object string           `json:"object"`
	Data   []ListModelsItem `json:"data"`
}

type ListModelsItem struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ProxyConfig struct {
	ProxyType int    `json:"proxy_type" mapstructure:"proxytype"`
	Proxy     string `json:"proxy" mapstructure:"proxy"`
	Username  string `json:"username" mapstructure:"username"`
	Password  string `json:"password" mapstructure:"password"`
}
