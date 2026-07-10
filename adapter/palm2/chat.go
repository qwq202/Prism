package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var geminiMaxImages = 16

const defaultVertexAIExpressEndpoint = "https://aiplatform.googleapis.com"
const defaultGeminiInteractionAPIVersion = "v1beta"

func getGeminiAPIVersion(model string) string {
	if globals.IsGeminiImageGenerationModel(model) {
		return "v1beta"
	}

	if strings.Contains(model, "preview") || strings.Contains(model, "exp") || strings.Contains(model, "latest") {
		return "v1beta"
	}

	return "v1"
}

func getVertexAIExpressAPIVersion(model string) string {
	if getGeminiAPIVersion(model) == "v1beta" {
		return "v1beta1"
	}

	return "v1"
}

func getVertexAIExpressModelPath(model string) string {
	model = strings.Trim(strings.TrimSpace(model), "/")
	if strings.HasPrefix(model, "publishers/") || strings.HasPrefix(model, "projects/") {
		return model
	}

	model = strings.TrimPrefix(model, "models/")
	return fmt.Sprintf("publishers/google/models/%s", model)
}

func (c *ChatInstance) GetVertexAIExpressChatEndpoint(model string, stream bool) string {
	endpoint := strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	if endpoint == "" {
		endpoint = defaultVertexAIExpressEndpoint
	}

	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}

	values := url.Values{}
	if stream {
		values.Set("alt", "sse")
	}
	values.Set("key", c.ApiKey)

	return fmt.Sprintf(
		"%s/%s/%s:%s?%s",
		endpoint,
		getVertexAIExpressAPIVersion(model),
		getVertexAIExpressModelPath(model),
		action,
		values.Encode(),
	)
}

func (c *ChatInstance) useGeminiInteractions(model string) bool {
	return !c.VertexAIExpress && globals.IsGeminiImageGenerationModel(model)
}

func (c *ChatInstance) GetGeminiInteractionsEndpoint() string {
	return fmt.Sprintf(
		"%s/%s/interactions",
		strings.TrimRight(strings.TrimSpace(c.Endpoint), "/"),
		defaultGeminiInteractionAPIVersion,
	)
}

func (c *ChatInstance) GetGeminiGenerateContentEndpoint(model string, stream bool) string {
	if model == globals.ChatBison001 {
		return fmt.Sprintf("%s/v1beta2/models/%s:generateMessage?key=%s", c.Endpoint, model, c.ApiKey)
	}

	version := getGeminiAPIVersion(model)

	if stream {
		return fmt.Sprintf("%s/%s/models/%s:streamGenerateContent?alt=sse&key=%s", c.Endpoint, version, model, c.ApiKey)
	}

	return fmt.Sprintf("%s/%s/models/%s:generateContent?key=%s", c.Endpoint, version, model, c.ApiKey)
}

func (c *ChatInstance) GetChatEndpoint(model string, stream bool) string {
	if c.VertexAIExpress {
		return c.GetVertexAIExpressChatEndpoint(model, stream)
	}

	if c.useGeminiInteractions(model) {
		return c.GetGeminiInteractionsEndpoint()
	}

	return c.GetGeminiGenerateContentEndpoint(model, stream)
}

func (c *ChatInstance) ConvertMessage(message []globals.Message) []PalmMessage {
	var result []PalmMessage
	for i, item := range message {
		if len(item.Content) == 0 {
			// palm model: message must include non empty content
			continue
		}

		if item.Role == globals.Tool {
			continue
		}

		if i > 0 && item.Role == result[len(result)-1].Author {
			// palm model: messages must alternate between authors
			result[len(result)-1].Content += " " + item.Content
			continue
		}

		result = append(result, PalmMessage{
			Author:  item.Role,
			Content: item.Content,
		})
	}
	return result
}

func (c *ChatInstance) GetPalm2ChatBody(props *adaptercommon.ChatProps) *PalmChatBody {
	return &PalmChatBody{
		Prompt: PalmPrompt{
			Messages: c.ConvertMessage(props.Message),
		},
	}
}

func (c *ChatInstance) GetGeminiChatBody(props *adaptercommon.ChatProps) *GeminiChatBody {
	config := GeminiConfig{
		Temperature:     props.Temperature,
		MaxOutputTokens: props.MaxTokens,
		TopP:            props.TopP,
		TopK:            props.TopK,
		ThinkingConfig:  getGeminiThinkingConfig(props),
	}
	if globals.IsGeminiImageGenerationModel(props.Model) {
		config.ResponseModalities = []string{"TEXT", "IMAGE"}
	}

	return &GeminiChatBody{
		SystemInstruction: c.GetGeminiSystemInstruction(props.Model, props.Message),
		Contents:          c.GetGeminiContents(props.Model, props.Message),
		CachedContent:     getGeminiCachedContent(props),
		Tools:             mergeGeminiTools(getGeminiBuiltinTools(props.EnableWebSearch, props.EnableURLContext), getGeminiTools(props.Tools)),
		ToolConfig:        getGeminiToolConfig(props.ToolChoice),
		GenerationConfig:  config,
	}
}

func getGeminiCachedContent(props *adaptercommon.ChatProps) string {
	if props == nil {
		return ""
	}

	for _, value := range []*string{props.CachedContent, props.CachedContentSnake} {
		if value == nil {
			continue
		}
		if normalized := strings.TrimSpace(*value); normalized != "" {
			return normalized
		}
	}
	return ""
}

func getStringFromMap(data interface{}, keys ...string) string {
	values, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}

	for _, key := range keys {
		if value, ok := values[key]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}

	return ""
}

func normalizeGeminiInteractionMimeType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image/jpeg", "image/jpg", "jpeg", "jpg":
		return "image/jpeg"
	default:
		return "image/png"
	}
}

type geminiInteractionImageModel string

const (
	geminiInteraction25FlashImage geminiInteractionImageModel = "gemini-2.5-flash-image"
	geminiInteraction31FlashImage geminiInteractionImageModel = "gemini-3.1-flash-image"
	geminiInteraction3ProImage    geminiInteractionImageModel = "gemini-3-pro-image"
)

var gemini25FlashInteractionAspectRatios = map[string]bool{
	"1:1": true, "2:3": true, "3:2": true, "3:4": true,
	"4:3": true, "4:5": true, "5:4": true, "9:16": true,
	"16:9": true, "21:9": true,
}

var gemini31FlashInteractionAspectRatios = map[string]bool{
	"1:1": true, "1:4": true, "1:8": true, "2:3": true,
	"3:2": true, "3:4": true, "4:1": true, "4:3": true,
	"4:5": true, "5:4": true, "8:1": true, "9:16": true,
	"16:9": true, "21:9": true,
}

var gemini31FlashInteractionImageSizes = map[string]bool{
	"512px": true, "1K": true, "2K": true, "4K": true,
}

var gemini3ProInteractionImageSizes = map[string]bool{
	"1K": true, "2K": true, "4K": true,
}

func getGeminiInteractionImageModel(model string) geminiInteractionImageModel {
	normalized := strings.ToLower(strings.TrimSpace(model))
	switch {
	case normalized == globals.Gemini31FlashImage || strings.Contains(normalized, globals.Gemini31FlashImage):
		return geminiInteraction31FlashImage
	case normalized == globals.Gemini3ProImage || strings.Contains(normalized, globals.Gemini3ProImage):
		return geminiInteraction3ProImage
	case normalized == globals.Gemini25FlashImage || strings.Contains(normalized, globals.Gemini25FlashImage):
		return geminiInteraction25FlashImage
	default:
		return ""
	}
}

func geminiInteractionAspectRatiosForModel(model string) map[string]bool {
	switch getGeminiInteractionImageModel(model) {
	case geminiInteraction31FlashImage:
		return gemini31FlashInteractionAspectRatios
	case geminiInteraction3ProImage, geminiInteraction25FlashImage:
		return gemini25FlashInteractionAspectRatios
	default:
		return gemini25FlashInteractionAspectRatios
	}
}

func normalizeGeminiInteractionAspectRatio(model string, value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "1:1"
	}

	if geminiInteractionAspectRatiosForModel(model)[normalized] {
		return normalized
	}

	return "1:1"
}

func geminiInteractionImageSizesForModel(model string) map[string]bool {
	switch getGeminiInteractionImageModel(model) {
	case geminiInteraction31FlashImage:
		return gemini31FlashInteractionImageSizes
	case geminiInteraction3ProImage:
		return gemini3ProInteractionImageSizes
	default:
		return nil
	}
}

func normalizeGeminiInteractionImageSizeValue(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "512", "512PX", "0.5K":
		return "512px"
	case "1K", "2K", "4K":
		return normalized
	default:
		return ""
	}
}

func normalizeGeminiInteractionImageSize(model string, value string) (string, bool) {
	allowed := geminiInteractionImageSizesForModel(model)
	if len(allowed) == 0 {
		return "", false
	}

	normalized := normalizeGeminiInteractionImageSizeValue(value)
	if normalized != "" && allowed[normalized] {
		return normalized, true
	}

	return "1K", true
}

func normalizeGeminiInteractionThinkingLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		return "high"
	case "minimal":
		return "minimal"
	default:
		return ""
	}
}

func geminiInteractionSupportsThinkingLevel(model string) bool {
	return getGeminiInteractionImageModel(model) == geminiInteraction31FlashImage
}

func getGeminiInteractionResponseFormat(props *adaptercommon.ChatProps) *GeminiInteractionResponseFormat {
	model := ""
	if props != nil {
		model = props.Model
	}

	responseFormat := &GeminiInteractionResponseFormat{
		Type:        "image",
		MimeType:    "image/png",
		AspectRatio: "1:1",
	}
	if imageSize, ok := normalizeGeminiInteractionImageSize(model, ""); ok {
		responseFormat.ImageSize = imageSize
	}
	if props == nil || props.ResponseFormat == nil {
		return responseFormat
	}

	responseFormat.Type = strings.ToLower(strings.TrimSpace(getStringFromMap(props.ResponseFormat, "type")))
	if responseFormat.Type != "image" {
		responseFormat.Type = "image"
	}
	responseFormat.MimeType = normalizeGeminiInteractionMimeType(getStringFromMap(props.ResponseFormat, "mime_type", "mimeType"))
	responseFormat.AspectRatio = normalizeGeminiInteractionAspectRatio(model, getStringFromMap(props.ResponseFormat, "aspect_ratio", "aspectRatio"))
	if imageSize, ok := normalizeGeminiInteractionImageSize(model, getStringFromMap(props.ResponseFormat, "image_size", "imageSize")); ok {
		responseFormat.ImageSize = imageSize
	} else {
		responseFormat.ImageSize = ""
	}

	return responseFormat
}

func getGeminiInteractionGenerationConfig(props *adaptercommon.ChatProps) *GeminiInteractionGenerationConfig {
	if props == nil || props.Thinking == nil {
		return nil
	}
	if !geminiInteractionSupportsThinkingLevel(props.Model) {
		return nil
	}

	level := normalizeGeminiInteractionThinkingLevel(getStringFromMap(props.Thinking, "thinking_level", "thinkingLevel"))
	if level == "" {
		return nil
	}

	return &GeminiInteractionGenerationConfig{ThinkingLevel: level}
}

var markdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)

func getGeminiInteractionInput(props *adaptercommon.ChatProps) string {
	if props == nil || len(props.Message) == 0 {
		return ""
	}

	prompt := strings.TrimSpace(props.Message[len(props.Message)-1].Content)
	textOnly := strings.TrimSpace(markdownImagePattern.ReplaceAllString(prompt, ""))
	if textOnly != "" {
		return textOnly
	}

	return prompt
}

func (c *ChatInstance) GetGeminiInteractionBody(props *adaptercommon.ChatProps) *GeminiInteractionBody {
	return &GeminiInteractionBody{
		Model:            props.Model,
		Input:            getGeminiInteractionInput(props),
		ResponseFormat:   getGeminiInteractionResponseFormat(props),
		GenerationConfig: getGeminiInteractionGenerationConfig(props),
	}
}

func getGeminiImageGenerateContentThinkingConfig(props *adaptercommon.ChatProps) *GeminiThinkingConfig {
	if props == nil || props.Thinking == nil {
		return nil
	}

	if !geminiInteractionSupportsThinkingLevel(props.Model) {
		return nil
	}

	level := normalizeGeminiInteractionThinkingLevel(getStringFromMap(props.Thinking, "thinking_level", "thinkingLevel"))
	if level == "" {
		return nil
	}

	return &GeminiThinkingConfig{ThinkingLevel: utils.ToPtr(level)}
}

func getGeminiImageGenerateContentResponseFormat(props *adaptercommon.ChatProps) *GeminiResponseFormat {
	if props == nil {
		return nil
	}

	format := getGeminiInteractionResponseFormat(props)
	if format == nil {
		return nil
	}

	image := &GeminiImageConfig{
		AspectRatio: format.AspectRatio,
		ImageSize:   format.ImageSize,
	}
	if image.AspectRatio == "" && image.ImageSize == "" {
		return nil
	}

	return &GeminiResponseFormat{Image: image}
}

func geminiInteractionsEndpointKey(endpoint string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

func (c *ChatInstance) geminiInteractionsUnsupported(props *adaptercommon.ChatProps) bool {
	if props == nil || len(props.GeminiInteractionsUnsupportedEndpoints) == 0 {
		return false
	}
	return props.GeminiInteractionsUnsupportedEndpoints[geminiInteractionsEndpointKey(c.Endpoint)]
}

func (c *ChatInstance) markGeminiInteractionsUnsupported(props *adaptercommon.ChatProps) {
	if props == nil {
		return
	}
	if props.GeminiInteractionsUnsupportedEndpoints == nil {
		props.GeminiInteractionsUnsupportedEndpoints = map[string]bool{}
	}
	props.GeminiInteractionsUnsupportedEndpoints[geminiInteractionsEndpointKey(c.Endpoint)] = true
}

func (c *ChatInstance) GetGeminiImageGenerateContentBody(props *adaptercommon.ChatProps) *GeminiChatBody {
	config := GeminiConfig{
		Temperature:        props.Temperature,
		MaxOutputTokens:    props.MaxTokens,
		TopP:               props.TopP,
		TopK:               props.TopK,
		ThinkingConfig:     getGeminiImageGenerateContentThinkingConfig(props),
		ResponseModalities: []string{"TEXT", "IMAGE"},
		ResponseFormat:     getGeminiImageGenerateContentResponseFormat(props),
	}

	content := ""
	if props != nil && len(props.Message) > 0 {
		content = props.Message[len(props.Message)-1].Content
	}

	return &GeminiChatBody{
		Contents: []GeminiContent{
			{
				Role:  GeminiUserType,
				Parts: getGeminiContent(nil, content, props.Model),
			},
		},
		GenerationConfig: config,
	}
}

func (c *ChatInstance) GetPalm2ChatResponse(data interface{}) (string, error) {
	if form := utils.MapToStruct[PalmChatResponse](data); form != nil {
		if len(form.Candidates) == 0 {
			return "", fmt.Errorf("palm2 error: the content violates content policy")
		}
		return form.Candidates[0].Content, nil
	}
	return "", fmt.Errorf("palm2 error: cannot parse response")
}

func appendGeminiInteractionImage(builder *strings.Builder, image GeminiInteractionImage) {
	data := strings.TrimSpace(image.Data)
	url := strings.TrimSpace(image.URL)
	if data == "" && url == "" {
		return
	}

	if builder.Len() > 0 {
		builder.WriteString("\n\n")
	}

	if url != "" {
		builder.WriteString("![image](")
		builder.WriteString(url)
		builder.WriteString(")")
		return
	}

	mimeType := strings.TrimSpace(image.MimeType)
	if mimeType == "" {
		mimeType = strings.TrimSpace(image.MimeTypeCamel)
	}
	if mimeType == "" {
		mimeType = "image/png"
	}

	builder.WriteString("![image](data:")
	builder.WriteString(mimeType)
	builder.WriteString(";base64,")
	builder.WriteString(data)
	builder.WriteString(")")
}

func appendGeminiInteractionContent(builder *strings.Builder, content GeminiInteractionContent) {
	contentType := strings.ToLower(strings.TrimSpace(content.Type))
	text := strings.TrimSpace(content.Text)
	if text != "" && (contentType == "" || strings.Contains(contentType, "text")) {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(text)
		return
	}

	if content.Data != "" || content.URL != "" || strings.Contains(contentType, "image") {
		appendGeminiInteractionImage(builder, GeminiInteractionImage{
			Data:          content.Data,
			MimeType:      content.MimeType,
			MimeTypeCamel: content.MimeTypeCamel,
			URL:           content.URL,
		})
	}
}

type geminiInteractionCollector struct {
	builder strings.Builder
	images  map[string]bool
}

func (c *geminiInteractionCollector) appendText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if c.builder.Len() > 0 {
		c.builder.WriteString("\n\n")
	}
	c.builder.WriteString(text)
}

func (c *geminiInteractionCollector) appendImage(image GeminiInteractionImage) {
	data := strings.TrimSpace(image.Data)
	url := strings.TrimSpace(image.URL)
	if data == "" && url == "" {
		return
	}

	mimeType := strings.TrimSpace(image.MimeType)
	if mimeType == "" {
		mimeType = strings.TrimSpace(image.MimeTypeCamel)
	}
	if mimeType == "" {
		mimeType = "image/png"
	}

	key := url
	if key == "" {
		key = mimeType + ":" + data
	}
	if c.images == nil {
		c.images = map[string]bool{}
	}
	if c.images[key] {
		return
	}
	c.images[key] = true

	appendGeminiInteractionImage(&c.builder, GeminiInteractionImage{
		Data:     data,
		MimeType: mimeType,
		URL:      url,
	})
}

func getMapStringValue(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		for itemKey, value := range data {
			if !strings.EqualFold(itemKey, key) {
				continue
			}
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func getMapValue(data map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		for itemKey, value := range data {
			if strings.EqualFold(itemKey, key) {
				return value, true
			}
		}
	}
	return nil, false
}

func isImageLikeURL(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "data:image/") ||
		strings.Contains(value, ".png") ||
		strings.Contains(value, ".jpg") ||
		strings.Contains(value, ".jpeg") ||
		strings.Contains(value, ".webp") ||
		strings.Contains(value, ".gif")
}

func isGeminiInteractionImageObject(key string, data map[string]interface{}) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	contentType := strings.ToLower(getMapStringValue(data, "type"))
	mimeType := strings.ToLower(getMapStringValue(data, "mime_type", "mimeType", "media_type", "mediaType"))
	url := getMapStringValue(data, "url", "uri", "image_url", "imageUrl")

	return strings.Contains(key, "image") ||
		strings.Contains(key, "inline") ||
		strings.Contains(contentType, "image") ||
		strings.HasPrefix(mimeType, "image/") ||
		isImageLikeURL(url)
}

func (c *geminiInteractionCollector) collect(key string, value interface{}, imageContext bool) {
	switch typed := value.(type) {
	case []interface{}:
		for _, item := range typed {
			c.collect(key, item, imageContext)
		}
	case map[string]interface{}:
		currentImageContext := imageContext || isGeminiInteractionImageObject(key, typed)

		if text := getMapStringValue(typed, "output_text", "outputText"); text != "" {
			c.appendText(text)
		}

		contentType := strings.ToLower(getMapStringValue(typed, "type"))
		if text := getMapStringValue(typed, "text"); text != "" && (contentType == "" || strings.Contains(contentType, "text")) {
			c.appendText(text)
		}

		data := getMapStringValue(typed, "data", "b64_json", "b64Json", "base64")
		url := getMapStringValue(typed, "url", "uri", "image_url", "imageUrl")
		mimeType := getMapStringValue(typed, "mime_type", "mimeType", "media_type", "mediaType")
		if currentImageContext || data != "" && strings.HasPrefix(strings.ToLower(mimeType), "image/") || isImageLikeURL(url) {
			c.appendImage(GeminiInteractionImage{
				Data:     data,
				MimeType: mimeType,
				URL:      url,
			})
		}

		for itemKey, itemValue := range typed {
			nextImageContext := currentImageContext ||
				strings.Contains(strings.ToLower(itemKey), "image") ||
				strings.EqualFold(itemKey, "inlineData") ||
				strings.EqualFold(itemKey, "inline_data")
			c.collect(itemKey, itemValue, nextImageContext)
		}
	}
}

func collectGeminiInteractionContent(data interface{}) string {
	collector := &geminiInteractionCollector{}
	collector.collect("", data, false)
	return strings.TrimSpace(collector.builder.String())
}

func formatGeminiCompatibleError(data interface{}) string {
	form := utils.MapToStruct[GeminiCompatibleErrorResponse](data)
	if form == nil || strings.TrimSpace(form.Error.Message) == "" {
		return ""
	}

	details := make([]string, 0, 3)
	if form.Error.Status != "" {
		details = append(details, "status: "+form.Error.Status)
	}
	if form.Error.Type != "" {
		details = append(details, "type: "+form.Error.Type)
	}
	if code := strings.TrimSpace(utils.ToString(form.Error.Code)); code != "" && code != "<nil>" && code != "0" {
		details = append(details, "code: "+code)
	}

	if len(details) == 0 {
		return form.Error.Message
	}
	return fmt.Sprintf("%s (%s)", form.Error.Message, strings.Join(details, ", "))
}

func summarizeGeminiInteractionResponse(data interface{}) string {
	switch typed := data.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 12 {
			keys = append(keys[:12], fmt.Sprintf("+%d more", len(keys)-12))
		}

		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			if strings.HasPrefix(key, "+") {
				parts = append(parts, key)
				continue
			}
			value, _ := getMapValue(typed, key)
			switch nested := value.(type) {
			case map[string]interface{}:
				nestedKeys := make([]string, 0, len(nested))
				for nestedKey := range nested {
					nestedKeys = append(nestedKeys, nestedKey)
				}
				sort.Strings(nestedKeys)
				if len(nestedKeys) > 6 {
					nestedKeys = append(nestedKeys[:6], fmt.Sprintf("+%d more", len(nestedKeys)-6))
				}
				parts = append(parts, fmt.Sprintf("%s:{%s}", key, strings.Join(nestedKeys, ",")))
			case []interface{}:
				parts = append(parts, fmt.Sprintf("%s:[%d]", key, len(nested)))
			default:
				parts = append(parts, key)
			}
		}
		return "keys=" + strings.Join(parts, " ")
	case []interface{}:
		return fmt.Sprintf("array_len=%d", len(typed))
	case nil:
		return "nil"
	default:
		return fmt.Sprintf("type=%T", data)
	}
}

func (c *ChatInstance) GetGeminiInteractionChunk(data interface{}) (*globals.Chunk, error) {
	if form := utils.MapToStruct[GeminiInteractionResponse](data); form != nil {
		var builder strings.Builder
		for _, output := range form.Output {
			appendGeminiInteractionContent(&builder, output)
		}
		for _, step := range form.Steps {
			if step.Type != "" && !strings.Contains(strings.ToLower(step.Type), "output") {
				continue
			}
			for _, content := range step.Content {
				appendGeminiInteractionContent(&builder, content)
			}
		}
		if strings.TrimSpace(form.OutputText) != "" {
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}
			builder.WriteString(strings.TrimSpace(form.OutputText))
		}
		if strings.TrimSpace(form.Text) != "" {
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}
			builder.WriteString(strings.TrimSpace(form.Text))
		}
		if form.OutputImage != nil {
			appendGeminiInteractionImage(&builder, *form.OutputImage)
		}
		for _, image := range form.GeneratedImages {
			appendGeminiInteractionImage(&builder, image)
		}

		content := strings.TrimSpace(builder.String())
		if content != "" {
			return &globals.Chunk{Content: content, Usage: getGeminiInteractionUsage(form)}, nil
		}
	}

	if content := collectGeminiInteractionContent(data); content != "" {
		var usage *globals.TokenUsage
		if form := utils.MapToStruct[GeminiInteractionResponse](data); form != nil {
			usage = getGeminiInteractionUsage(form)
		}
		return &globals.Chunk{Content: content, Usage: usage}, nil
	}

	if message := formatGeminiCompatibleError(data); message != "" {
		return nil, fmt.Errorf("gemini error: %s", message)
	}

	return nil, fmt.Errorf("gemini interactions: cannot parse response (%s)", summarizeGeminiInteractionResponse(data))
}

func getGeminiInteractionUsage(form *GeminiInteractionResponse) *globals.TokenUsage {
	if form == nil {
		return nil
	}
	return getGeminiUsage(form.UsageMetadata, form.UsageMetadataSnake)
}

func getGeminiUsage(metadata ...*GeminiUsageMetadata) *globals.TokenUsage {
	for _, item := range metadata {
		usage := item.TokenUsage()
		if !usage.IsEmpty() {
			return usage
		}
	}
	return nil
}

func (c *ChatInstance) buildGeminiChunk(model string, parts []GeminiChatPart, stream bool, usage ...*globals.TokenUsage) *globals.Chunk {
	content := c.GetGeminiChatText(model, parts)
	if stream {
		content = c.GetGeminiStreamText(model, parts)
	}
	var tokenUsage *globals.TokenUsage
	if len(usage) > 0 {
		tokenUsage = usage[0]
	}

	return &globals.Chunk{
		Content:              content,
		ToolCall:             getGeminiToolCalls(parts),
		GeminiHiddenMetadata: getGeminiHiddenMetadataFromParts(parts),
		Usage:                tokenUsage,
	}
}

func (c *ChatInstance) GetGeminiChunk(model string, data interface{}) (*globals.Chunk, error) {
	if form := utils.MapToStruct[GeminiChatResponse](data); form != nil {
		usage := getGeminiUsage(form.UsageMetadata, form.UsageMetadataSnake)
		if len(form.Candidates) != 0 {
			parts := form.Candidates[0].Content.Parts
			return c.buildGeminiChunk(model, parts, false, usage), nil
		}
		if !usage.IsEmpty() {
			return &globals.Chunk{Usage: usage}, nil
		}
	}

	if message := formatGeminiCompatibleError(data); message != "" {
		return nil, fmt.Errorf("gemini error: %s", message)
	}

	return nil, fmt.Errorf("gemini: cannot parse response")
}

func (c *ChatInstance) GetGeminiChatResponse(data interface{}) (string, error) {
	chunk, err := c.GetGeminiChunk("", data)
	if err != nil {
		return "", err
	}

	return chunk.Content, nil
}

func (c *ChatInstance) CreateChatRequest(props *adaptercommon.ChatProps) (string, error) {
	uri := c.GetChatEndpoint(props.Model, false)

	if props.Model == globals.ChatBison001 {
		data, err := utils.Post(uri, map[string]string{
			"Content-Type": "application/json",
		}, c.GetPalm2ChatBody(props), props.Proxy)

		if err != nil {
			return "", fmt.Errorf("palm2 error: %s", err.Error())
		}
		return c.GetPalm2ChatResponse(data)
	}

	chunk, err := c.CreateGeminiChatRequest(props)
	if err != nil {
		return "", err
	}

	return chunk.Content, nil
}

// CreateStreamChatRequest is the stream request for palm2
func (c *ChatInstance) CreateStreamChatRequest(props *adaptercommon.ChatProps, callback globals.Hook) error {
	if globals.IsGeminiImageGenerationModel(props.Model) {
		chunk, err := c.CreateGeminiChatRequest(props)
		if err != nil {
			return err
		}
		return callback(chunk)
	}

	// Handle chat models
	if props.Model == globals.ChatBison001 {
		response, err := c.CreateChatRequest(props)
		if err != nil {
			return err
		}

		for _, item := range utils.SplitItem(response, " ") {
			if err := callback(&globals.Chunk{Content: item}); err != nil {
				return err
			}
		}
		return nil
	}

	ticks := 0
	c.isFirstReasoning = true
	c.isReasonOver = false
	c.pendingThoughtImage = ""
	c.geminiStreamHasFinalImage = false
	scanErr := utils.EventScanner(&utils.EventScannerProps{
		Method: "POST",
		Uri:    c.GetChatEndpoint(props.Model, true),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: c.GetGeminiChatBody(props),
		Callback: func(data string) error {
			ticks += 1

			if form := utils.UnmarshalForm[GeminiStreamResponse](data); form != nil {
				usage := getGeminiUsage(form.UsageMetadata, form.UsageMetadataSnake)
				if len(form.Candidates) != 0 && len(form.Candidates[0].Content.Parts) != 0 {
					parts := form.Candidates[0].Content.Parts
					return callback(c.buildGeminiChunk(props.Model, parts, true, usage))
				}
				if !usage.IsEmpty() {
					return callback(&globals.Chunk{Usage: usage})
				}
				return nil
			}

			if form := utils.UnmarshalForm[GeminiChatErrorResponse](data); form != nil {
				return fmt.Errorf("gemini error: %s (code: %d, status: %s)", form.Error.Message, form.Error.Code, form.Error.Status)
			}

			return nil
		},
	}, props.Proxy)

	if scanErr != nil {
		if scanErr.Error != nil && strings.Contains(scanErr.Error.Error(), "status code: 404") {
			// downgrade to non-stream request
			chunk, err := c.CreateGeminiChatRequest(props)
			if err != nil {
				return err
			}
			return callback(chunk)
		}

		if scanErr.Body != "" {
			if form := utils.UnmarshalForm[GeminiChatErrorResponse](scanErr.Body); form != nil {
				return fmt.Errorf("gemini error: %s (code: %d, status: %s)", form.Error.Message, form.Error.Code, form.Error.Status)
			}
			return fmt.Errorf("gemini error: %s", scanErr.Body)
		}
		return fmt.Errorf("gemini error: %v", scanErr.Error)
	}

	if ticks == 0 {
		return errors.New("no response")
	}

	if !c.isFirstReasoning && !c.isReasonOver {
		if err := callback(&globals.Chunk{Content: "\n</think>\n\n"}); err != nil {
			return err
		}
	}
	if fallbackImage := c.takeGeminiStreamFallbackImage(); fallbackImage != "" {
		if err := callback(&globals.Chunk{Content: fallbackImage}); err != nil {
			return err
		}
	}

	return nil
}

func (c *ChatInstance) CreateGeminiChatRequest(props *adaptercommon.ChatProps) (*globals.Chunk, error) {
	if c.useGeminiInteractions(props.Model) && !c.geminiInteractionsUnsupported(props) {
		data, err := utils.Post(c.GetGeminiInteractionsEndpoint(), map[string]string{
			"Content-Type":   "application/json",
			"x-goog-api-key": c.ApiKey,
		}, c.GetGeminiInteractionBody(props), props.Proxy)

		if err != nil {
			interactionErr := fmt.Errorf("gemini interactions error: %s", err.Error())
			if shouldDowngradeGeminiInteractions(interactionErr) {
				c.markGeminiInteractionsUnsupported(props)
				return c.CreateGeminiGenerateContentRequest(props)
			}
			return nil, interactionErr
		}

		chunk, err := c.GetGeminiInteractionChunk(data)
		if err == nil {
			return chunk, nil
		}
		if shouldDowngradeGeminiInteractions(err) {
			c.markGeminiInteractionsUnsupported(props)
			globals.Debug(fmt.Sprintf(
				"[gemini] interactions unsupported for endpoint %s, downgrading to generateContent (model: %s, error: %s)",
				strings.TrimRight(strings.TrimSpace(c.Endpoint), "/"),
				props.Model,
				err.Error(),
			))
			return c.CreateGeminiGenerateContentRequest(props)
		}
		return nil, err
	}

	return c.CreateGeminiGenerateContentRequest(props)
}

func shouldDowngradeGeminiInteractions(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "invalid url") ||
		strings.Contains(message, "404") ||
		strings.Contains(message, "not found") ||
		strings.Contains(message, "unsupported") ||
		strings.Contains(message, "unknown") && strings.Contains(message, "interactions")
}

func (c *ChatInstance) CreateGeminiGenerateContentRequest(props *adaptercommon.ChatProps) (*globals.Chunk, error) {
	body := interface{}(c.GetGeminiChatBody(props))
	if globals.IsGeminiImageGenerationModel(props.Model) && !c.VertexAIExpress {
		body = c.GetGeminiImageGenerateContentBody(props)
	}

	data, err := utils.Post(c.GetGeminiGenerateContentEndpoint(props.Model, false), map[string]string{
		"Content-Type": "application/json",
	}, body, props.Proxy)

	if err != nil {
		return nil, fmt.Errorf("gemini error: %s", err.Error())
	}

	return c.GetGeminiChunk(props.Model, data)
}

func (c *ChatInstance) GetLatestPrompt(props *adaptercommon.ChatProps) string {
	if len(props.Message) == 0 {
		return ""
	}
	return props.Message[len(props.Message)-1].Content
}
