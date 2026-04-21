package globals

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const ChatMaxThread = 5
const AnonymousMaxThread = 1

var HttpMaxTimeout = 30 * time.Minute

var AllowedOrigins []string

var DebugMode bool
var NotifyUrl = ""
var ArticlePermissionGroup []string
var GenerationPermissionGroup []string
var CacheAcceptedModels []string
var CacheAcceptedExpire int64
var CacheAcceptedSize int64
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
var CloseRelay bool

var SearchApiKey string
var SearchCrop bool
var SearchCropLength int
var SearchMaxResults int
var SearchTopic string
var SearchDepth string
var TaskModel string
var VisionModelResolver func(string) bool

func IsConfiguredVisionModel(model string) bool {
	if VisionModelResolver == nil {
		return false
	}

	return VisionModelResolver(model)
}

func OriginIsAllowed(uri string) bool {
	if len(AllowedOrigins) == 0 {
		// if allowed origins is empty, allow all origins
		return true
	}

	instance, _ := url.Parse(uri)
	if instance == nil {
		return false
	}

	if instance.Hostname() == "localhost" || instance.Scheme == "file" {
		return true
	}

	if strings.HasPrefix(instance.Host, "www.") {
		instance.Host = instance.Host[4:]
	}

	return in(instance.Host, AllowedOrigins)
}

func OriginIsOpen(c *gin.Context) bool {
	return strings.HasPrefix(c.Request.URL.Path, "/v1") || strings.HasPrefix(c.Request.URL.Path, "/dashboard") || strings.HasPrefix(c.Request.URL.Path, "/mj")
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
	ClaudeSlack                  = "claude-slack"
	SparkDeskLite                = "spark-desk-lite"
	SparkDeskPro                 = "spark-desk-pro"
	SparkDeskPro128K             = "spark-desk-pro-128k"
	SparkDeskMax                 = "spark-desk-max"
	SparkDeskMax32K              = "spark-desk-max-32k"
	SparkDeskV4Ultra             = "spark-desk-4.0-ultra"
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
	Gemini3Flash                 = "gemini-3-flash"
	Gemini3ProPreview            = "gemini-3-pro-preview"
	Gemini3ProImagePreview       = "gemini-3-pro-image-preview"
	GeminiExp1206                = "gemini-exp-1206"
	GoogleImagen002              = "imagen-3.0-generate-002"
	ZhiPuChatGLM4                = "glm-4"
	ZhiPuChatGLM4Vision          = "glm-4v"
	ZhiPuChatGLM3Turbo           = "glm-3-turbo"
	ZhiPuChatGLMTurbo            = "zhipu-chatglm-turbo"
	ZhiPuChatGLMPro              = "zhipu-chatglm-pro"
	ZhiPuChatGLMStd              = "zhipu-chatglm-std"
	ZhiPuChatGLMLite             = "zhipu-chatglm-lite"
	QwenTurbo                    = "qwen-turbo"
	QwenPlus                     = "qwen-plus"
	QwenTurboNet                 = "qwen-turbo-net"
	QwenPlusNet                  = "qwen-plus-net"
	Midjourney                   = "midjourney"
	MidjourneyFast               = "midjourney-fast"
	MidjourneyTurbo              = "midjourney-turbo"
	Hunyuan                      = "hunyuan"
	GPT360V9                     = "360-gpt-v9"
	Baichuan53B                  = "baichuan-53b"
	SkylarkLite                  = "skylark-lite-public"
	SkylarkPlus                  = "skylark-plus-public"
	SkylarkPro                   = "skylark-pro-public"
	SkylarkChat                  = "skylark-chat"
	DeepseekV3                   = "deepseek-chat"
	DeepseekR1                   = "deepseek-reasoner"
)

var OpenAIDalleModels = []string{
	Dalle, Dalle2, Dalle3, GPTImage1,
}

var GoogleImagenModels = []string{
	GoogleImagen002,
}

var VisionModels = []string{
	GPT4VisionPreview, GPT41106VisionPreview, GPT4Turbo, GPT4Turbo20240409, GPT4O, GPT4O20240513, // openai
	GeminiProVision, Gemini15Pro002, Gemini15Flash002, Gemini15ProLatest, Gemini15FlashLatest,
	Gemini20Flash, Gemini20Flash001, Gemini20FlashLite,
	Gemini25Flash, Gemini25Pro, Gemini25FlashLitePreview, "gemini-2.5-flash-lite", "gemini-2.5-flash-preview-09-2025",
	Gemini3Flash, Gemini3ProPreview, Gemini3ProImagePreview, "gemini-3-flash-preview", "gemini-3.1-pro-preview", "gemini-3.1-pro-preview-customtools", "gemini-3.1-flash-lite-preview", "gemini-3.1-flash-image-preview", // gemini
	Claude3,             // anthropic
	ZhiPuChatGLM4Vision, // chatglm
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

func IsGeminiModel(model string) bool {
	return model == GeminiPro ||
		model == GeminiProVision ||
		strings.HasPrefix(model, "gemini-")
}

func IsXAIModel(model string) bool {
	return strings.HasPrefix(strings.TrimSpace(strings.ToLower(model)), "grok")
}

func IsGeminiNoThinkingModel(model string) bool {
	return strings.HasSuffix(strings.TrimSpace(model), "-nothinking")
}

func SupportGeminiThinkingLevel(model string) bool {
	if IsGeminiNoThinkingModel(model) {
		return false
	}

	return model == "gemini-3-flash-preview" ||
		strings.HasPrefix(model, "gemini-3-flash-preview-") ||
		model == "gemini-3.1-flash-lite-preview" ||
		strings.HasPrefix(model, "gemini-3.1-flash-lite-preview-") ||
		model == "gemini-3.1-pro-preview" ||
		strings.HasPrefix(model, "gemini-3.1-pro-preview-") ||
		model == "gemini-3.1-pro-preview-customtools" ||
		strings.HasPrefix(model, "gemini-3.1-pro-preview-customtools-") ||
		model == "gemini-3.1-flash-image-preview" ||
		strings.HasPrefix(model, "gemini-3.1-flash-image-preview-") ||
		model == "gemini-3-pro-image-preview" ||
		strings.HasPrefix(model, "gemini-3-pro-image-preview-") ||
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

func IsGoogleImagenModel(model string) bool {
	// using image generation api if model is in imagen models
	return in(model, GoogleImagenModels)
}

func IsVisionModel(model string) bool {
	return in(model, VisionModels) && !in(model, VisionSkipModels)
}

func IsVideoModel(model string) bool {
	return in(model, VideoModels)
}
