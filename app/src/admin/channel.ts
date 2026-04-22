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
  "openai-responses": "OpenAI Responses",
  xai: "xAI Grok",
  azure: "Azure OpenAI",
  claude: "Anthropic Claude",
  "glm-coding-plan-cn": "GLM Coding Plan（CN）",
  "minimax-token-plan-cn": "MiniMax Token Plan（CN）",
  palm: "Google Gemini",
  midjourney: "Midjourney Proxy",
  deepseek: "深度求索 DeepSeek",
};

export const ShortChannelTypes: Record<string, string> = {
  openai: "OpenAI",
  "openai-responses": "OpenAI Responses",
  xai: "xAI",
  azure: "Azure",
  claude: "Claude",
  "glm-coding-plan-cn": "GLM Coding",
  "minimax-token-plan-cn": "MiniMax",
  palm: "Gemini",
  midjourney: "Midjourney",
  deepseek: "DeepSeek",
};

export const ChannelInfos: Record<string, ChannelInfo> = {
  openai: {
    endpoint: "https://api.openai.com",
    format: "<api-key>",
    models: [
      "gpt-3.5-turbo",
      "gpt-3.5-turbo-instruct",
      "gpt-3.5-turbo-0613",
      "gpt-3.5-turbo-0301",
      "gpt-3.5-turbo-1106",
      "gpt-3.5-turbo-0125",
      "gpt-3.5-turbo-16k",
      "gpt-3.5-turbo-16k-0613",
      "gpt-3.5-turbo-16k-0301",
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
  "openai-responses": {
    endpoint: "https://api.openai.com",
    format: "<api-key>",
    models: [
      "gpt-4o",
      "gpt-4o-mini",
      "gpt-4.1",
      "gpt-4.1-mini",
      "gpt-4.1-nano",
      "gpt-5",
      "gpt-5-mini",
      "gpt-5-nano",
      "o1",
      "o1-mini",
      "o3",
      "o4-mini",
    ],
  },
  xai: {
    endpoint: "https://api.x.ai",
    format: "<api-key>",
    models: ["grok-4.20-reasoning", "grok-4.20-mini", "grok-4-1-fast"],
    description:
      "> xAI 渠道基于 **OpenAI Responses API** 兼容格式，请将接入点填写为 *https://api.x.ai* 或其反代地址，系统会自动请求 */v1/responses*。 \n" +
      "> 已内置适配 xAI 原生 **Web Search** 与 **X Search** 两个独立开关，用户可在聊天界面分别控制是否开启。 \n" +
      "> 常用模型可填写如 **grok-4.20-reasoning**、**grok-4-1-fast** 等 Grok 模型。\n",
  },
  azure: {
    endpoint: "2023-12-01-preview",
    format: "<api-key>|<api-endpoint>",
    description:
      "> Azure 密钥 API Key 1 和 API Key 2 任填一个即可，密钥格式为 **api-key|api-endpoint**, api-endpoint 为 Azure 的 **API 端点**。\n" +
      "> 接入点填 **API Version**，如 2023-12-01-preview。\n" +
      "Azure 模型名称忽略点号等问题内部已经进行适配，无需额外任何设置。",
    models: [
      "gpt-3.5-turbo",
      "gpt-3.5-turbo-instruct",
      "gpt-3.5-turbo-0613",
      "gpt-3.5-turbo-0301",
      "gpt-3.5-turbo-1106",
      "gpt-3.5-turbo-0125",
      "gpt-3.5-turbo-16k",
      "gpt-3.5-turbo-16k-0613",
      "gpt-3.5-turbo-16k-0301",
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
      "> Anthropic Claude 密钥即为 **x-api-key**，Anthropic 对请求 IP 地域有限制，可能出现 **Request not allowed** 的错误，请尝试更换 IP 或者使用代理。\n",
    models: [
      "claude-instant-1.2",
      "claude-2",
      "claude-2.1",
      "claude-3-opus-20240229",
      "claude-3-sonnet-20240229",
      "claude-3-haiku-20240307",
    ],
  },
  "glm-coding-plan-cn": {
    endpoint: "https://open.bigmodel.cn/api/anthropic",
    format: "<x-api-key>",
    description:
      "> GLM Coding Plan（CN）渠道基于 **Anthropic / Claude API** 兼容格式，接入点请填写 *https://open.bigmodel.cn/api/anthropic* 或其反代地址。 \n" +
      "> 密钥请填写智谱 API Key，系统会按 **x-api-key** 方式请求官方 *`/v1/messages`* 接口。 \n" +
      "> 官方当前推荐模型包括 **glm-5.1**、**glm-5**、**glm-4.7**，编码套餐文档中也常见 **glm-4.5-air**。 \n",
    models: ["glm-5.1", "glm-5", "glm-4.7", "glm-4.5-air"],
  },
  "minimax-token-plan-cn": {
    endpoint: "https://api.minimaxi.com/anthropic",
    format: "<x-api-key>",
    description:
      "> MiniMax Token Plan（CN）渠道基于 **Anthropic API** 兼容格式，接入点请填写 *https://api.minimaxi.com/anthropic* 或其反代地址。 \n" +
      "> 密钥需使用 **Token Plan 专属 API Key**，与 MiniMax 按量计费 API Key **不可互通**。 \n" +
      "> 当前常用文本模型可填写 **MiniMax-M2.7**，极速版套餐也可使用 **MiniMax-M2.7-highspeed**。 \n",
    models: ["MiniMax-M2.7", "MiniMax-M2.7-highspeed"],
  },
  palm: {
    endpoint: "https://generativelanguage.googleapis.com",
    format: "<api-key>",
    models: [
      "chat-bison-001",
      "gemini-1.5-pro-002",
      "gemini-1.5-flash-002",
      "gemini-2.0-flash",
      "gemini-2.0-flash-001",
      "gemini-2.0-flash-lite",
      "gemini-2.5-flash",
      "gemini-2.5-flash-preview-09-2025",
      "gemini-2.5-pro",
      "gemini-2.5-flash-lite",
      "gemini-2.5-flash-lite-preview-06-17",
      "gemini-2.5-flash-lite-preview-09-2025",
      "gemini-3-flash",
      "gemini-3-flash-preview",
      "gemini-3-pro-preview",
      "gemini-3-pro-image-preview",
      "gemini-3.1-pro-preview",
      "gemini-3.1-pro-preview-customtools",
      "gemini-3.1-flash-lite-preview",
      "gemini-3.1-flash-image-preview",
      "gemini-1.5-pro-latest",
      "gemini-1.5-flash-latest",
    ],
    description:
      "> Google Gemini 密钥格式为 **api-key**，接入点填写 *https://generativelanguage.googleapis.com* 或其反代地址。 \n" +
      "> 系统已适配当前官方 `generateContent` / `streamGenerateContent` 请求结构，并支持 `system_instruction` 与 function calling。 \n" +
      "> Gemini 2.5 系列使用 `thinkingBudget`，Gemini 3 系列使用官方推荐的 `thinkingLevel` 参数，系统会按模型自动选择。 \n" +
      "> 为兼容官方稳定版与预览版模型，系统会自动在 `v1` 与 `v1beta` 之间选择合适的 Gemini API 版本。 \n" +
      "> Google 对请求 IP 地域有限制，可能出现 **User Location Is Not Supported** 的错误，可通过可用地区 IP 或反代接入解决。\n",
  },
  midjourney: {
    endpoint: "https://your.midjourney.proxy",
    format: "<mj-api-secret>|<white-list>",
    models: ["midjourney", "midjourney-fast", "midjourney-turbo"],
    description:
      "> 请参考 [midjourney-proxy](https://github.com/novicezk/midjourney-proxy) 项目填入参数，可设置白名单 *white-list* 以限制回调 IP \n" +
      "> 密钥举例： password|localhost,127.0.0.1,196.128.0.31\n" +
      "> 密钥即为 *mj-api-secret* （如果没有设置 secret 请填 `null` ） \n" +
      "> 白名单即为 *white-list*（如果没有回调 IP 白名单默认接收所有 IP 的回调，不需要加 | 以及后面的内容） \n" +
      "> 接入点填写你的 Midjourney Proxy 的部署地址，如 *http://localhost:8080*, *https://example.com* \n" +
      "> 注意：**请在系统设置中设置后端的公网 IP / 域名，否则无法接收回调报错 please provide available notify url** \n",
  },
  deepseek: {
    endpoint: "https://api.deepseek.com",
    format: "<api-key>",
    models: ["deepseek-chat", "deepseek-reasoner"],
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
