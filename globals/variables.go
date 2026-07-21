package globals

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const ChatMaxThread = 5
const AnonymousMaxThread = 1

var HttpMaxTimeout = 30 * time.Minute

var AllowedOrigins []string

var DebugMode bool
var NotifyUrl = ""
var CacheAcceptedModels []string
var CacheAcceptedExpire int64
var CacheAcceptedSize int64
var PromptCacheEnabled = true
var PromptCacheMinTokens int
var AcceptImageStore bool
var AcceptPromptStore bool
var StorageMode = "local"
var StorageS3Endpoint string
var StorageS3Region string
var StorageS3Bucket string
var StorageS3AccessKey string
var StorageS3SecretKey string
var StorageS3PublicBaseURL string
var StorageS3ForcePathStyle bool
var StorageR2AccountID string
var StorageR2Jurisdiction string
var StorageR2Bucket string
var StorageR2AccessKey string
var StorageR2SecretKey string
var StorageR2PublicBaseURL string
var OrphanCleanupEnabled bool
var OrphanCleanupInterval int64
var CloseRegistration bool

var SearchApiKey string
var SearchCrop bool
var SearchCropLength int
var SearchMaxResults int
var SearchTopic string
var SearchDepth string
var taskModel string
var taskModelMu sync.RWMutex
var VisionModelResolver func(string) bool

func SetTaskModel(model string) {
	taskModelMu.Lock()
	defer taskModelMu.Unlock()
	taskModel = strings.TrimSpace(model)
}

func GetTaskModel() string {
	taskModelMu.RLock()
	defer taskModelMu.RUnlock()
	return taskModel
}

func IsConfiguredVisionModel(model string) bool {
	if VisionModelResolver == nil {
		return false
	}

	return VisionModelResolver(model)
}

func OriginIsAllowed(uri string) bool {
	instance, err := url.Parse(uri)
	if err != nil || instance == nil {
		return false
	}

	if isLocalOriginHostname(instance.Hostname()) || instance.Scheme == "file" {
		return true
	}

	if len(AllowedOrigins) == 0 {
		return false
	}

	if strings.HasPrefix(instance.Host, "www.") {
		instance.Host = instance.Host[4:]
	}

	return in(instance.Host, AllowedOrigins)
}

func isLocalOriginHostname(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func OriginIsOpen(c *gin.Context) bool {
	return strings.HasPrefix(c.Request.URL.Path, "/v1") || strings.HasPrefix(c.Request.URL.Path, "/dashboard")
}

const (
	GPT3Turbo                    = "gpt-3.5-turbo"
	GPT3TurboInstruct            = "gpt-3.5-turbo-instruct"
	GPT3Turbo0613                = "gpt-3.5-turbo-0613"
	GPT3Turbo0301                = "gpt-3.5-turbo-0301"
	GPT3Turbo1106                = "gpt-3.5-turbo-1106"
	GPT3Turbo0125                = "gpt-3.5-turbo-0125"
	GPT3Turbo16k                 = "gpt-3.5-turbo-16k"
	GPT3Turbo16k0613             = "gpt-3.5-turbo-16k-0613"
	GPT3Turbo16k0301             = "gpt-3.5-turbo-16k-0301"
	GPT4                         = "gpt-4"
	GPT4All                      = "gpt-4-all"
	GPT4Vision                   = "gpt-4-v"
	GPT4Dalle                    = "gpt-4-dalle"
	GPT40314                     = "gpt-4-0314"
	GPT40613                     = "gpt-4-0613"
	GPT41106Preview              = "gpt-4-1106-preview"
	GPT40125Preview              = "gpt-4-0125-preview"
	GPT4TurboPreview             = "gpt-4-turbo-preview"
	GPT4VisionPreview            = "gpt-4-vision-preview"
	GPT4Turbo                    = "gpt-4-turbo"
	GPT4Turbo20240409            = "gpt-4-turbo-2024-04-09"
	GPT41106VisionPreview        = "gpt-4-1106-vision-preview"
	GPT432k                      = "gpt-4-32k"
	GPT432k0314                  = "gpt-4-32k-0314"
	GPT432k0613                  = "gpt-4-32k-0613"
	GPT4O                        = "gpt-4o"
	GPT4O20240513                = "gpt-4o-2024-05-13"
	GPTImage1                    = "gpt-image-1"
	Sora2                        = "sora-2"
	Dalle                        = "dalle"
	Dalle2                       = "dall-e-2"
	Dalle3                       = "dall-e-3"
	Claude1                      = "claude-1"
	Claude1100k                  = "claude-1.3"
	Claude2                      = "claude-1-100k"
	Claude2100k                  = "claude-2"
	Claude2200k                  = "claude-2.1"
	Claude3                      = "claude-3"
	ChatBison001                 = "chat-bison-001"
	GeminiPro                    = "gemini-pro"
	GeminiProVision              = "gemini-pro-vision"
	Gemini15Pro002               = "gemini-1.5-pro-002"
	Gemini15Flash002             = "gemini-1.5-flash-002"
	Gemini15ProLatest            = "gemini-1.5-pro-latest"
	Gemini15FlashLatest          = "gemini-1.5-flash-latest"
	Gemini20FlashLite            = "gemini-2.0-flash-lite"
	Gemini20ProExp               = "gemini-2.0-pro-exp-02-05"
	Gemini20Flash                = "gemini-2.0-flash"
	Gemini20FlashExp             = "gemini-2.0-flash-exp"
	Gemini20Flash001             = "gemini-2.0-flash-001"
	Gemini20FlashThinkingExp     = "gemini-2.0-flash-thinking-exp-01-21"
	Gemini20FlashLitePreview     = "gemini-2.0-flash-lite-preview-02-05"
	Gemini20FlashThinkingExp1219 = "gemini-2.0-flash-thinking-exp-1219"
	Gemini25Flash                = "gemini-2.5-flash"
	Gemini25Pro                  = "gemini-2.5-pro"
	Gemini25FlashLitePreview     = "gemini-2.5-flash-lite-preview-06-17"
	Gemini25FlashImage           = "gemini-2.5-flash-image"
	Gemini35Flash                = "gemini-3.5-flash"
	Gemini36Flash                = "gemini-3.6-flash"
	Gemini3Flash                 = "gemini-3-flash"
	Gemini3ProPreview            = "gemini-3-pro-preview"
	Gemini31FlashLiteImage       = "gemini-3.1-flash-lite-image"
	Gemini31FlashImage           = "gemini-3.1-flash-image"
	Gemini3ProImage              = "gemini-3-pro-image"
	GeminiExp1206                = "gemini-exp-1206"
	DeepseekV4Flash              = "deepseek-v4-flash"
	DeepseekV4Pro                = "deepseek-v4-pro"
	MiMoV25                      = "mimo-v2.5"
	MiMoV25Pro                   = "mimo-v2.5-pro"
	MiMoV2Omni                   = "mimo-v2-omni"
	MiMoV2Pro                    = "mimo-v2-pro"
)

var OpenAIDalleModels = []string{
	Dalle, Dalle2, Dalle3, GPTImage1,
}

var GeminiImageGenerationModels = []string{
	Gemini25FlashImage, Gemini31FlashLiteImage, Gemini31FlashImage, Gemini3ProImage,
}

var VisionModels = []string{
	GPT4VisionPreview, GPT41106VisionPreview, GPT4Turbo, GPT4Turbo20240409, GPT4O, GPT4O20240513, // openai
	GeminiProVision, Gemini15Pro002, Gemini15Flash002, Gemini15ProLatest, Gemini15FlashLatest,
	Gemini20Flash, Gemini20Flash001, Gemini20FlashLite,
	Gemini25Flash, Gemini25Pro, Gemini25FlashLitePreview, "gemini-2.5-flash-lite", "gemini-2.5-flash-preview-09-2025",
	Gemini25FlashImage, Gemini35Flash, Gemini36Flash, Gemini3Flash, Gemini3ProPreview, Gemini31FlashLiteImage, Gemini31FlashImage, Gemini3ProImage, "gemini-3-flash-preview", "gemini-3.1-pro-preview", "gemini-3.1-pro-preview-customtools", "gemini-3.1-flash-lite-preview", // gemini
	Claude3,                                    // anthropic
	MiMoV25, MiMoV25Pro, MiMoV2Omni, MiMoV2Pro, // xiaomi mimo
	"grok-4.5", "grok-build-latest", // xAI
}

var VisionSkipModels = []string{
	GPT4TurboPreview,
}

var VideoModels = []string{
	Sora2,
}

func in(value string, slice []string) bool {
	for _, item := range slice {
		if item == value || strings.Contains(value, item) {
			return true
		}
	}
	return false
}

func IsOpenAIDalleModel(model string) bool {
	// using image generation api if model is in dalle models
	return in(model, OpenAIDalleModels) && !strings.Contains(model, "gpt-4-dalle")
}

func IsDrawingModel(channelType, model string) bool {
	normalizedChannel := strings.TrimSpace(strings.ToLower(channelType))
	switch normalizedChannel {
	case OpenAIChannelType, AzureOpenAIChannelType:
		return IsOpenAIDalleModel(model)
	case PalmChannelType, GeminiEnterpriseAgentPlatformChannelType:
		return IsGeminiImageGenerationModel(model)
	case "":
		return IsOpenAIDalleModel(model) || IsGeminiImageGenerationModel(model)
	default:
		return false
	}
}

func IsGeminiModel(model string) bool {
	return model == GeminiPro ||
		model == GeminiProVision ||
		strings.HasPrefix(model, "gemini-")
}

func IsXAIModel(model string) bool {
	return strings.HasPrefix(strings.TrimSpace(strings.ToLower(model)), "grok")
}

func IsOpenAIResponsesNativeWebModel(model string) bool {
	return CapabilitiesFor(OpenAIResponsesChannelType, model).NativeWebSearch
}

func IsOpenAIGPT54Model(model string) bool {
	return isOpenAIGPT54Model(normalizeModelName(model))
}

func IsOpenAIGPT56Model(model string) bool {
	return isOpenAIGPT56Model(normalizeModelName(model))
}

func GetOpenAIResponsesReasoningEfforts(model string) []string {
	return CapabilitiesFor(OpenAIResponsesChannelType, model).ReasoningEfforts
}

func SupportOpenAIResponsesReasoningControl(model string) bool {
	return CapabilitiesFor(OpenAIResponsesChannelType, model).ReasoningControl
}

func SupportXiaomiTokenPlanThinkingControl(model string) bool {
	return CapabilitiesFor(XiaomiTokenPlanCNChannelType, model).ReasoningControl
}

func SupportXAIReasoningControl(model string) bool {
	return CapabilitiesFor(XAIChannelType, model).ReasoningControl
}

func NormalizeXAIReasoningEffort(model string, effort string) string {
	capabilities := CapabilitiesFor(XAIChannelType, model)
	return NormalizeReasoningEffort(capabilities, effort)
}

func NormalizeOpenAIResponsesReasoningEffort(model string, effort string, nativeWebEnabled bool) string {
	capabilities := CapabilitiesFor(OpenAIResponsesChannelType, model)
	normalized := NormalizeReasoningEffort(capabilities, effort)
	if nativeWebEnabled && normalizeModelName(model) == "gpt-5" && normalized == "minimal" {
		return "low"
	}

	return normalized
}

func NormalizeXiaomiTokenPlanThinkingEffort(model string, effort string) string {
	capabilities := CapabilitiesFor(XiaomiTokenPlanCNChannelType, model)
	normalized := NormalizeReasoningEffort(capabilities, effort)
	if normalized == "" && strings.TrimSpace(effort) == "" {
		return "none"
	}

	return normalized
}

func NormalizeOpenAIResponsesReasoningSummary(summary string) string {
	normalized := strings.TrimSpace(strings.ToLower(summary))
	switch normalized {
	case "":
		return "detailed"
	case "none", "concise", "auto", "detailed":
		return normalized
	default:
		return "detailed"
	}
}

func IsGeminiNoThinkingModel(model string) bool {
	return strings.HasSuffix(strings.TrimSpace(model), "-nothinking")
}

func NormalizeDeepseekModel(model string) string {
	return strings.TrimSpace(strings.ToLower(model))
}

func IsDeepseekV4Model(model string) bool {
	normalized := NormalizeDeepseekModel(model)
	return normalized == DeepseekV4Flash || normalized == DeepseekV4Pro
}

func IsDeepseekReasoningReplayModel(model string) bool {
	return IsDeepseekV4Model(model)
}

func NormalizeDeepseekReasoningEffort(effort string) string {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "max", "xhigh":
		return "max"
	case "high", "medium", "low":
		return "high"
	default:
		return "high"
	}
}

func IsDeepseekThinkingDisabled(thinking interface{}) bool {
	if thinking == nil {
		return false
	}

	data, err := json.Marshal(thinking)
	if err != nil {
		return false
	}

	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(payload.Type), "disabled")
}

func SupportGeminiThinkingLevel(model string) bool {
	if IsGeminiNoThinkingModel(model) {
		return false
	}

	return model == Gemini36Flash ||
		strings.HasPrefix(model, "gemini-3.6-flash-") ||
		model == Gemini35Flash ||
		strings.HasPrefix(model, "gemini-3.5-flash-") ||
		model == "gemini-3-flash-preview" ||
		strings.HasPrefix(model, "gemini-3-flash-preview-") ||
		model == "gemini-3.1-flash-lite-preview" ||
		strings.HasPrefix(model, "gemini-3.1-flash-lite-preview-") ||
		model == "gemini-3.1-pro-preview" ||
		strings.HasPrefix(model, "gemini-3.1-pro-preview-") ||
		model == "gemini-3.1-pro-preview-customtools" ||
		strings.HasPrefix(model, "gemini-3.1-pro-preview-customtools-") ||
		model == Gemini31FlashLiteImage ||
		strings.HasPrefix(model, Gemini31FlashLiteImage+"-") ||
		model == Gemini31FlashImage ||
		strings.HasPrefix(model, Gemini31FlashImage+"-") ||
		model == Gemini3ProImage ||
		strings.HasPrefix(model, Gemini3ProImage+"-") ||
		model == "gemini-3-pro-preview" ||
		strings.HasPrefix(model, "gemini-3-pro-preview-")
}

func SupportGeminiThinkingBudget(model string) bool {
	if IsGeminiNoThinkingModel(model) {
		return false
	}

	return model == "gemini-2.5-flash" ||
		strings.HasPrefix(model, "gemini-2.5-flash-preview-") ||
		model == "gemini-2.5-flash-lite" ||
		strings.HasPrefix(model, "gemini-2.5-flash-lite-preview-") ||
		model == "gemini-2.5-pro" ||
		strings.HasPrefix(model, "gemini-2.5-pro-preview-") ||
		strings.HasPrefix(model, "gemini-2.5-pro-exp-")
}

func IsGeminiImageGenerationModel(model string) bool {
	return in(model, GeminiImageGenerationModels)
}

func IsVisionModel(model string) bool {
	if in(model, VisionSkipModels) {
		return IsConfiguredVisionModel(model)
	}

	return in(model, VisionModels) || IsConfiguredVisionModel(model)
}

func IsVideoModel(model string) bool {
	return in(model, VideoModels)
}
