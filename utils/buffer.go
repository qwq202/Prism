package utils

import (
	"chat/globals"
	"fmt"
	"strings"
	"time"
)

type Charge interface {
	GetType() string
	GetModels() []string
	GetInput() float32
	GetOutput() float32
	SupportAnonymous() bool
	IsBilling() bool
	IsBillingType(t string) bool
	GetLimit() float32
}

type ImageBillingEstimator interface {
	EstimateImageQuota(referenceImages int, responseFormat interface{}, outputImages int) float32
	GetImageBillingDetail(referenceImages int, responseFormat interface{}, outputImages int) map[string]interface{}
}

type ImageUsageBillingEstimator interface {
	EstimateImageQuotaWithUsage(referenceImages int, responseFormat interface{}, outputImages int, usage *globals.TokenUsage) float32
	GetImageBillingDetailWithUsage(referenceImages int, responseFormat interface{}, outputImages int, usage *globals.TokenUsage) map[string]interface{}
}

type TokenCacheCharge interface {
	GetCacheHit() (float32, bool)
	GetCacheMiss() (float32, bool)
}

type Buffer struct {
	Model                string                        `json:"model"`
	Quota                float32                       `json:"quota"`
	Data                 string                        `json:"data"`
	Latest               string                        `json:"latest"`
	Cursor               int                           `json:"cursor"`
	Times                int                           `json:"times"`
	InputTokens          int                           `json:"input_tokens"`
	Images               Images                        `json:"images"`
	ToolCalls            *globals.ToolCalls            `json:"tool_calls"`
	ToolCallsCursor      int                           `json:"tool_calls_cursor"`
	FunctionCall         *globals.FunctionCall         `json:"function_call"`
	ReasoningContent     string                        `json:"reasoning_content,omitempty"`
	Usage                *globals.TokenUsage           `json:"usage,omitempty"`
	GeminiHiddenMetadata *globals.GeminiHiddenMetadata `json:"gemini_hidden_metadata,omitempty"`
	ClaudeHiddenMetadata *globals.ClaudeHiddenMetadata `json:"claude_hidden_metadata,omitempty"`
	StartTime            *time.Time                    `json:"-"`
	Prompts              string                        `json:"prompts"`
	TokenName            string                        `json:"-"`
	ChannelId            int                           `json:"-"`
	ChannelName          string                        `json:"-"`
	Charge               Charge                        `json:"-"`
	ResponseFormat       interface{}                   `json:"-"`
	ReferenceImages      int                           `json:"-"`
	VisionRecall         bool                          `json:"-"`
}

func initInputToken(model string, history []globals.Message) int {
	if globals.IsVisionModel(model) {
		for _, message := range history {
			if message.Role == globals.User {
				content, _ := ExtractImages(message.Content, true)
				message.Content = content
			}
		}

		history = Each(history, func(message globals.Message) globals.Message {
			if message.Role == globals.User {
				raw, _ := ExtractImages(message.Content, true)
				return globals.Message{
					Role:         message.Role,
					Content:      raw,
					Name:         message.Name,
					FunctionCall: message.FunctionCall,
					ToolCalls:    message.ToolCalls,
					ToolCallId:   message.ToolCallId,
				}
			}

			return message
		})
	}

	return NumTokensFromMessages(history, model, false)
}

func CountReferenceImagesFromMessages(history []globals.Message) int {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role != globals.User {
			continue
		}
		_, urls := ExtractImages(history[i].Content, true)
		return len(urls)
	}
	return 0
}

func NewBuffer(model string, history []globals.Message, charge Charge) *Buffer {
	token := initInputToken(model, history)

	return &Buffer{
		Model:           model,
		Quota:           CountInputQuota(charge, token),
		InputTokens:     token,
		ReferenceImages: CountReferenceImagesFromMessages(history),
		Charge:          charge,
		FunctionCall:    nil,
		ToolCalls:       nil,
		ToolCallsCursor: 0,
		StartTime:       ToPtr(time.Now()),
	}
}

func (b *Buffer) GetCursor() int {
	return b.Cursor
}

func (b *Buffer) GetQuota() float32 {
	return b.Quota + CountOutputToken(b.Charge, b.CountOutputToken(true))
}

func (b *Buffer) GetRecordQuota() float32 {
	if b.Charge == nil {
		return 0
	}

	if b.Charge.IsBillingType(globals.ImageBilling) {
		if estimator, ok := b.Charge.(ImageBillingEstimator); ok {
			outputImages := b.CountRecordOutputImages()
			if usageEstimator, ok := b.Charge.(ImageUsageBillingEstimator); ok {
				return usageEstimator.EstimateImageQuotaWithUsage(
					b.ReferenceImages,
					b.ResponseFormat,
					outputImages,
					b.Usage,
				)
			}
			return estimator.EstimateImageQuota(b.ReferenceImages, b.ResponseFormat, outputImages)
		}
	}

	if b.Charge.IsBillingType(globals.TokenBilling) && !b.Usage.IsEmpty() {
		return CountRecordInputQuota(b.Charge, b.CountInputToken(), b.Usage) +
			CountOutputToken(b.Charge, b.CountRecordOutputToken())
	}

	return b.Quota + CountOutputToken(b.Charge, b.CountOutputToken(false))
}

func (b *Buffer) Write(data string) string {
	b.Data += data
	b.Cursor += len(data)
	b.Times++
	b.Latest = data
	return data
}

func (b *Buffer) WriteChunk(data *globals.Chunk) string {
	if data == nil {
		return ""
	}

	b.Write(data.Content)
	b.AddToolCalls(data.ToolCall)
	b.SetFunctionCall(data.FunctionCall)
	b.AddReasoningContent(data.ReasoningContent)
	b.SetGeminiHiddenMetadata(data.GeminiHiddenMetadata)
	b.SetClaudeHiddenMetadata(data.ClaudeHiddenMetadata)
	b.SetUsage(data.Usage)

	return data.Content
}

func (b *Buffer) GetChunk() string {
	return b.Latest
}

func (b *Buffer) InitVisionRecall() {
	// set the vision recall flag to true to prevent the buffer from adding more images of retrying the channel
	b.VisionRecall = true
}

func (b *Buffer) AddImage(image *Image) {
	if image == nil || b.VisionRecall {
		return
	}

	b.Images = append(b.Images, *image)

	tokens := image.CountTokens(b.Model)
	b.InputTokens += tokens

	if b.Charge.IsBillingType(globals.TokenBilling) {
		b.Quota += float32(tokens) / 1000 * b.Charge.GetInput()
	}
}

func (b *Buffer) GetImages() Images {
	return b.Images
}

func (b *Buffer) SetToolCalls(toolCalls *globals.ToolCalls) {
	b.ToolCalls = toolCalls
}

func hitTool(tool globals.ToolCall, tools globals.ToolCalls) (int, *globals.ToolCall) {
	for i, t := range tools {
		if t.Id == tool.Id {
			return i, &t
		}
	}

	if len(tool.Type) == 0 && len(tool.Id) == 0 {
		length := len(tools)

		if length > 0 {
			// if the tool is empty, return the last tool as the hit
			return length - 1, &tools[length-1]
		}
	}

	return 0, nil
}

func appendTool(tool globals.ToolCall, chunk globals.ToolCall) string {
	from := ToString(tool.Function.Arguments)
	to := ToString(chunk.Function.Arguments)

	return from + to
}

func mixTools(source *globals.ToolCalls, target *globals.ToolCalls) *globals.ToolCalls {
	if source == nil {
		return target
	}

	tools := make(globals.ToolCalls, 0)
	arr := Collect[globals.ToolCall](*source, *target)

	for _, tool := range arr {
		idx, hit := hitTool(tool, tools)

		if hit != nil {
			tools[idx].Function.Arguments = appendTool(tools[idx], tool)
		} else {
			tools = append(tools, tool)
		}
	}

	return &tools
}

func (b *Buffer) AddToolCalls(toolCalls *globals.ToolCalls) {
	if toolCalls == nil {
		return
	}

	b.ToolCalls = mixTools(b.ToolCalls, toolCalls)
}

func (b *Buffer) SetFunctionCall(functionCall *globals.FunctionCall) {
	if functionCall == nil {
		return
	}

	b.FunctionCall = functionCall
}

func (b *Buffer) AddReasoningContent(reasoning *string) {
	if reasoning == nil || len(*reasoning) == 0 {
		return
	}

	b.ReasoningContent += *reasoning
}

func (b *Buffer) GetReasoningContent() *string {
	if len(b.ReasoningContent) == 0 {
		return nil
	}

	return ToPtr(b.ReasoningContent)
}

func cloneTokenUsage(usage *globals.TokenUsage) *globals.TokenUsage {
	if usage == nil {
		return nil
	}

	normalized := globals.NormalizeTokenUsage(usage)
	cloned := *normalized
	if normalized.PromptTokensDetails != nil {
		details := *normalized.PromptTokensDetails
		cloned.PromptTokensDetails = &details
	}
	return &cloned
}

func mergeTokenUsage(left *globals.TokenUsage, right *globals.TokenUsage) *globals.TokenUsage {
	left = globals.NormalizeTokenUsage(left)
	right = globals.NormalizeTokenUsage(right)
	if left.IsEmpty() {
		return cloneTokenUsage(right)
	}
	if right.IsEmpty() {
		return cloneTokenUsage(left)
	}

	return &globals.TokenUsage{
		PromptTokens:          left.PromptTokens + right.PromptTokens,
		CompletionTokens:      left.CompletionTokens + right.CompletionTokens,
		TotalTokens:           left.TotalTokens + right.TotalTokens,
		ImageTokens:           left.ImageTokens + right.ImageTokens,
		PromptCacheHitTokens:  left.PromptCacheHitTokens + right.PromptCacheHitTokens,
		PromptCacheMissTokens: left.PromptCacheMissTokens + right.PromptCacheMissTokens,
		PromptCacheWriteTokens: left.PromptCacheWriteTokens +
			right.PromptCacheWriteTokens,
		CompletionTokensDetails: globals.CompletionTokensDetails{
			ReasoningTokens: left.CompletionTokensDetails.ReasoningTokens + right.CompletionTokensDetails.ReasoningTokens,
		},
	}
}

func (b *Buffer) SetUsage(usage *globals.TokenUsage) {
	if usage.IsEmpty() {
		return
	}

	b.Usage = cloneTokenUsage(usage)
}

func (b *Buffer) AddUsage(usage *globals.TokenUsage) {
	if usage.IsEmpty() {
		return
	}

	b.Usage = mergeTokenUsage(b.Usage, usage)
}

func (b *Buffer) MergeUsage(source *Buffer) {
	if source == nil {
		return
	}

	b.AddUsage(source.Usage)
}

func (b *Buffer) GetUsage() *globals.TokenUsage {
	return cloneTokenUsage(b.Usage)
}

func (b *Buffer) GetBillingDetail() string {
	detail := make(map[string]interface{})
	if !b.Usage.IsEmpty() {
		detail["official_usage"] = b.Usage
	}
	if b.Charge != nil && b.Charge.IsBillingType(globals.ImageBilling) {
		if estimator, ok := b.Charge.(ImageBillingEstimator); ok {
			outputImages := b.CountRecordOutputImages()
			if usageEstimator, ok := b.Charge.(ImageUsageBillingEstimator); ok {
				detail["image_billing"] = usageEstimator.GetImageBillingDetailWithUsage(
					b.ReferenceImages,
					b.ResponseFormat,
					outputImages,
					b.Usage,
				)
			} else {
				detail["image_billing"] = estimator.GetImageBillingDetail(
					b.ReferenceImages,
					b.ResponseFormat,
					outputImages,
				)
			}
		}
	}

	if len(detail) == 0 {
		return ""
	}

	return Marshal(detail)
}

func cloneGeminiHiddenMetadata(metadata *globals.GeminiHiddenMetadata) *globals.GeminiHiddenMetadata {
	return metadata.Normalized(globals.GeminiThoughtSignatureLimit)
}

func cloneClaudeHiddenMetadata(metadata *globals.ClaudeHiddenMetadata) *globals.ClaudeHiddenMetadata {
	return metadata.Normalized(globals.ClaudeThinkingBlockLimit)
}

func (b *Buffer) SetGeminiHiddenMetadata(metadata *globals.GeminiHiddenMetadata) {
	b.GeminiHiddenMetadata = globals.MergeGeminiHiddenMetadata(
		globals.GeminiThoughtSignatureLimit,
		b.GeminiHiddenMetadata,
		metadata,
	)
}

func (b *Buffer) GetGeminiHiddenMetadata() *globals.GeminiHiddenMetadata {
	return cloneGeminiHiddenMetadata(b.GeminiHiddenMetadata)
}

func (b *Buffer) HasGeminiHiddenMetadata() bool {
	return !b.GeminiHiddenMetadata.IsEmpty()
}

func (b *Buffer) SetClaudeHiddenMetadata(metadata *globals.ClaudeHiddenMetadata) {
	b.ClaudeHiddenMetadata = globals.MergeClaudeHiddenMetadata(
		globals.ClaudeThinkingBlockLimit,
		b.ClaudeHiddenMetadata,
		metadata,
	)
}

func (b *Buffer) GetClaudeHiddenMetadata() *globals.ClaudeHiddenMetadata {
	return cloneClaudeHiddenMetadata(b.ClaudeHiddenMetadata)
}

func (b *Buffer) HasClaudeHiddenMetadata() bool {
	return !b.ClaudeHiddenMetadata.IsEmpty()
}

func (b *Buffer) GetToolCalls() *globals.ToolCalls {
	calls := b.ToolCalls
	b.ToolCalls = nil

	return calls
}

func (b *Buffer) GetFunctionCall() *globals.FunctionCall {
	return b.FunctionCall
}

func (b *Buffer) IsFunctionCalling() bool {
	return b.FunctionCall != nil || b.ToolCalls != nil
}

// HasVisiblePayload indicates whether the streamed result contains visible text
// or tool/function payload. Hidden Gemini metadata alone does not count.
func (b *Buffer) HasVisiblePayload() bool {
	return b.Cursor > 0 || b.IsFunctionCalling()
}

func (b *Buffer) HasHiddenMetadataOnly() bool {
	return !b.HasVisiblePayload() && (b.HasGeminiHiddenMetadata() || b.HasClaudeHiddenMetadata())
}

func (b *Buffer) IsEmpty() bool {
	return !b.HasVisiblePayload()
}

func (b *Buffer) GetModel() string {
	return b.Model
}

func (b *Buffer) GetCharge() Charge {
	return b.Charge
}

func (b *Buffer) ToChargeInfo() string {
	switch b.Charge.GetType() {
	case globals.TokenBilling:
		return fmt.Sprintf(
			"input tokens: %0.4f quota / 1k tokens\n"+
				"output tokens: %0.4f quota / 1k tokens\n",
			b.Charge.GetInput(), b.Charge.GetOutput(),
		)
	case globals.TimesBilling:
		return fmt.Sprintf("%f quota per request\n", b.Charge.GetLimit())
	case globals.ImageBilling:
		return fmt.Sprintf("%f quota per generated image\n", b.Charge.GetLimit())
	case globals.NonBilling:
		return "no cost"
	}

	return ""
}

func (b *Buffer) SetPrompts(prompts interface{}) {
	b.Prompts = ToString(prompts)
}

func (b *Buffer) Read() string {
	return b.Data
}

func (b *Buffer) ReadBytes() []byte {
	return []byte(b.Data)
}

func (b *Buffer) ReadWithDefault(_default string) string {
	if b.IsEmpty() || (len(strings.TrimSpace(b.Data)) == 0 && !b.IsFunctionCalling()) {
		return _default
	}
	return b.Data
}

func (b *Buffer) ReadTimes() int {
	return b.Times
}

func (b *Buffer) SetInputTokens(tokens int) {
	b.InputTokens = tokens
}

func (b *Buffer) CountInputToken() int {
	return b.InputTokens
}

func (b *Buffer) CountRecordInputToken() int {
	tokens := b.CountInputToken()
	if b.Usage.IsEmpty() {
		return tokens
	}

	usageTokens := b.Usage.PromptTokens
	if usageTokens == 0 {
		usageTokens = b.Usage.PromptCacheHitTokens +
			b.Usage.PromptCacheMissTokens +
			b.Usage.PromptCacheWriteTokens
	}
	if usageTokens > tokens {
		return usageTokens
	}
	return tokens
}

func (b *Buffer) CountOutputToken(running bool) int {
	if running {
		// performance optimization:
		// if the buffer is still running, the output token counted using the times instead
		return b.Times
	}

	return NumTokensFromResponse(b.Read(), b.Model)
}

func (b *Buffer) CountRecordOutputToken() int {
	tokens := b.CountOutputToken(false)
	if b.Usage.IsEmpty() {
		return tokens
	}

	usageTokens := b.Usage.CompletionTokens
	if usageTokens == 0 {
		promptTokens := b.Usage.PromptTokens
		if promptTokens == 0 {
			promptTokens = b.Usage.PromptCacheHitTokens +
				b.Usage.PromptCacheMissTokens +
				b.Usage.PromptCacheWriteTokens
		}
		if b.Usage.TotalTokens > promptTokens {
			usageTokens = b.Usage.TotalTokens - promptTokens
		}
	}
	if reasoningTokens := b.Usage.CompletionTokensDetails.ReasoningTokens; reasoningTokens > usageTokens {
		usageTokens = reasoningTokens
	}
	if usageTokens > tokens {
		return usageTokens
	}
	return tokens
}

func (b *Buffer) CountRecordOutputImages() int {
	if b == nil || strings.TrimSpace(b.Read()) == "" {
		return 0
	}

	_, urls := ExtractImages(b.Read(), true)
	if len(urls) > 0 {
		return len(urls)
	}
	return 1
}

func (b *Buffer) CountToken() int {
	return b.CountInputToken() + b.CountOutputToken(true)
}

func (b *Buffer) GetDuration() float32 {
	if b.StartTime == nil {
		return 0
	}

	return float32(time.Since(*b.StartTime).Seconds())
}

func (b *Buffer) GetStartTime() *time.Time {
	return b.StartTime
}

func (b *Buffer) GetPrompts() string {
	return b.Prompts
}

func (b *Buffer) GetTokenName() string {
	if len(b.TokenName) == 0 {
		return globals.WebTokenType
	}

	return b.TokenName
}

func (b *Buffer) SetTokenName(tokenName string) {
	b.TokenName = tokenName
}

func (b *Buffer) SetChannel(id int, name string) {
	b.ChannelId = id
	b.ChannelName = name
}

func (b *Buffer) SetResponseFormat(format interface{}) {
	b.ResponseFormat = format
}

func (b *Buffer) GetChannelId() int {
	return b.ChannelId
}

func (b *Buffer) GetChannelName() string {
	return b.ChannelName
}

func (b *Buffer) GetRecordPrompts() string {
	if !globals.AcceptPromptStore {
		return ""
	}

	return b.GetPrompts()
}

func (b *Buffer) GetRecordResponsePrompts() string {
	if !globals.AcceptPromptStore {
		return ""
	}

	return b.Read()
}
