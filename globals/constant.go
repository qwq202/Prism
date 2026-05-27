package globals

const (
	System    = "system"
	User      = "user"
	Assistant = "assistant"
	Tool      = "tool"
	Function  = "function"
)

const (
	OpenAIChannelType                        = "openai"
	OpenRouterChannelType                    = "openrouter"
	OpenAIResponsesChannelType               = "openai-responses"
	XAIChannelType                           = "xai"
	AzureOpenAIChannelType                   = "azure"
	ClaudeChannelType                        = "claude"
	GLMCodingPlanCNChannelType               = "glm-coding-plan-cn"
	MiniMaxTokenPlanCNChannelType            = "minimax-token-plan-cn"
	XiaomiTokenPlanCNChannelType             = "xiaomi-token-plan-cn"
	PalmChannelType                          = "palm"
	GeminiEnterpriseAgentPlatformChannelType = "gemini-enterprise-agent-platform"
	DeepseekChannelType                      = "deepseek"
)

const (
	NonBilling   = "non-billing"
	TimesBilling = "times-billing"
	TokenBilling = "token-billing"
)

const (
	AnonymousType = "anonymous"
	NormalType    = "normal"
	BasicType     = "basic"    // basic subscription
	StandardType  = "standard" // standard subscription
	ProType       = "pro"      // pro subscription
	AdminType     = "admin"
)

const (
	NoneProxyType = iota
	HttpProxyType
	HttpsProxyType
	Socks5ProxyType
)

const (
	WebTokenType = "web"
	SystemToken  = "system"
)
