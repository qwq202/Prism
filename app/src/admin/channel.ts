import { getUniqueList } from "@/utils/base.ts";
import {
  AnonymousType,
  BasicType,
  NormalType,
  ProType,
  StandardType,
} from "@/utils/groups.ts";

export type Channel = {
  id: number;
  name: string;
  type: string;
  models: string[];
  priority: number;
  weight: number;
  retry: number;
  secret: string;
  endpoint: string;
  mapper: string;
  state: boolean;
  group?: string[];
  proxy?: {
    proxy: string;
    proxy_type: number;
    username: string;
    password: string;
  };
  first_message_as_user?: boolean;
  merge_consecutive_user_messages?: boolean;
};

export enum proxyType {
  NoneProxy = 0,
  HttpProxy = 1,
  HttpsProxy = 2,
  Socks5Proxy = 3,
}

export const ProxyTypes: Record<number, string> = {
  [proxyType.NoneProxy]: "None Proxy",
  [proxyType.HttpProxy]: "HTTP Proxy",
  [proxyType.HttpsProxy]: "HTTPS Proxy",
  [proxyType.Socks5Proxy]: "SOCKS5 Proxy",
};

export type ChannelInfo = {
  description?: string;
  endpoint: string;
  format: string;
  models: string[];
};

export const ChannelTypes: Record<string, string> = {
  openai: "OpenAI",
  openrouter: "OpenRouter",
  siliconflow: "硅基流动 SiliconFlow",
  "openai-responses": "OpenAI Responses",
  xai: "xAI Grok",
  azure: "Azure OpenAI",
  claude: "Anthropic Claude",
  "glm-coding-plan-cn": "GLM Coding Plan（CN）",
  "minimax-token-plan-cn": "MiniMax Token Plan（CN）",
  "xiaomi-mimo": "Xiaomi MiMo",
  "xiaomi-token-plan-cn": "Xiaomi Token Plan (China)",
  palm: "Google Gemini",
  "gemini-enterprise-agent-platform": "Gemini Enterprise Agent Platform",
  deepseek: "深度求索 DeepSeek",
};

export const ShortChannelTypes: Record<string, string> = {
  openai: "OpenAI",
  openrouter: "OpenRouter",
  siliconflow: "SiliconFlow",
  "openai-responses": "OpenAI Responses",
  xai: "xAI",
  azure: "Azure",
  claude: "Claude",
  "glm-coding-plan-cn": "GLM Coding",
  "minimax-token-plan-cn": "MiniMax",
  "xiaomi-mimo": "MiMo",
  "xiaomi-token-plan-cn": "Xiaomi TP",
  palm: "Gemini",
  "gemini-enterprise-agent-platform": "Gemini AP",
  deepseek: "DeepSeek",
};

export const ChannelInfos: Record<string, ChannelInfo> = {
  openai: {
    endpoint: "https://api.openai.com",
    format: "<api-key>",
    models: [
      "gpt-4",
      "gpt-4-0314",
      "gpt-4-0613",
      "gpt-4-1106-preview",
      "gpt-4-0125-preview",
      "gpt-4-turbo-preview",
      "gpt-4-vision-preview",
      "gpt-4-1106-vision-preview",
      "gpt-4-turbo",
      "gpt-4-turbo-2024-04-09",
      "gpt-4-32k",
      "gpt-4-32k-0314",
      "gpt-4-32k-0613",
      "gpt-4o",
      "gpt-4o-2024-05-13",
      "gpt-4o-mini",
      "gpt-4o-2024-08-06",
      "gpt-4o-mini-2024-07-18",
      "dalle",
      "dall-e-2",
      "dall-e-3",
    ],
  },
  openrouter: {
    endpoint: "https://openrouter.ai/api/v1",
    format: "<openrouter-api-key>",
    models: [
      "~openai/gpt-latest",
      "openai/gpt-4o",
      "openai/gpt-4o-mini",
      "anthropic/claude-sonnet-4.5",
      "anthropic/claude-haiku-4.5",
      "google/gemini-2.5-pro",
      "google/gemini-2.5-flash",
      "deepseek/deepseek-chat-v3.1",
    ],
    description:
      "> OpenRouter 渠道基于 **OpenAI Chat Completions API** 兼容格式，接入点填写 `https://openrouter.ai/api/v1` 或其反代地址，系统会请求 *`/chat/completions`*。 \n" +
      "> 密钥填写 OpenRouter API Key，系统会按 **Authorization: Bearer** 鉴权，并附加 **X-OpenRouter-Title: Prism** 用于应用归因。 \n" +
      "> 模型名使用 OpenRouter 官方模型 slug，例如 **openai/gpt-4o**、**anthropic/claude-sonnet-4.5**、**google/gemini-2.5-flash**，也可以从 OpenRouter 模型列表复制后手动添加。\n",
  },
  siliconflow: {
    endpoint: "https://api.siliconflow.cn/v1",
    format: "<siliconflow-api-key>",
    models: [
      "Pro/zai-org/GLM-4.7",
      "Pro/zai-org/GLM-5",
      "deepseek-ai/DeepSeek-V3.2",
      "deepseek-ai/DeepSeek-V4-Flash",
      "Qwen/Qwen3-32B",
      "Qwen/Qwen3-Coder-480B-A35B-Instruct",
    ],
    description:
      "> 硅基流动渠道基于 **OpenAI Chat Completions API** 兼容格式，接入点默认填写 `https://api.siliconflow.cn/v1`，系统会请求 *`/chat/completions`*。 \n" +
      "> 密钥填写 SiliconFlow API Key，系统会按 **Authorization: Bearer** 鉴权。官方同时提供 *`/v1/models`* 模型列表接口，可从控制台或接口复制模型名后添加。 \n" +
      "> 常见模型包括 **Pro/zai-org/GLM-4.7**、**deepseek-ai/DeepSeek-V3.2**、**deepseek-ai/DeepSeek-V4-Flash**、**Qwen/Qwen3** 系列，具体以硅基流动模型列表和你的账号权限为准。\n",
  },
  "openai-responses": {
    endpoint: "https://api.openai.com",
    format: "<api-key>",
    models: [
      "gpt-4o",
      "gpt-4o-mini",
      "gpt-4.1",
      "gpt-4.1-mini",
      "gpt-4.1-nano",
      "gpt-5.5",
      "gpt-5.4",
      "gpt-5.4-pro",
      "gpt-5.4-mini",
      "gpt-5.4-nano",
      "gpt-5.3-chat-latest",
      "gpt-5.2",
      "gpt-5.2-pro",
      "gpt-5.2-chat-latest",
      "gpt-5.1",
      "gpt-5",
      "gpt-5-pro",
      "gpt-4.5-preview",
      "gpt-5-mini",
      "gpt-5-nano",
      "o3",
      "o1",
      "o1-mini",
      "o4-mini",
    ],
  },
  xai: {
    endpoint: "https://api.x.ai",
    format: "<api-key>",
    models: [
      "grok-4.20-reasoning",
      "grok-4.20-mini",
      "grok-4-1-fast-reasoning",
      "grok-4-1-fast",
    ],
    description:
      "> xAI 渠道基于 **OpenAI Responses API** 兼容格式，请将接入点填写为 `https://api.x.ai` 或其反代地址，系统会自动请求 `/v1/responses`。 \n" +
      "> 系统已按 xAI 当前文档改为将 **system prompt** 保留在 `input` 中，而不是使用 xAI 暂不支持的 `instructions` 字段。 \n" +
      "> 已内置适配 xAI 原生 **Web Search** 与 **X Search** 两个独立开关，并会按官方方式自动开启 **view_image / view_x_video** 所需的图像与视频理解能力。 \n" +
      "> 常用模型可填写如 **grok-4.20-reasoning**、**grok-4-1-fast-reasoning**、**grok-4-1-fast** 等 Grok 模型。\n",
  },
  azure: {
    endpoint: "2023-12-01-preview",
    format: "<api-key>|<api-endpoint>",
    description:
      "> Azure 密钥 API Key 1 和 API Key 2 任填一个即可，密钥格式为 **api-key|api-endpoint**, api-endpoint 为 Azure 的 **API 端点**。\n" +
      "> 接入点填 **API Version**，如 2023-12-01-preview。\n" +
      "Azure 模型名称忽略点号等问题内部已经进行适配，无需额外任何设置。",
    models: [
      "gpt-4",
      "gpt-4-0314",
      "gpt-4-0613",
      "gpt-4-1106-preview",
      "gpt-4-0125-preview",
      "gpt-4-turbo-preview",
      "gpt-4-vision-preview",
      "gpt-4-1106-vision-preview",
      "gpt-4-turbo",
      "gpt-4-turbo-2024-04-09",
      "gpt-4-32k",
      "gpt-4-32k-0314",
      "gpt-4-32k-0613",
      "dalle",
      "dall-e-2",
      "dall-e-3",
    ],
  },
  claude: {
    endpoint: "https://api.anthropic.com",
    format: "<x-api-key>",
    description:
      "> Anthropic Claude 密钥即为 **x-api-key**，接入点填写 `https://api.anthropic.com` 或其反代地址，系统会请求官方 *`/v1/messages`* 接口。 \n" +
      "> 系统现已适配新版 **Messages API** 的 **tools / tool_choice / thinking** 与完整 **SSE content block** 流式事件，可用于 Claude 原生工具调用与 extended thinking。 \n" +
      "> 如果同时开启工具调用与 thinking，系统会按官方要求自动启用 **interleaved thinking** beta 头。Anthropic 对请求 IP 地域有限制，可能出现 **Request not allowed** 的错误，请尝试更换 IP 或者使用代理。\n",
    models: [
      "claude-opus-4-1-20250805",
      "claude-opus-4-20250514",
      "claude-sonnet-4-20250514",
      "claude-3-7-sonnet-20250219",
      "claude-3-5-haiku-20241022",
    ],
  },
  "glm-coding-plan-cn": {
    endpoint: "https://open.bigmodel.cn/api/anthropic",
    format: "<x-api-key>",
    description:
      "> GLM Coding Plan（CN）渠道基于 **Anthropic / Claude API** 兼容格式，接入点请填写 `https://open.bigmodel.cn/api/anthropic` 或其反代地址。 \n" +
      "> 密钥请填写智谱 API Key，系统会按 **x-api-key** 方式请求官方 *`/v1/messages`* 接口。 \n" +
      "> 官方当前推荐模型包括 **glm-5.1**、**glm-5**、**glm-4.7**，编码套餐文档中也常见 **glm-4.5-air**。 \n",
    models: ["glm-5.1", "glm-5", "glm-4.7", "glm-4.5-air"],
  },
  "minimax-token-plan-cn": {
    endpoint: "https://api.minimaxi.com/anthropic",
    format: "<x-api-key>",
    description:
      "> MiniMax Token Plan（CN）渠道基于 **Anthropic API** 兼容格式，接入点请填写 `https://api.minimaxi.com/anthropic` 或其反代地址。 \n" +
      "> 密钥需使用 **Token Plan 专属 API Key**，与 MiniMax 按量计费 API Key **不可互通**。 \n" +
      "> 系统现已适配 MiniMax 当前 Anthropic 兼容链路里的 **tools / tool_choice / thinking** 与完整 **SSE content block** 流式事件，可用于 Tool Use 与交错思维链。 \n" +
      "> 官方文档当前同时出现 **MiniMax-M2.1 / MiniMax-M2.1-highspeed / MiniMax-M2** 与快速接入页里的 **MiniMax-M2.7 / MiniMax-M2.7-highspeed**，这里一并保留，方便兼容不同套餐文档口径。 \n",
    models: [
      "MiniMax-M2.1",
      "MiniMax-M2.1-highspeed",
      "MiniMax-M2",
      "MiniMax-M2.7",
      "MiniMax-M2.7-highspeed",
    ],
  },
  "xiaomi-mimo": {
    endpoint: "https://api.xiaomimimo.com/v1",
    format: "<mimo-api-key>",
    description:
      "> Xiaomi MiMo 渠道基于开放平台 **OpenAI Chat Completions API** 兼容格式，接入点默认填写 `https://api.xiaomimimo.com/v1`，系统会请求 *`/chat/completions`*。 \n" +
      "> 密钥填写开放平台 API Key，系统会按官方 **api-key** 请求头鉴权。 \n" +
      "> 系统已适配 **max_completion_tokens**、**thinking**、**reasoning_content**、函数调用与流式 Chat Completions 返回。 \n" +
      "> 默认模型按官方模型列表保留 **mimo-v2.5-pro**、**mimo-v2.5**、**mimo-v2-flash**；**mimo-v2-pro** 和 **mimo-v2-omni** 已进入下线窗口，不再作为默认模型。\n",
    models: ["mimo-v2.5-pro", "mimo-v2.5", "mimo-v2-flash"],
  },
  "xiaomi-token-plan-cn": {
    endpoint: "https://token-plan-cn.xiaomimimo.com/v1",
    format: "<tp-api-key>",
    description:
      "> Xiaomi Token Plan (China) 渠道基于 **OpenAI Chat Completions API** 兼容格式，接入点默认填写 `https://token-plan-cn.xiaomimimo.com/v1`，系统会请求 *`/chat/completions`*。 \n" +
      "> 密钥需使用 **Token Plan 专属 API Key**，格式通常为 **tp-xxxxx**，与按量计费 API Key **sk-xxxxx** 不可互通。 \n" +
      "> 系统已单独适配小米 Token Plan 的 **api-key** 鉴权、**max_completion_tokens**、**thinking**、**reasoning_content** 与流式 Chat Completions 返回。 \n" +
      "> 默认模型按官方快速接入和模型列表保留 **mimo-v2.5-pro**、**mimo-v2.5**、**mimo-v2-pro**、**mimo-v2-omni**、**mimo-v2-flash**。\n",
    models: [
      "mimo-v2.5-pro",
      "mimo-v2.5",
      "mimo-v2-pro",
      "mimo-v2-omni",
      "mimo-v2-flash",
    ],
  },
  palm: {
    endpoint: "https://generativelanguage.googleapis.com",
    format: "<api-key>",
    models: [
      "gemini-2.5-flash",
      "gemini-2.5-flash-preview-09-2025",
      "gemini-2.5-pro",
      "gemini-2.5-flash-lite",
      "gemini-2.5-flash-lite-preview-06-17",
      "gemini-2.5-flash-lite-preview-09-2025",
      "gemini-3.5-flash",
      "gemini-3-flash",
      "gemini-3-flash-preview",
      "gemini-3-pro-preview",
      "gemini-3-pro-image",
      "gemini-2.5-flash-image",
      "gemini-3.1-pro-preview",
      "gemini-3.1-pro-preview-customtools",
      "gemini-3.1-flash-lite-preview",
      "gemini-3.1-flash-image",
    ],
    description:
      "> Google Gemini 密钥格式为 **api-key**，接入点填写 `https://generativelanguage.googleapis.com` 或其反代地址。 \n" +
      "> 系统已适配当前官方 `generateContent` / `streamGenerateContent` 请求结构，并支持 `system_instruction`、Google Search、URL Context 与 function calling。 \n" +
      "> Gemini 原生图像生成模型（如 **gemini-3.1-flash-image**、**gemini-3-pro-image**、**gemini-2.5-flash-image**）使用 `generateContent` 并解析 `inlineData` 图片输出。 \n" +
      "> Gemini 2.5 系列使用 `thinkingBudget`，Gemini 3 系列使用官方推荐的 `thinkingLevel` 参数，系统会按模型自动选择。 \n" +
      "> 为兼容官方稳定版与预览版模型，系统会自动在 `v1` 与 `v1beta` 之间选择合适的 Gemini API 版本。 \n" +
      "> Google 对请求 IP 地域有限制，可能出现 **User Location Is Not Supported** 的错误，可通过可用地区 IP 或反代接入解决。\n",
  },
  "gemini-enterprise-agent-platform": {
    endpoint: "https://aiplatform.googleapis.com",
    format: "<express-mode-api-key>",
    models: [
      "gemini-2.5-flash",
      "gemini-2.5-flash-lite",
      "gemini-2.5-pro",
      "gemini-2.5-flash-preview-09-2025",
      "gemini-2.5-flash-lite-preview-09-2025",
      "gemini-3.5-flash",
      "gemini-3-flash",
      "gemini-3-flash-preview",
      "gemini-3-pro-preview",
      "gemini-3-pro-image",
      "gemini-2.5-flash-image",
      "gemini-3.1-pro-preview",
      "gemini-3.1-pro-preview-customtools",
      "gemini-3.1-flash-lite-preview",
      "gemini-3.1-flash-image",
    ],
    description:
      "> Gemini Enterprise Agent Platform 渠道基于 **Vertex AI Express Mode** 的 Gemini API，接入点填写 `https://aiplatform.googleapis.com` 或其反代地址。 \n" +
      "> 密钥请填写 Google Cloud Express Mode API Key；系统会请求 *`/v1/publishers/google/models/{model}:generateContent`*，流式请求使用 *`streamGenerateContent?alt=sse`*。 \n" +
      "> 预览、实验和 latest 模型会自动切换到 Agent Platform 的 *`v1beta1`* API 版本；稳定模型默认使用 *`v1`*。 \n" +
      "> 请求体沿用官方 Gemini `contents` / `systemInstruction` / `tools` / `generationConfig` 结构，并支持当前 Gemini 思考预算、URL Context、Google Search、function calling 与原生 `inlineData` 图像输出。 \n" +
      "> 该渠道不自动刷新 OAuth/ADC 访问令牌；如果要使用标准项目/区域 Vertex AI OAuth 链路，请先通过 Express Mode API Key 或反代接入。\n",
  },
  deepseek: {
    endpoint: "https://api.deepseek.com",
    format: "<api-key>",
    models: ["deepseek-v4-pro", "deepseek-v4-flash"],
    description:
      "> DeepSeek 渠道使用官方 **Chat Completions API**，接入点填写 `https://api.deepseek.com` 或其反代地址，系统会请求 *`/chat/completions`*。 \n" +
      "> 官方已发布 **DeepSeek-V4** 预览模型 **deepseek-v4-pro** 与 **deepseek-v4-flash**，二者均支持 1M 上下文与 Thinking / Non-Thinking 双模式。 \n" +
      "> 系统现已适配官方最新的 **thinking**、**reasoning_effort**、**response_format**、**stream_options**、**logprobs / top_logprobs**、**tools / tool_choice** 等参数。 \n",
  },
};

export const defaultChannelModels: string[] = getUniqueList(
  Object.values(ChannelInfos).flatMap((info) => info.models),
);

export const channelGroups: string[] = [
  AnonymousType,
  NormalType,
  BasicType,
  StandardType,
  ProType,
];

export function getChannelInfo(type?: string): ChannelInfo {
  if (type && type in ChannelInfos) return ChannelInfos[type];
  return ChannelInfos.openai;
}

export function getChannelType(type?: string): string {
  if (type && type in ChannelTypes) return ChannelTypes[type];
  return ChannelTypes.openai;
}

export function getShortChannelType(type?: string): string {
  if (type && type in ShortChannelTypes) return ShortChannelTypes[type];
  return ShortChannelTypes.openai;
}
