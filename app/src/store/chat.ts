import { createSelector, createSlice } from "@reduxjs/toolkit";
import {
  AssistantRole,
  ConversationInstance,
  Model,
  MessageToolCall,
  UserRole,
} from "@/api/types.tsx";
import { Message } from "@/api/types.tsx";
import { AppDispatch, RootState } from "./index.ts";
import {
  getArrayMemory,
  getBooleanMemory,
  getMemory,
  getNumberMemory,
  setArrayMemory,
  setMemory,
  setNumberMemory,
} from "@/utils/memory.ts";
import {
  getOfflineModels,
  loadPreferenceModels,
  setOfflineModels,
} from "@/conf/storage.ts";
import { isDrawingModel } from "@/conf/model.ts";
import {
  getMaintainedReasoningEfforts,
  normalizeConfiguredReasoningEfforts,
} from "@/conf/reasoning.ts";
import {
  deleteConversation as doDeleteConversation,
  deleteAllConversations as doDeleteAllConversations,
  favoriteConversation as doFavoriteConversation,
  renameConversation as doRenameConversation,
  retitleConversation as doRetitleConversation,
  fetchConversation,
  fetchConversationList,
} from "@/api/history.ts";
import {
  clearCachedConversation,
  clearCachedConversations,
  getConversationListCacheGeneration,
  getCachedConversation,
  getCachedConversationList,
  isCurrentConversationListCacheGeneration,
  removeCachedConversationFromList,
  setCachedConversation,
} from "@/utils/conversation-cache.ts";
import { logClientEvent } from "@/utils/client-logger.ts";
import {
  clearPendingChatRequests,
  clearPendingChatRequestsForConversation,
  createChatRequestID,
  enqueuePendingChatRequest,
  getChatOutboxScope,
  getPendingChatRequests,
} from "@/utils/chat-outbox.ts";
import { selectAuthenticated } from "@/store/auth.ts";
import { isHiddenToolCallName } from "@/api/tool-calls.ts";
import {
  type AskUserResult,
  isPendingAskUserToolCall,
} from "@/api/ask-user.ts";
import { CustomMask, Mask } from "@/masks/types.ts";
import { listMasks } from "@/api/mask.ts";
import { useDispatch, useSelector } from "react-redux";
import { useMemo } from "react";
import { ChatProps, ConnectionStack, StreamMessage } from "@/api/connection.ts";
import { useTranslation } from "react-i18next";
import {
  buildPersonalizationInstruction,
  contextSelector,
  frequencyPenaltySelector,
  historySelector,
  maxTokensSelector,
  memoryEnabledSelector,
  memoryHistoryEnabledSelector,
  personaAboutUserSelector,
  personaCustomInstructionSelector,
  personaEmojiSelector,
  personaEnthusiasmSelector,
  personaListsSelector,
  personaNicknameSelector,
  personaOccupationSelector,
  personaStyleSelector,
  personaWarmthSelector,
  presencePenaltySelector,
  repetitionPenaltySelector,
  temperatureSelector,
  topKSelector,
  topPSelector,
} from "@/store/settings.ts";

function resolveOpenAIReasoningEffortForRequest(
  supportModels: Model[],
  model: string,
  effort: string,
  nativeWebEnabled: boolean,
): string | undefined {
  const capabilities = getOpenAIResponsesCapabilities(supportModels, model);
  const normalized = normalizeOpenAIResponsesReasoningEffort(
    supportModels,
    model,
    effort,
  );
  if (!normalized) {
    const requested = (effort || "").trim().toLowerCase();
    if (!requested || requested === "none") return undefined;

    const fallback = capabilities.reasoningEfforts.find(
      (item) => item !== "none",
    );
    console.warn("[openai-responses] unsupported reasoning effort fallback", {
      model,
      requested,
      fallback,
      supported: capabilities.reasoningEfforts,
    });

    return fallback;
  }

  if (
    nativeWebEnabled &&
    model.trim().toLowerCase() === "gpt-5" &&
    normalized === "minimal"
  ) {
    return "low";
  }

  return normalized;
}

export type ConversationSerialized = {
  model?: string;
  messages: Message[];
  updated_at?: string;
  cache_complete?: boolean;
  server_synced?: boolean;
  local_pending_until?: number;
  local_revision?: number;
};

export type ConnectionEvent = {
  id: number;
  event: string;
  index?: number;
  message?: string;
  toolCallId?: string;
  askUserResult?: AskUserResult;
};

type initialStateType = {
  history: ConversationInstance[];
  messages: Message[];
  conversations: Record<number, ConversationSerialized>;
  model: string;
  web: boolean;
  gemini_google_search: boolean;
  gemini_url_context: boolean;
  xai_web_search: boolean;
  xai_x_search: boolean;
  openai_responses_web_search: boolean;
  fetch: boolean;
  learning_mode: boolean;
  gemini_thinking_budget: number;
  deepseek_thinking_enabled_by_model: Record<string, boolean>;
  deepseek_reasoning_effort_by_model: Record<string, string>;
  openai_reasoning_effort: string;
  current: number;
  model_list: string[];
  market: boolean;
  mask_item: Mask | null;
  custom_masks: CustomMask[];
  support_models: Model[];
  loadingConversationId: number | null;
  active_requests: Record<number, string>;
};

const defaultConversation: ConversationSerialized = { messages: [] };
const localMutationProtectionMs = 10_000;
let conversationNavigationRevision = 0;
const conversationDetailRequestSeq = new Map<number, number>();

function bumpConversationNavigationRevision(): number {
  conversationNavigationRevision += 1;
  return conversationNavigationRevision;
}

function getConversationNavigationRevision(): number {
  return conversationNavigationRevision;
}

function nextConversationDetailRequestSeq(id: number): number {
  const seq = (conversationDetailRequestSeq.get(id) ?? 0) + 1;
  conversationDetailRequestSeq.set(id, seq);
  return seq;
}

function isLatestConversationDetailRequest(id: number, seq: number): boolean {
  return conversationDetailRequestSeq.get(id) === seq;
}

function resetLocalConversationState(state: initialStateType) {
  bumpConversationNavigationRevision();
  state.history = [];
  state.messages = [];
  state.conversations = { [-1]: { ...defaultConversation } };
  state.current = -1;
  state.loadingConversationId = null;
  state.active_requests = {};
  state.mask_item = null;
  setNumberMemory("history_conversation", -1);
}

function replaceWithRemoteConversationHistory(
  state: initialStateType,
  incoming: ConversationInstance[],
) {
  const stable = preserveLocalHistoryKeys(
    dedupeStableHistory(incoming),
    state.history,
  );
  const remoteIds = new Set(stable.map((item) => item.id));
  const pending = state.history.find((item) => item.id === -1);
  const currentMissing = state.current !== -1 && !remoteIds.has(state.current);
  const currentConversation = currentMissing
    ? state.conversations[state.current]
    : undefined;
  const currentHistory = currentMissing
    ? state.history.find((item) => item.id === state.current)
    : undefined;

  let nextHistory =
    state.current === -1 &&
    pending &&
    state.conversations[-1]?.messages.length > 0
      ? [pending, ...stable]
      : stable;

  if (currentMissing && currentConversation) {
    nextHistory = [
      buildConversationHistoryEntry(
        state.current,
        currentConversation,
        currentHistory,
      ),
      ...nextHistory.filter((item) => item.id !== state.current),
    ];
  }

  state.history = nextHistory;

  if (currentMissing && !currentConversation) {
    closeConversationConnection(state.current);
    bumpConversationNavigationRevision();
    state.current = -1;
    state.messages = [];
    state.loadingConversationId = null;
    state.mask_item = null;
    setNumberMemory("history_conversation", -1);
  }
}

function getConversationHistoryName(
  conversation?: ConversationSerialized,
  fallback?: string,
): string {
  const fallbackName = fallback?.trim();
  if (fallbackName) return fallbackName;

  const firstUserMessage =
    conversation?.messages.find((item) => item.role === UserRole)?.content ??
    conversation?.messages[0]?.content ??
    "";

  return firstUserMessage.trim();
}

function buildConversationHistoryEntry(
  id: number,
  conversation?: ConversationSerialized,
  fallback?: ConversationInstance,
): ConversationInstance {
  const entry: ConversationInstance = {
    id,
    name: getConversationHistoryName(conversation, fallback?.name),
    message: fallback?.message ?? [],
  };
  const model = conversation?.model ?? fallback?.model;

  if (model) entry.model = model;
  if (fallback?.shared !== undefined) entry.shared = fallback.shared;
  if (fallback?.favorite !== undefined) entry.favorite = fallback.favorite;
  entry.updated_at = conversation?.updated_at ?? fallback?.updated_at;
  if (fallback?.local_key) entry.local_key = fallback.local_key;

  return entry;
}

function dedupeStableHistory(
  history: ConversationInstance[],
): ConversationInstance[] {
  const seen = new Set<number>();
  return history.filter((item) => {
    if (item.id === -1) return false;
    if (seen.has(item.id)) return false;

    seen.add(item.id);
    return true;
  });
}

function preserveLocalHistoryKeys(
  incoming: ConversationInstance[],
  localHistory: ConversationInstance[],
): ConversationInstance[] {
  const localKeys = new Map(
    localHistory
      .filter((item) => item.local_key)
      .map((item) => [item.id, item.local_key as string]),
  );

  return incoming.map((item) => {
    const localKey = localKeys.get(item.id);
    return localKey ? { ...item, local_key: localKey } : item;
  });
}

function promotePendingConversationHistory(
  history: ConversationInstance[],
  id: number,
  conversation: ConversationSerialized,
): ConversationInstance[] {
  const pending = history.find((item) => item.id === -1);
  const existing = history.find((item) => item.id === id);
  const promoted = buildConversationHistoryEntry(
    id,
    conversation,
    pending ?? existing,
  );
  const rest = history.filter((item) => item.id !== -1 && item.id !== id);

  return [promoted, ...rest];
}

function reconcileConversationHistory(
  incoming: ConversationInstance[],
  current: number,
  localHistory: ConversationInstance[],
  conversations: Record<number, ConversationSerialized>,
): ConversationInstance[] {
  const stable = preserveLocalHistoryKeys(
    dedupeStableHistory(incoming),
    localHistory,
  );

  if (current === -1) {
    const pending = localHistory.find((item) => item.id === -1);
    if (pending && conversations[-1]?.messages.length > 0) {
      return [pending, ...stable];
    }

    return stable;
  }

  if (stable.some((item) => item.id === current)) {
    return stable;
  }

  const activeConversation = conversations[current];
  const existing =
    localHistory.find((item) => item.id === current) ??
    localHistory.find((item) => item.id === -1);

  if (!activeConversation && !existing) return stable;

  return [
    buildConversationHistoryEntry(current, activeConversation, existing),
    ...stable,
  ];
}

function shouldReplaceConversation(
  currentConversation: ConversationSerialized | undefined,
  incoming: ConversationInstance,
  requestedRevision: number,
): boolean {
  if (!currentConversation) return true;
  if (getConversationLocalRevision(currentConversation) !== requestedRevision)
    return false;

  const currentVersion = parseConversationVersion(
    currentConversation.updated_at,
  );
  const incomingVersion = parseConversationVersion(incoming.updated_at);

  if (hasPendingLocalMutation(currentConversation)) {
    if (isStreamingConversation(currentConversation)) return false;
    if (currentVersion !== undefined && incomingVersion !== undefined) {
      return incomingVersion > currentVersion;
    }

    return false;
  }

  if (currentConversation.cache_complete === false) return true;

  if (incoming.message.length < currentConversation.messages.length) {
    if (isStreamingConversation(currentConversation)) return false;
    if (currentVersion !== undefined && incomingVersion !== undefined) {
      return incomingVersion > currentVersion;
    }

    return false;
  }

  if (currentVersion !== undefined && incomingVersion !== undefined) {
    if (incomingVersion !== currentVersion) {
      return incomingVersion > currentVersion;
    }
  }

  return incoming.message.length >= currentConversation.messages.length;
}

function shouldReplaceCachedConversation(
  currentConversation: ConversationSerialized | undefined,
  incomingConversation: ConversationSerialized,
): boolean {
  if (!currentConversation) return true;
  return (
    incomingConversation.cache_complete !== false ||
    currentConversation.cache_complete === false
  );
}

function markConversationPending(conversation: ConversationSerialized) {
  conversation.local_revision = getConversationLocalRevision(conversation) + 1;
  conversation.local_pending_until = Date.now() + localMutationProtectionMs;
  conversation.cache_complete = false;
  conversation.server_synced = false;
}

function hasPendingLocalMutation(
  conversation: ConversationSerialized,
): boolean {
  return (conversation.local_pending_until ?? 0) > Date.now();
}

function getConversationLocalRevision(
  conversation: ConversationSerialized | undefined,
): number {
  return conversation?.local_revision ?? 0;
}

function isStreamingConversation(
  conversation: ConversationSerialized,
): boolean {
  const last = conversation.messages[conversation.messages.length - 1];
  return (
    last?.role === AssistantRole &&
    (last.end === false || last.status === "streaming")
  );
}

function summarizeIncomingStreamMessage(
  message: StreamMessage,
): Record<string, unknown> {
  return {
    conversation: message.conversation,
    message_length:
      typeof message.message === "string" ? message.message.length : undefined,
    end: message.end,
    keyword: message.keyword,
    quota: message.quota,
    plan: message.plan,
    has_title: Boolean(message.title),
    response_type: message.response_type,
    search_query_count: message.search_query?.search_queries.length,
    search_result_count: message.search_result?.search_results.length,
    search_index_count: message.search_index?.search_indexes.length,
    tool_call: message.tool_call
      ? {
          name: message.tool_call.name,
          status: message.tool_call.status,
          has_arguments: Boolean(message.tool_call.arguments),
          has_result: Boolean(message.tool_call.result),
          has_error: Boolean(message.tool_call.error),
        }
      : undefined,
  };
}

function hasAssistantStreamPayload(message: StreamMessage): boolean {
  return (
    (message.message ?? "").length > 0 ||
    Boolean(message.keyword) ||
    Boolean(message.tool_call)
  );
}

function hasAssistantStreamUpdate(message: StreamMessage): boolean {
  return (
    hasAssistantStreamPayload(message) ||
    message.end === true ||
    (typeof message.quota === "number" && message.quota > 0) ||
    message.plan === true
  );
}

function parseConversationVersion(
  value: string | undefined,
): number | undefined {
  const trimmed = value?.trim();
  if (!trimmed) return undefined;

  const normalized = trimmed.includes("T")
    ? trimmed
    : trimmed.replace(" ", "T");
  const parsed = Date.parse(normalized);

  return Number.isFinite(parsed) ? parsed : undefined;
}

export function inModel(supportModels: Model[], model: string): boolean {
  return (
    model.length > 0 &&
    supportModels.filter((item: Model) => item.id === model).length > 0
  );
}

export function getChatSupportModels(supportModels: Model[]): Model[] {
  return supportModels.filter((item) => !isDrawingModel(item));
}

export function inChatModel(supportModels: Model[], model: string): boolean {
  return inModel(getChatSupportModels(supportModels), model);
}

export function getModel(
  supportModels: Model[],
  model: string | undefined | null,
): string {
  const chatModels = getChatSupportModels(supportModels);
  if (chatModels.length === 0) return "";
  return model && inModel(chatModels, model) ? model : chatModels[0].id;
}

export function getModelList(
  supportModels: Model[],
  models: string[],
): string[] {
  const chatModels = getChatSupportModels(supportModels);
  return models.filter((item) => inModel(chatModels, item));
}

export function isGeminiModelId(model: string | undefined | null): boolean {
  if (!model) return false;
  return (
    model === "gemini-pro" ||
    model === "gemini-pro-vision" ||
    model.startsWith("gemini-")
  );
}

export function isXAIModelId(model: string | undefined | null): boolean {
  return !!model && model.toLowerCase().startsWith("grok");
}

export function isDeepSeekV4ModelId(model: string | undefined | null): boolean {
  return getDeepSeekV4ModelKey(model) !== undefined;
}

function getDeepSeekV4ModelKey(
  model: string | undefined | null,
): string | undefined {
  if (!model) return undefined;
  const normalized = model.trim().toLowerCase();
  return normalized === "deepseek-v4-flash" || normalized === "deepseek-v4-pro"
    ? normalized
    : undefined;
}

export type OpenAIResponsesCapabilities = {
  nativeWeb: boolean;
  reasoningEfforts: string[];
  reasoningSummary: boolean;
};

function emptyOpenAIResponsesCapabilities(): OpenAIResponsesCapabilities {
  return { nativeWeb: false, reasoningEfforts: [], reasoningSummary: false };
}

function isXiaomiMiMoModel(model: string): boolean {
  const normalized = model
    .trim()
    .toLowerCase()
    .replace(/^xiaomi\//, "");
  return normalized.startsWith("mimo-v2") && !normalized.includes("tts");
}

function restrictMaintainedReasoningCapabilities(
  model: Model,
  capabilities: OpenAIResponsesCapabilities,
): OpenAIResponsesCapabilities {
  if (model.reasoning_configurable !== false) return capabilities;

  const configured = normalizeConfiguredReasoningEfforts(
    model.reasoning_efforts,
  );
  if (configured.length === 0) return capabilities;

  return {
    ...capabilities,
    reasoningEfforts: capabilities.reasoningEfforts.filter((effort) =>
      configured.includes(effort),
    ),
  };
}

export function getOpenAIResponsesCapabilities(
  supportModels: Model[],
  model: string | undefined | null,
): OpenAIResponsesCapabilities {
  if (!model) {
    return emptyOpenAIResponsesCapabilities();
  }
  const current = supportModels.find((item) => item.id === model);
  if (!current) {
    return emptyOpenAIResponsesCapabilities();
  }

  const channelType = (current.channel_type || "").toLowerCase();
  const normalized = model.trim().toLowerCase();
  if (channelType === "xiaomi-mimo" || channelType === "xiaomi-token-plan-cn") {
    return isXiaomiMiMoModel(normalized)
      ? restrictMaintainedReasoningCapabilities(current, {
          nativeWeb: false,
          reasoningEfforts: ["none", "high"],
          reasoningSummary: false,
        })
      : emptyOpenAIResponsesCapabilities();
  }

  if (channelType === "xai") {
    const maintained = getMaintainedReasoningEfforts(normalized);
    return maintained && maintained.length > 0
      ? restrictMaintainedReasoningCapabilities(current, {
          nativeWeb: false,
          reasoningEfforts: maintained,
          reasoningSummary: false,
        })
      : emptyOpenAIResponsesCapabilities();
  }

  if (current.reasoning_model && current.reasoning_configurable !== false) {
    const reasoningEfforts = normalizeConfiguredReasoningEfforts(
      current.reasoning_efforts,
    );
    if (reasoningEfforts.length > 0) {
      return {
        nativeWeb: false,
        reasoningEfforts,
        reasoningSummary: false,
      };
    }
  }

  if (channelType !== "openai-responses") {
    return emptyOpenAIResponsesCapabilities();
  }

  if (normalized === "gpt-5.6" || normalized.startsWith("gpt-5.6-")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh", "max"],
      reasoningSummary: true,
    });
  }
  if (normalized === "gpt-5.5" || normalized.startsWith("gpt-5.5-")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    });
  }
  if (normalized.startsWith("gpt-5.4-pro")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["medium", "high", "xhigh"],
      reasoningSummary: true,
    });
  }
  if (normalized.startsWith("gpt-5.4-mini")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    });
  }
  if (normalized.startsWith("gpt-5.4-nano")) {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "gpt-5.2-pro" || normalized.startsWith("gpt-5.2-pro-")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["medium", "high", "xhigh"],
      reasoningSummary: true,
    });
  }
  if (normalized === "gpt-5.2-chat-latest") {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "gpt-5-pro" || normalized.startsWith("gpt-5-pro-")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["high"],
      reasoningSummary: true,
    });
  }
  if (normalized === "gpt-5-mini" || normalized.startsWith("gpt-5-mini-")) {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "gpt-5-nano" || normalized.startsWith("gpt-5-nano-")) {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized.startsWith("gpt-5.4")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    });
  }
  if (normalized.startsWith("gpt-5.2")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    });
  }
  if (normalized.startsWith("gpt-5.1")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high"],
      reasoningSummary: true,
    });
  }
  if (normalized === "gpt-5") {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["minimal", "low", "medium", "high"],
      reasoningSummary: true,
    });
  }
  if (normalized === "gpt-5.3-chat-latest") {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "o3" || normalized.startsWith("o3-")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: true,
      reasoningEfforts: ["low", "medium", "high"],
      reasoningSummary: true,
    });
  }
  if (normalized === "o1" || normalized.startsWith("o1-")) {
    return restrictMaintainedReasoningCapabilities(current, {
      nativeWeb: false,
      reasoningEfforts: ["low", "medium", "high"],
      reasoningSummary: true,
    });
  }
  if (normalized.startsWith("gpt-4.5")) {
    return emptyOpenAIResponsesCapabilities();
  }

  return emptyOpenAIResponsesCapabilities();
}

export function isOpenAIResponsesNativeWebModel(
  supportModels: Model[],
  model: string | undefined | null,
): boolean {
  return getOpenAIResponsesCapabilities(supportModels, model).nativeWeb;
}

export function supportsOpenAIResponsesReasoningControl(
  supportModels: Model[],
  model: string | undefined | null,
): boolean {
  return (
    getOpenAIResponsesCapabilities(supportModels, model).reasoningEfforts
      .length > 0
  );
}

export function normalizeOpenAIResponsesReasoningEffort(
  supportModels: Model[],
  model: string | undefined | null,
  effort: string | undefined | null,
): string | undefined {
  const capabilities = getOpenAIResponsesCapabilities(supportModels, model);
  const normalized = (effort || "").trim().toLowerCase();
  if (!normalized) return undefined;
  return capabilities.reasoningEfforts.includes(normalized)
    ? normalized
    : undefined;
}

export function normalizeDeepSeekReasoningEffort(
  effort: string | undefined | null,
): string {
  const normalized = (effort || "").trim().toLowerCase();
  if (normalized === "max" || normalized === "xhigh") return "max";
  return "high";
}

function getDeepSeekThinkingMemoryKey(model: string): string {
  return `deepseek_thinking_enabled:${model}`;
}

function getDeepSeekReasoningEffortMemoryKey(model: string): string {
  return `deepseek_reasoning_effort:${model}`;
}

function getInitialDeepSeekThinkingEnabledByModel(
  currentModel: string,
): Record<string, boolean> {
  const currentKey = getDeepSeekV4ModelKey(currentModel);

  return {
    "deepseek-v4-flash": getBooleanMemory(
      getDeepSeekThinkingMemoryKey("deepseek-v4-flash"),
      currentKey === "deepseek-v4-flash"
        ? getMemory("deepseek_thinking_enabled") !== "false"
        : true,
    ),
    "deepseek-v4-pro": getBooleanMemory(
      getDeepSeekThinkingMemoryKey("deepseek-v4-pro"),
      currentKey === "deepseek-v4-pro"
        ? getMemory("deepseek_thinking_enabled") !== "false"
        : true,
    ),
  };
}

function getInitialDeepSeekReasoningEffortByModel(
  currentModel: string,
): Record<string, string> {
  const currentKey = getDeepSeekV4ModelKey(currentModel);

  return {
    "deepseek-v4-flash": normalizeDeepSeekReasoningEffort(
      getMemory(getDeepSeekReasoningEffortMemoryKey("deepseek-v4-flash")) ||
        (currentKey === "deepseek-v4-flash"
          ? getMemory("deepseek_reasoning_effort")
          : "high"),
    ),
    "deepseek-v4-pro": normalizeDeepSeekReasoningEffort(
      getMemory(getDeepSeekReasoningEffortMemoryKey("deepseek-v4-pro")) ||
        (currentKey === "deepseek-v4-pro"
          ? getMemory("deepseek_reasoning_effort")
          : "high"),
    ),
  };
}

function getDeepSeekThinkingEnabledForModel(
  enabledByModel: Record<string, boolean>,
  model: string | undefined | null,
): boolean {
  const key = getDeepSeekV4ModelKey(model);
  return key ? (enabledByModel[key] ?? true) : false;
}

function getDeepSeekReasoningEffortForModel(
  effortByModel: Record<string, string>,
  model: string | undefined | null,
): string {
  const key = getDeepSeekV4ModelKey(model);
  return normalizeDeepSeekReasoningEffort(key ? effortByModel[key] : "high");
}

export function isGeminiNoThinkingModel(
  model: string | undefined | null,
): boolean {
  return !!model && model.endsWith("-nothinking");
}

export function supportsGeminiThinkingBudgetControl(
  model: string | undefined | null,
): boolean {
  if (!model) return false;
  if (isGeminiNoThinkingModel(model)) return false;
  return (
    model === "gemini-2.5-flash" ||
    model.startsWith("gemini-2.5-flash-preview-") ||
    model === "gemini-2.5-flash-lite" ||
    model.startsWith("gemini-2.5-flash-lite-preview-") ||
    model === "gemini-2.5-pro" ||
    model.startsWith("gemini-2.5-pro-preview-") ||
    model.startsWith("gemini-2.5-pro-exp-") ||
    model === "gemini-3.6-flash" ||
    model.startsWith("gemini-3.6-flash-") ||
    model === "gemini-3.5-flash" ||
    model.startsWith("gemini-3.5-flash-") ||
    model === "gemini-3-flash-preview" ||
    model.startsWith("gemini-3-flash-preview-") ||
    model === "gemini-3.1-flash-lite-preview" ||
    model.startsWith("gemini-3.1-flash-lite-preview-") ||
    model === "gemini-3.1-pro-preview" ||
    model.startsWith("gemini-3.1-pro-preview-") ||
    model === "gemini-3.1-pro-preview-customtools" ||
    model.startsWith("gemini-3.1-pro-preview-customtools-") ||
    model === "gemini-3.1-flash-image" ||
    model.startsWith("gemini-3.1-flash-image-") ||
    model === "gemini-3-pro-image" ||
    model.startsWith("gemini-3-pro-image-") ||
    model === "gemini-3-pro-preview" ||
    model.startsWith("gemini-3-pro-preview-")
  );
}

const toolStatusPriority: Record<string, number> = {
  start: 0,
  executing: 1,
  pending: 2,
  success: 3,
  error: 3,
};

function findPendingAskUserToolCall(
  messages?: Message[],
): MessageToolCall | undefined {
  if (!messages) return undefined;
  for (let index = messages.length - 1; index >= 0; index--) {
    const message = messages[index];
    if (message.role === "tool") continue;
    if (message.role !== AssistantRole) return undefined;
    const pending = message.tool_calls?.find(isPendingAskUserToolCall);
    if (pending) return pending;
    return undefined;
  }
  return undefined;
}

function normalizeToolArguments(argumentsText?: string): string {
  if (!argumentsText) return "";
  return typeof argumentsText === "string"
    ? argumentsText
    : JSON.stringify(argumentsText);
}

function mergeToolArguments(existing: string, incoming: string): string {
  if (!incoming) return existing;
  if (!existing) return incoming;
  if (existing === incoming) return existing;
  if (incoming.startsWith(existing)) return incoming;
  if (existing.startsWith(incoming)) return existing;
  if (existing.includes(incoming)) return existing;
  return `${existing}${incoming}`;
}

function upsertToolCall(
  current: MessageToolCall[] | undefined,
  incoming: NonNullable<StreamMessage["tool_call"]>,
): MessageToolCall[] {
  const next = current ? [...current] : [];
  const id = incoming.id?.trim() || "";
  const name = incoming.name.trim();
  if (isHiddenToolCallName(name)) {
    return next;
  }
  let hitIndex = -1;

  if (id) {
    hitIndex = next.findIndex((item) => item.id === id);
  }

  if (hitIndex < 0) {
    hitIndex = next.findIndex((item) => item.function.name === name);
  }

  const base: MessageToolCall =
    hitIndex >= 0
      ? next[hitIndex]
      : {
          index: next.length,
          type: "function",
          id,
          function: {
            name,
            arguments: "",
          },
        };

  const merged: MessageToolCall = {
    ...base,
    id: id || base.id,
    function: {
      name: name || base.function.name,
      arguments: mergeToolArguments(
        base.function.arguments,
        normalizeToolArguments(incoming.arguments),
      ),
    },
    status:
      (toolStatusPriority[incoming.status] ?? 0) >=
      (toolStatusPriority[base.status ?? "start"] ?? 0)
        ? incoming.status
        : base.status,
    result: incoming.result ?? base.result,
    error: incoming.error ?? base.error,
  };

  if (hitIndex >= 0) {
    next[hitIndex] = merged;
  } else {
    next.push(merged);
  }

  return next;
}

function finalizePendingToolCalls(
  current: MessageToolCall[] | undefined,
): MessageToolCall[] | undefined {
  if (!current || current.length === 0) return current;

  let changed = false;
  const next = current.map((toolCall) => {
    if (toolCall.status !== "start" && toolCall.status !== "executing") {
      return toolCall;
    }

    changed = true;
    return {
      ...toolCall,
      status: toolCall.error ? "error" : "success",
    } as MessageToolCall;
  });

  return changed ? next : current;
}

export const stack = new ConnectionStack();

function closeConversationConnection(id: number) {
  stack.close(id);
}

function closeAllConversationConnections() {
  stack.closeAll();
}

const offline = loadPreferenceModels(getOfflineModels());
const initialModel = getModel(offline, getMemory("model"));
const chatSlice = createSlice({
  name: "chat",
  initialState: {
    history: [],
    messages: [],
    conversations: {
      [-1]: { ...defaultConversation },
    },
    web: getBooleanMemory("web", false),
    gemini_google_search: getBooleanMemory("gemini_google_search", false),
    gemini_url_context: getBooleanMemory("gemini_url_context", false),
    xai_web_search: getBooleanMemory("xai_web_search", false),
    xai_x_search: getBooleanMemory("xai_x_search", false),
    openai_responses_web_search: getBooleanMemory(
      "openai_responses_web_search",
      false,
    ),
    fetch: getBooleanMemory("fetch", false),
    learning_mode: getBooleanMemory("learning_mode", false),
    gemini_thinking_budget: getNumberMemory("gemini_thinking_budget", 0),
    deepseek_thinking_enabled_by_model:
      getInitialDeepSeekThinkingEnabledByModel(initialModel),
    deepseek_reasoning_effort_by_model:
      getInitialDeepSeekReasoningEffortByModel(initialModel),
    openai_reasoning_effort: getMemory("openai_reasoning_effort") || "none",
    current: -1,
    loadingConversationId: null,
    active_requests: {},
    model: initialModel,
    model_list: getModelList(offline, getArrayMemory("model_mark_list")),
    market: false,
    mask_item: null,
    custom_masks: [],
    support_models: offline,
  } as initialStateType,
  reducers: {
    createMessage: (state, action) => {
      const { id, role, content, model } = action.payload as {
        id: number;
        role: string;
        content?: string;
        model?: string;
      };

      const conversation = state.conversations[id];
      if (!conversation) return;
      markConversationPending(conversation);

      if (role === AssistantRole && model) {
        conversation.model = model;
      }

      conversation.messages.push({
        role: role ?? AssistantRole,
        content: content ?? "",
        model,
        end: role === AssistantRole ? false : undefined,
      });
    },
    fillMaskItem: (state) => {
      const conversation = state.conversations[-1];

      if (state.mask_item && conversation.messages.length === 0) {
        conversation.messages = [...state.mask_item.context];
        state.mask_item = null;
      }
    },
    updateMessage: (state, action) => {
      const { id, message, model } = action.payload as {
        id: number;
        message: StreamMessage;
        model?: string;
      };
      const conversation = state.conversations[id];
      if (!conversation) return;
      const content = message.message ?? "";
      const hasPayload = hasAssistantStreamPayload(message);
      const hasUpdate = hasAssistantStreamUpdate(message);
      const last = conversation.messages[conversation.messages.length - 1];
      const hasAssistantTarget = last?.role === AssistantRole;

      if (!hasAssistantTarget && !hasPayload) return;
      if (hasAssistantTarget && !hasUpdate) return;

      markConversationPending(conversation);

      if (!hasAssistantTarget) {
        if (model) {
          conversation.model = model;
        }
        conversation.messages.push({
          role: AssistantRole,
          content: "",
          model,
          keyword: message.keyword,
          quota: message.quota,
          end: message.end,
          plan: message.plan,
        });
      }

      const instance = conversation.messages[conversation.messages.length - 1];
      if (content.length > 0) instance.content += content;
      if (!instance.model && model) instance.model = model;
      if (message.keyword) instance.keyword = message.keyword;
      if (typeof message.quota === "number" && message.quota > 0) {
        instance.quota = message.quota;
      }
      if (message.tool_call) {
        instance.tool_calls = upsertToolCall(
          instance.tool_calls,
          message.tool_call,
        );
      }
      if (message.end) {
        instance.end = message.end;
        instance.tool_calls = finalizePendingToolCalls(instance.tool_calls);
        delete state.active_requests[id];
      }
      if (hasPayload || message.end === true || message.plan === true) {
        instance.plan = message.plan;
      }
    },
    removeMessage: (state, action) => {
      const { id, idx } = action.payload as { id: number; idx: number };
      const conversation = state.conversations[id];
      if (!conversation) return;
      markConversationPending(conversation);

      conversation.messages.splice(idx, 1);
    },
    restartMessage: (state, action) => {
      const { id, model } = action.payload as { id: number; model?: string };
      const conversation = state.conversations[id];
      if (!conversation || conversation.messages.length === 0) return;
      markConversationPending(conversation);

      if (model) {
        conversation.model = model;
      }

      conversation.messages.push({
        role: AssistantRole,
        content: "",
        model,
        end: false,
      });
    },
    answerAskUserMessage: (state, action) => {
      const { id, toolCallId, result, model } = action.payload as {
        id: number;
        toolCallId: string;
        result: AskUserResult;
        model?: string;
      };
      const conversation = state.conversations[id];
      if (!conversation) return;

      const pending = findPendingAskUserToolCall(conversation.messages);
      if (!pending || pending.id !== toolCallId) return;

      markConversationPending(conversation);
      pending.result = JSON.stringify(result);
      pending.status = "success";
      conversation.messages.push({
        role: AssistantRole,
        content: "",
        model,
        end: false,
      });
    },
    editMessage: (state, action) => {
      const { id, idx, message } = action.payload as {
        id: number;
        idx: number;
        message: string;
      };
      const conversation = state.conversations[id];
      if (!conversation || conversation.messages.length <= idx) return;
      markConversationPending(conversation);

      conversation.messages[idx].content = message;
    },
    stopMessage: (state, action) => {
      const { id } = action.payload as { id: number };
      const conversation = state.conversations[id];
      if (!conversation || conversation.messages.length === 0) return;
      markConversationPending(conversation);

      conversation.messages[conversation.messages.length - 1].end = true;
      delete state.active_requests[id];
    },
    startGenerationRequest: (state, action) => {
      const { id, requestId } = action.payload as {
        id: number;
        requestId: string;
      };
      state.active_requests[id] = requestId;
    },
    finishGenerationRequest: (state, action) => {
      const { id, requestId } = action.payload as {
        id: number;
        requestId?: string;
      };
      const activeRequestId = state.active_requests[id];
      if (!activeRequestId) return;
      if (requestId && activeRequestId !== requestId) return;
      delete state.active_requests[id];
    },
    raiseConversation: (state, action) => {
      // raise conversation `-1` to target id
      const id = action.payload as number;
      const conversation = state.conversations[-1];
      if (!conversation || id === -1) return;

      state.conversations[id] = conversation;
      if (state.current === -1) state.current = id;
      state.history = promotePendingConversationHistory(
        state.history,
        id,
        conversation,
      );
      if (state.active_requests[-1]) {
        state.active_requests[id] = state.active_requests[-1];
        delete state.active_requests[-1];
      }

      state.conversations[-1] = { ...defaultConversation };
    },
    importConversation: (state, action) => {
      const { conversation, id } = action.payload as {
        conversation: ConversationSerialized;
        id: number;
      };

      if (state.conversations[id]) return;
      state.conversations[id] = conversation;
    },
    setConversation: (state, action) => {
      const { conversation, id } = action.payload as {
        conversation: ConversationSerialized;
        id: number;
      };

      if (
        !shouldReplaceCachedConversation(state.conversations[id], conversation)
      ) {
        return;
      }

      state.conversations[id] = conversation;
      if (state.current === id) {
        state.loadingConversationId = null;
      }
    },
    setRemoteConversation: (state, action) => {
      const { conversation, id, requestedRevision } = action.payload as {
        conversation: ConversationInstance;
        id: number;
        requestedRevision: number;
      };

      if (
        !shouldReplaceConversation(
          state.conversations[id],
          conversation,
          requestedRevision,
        )
      ) {
        return;
      }

      const nextConversation = {
        model: conversation.model,
        messages: conversation.message,
        updated_at: conversation.updated_at,
        cache_complete: true,
        server_synced: true,
      };
      state.conversations[id] = nextConversation;
      if (state.current === id) {
        state.loadingConversationId = null;
      }
      void setCachedConversation(id, nextConversation);

      const index = state.history.findIndex((item) => item.id === id);
      const previous = index >= 0 ? state.history[index] : undefined;
      const next = {
        id,
        name: conversation.name || previous?.name || "",
        message: previous?.message ?? [],
        model: conversation.model ?? previous?.model,
        shared: conversation.shared ?? previous?.shared,
        favorite: conversation.favorite ?? previous?.favorite,
        updated_at: conversation.updated_at ?? previous?.updated_at,
        local_key: previous?.local_key ?? conversation.local_key,
      };

      if (index >= 0) {
        state.history[index] = next;
        return;
      }

      state.history = [next, ...state.history];
    },
    deleteRemoteConversation: (state, action) => {
      const { id, requestedRevision } = action.payload as {
        id: number;
        requestedRevision: number;
      };

      if (id === -1) return;

      const conversation = state.conversations[id];
      if (
        conversation &&
        (getConversationLocalRevision(conversation) !== requestedRevision ||
          isStreamingConversation(conversation))
      ) {
        return;
      }

      state.history = state.history.filter((item) => item.id !== id);

      if (state.current === id) {
        bumpConversationNavigationRevision();
        state.current = -1;
        state.messages = [];
        state.loadingConversationId = null;
        state.mask_item = null;
        setNumberMemory("history_conversation", -1);
      }

      if (getNumberMemory("history_conversation", -1) === id) {
        setNumberMemory("history_conversation", -1);
      }

      void clearCachedConversation(id);
      if (!state.conversations[id]) return;

      closeConversationConnection(id);
      delete state.active_requests[id];
      delete state.conversations[id];
    },
    deleteConversation: (state, action) => {
      const id = action.payload as number;

      if (id === -1) return;

      state.history = state.history.filter((item) => item.id !== id);

      if (state.current === id) {
        bumpConversationNavigationRevision();
        state.current = -1;
        state.messages = [];
        state.loadingConversationId = null;
        state.mask_item = null;
        setNumberMemory("history_conversation", -1);
      }

      if (getNumberMemory("history_conversation", -1) === id) {
        setNumberMemory("history_conversation", -1);
      }

      if (!state.conversations[id]) return;

      closeConversationConnection(id);
      delete state.active_requests[id];
      delete state.conversations[id];
    },
    deleteAllConversation: (state) => {
      closeAllConversationConnections();
      resetLocalConversationState(state);
    },
    setHistory: (state, action) => {
      state.history = reconcileConversationHistory(
        action.payload as ConversationInstance[],
        state.current,
        state.history,
        state.conversations,
      );
    },
    setRemoteHistory: (state, action) => {
      replaceWithRemoteConversationHistory(
        state,
        action.payload as ConversationInstance[],
      );
    },
    preflightHistory: (state, action) => {
      const { localKey, name } = action.payload as {
        localKey: string;
        name: string;
      };

      // add a new history at the beginning
      state.history = [
        { id: -1, name, message: [], local_key: localKey },
        ...state.history.filter((item) => item.id !== -1),
      ];
    },
    renameHistory: (state, action) => {
      const { id, name } = action.payload as { id: number; name: string };
      const conversation = state.history.find((item) => item.id === id);
      if (conversation) conversation.name = name;
    },
    favoriteHistory: (state, action) => {
      const { id, favorite } = action.payload as {
        id: number;
        favorite: boolean;
      };
      const conversation = state.history.find((item) => item.id === id);
      if (conversation) conversation.favorite = favorite;
    },
    upsertHistory: (state, action) => {
      const incoming = action.payload as ConversationInstance;
      if (incoming.id === -1) return;

      const index = state.history.findIndex((item) => item.id === incoming.id);
      const previous = index >= 0 ? state.history[index] : undefined;
      const next = {
        id: incoming.id,
        name: incoming.name || previous?.name || "",
        message: incoming.message ?? previous?.message ?? [],
        model: incoming.model ?? previous?.model,
        shared: incoming.shared ?? previous?.shared,
        favorite: incoming.favorite ?? previous?.favorite,
        updated_at: incoming.updated_at ?? previous?.updated_at,
        local_key: previous?.local_key ?? incoming.local_key,
      };

      if (index >= 0) {
        state.history[index] = next;
        return;
      }

      state.history = [next, ...state.history];
    },
    setModel: (state, action) => {
      const model = action.payload as string;
      if (!model || model === "") return;
      if (!inChatModel(state.support_models, model)) return;

      // if model is not in model list, add it
      // if (!state.model_list.includes(model)) {
      //   console.log("[model] auto add model to list:", model);
      //   state.model_list.push(model);
      //   setArrayMemory("model_mark_list", state.model_list);
      // }

      setMemory("model", model as string);
      state.model = action.payload as string;

      const conversation = state.conversations[state.current];
      if (conversation) {
        markConversationPending(conversation);
        conversation.model = model;
      }

      const historyConversation = state.history.find(
        (item) => item.id === state.current,
      );
      if (historyConversation) {
        historyConversation.model = model;
      }
    },
    setWeb: (state, action) => {
      setMemory("web", action.payload ? "true" : "false");
      state.web = action.payload as boolean;
    },
    toggleWeb: (state) => {
      const web = !state.web;
      setMemory("web", web ? "true" : "false");
      state.web = web;
    },
    setGeminiGoogleSearch: (state, action) => {
      setMemory("gemini_google_search", action.payload ? "true" : "false");
      state.gemini_google_search = action.payload as boolean;
    },
    setGeminiURLContext: (state, action) => {
      setMemory("gemini_url_context", action.payload ? "true" : "false");
      state.gemini_url_context = action.payload as boolean;
    },
    setXAIWebSearch: (state, action) => {
      setMemory("xai_web_search", action.payload ? "true" : "false");
      state.xai_web_search = action.payload as boolean;
    },
    setXAIXSearch: (state, action) => {
      setMemory("xai_x_search", action.payload ? "true" : "false");
      state.xai_x_search = action.payload as boolean;
    },
    setOpenAIResponsesWebSearch: (state, action) => {
      setMemory(
        "openai_responses_web_search",
        action.payload ? "true" : "false",
      );
      state.openai_responses_web_search = action.payload as boolean;
    },
    setFetch: (state, action) => {
      setMemory("fetch", action.payload ? "true" : "false");
      state.fetch = action.payload as boolean;
    },
    setLearningMode: (state, action) => {
      setMemory("learning_mode", action.payload ? "true" : "false");
      state.learning_mode = action.payload as boolean;
    },
    setGeminiThinkingBudget: (state, action) => {
      setNumberMemory("gemini_thinking_budget", action.payload as number);
      state.gemini_thinking_budget = action.payload as number;
    },
    setDeepSeekThinkingEnabled: (state, action) => {
      const enabled = action.payload as boolean;
      const modelKey = getDeepSeekV4ModelKey(state.model);
      if (!modelKey) return;

      setMemory(
        getDeepSeekThinkingMemoryKey(modelKey),
        enabled ? "true" : "false",
      );
      state.deepseek_thinking_enabled_by_model[modelKey] = enabled;
    },
    setDeepSeekReasoningEffort: (state, action) => {
      const effort = normalizeDeepSeekReasoningEffort(action.payload as string);
      const modelKey = getDeepSeekV4ModelKey(state.model);
      if (!modelKey) return;

      setMemory(getDeepSeekReasoningEffortMemoryKey(modelKey), effort);
      state.deepseek_reasoning_effort_by_model[modelKey] = effort;
    },
    setOpenAIReasoningEffort: (state, action) => {
      setMemory("openai_reasoning_effort", action.payload as string);
      state.openai_reasoning_effort = action.payload as string;
    },
    setCurrent: (state, action) => {
      const current = action.payload as number;
      state.current = current;
      state.loadingConversationId = null;

      const conversation = state.conversations[current];
      if (!conversation) return;
      if (
        conversation.model &&
        inChatModel(state.support_models, conversation.model)
      ) {
        state.model = conversation.model;
      }
    },
    setActiveConversation: (state, action) => {
      const current = action.payload as number;
      state.current = current;
      state.loadingConversationId =
        current !== -1 && !state.conversations[current] ? current : null;

      const conversation = state.conversations[current];
      if (
        conversation?.model &&
        inChatModel(state.support_models, conversation.model)
      ) {
        state.model = conversation.model;
      }
    },
    clearConversationLoading: (state, action) => {
      const id = action.payload as number;
      if (state.loadingConversationId === id) {
        state.loadingConversationId = null;
      }
    },
    setModelList: (state, action) => {
      const models = action.payload as string[];
      state.model_list = models.filter((item) =>
        inChatModel(state.support_models, item),
      );
      setArrayMemory("model_mark_list", state.model_list);
    },
    addModelList: (state, action) => {
      const model = action.payload as string;
      if (
        inChatModel(state.support_models, model) &&
        !state.model_list.includes(model)
      ) {
        state.model_list.push(model);
        setArrayMemory("model_mark_list", state.model_list);
      }
    },
    removeModelList: (state, action) => {
      const model = action.payload as string;
      if (
        inChatModel(state.support_models, model) &&
        state.model_list.includes(model)
      ) {
        state.model_list = state.model_list.filter((item) => item !== model);
        setArrayMemory("model_mark_list", state.model_list);
      }
    },
    setMaskItem: (state, action) => {
      state.mask_item = action.payload as Mask;
    },
    startMaskedConversation: (state, action) => {
      const mask = action.payload as Mask;
      closeConversationConnection(-1);
      bumpConversationNavigationRevision();

      const nextConversation: ConversationSerialized = {
        ...defaultConversation,
      };
      if (mask.model) {
        nextConversation.model = mask.model;
        if (inChatModel(state.support_models, mask.model)) {
          state.model = mask.model;
          setMemory("model", mask.model);
        }
      }

      state.current = -1;
      state.messages = [];
      state.loadingConversationId = null;
      state.history = state.history.filter((item) => item.id !== -1);
      state.conversations[-1] = nextConversation;
      state.mask_item = mask;
      setNumberMemory("history_conversation", -1);
    },
    clearMaskItem: (state) => {
      state.mask_item = null;
    },
    setCustomMasks: (state, action) => {
      state.custom_masks = action.payload as CustomMask[];
    },
    setSupportModels: (state, action) => {
      const models = action.payload as Model[];
      const maskedModel =
        state.current === -1 ? state.conversations[-1]?.model : undefined;
      const preferredModel =
        maskedModel && inChatModel(models, maskedModel)
          ? maskedModel
          : getMemory("model");

      state.support_models = models;
      state.model = getModel(models, preferredModel);
      state.model_list = getModelList(
        models,
        getArrayMemory("model_mark_list"),
      );

      setOfflineModels(models);
    },
  },
  extraReducers: (builder) => {
    builder.addCase("auth/logout", (state) => {
      resetLocalConversationState(state);
    });
  },
});

export const {
  setHistory,
  setRemoteHistory,
  renameHistory,
  favoriteHistory,
  upsertHistory,
  setCurrent,
  setActiveConversation,
  clearConversationLoading,
  setModel,
  setWeb,
  toggleWeb,
  setGeminiGoogleSearch,
  setGeminiURLContext,
  setXAIWebSearch,
  setXAIXSearch,
  setOpenAIResponsesWebSearch,
  setFetch,
  setLearningMode,
  setGeminiThinkingBudget,
  setDeepSeekThinkingEnabled,
  setDeepSeekReasoningEffort,
  setOpenAIReasoningEffort,
  setModelList,
  addModelList,
  removeModelList,
  setCustomMasks,
  setSupportModels,
  setMaskItem,
  startMaskedConversation,
  clearMaskItem,
  fillMaskItem,
  createMessage,
  updateMessage,
  removeMessage,
  restartMessage,
  answerAskUserMessage,
  editMessage,
  stopMessage,
  startGenerationRequest,
  finishGenerationRequest,
  raiseConversation,
  importConversation,
  setConversation,
  setRemoteConversation,
  deleteRemoteConversation,
  deleteConversation,
  deleteAllConversation,
  preflightHistory,
} = chatSlice.actions;
export const selectHistory = (state: RootState): ConversationInstance[] =>
  state.chat.history;
export const selectConversations = (
  state: RootState,
): Record<number, ConversationSerialized> => state.chat.conversations;
export const selectModel = (state: RootState): string => state.chat.model;
export const selectWeb = (state: RootState): boolean => state.chat.web;
export const selectGeminiGoogleSearch = (state: RootState): boolean =>
  state.chat.gemini_google_search;
export const selectGeminiURLContext = (state: RootState): boolean =>
  state.chat.gemini_url_context;
export const selectXAIWebSearch = (state: RootState): boolean =>
  state.chat.xai_web_search;
export const selectXAIXSearch = (state: RootState): boolean =>
  state.chat.xai_x_search;
export const selectOpenAIResponsesWebSearch = (state: RootState): boolean =>
  state.chat.openai_responses_web_search;
export const selectFetch = (state: RootState): boolean => state.chat.fetch;
export const selectLearningMode = (state: RootState): boolean =>
  state.chat.learning_mode;
export const selectGeminiThinkingBudget = (state: RootState): number =>
  state.chat.gemini_thinking_budget;
export const selectDeepSeekThinkingEnabled = (state: RootState): boolean =>
  getDeepSeekThinkingEnabledForModel(
    state.chat.deepseek_thinking_enabled_by_model,
    state.chat.model,
  );
export const selectDeepSeekReasoningEffort = (state: RootState): string =>
  getDeepSeekReasoningEffortForModel(
    state.chat.deepseek_reasoning_effort_by_model,
    state.chat.model,
  );
export const selectDeepSeekThinkingEnabledByModel = (
  state: RootState,
): Record<string, boolean> => state.chat.deepseek_thinking_enabled_by_model;
export const selectDeepSeekReasoningEffortByModel = (
  state: RootState,
): Record<string, string> => state.chat.deepseek_reasoning_effort_by_model;
export const selectOpenAIReasoningEffort = (state: RootState): string =>
  state.chat.openai_reasoning_effort;
export const selectCurrent = (state: RootState): number => state.chat.current;
export const selectCurrentGenerationActive = (state: RootState): boolean =>
  Boolean(state.chat.active_requests[state.chat.current]);
export const selectConversationLoading = (state: RootState): boolean =>
  state.chat.loadingConversationId === state.chat.current;
export const selectModelList = (state: RootState): string[] =>
  state.chat.model_list;
export const selectCustomMasks = (state: RootState): CustomMask[] =>
  state.chat.custom_masks;
export const selectSupportModels = (state: RootState): Model[] =>
  state.chat.support_models;
export const selectChatSupportModels = createSelector(
  [selectSupportModels],
  getChatSupportModels,
);
export const selectMaskItem = (state: RootState): Mask | null =>
  state.chat.mask_item;

export function useConversation(): ConversationSerialized | undefined {
  const conversations = useSelector(selectConversations);
  const current = useSelector(selectCurrent);

  return useMemo(() => conversations[current], [conversations, current]);
}

export function useConversationActions() {
  const dispatch = useDispatch();
  const conversations = useSelector(selectConversations);
  const current = useSelector(selectCurrent);
  const mask = useSelector(selectMaskItem);

  const applyConversationListResult = async (
    resp: Awaited<ReturnType<typeof fetchConversationList>>,
  ): Promise<boolean> => {
    if (
      !(await isCurrentConversationListCacheGeneration(resp.cacheGeneration))
    ) {
      return false;
    }

    if (!resp.fromCache || resp.conversations.length > 0) {
      dispatch(
        resp.fromCache
          ? setHistory(resp.conversations)
          : setRemoteHistory(resp.conversations),
      );
    }

    return true;
  };

  const refreshConversationDetail = async (
    id: number,
    options?: { activate?: boolean; navigationRevision?: number },
  ) => {
    if (id === -1) return;
    const activate = options?.activate ?? true;
    const requestedRevision = getConversationLocalRevision(conversations[id]);
    const requestSeq = nextConversationDetailRequestSeq(id);

    const result = await fetchConversation(id);
    if (!isLatestConversationDetailRequest(id, requestSeq)) return;

    if (result.status === "not_found") {
      const list = await fetchConversationList();
      if (!isLatestConversationDetailRequest(id, requestSeq)) return;
      if (!(await applyConversationListResult(list))) return;

      if (!list.fromCache) {
        if (!list.conversations.some((item) => item.id === id)) {
          dispatch(deleteRemoteConversation({ id, requestedRevision }));
          return;
        }
      }

      dispatch(clearConversationLoading(id));
      return;
    }
    if (result.status !== "ok") {
      dispatch(clearConversationLoading(id));
      return;
    }

    const data = result.conversation;
    dispatch(
      setRemoteConversation({
        conversation: data,
        id,
        requestedRevision,
      }),
    );
    if (
      activate &&
      getNumberMemory("history_conversation", -1) === id &&
      (options?.navigationRevision === undefined ||
        options.navigationRevision === getConversationNavigationRevision())
    ) {
      dispatch(setCurrent(id));
    }
  };

  const showConversation = async (
    id: number,
    options?: { refreshRemote?: boolean; useCache?: boolean },
  ) => {
    const refreshRemote = options?.refreshRemote ?? true;
    const navigationRevision = bumpConversationNavigationRevision();
    setNumberMemory("history_conversation", id);

    if (id === -1) {
      if (current === -1 && conversations[-1].messages.length === 0) {
        mask && dispatch(clearMaskItem());
      }
      dispatch(setActiveConversation(id));
      return;
    }

    let restoredConversation = conversations[id];
    let restored = Boolean(restoredConversation);
    dispatch(setActiveConversation(id));
    if (!restored && options?.useCache) {
      const cached = await getCachedConversation(id);
      if (
        navigationRevision !== getConversationNavigationRevision() ||
        getNumberMemory("history_conversation", -1) !== id
      ) {
        return;
      }
      if (cached) {
        restoredConversation = {
          model: cached.model,
          messages: cached.messages,
          updated_at: cached.updated_at,
          cache_complete: cached.cache_complete === true,
          server_synced: cached.server_synced === true,
        };
        dispatch(
          setConversation({
            conversation: restoredConversation,
            id,
          }),
        );
        restored = true;
      }
    }

    if (current === -1 && conversations[-1].messages.length === 0) {
      mask && dispatch(clearMaskItem());
    }

    if (restored) {
      if (refreshRemote) {
        void refreshConversationDetail(id, {
          activate: false,
        });
      }
      return;
    }

    if (!refreshRemote) {
      dispatch(clearConversationLoading(id));
      return;
    }

    await refreshConversationDetail(id, {
      activate: true,
      navigationRevision,
    });
  };

  return {
    toggle: async (id: number) => {
      await showConversation(id, { useCache: true });
    },
    rename: async (id: number, name: string) => {
      const resp = await doRenameConversation(id, name);
      resp.status && dispatch(renameHistory({ id, name }));

      return resp;
    },
    favorite: async (id: number, favorite: boolean) => {
      const resp = await doFavoriteConversation(id, favorite);
      resp.status && dispatch(favoriteHistory({ id, favorite }));

      return resp;
    },
    retitle: async (id: number) => {
      const resp = await doRetitleConversation(id);
      const data = resp.data;
      const name =
        data && typeof data === "object" && "name" in data
          ? data.name
          : undefined;
      if (resp.status && typeof name === "string" && name.length > 0) {
        dispatch(renameHistory({ id, name }));
      }

      return resp;
    },
    remove: async (id: number) => {
      const state = await doDeleteConversation(id);
      if (state) {
        await clearPendingChatRequestsForConversation(id);
        await clearCachedConversation(id);
        await removeCachedConversationFromList(id);
        dispatch(deleteConversation(id));
      }

      return state;
    },
    removeAll: async () => {
      const state = await doDeleteAllConversations();
      if (state) {
        await clearPendingChatRequests();
        await clearCachedConversations();
        dispatch(deleteAllConversation());
      }

      return state;
    },
    refresh: async (options?: { useCache?: boolean }) => {
      const useCache = options?.useCache ?? true;
      if (useCache) {
        const cacheGeneration = await getConversationListCacheGeneration();
        const cached = await getCachedConversationList();
        if (
          cached &&
          (await isCurrentConversationListCacheGeneration(cacheGeneration))
        ) {
          dispatch(setHistory(cached));
        }
      }

      const resp = await fetchConversationList();
      if (!(await applyConversationListResult(resp))) return [];

      const activeConversation = getNumberMemory(
        "history_conversation",
        current,
      );
      if (!resp.fromCache && activeConversation !== -1) {
        await refreshConversationDetail(activeConversation, {
          activate: false,
        });
      }

      return resp.conversations;
    },
    restore: async (options?: { useCache?: boolean }) => {
      const useCache = options?.useCache ?? true;
      const cacheGeneration = await getConversationListCacheGeneration();
      const cached = useCache ? await getCachedConversationList() : undefined;
      const stored = getNumberMemory("history_conversation", -1);
      if (
        cached &&
        (await isCurrentConversationListCacheGeneration(cacheGeneration))
      ) {
        dispatch(setHistory(cached));
        if (
          stored !== -1 &&
          getNumberMemory("history_conversation", -1) === stored &&
          current !== stored &&
          cached.some((item) => item.id === stored)
        ) {
          void showConversation(stored, {
            refreshRemote: false,
            useCache: true,
          });
        }
      }

      const resp = await fetchConversationList();
      if (!(await applyConversationListResult(resp))) return [];

      if (
        stored !== -1 &&
        getNumberMemory("history_conversation", -1) === stored
      ) {
        await showConversation(stored, { useCache: true });
      }

      return resp.conversations;
    },
    mask: (mask: Mask) => {
      dispatch(startMaskedConversation(mask));
    },
    selected: (model?: string) => {
      dispatch(setModel(model ?? ""));
    },
  };
}

export function useMessageActions() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const { refresh } = useConversationActions();
  const current = useSelector(selectCurrent);
  const conversations = useSelector(selectConversations);
  const conversationLoading = useSelector(selectConversationLoading);
  const mask = useSelector(selectMaskItem);

  const model = useSelector(selectModel);
  const web = useSelector(selectWeb);
  const gemini_google_search = useSelector(selectGeminiGoogleSearch);
  const gemini_url_context = useSelector(selectGeminiURLContext);
  const xai_web_search = useSelector(selectXAIWebSearch);
  const xai_x_search = useSelector(selectXAIXSearch);
  const openai_responses_web_search = useSelector(
    selectOpenAIResponsesWebSearch,
  );
  const fetch = useSelector(selectFetch);
  const learning_mode = useSelector(selectLearningMode);
  const gemini_thinking_budget = useSelector(selectGeminiThinkingBudget);
  const deepseek_thinking_enabled_by_model = useSelector(
    selectDeepSeekThinkingEnabledByModel,
  );
  const deepseek_reasoning_effort_by_model = useSelector(
    selectDeepSeekReasoningEffortByModel,
  );
  const openai_reasoning_effort = useSelector(selectOpenAIReasoningEffort);
  const support_models = useSelector(selectSupportModels);
  const chat_support_models = useSelector(selectChatSupportModels);
  const history = useSelector(historySelector);
  const context = useSelector(contextSelector);
  const max_tokens = useSelector(maxTokensSelector);
  const temperature = useSelector(temperatureSelector);
  const top_p = useSelector(topPSelector);
  const top_k = useSelector(topKSelector);
  const presence_penalty = useSelector(presencePenaltySelector);
  const frequency_penalty = useSelector(frequencyPenaltySelector);
  const repetition_penalty = useSelector(repetitionPenaltySelector);
  const persona_style = useSelector(personaStyleSelector);
  const persona_warmth = useSelector(personaWarmthSelector);
  const persona_enthusiasm = useSelector(personaEnthusiasmSelector);
  const persona_lists = useSelector(personaListsSelector);
  const persona_emoji = useSelector(personaEmojiSelector);
  const persona_custom_instruction = useSelector(
    personaCustomInstructionSelector,
  );
  const persona_nickname = useSelector(personaNicknameSelector);
  const persona_occupation = useSelector(personaOccupationSelector);
  const persona_about_user = useSelector(personaAboutUserSelector);
  const memory_enabled = useSelector(memoryEnabledSelector);
  const memory_history_enabled = useSelector(memoryHistoryEnabledSelector);
  const authenticated = useSelector(selectAuthenticated);

  const personalizationInstruction = buildPersonalizationInstruction({
    persona_style,
    persona_warmth,
    persona_enthusiasm,
    persona_lists,
    persona_emoji,
    persona_custom_instruction,
    persona_nickname,
    persona_occupation,
    persona_about_user,
  });

  return {
    send: async (
      message: string,
      using_model?: string,
      requestOptions?: Pick<ChatProps, "response_format" | "thinking">,
    ) => {
      if (conversationLoading) {
        logClientEvent(
          "chat.action",
          "send-blocked-loading",
          {
            current,
            loading_conversation_id: conversationLoading,
          },
          "warn",
        );
        return false;
      }

      if (findPendingAskUserToolCall(conversations[current]?.messages)) {
        logClientEvent(
          "chat.action",
          "send-blocked-pending-question",
          { current },
          "warn",
        );
        return false;
      }

      const conversationModel =
        current === -1 ? conversations[-1]?.model : undefined;
      const targetModel =
        using_model ||
        (conversationModel && inModel(chat_support_models, conversationModel)
          ? conversationModel
          : model);
      const enableGeminiNativeWeb = isGeminiModelId(targetModel);
      const enableXAINativeWeb = isXAIModelId(targetModel);
      const enableDeepSeekThinkingControl = isDeepSeekV4ModelId(targetModel);
      const openAIReasoningCapabilities = getOpenAIResponsesCapabilities(
        support_models,
        targetModel,
      );
      const enableOpenAINativeWeb = openAIReasoningCapabilities.nativeWeb;
      const enableOpenAIReasoningControl =
        openAIReasoningCapabilities.reasoningEfforts.length > 0;
      const targetDeepSeekThinkingEnabled = getDeepSeekThinkingEnabledForModel(
        deepseek_thinking_enabled_by_model,
        targetModel,
      );
      const targetDeepSeekReasoningEffort = getDeepSeekReasoningEffortForModel(
        deepseek_reasoning_effort_by_model,
        targetModel,
      );
      const openAIReasoningEffortForRequest =
        resolveOpenAIReasoningEffortForRequest(
          support_models,
          targetModel,
          openai_reasoning_effort,
          enableOpenAINativeWeb && openai_responses_web_search,
        );

      const shouldPreflightHistory =
        current === -1 && conversations[-1].messages.length === 0;
      const requestSummary = {
        current,
        target_model: targetModel,
        message_length: message.length,
        new_conversation: current === -1,
        should_preflight_history: shouldPreflightHistory,
        existing_message_count: conversations[current]?.messages.length ?? 0,
        web: enableGeminiNativeWeb
          ? gemini_google_search || gemini_url_context
          : enableXAINativeWeb
            ? xai_web_search || xai_x_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : web,
        web_search: enableGeminiNativeWeb
          ? gemini_google_search
          : enableXAINativeWeb
            ? xai_web_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : false,
        url_context: enableGeminiNativeWeb ? gemini_url_context : false,
        x_search: enableXAINativeWeb ? xai_x_search : false,
        fetch: enableGeminiNativeWeb ? false : fetch,
        learning_mode,
        context: history,
        ignore_context: !context,
        memory_enabled,
        memory_history_enabled,
        max_tokens: max_tokens > 0 ? max_tokens : undefined,
        temperature,
        top_p,
        top_k,
        presence_penalty,
        frequency_penalty,
        repetition_penalty,
        gemini_thinking_budget: supportsGeminiThinkingBudgetControl(targetModel)
          ? gemini_thinking_budget
          : undefined,
        deepseek_thinking_enabled: enableDeepSeekThinkingControl
          ? targetDeepSeekThinkingEnabled
          : undefined,
        deepseek_reasoning_effort:
          enableDeepSeekThinkingControl && targetDeepSeekThinkingEnabled
            ? targetDeepSeekReasoningEffort
            : undefined,
        openai_reasoning_effort: enableOpenAIReasoningControl
          ? openAIReasoningEffortForRequest
          : undefined,
        openai_reasoning_summary: openAIReasoningCapabilities.reasoningSummary
          ? "detailed"
          : undefined,
        has_response_format: Boolean(requestOptions?.response_format),
        has_thinking: Boolean(requestOptions?.thinking),
      };

      logClientEvent("chat.action", "send-start", requestSummary);

      const outboxScope = getChatOutboxScope();
      stack.bindAuthScope(outboxScope);
      if (!stack.hasConnection(current)) {
        stack.createConnection(current);
      }

      const requestID = authenticated ? createChatRequestID() : undefined;
      const chatProps: ChatProps = {
        type: "chat",
        request_id: requestID,
        message,
        web: enableGeminiNativeWeb
          ? gemini_google_search || gemini_url_context
          : enableXAINativeWeb
            ? xai_web_search || xai_x_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : web,
        web_search: enableGeminiNativeWeb
          ? gemini_google_search
          : enableXAINativeWeb
            ? xai_web_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : false,
        url_context: enableGeminiNativeWeb ? gemini_url_context : false,
        x_search: enableXAINativeWeb ? xai_x_search : false,
        fetch: enableGeminiNativeWeb ? false : fetch,
        learning_mode,
        gemini_thinking_budget: supportsGeminiThinkingBudgetControl(targetModel)
          ? gemini_thinking_budget
          : undefined,
        deepseek_thinking_enabled: enableDeepSeekThinkingControl
          ? targetDeepSeekThinkingEnabled
          : undefined,
        deepseek_reasoning_effort:
          enableDeepSeekThinkingControl && targetDeepSeekThinkingEnabled
            ? targetDeepSeekReasoningEffort
            : undefined,
        openai_reasoning_effort: enableOpenAIReasoningControl
          ? openAIReasoningEffortForRequest
          : undefined,
        openai_reasoning_summary: openAIReasoningCapabilities.reasoningSummary
          ? "detailed"
          : undefined,
        response_format: requestOptions?.response_format,
        thinking: requestOptions?.thinking,
        mask_context:
          current === -1 && mask?.context.length ? mask.context : undefined,
        model: targetModel,
        context: history,
        ignore_context: !context,
        custom_instruction: personalizationInstruction || undefined,
        memory_enabled,
        memory_history_enabled,
        max_tokens: max_tokens > 0 ? max_tokens : undefined,
        temperature,
        top_p,
        top_k,
        presence_penalty,
        frequency_penalty,
        repetition_penalty,
      };

      if (requestID) {
        try {
          await enqueuePendingChatRequest({
            requestId: requestID,
            conversationId: current,
            props: chatProps,
            createdAt: Date.now(),
            scope: outboxScope,
          });
        } catch (error) {
          logClientEvent(
            "chat.action",
            "outbox-persist-failed",
            {
              ...requestSummary,
              request_id: requestID,
              error: String(error),
            },
            "error",
          );
          return false;
        }
      }

      if (current === -1 && mask && mask.context.length > 0) {
        stack.sendMaskEvent(current, t, mask);
        dispatch(fillMaskItem());
      }

      const state = stack.send(current, t, chatProps, {
        scope: outboxScope,
      });
      if (!state) {
        logClientEvent(
          "chat.action",
          "send-failed-no-connection",
          requestSummary,
          "error",
        );
        return false;
      }

      if (shouldPreflightHistory) {
        dispatch(
          preflightHistory({
            localKey: `pending:${Date.now()}:${Math.random()
              .toString(36)
              .slice(2)}`,
            name: message,
          }),
        );
        logClientEvent("chat.action", "preflight-history", {
          current,
          target_model: targetModel,
        });
      }

      dispatch(
        createMessage({ id: current, role: UserRole, content: message }),
      );
      dispatch(
        createMessage({
          id: current,
          role: AssistantRole,
          model: targetModel,
        }),
      );
      dispatch(
        startGenerationRequest({
          id: current,
          requestId:
            requestID ?? `local:${Date.now()}:${Math.random().toString(36)}`,
        }),
      );

      logClientEvent("chat.action", "send-dispatched", {
        current,
        target_model: targetModel,
      });
      return true;
    },
    resumePending: async () => {
      if (!authenticated) return 0;
      const scope = getChatOutboxScope();
      stack.bindAuthScope(scope);
      const pending = await getPendingChatRequests(scope);
      for (const request of pending) {
        stack.send(request.conversationId, t, request.props, {
          scope: request.scope,
          resumed: true,
        });
      }
      if (pending.length > 0) {
        logClientEvent("chat.action", "outbox-resumed", {
          count: pending.length,
          request_ids: pending.map((item) => item.requestId),
        });
      }
      return pending.length;
    },
    stop: () => {
      if (!stack.hasConnection(current)) {
        logClientEvent(
          "chat.action",
          "stop-ignored-no-connection",
          {
            current,
          },
          "warn",
        );
        return;
      }
      logClientEvent("chat.action", "stop", {
        current,
      });
      stack.sendStopEvent(current, t);
      dispatch(stopMessage(current));
    },
    restart: () => {
      if (conversationLoading) {
        logClientEvent(
          "chat.action",
          "restart-blocked-loading",
          {
            current,
            loading_conversation_id: conversationLoading,
          },
          "warn",
        );
        return;
      }

      const enableGeminiNativeWeb = isGeminiModelId(model);
      const enableXAINativeWeb = isXAIModelId(model);
      const enableDeepSeekThinkingControl = isDeepSeekV4ModelId(model);
      const openAIReasoningCapabilities = getOpenAIResponsesCapabilities(
        support_models,
        model,
      );
      const enableOpenAINativeWeb = openAIReasoningCapabilities.nativeWeb;
      const enableOpenAIReasoningControl =
        openAIReasoningCapabilities.reasoningEfforts.length > 0;
      const currentDeepSeekThinkingEnabled = getDeepSeekThinkingEnabledForModel(
        deepseek_thinking_enabled_by_model,
        model,
      );
      const currentDeepSeekReasoningEffort = getDeepSeekReasoningEffortForModel(
        deepseek_reasoning_effort_by_model,
        model,
      );
      const openAIReasoningEffortForRequest =
        resolveOpenAIReasoningEffortForRequest(
          support_models,
          model,
          openai_reasoning_effort,
          enableOpenAINativeWeb && openai_responses_web_search,
        );
      if (!stack.hasConnection(current)) {
        stack.createConnection(current);
      }
      logClientEvent("chat.action", "restart", {
        current,
        model,
        message_count: conversations[current]?.messages.length ?? 0,
      });
      stack.sendRestartEvent(current, t, {
        web: enableGeminiNativeWeb
          ? gemini_google_search || gemini_url_context
          : enableXAINativeWeb
            ? xai_web_search || xai_x_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : web,
        web_search: enableGeminiNativeWeb
          ? gemini_google_search
          : enableXAINativeWeb
            ? xai_web_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : false,
        url_context: enableGeminiNativeWeb ? gemini_url_context : false,
        x_search: enableXAINativeWeb ? xai_x_search : false,
        fetch: enableGeminiNativeWeb ? false : fetch,
        learning_mode,
        gemini_thinking_budget: supportsGeminiThinkingBudgetControl(model)
          ? gemini_thinking_budget
          : undefined,
        deepseek_thinking_enabled: enableDeepSeekThinkingControl
          ? currentDeepSeekThinkingEnabled
          : undefined,
        deepseek_reasoning_effort:
          enableDeepSeekThinkingControl && currentDeepSeekThinkingEnabled
            ? currentDeepSeekReasoningEffort
            : undefined,
        openai_reasoning_effort: enableOpenAIReasoningControl
          ? openAIReasoningEffortForRequest
          : undefined,
        openai_reasoning_summary: openAIReasoningCapabilities.reasoningSummary
          ? "detailed"
          : undefined,
        model,
        context: history,
        ignore_context: !context,
        custom_instruction: personalizationInstruction || undefined,
        memory_enabled,
        memory_history_enabled,
        max_tokens: max_tokens > 0 ? max_tokens : undefined,
        temperature,
        top_p,
        top_k,
        presence_penalty,
        frequency_penalty,
        repetition_penalty,
        message: "",
      });

      // remove the last message if it's from assistant and create a new message
      dispatch(restartMessage({ id: current, model }));
      dispatch(
        startGenerationRequest({
          id: current,
          requestId: `restart:${Date.now()}:${Math.random().toString(36)}`,
        }),
      );
    },
    answerAskUser: async (toolCallId: string, result: AskUserResult) => {
      if (conversationLoading) {
        logClientEvent(
          "chat.action",
          "ask-user-answer-blocked-loading",
          { current, loading_conversation_id: conversationLoading },
          "warn",
        );
        return false;
      }

      const pending = findPendingAskUserToolCall(
        conversations[current]?.messages,
      );
      if (!pending || pending.id !== toolCallId) {
        logClientEvent(
          "chat.action",
          "ask-user-answer-stale",
          { current, tool_call_id: toolCallId },
          "warn",
        );
        return false;
      }

      const answerModel = conversations[current]?.model || model;
      const enableGeminiNativeWeb = isGeminiModelId(answerModel);
      const enableXAINativeWeb = isXAIModelId(answerModel);
      const enableDeepSeekThinkingControl = isDeepSeekV4ModelId(answerModel);
      const openAIReasoningCapabilities = getOpenAIResponsesCapabilities(
        support_models,
        answerModel,
      );
      const enableOpenAINativeWeb = openAIReasoningCapabilities.nativeWeb;
      const enableOpenAIReasoningControl =
        openAIReasoningCapabilities.reasoningEfforts.length > 0;
      const currentDeepSeekThinkingEnabled = getDeepSeekThinkingEnabledForModel(
        deepseek_thinking_enabled_by_model,
        answerModel,
      );
      const currentDeepSeekReasoningEffort = getDeepSeekReasoningEffortForModel(
        deepseek_reasoning_effort_by_model,
        answerModel,
      );
      const openAIReasoningEffortForRequest =
        resolveOpenAIReasoningEffortForRequest(
          support_models,
          answerModel,
          openai_reasoning_effort,
          enableOpenAINativeWeb && openai_responses_web_search,
        );

      if (!stack.hasConnection(current)) stack.createConnection(current);
      stack.sendToolResultEvent(current, t, toolCallId, result, {
        message: "",
        model: answerModel,
        web: enableGeminiNativeWeb
          ? gemini_google_search || gemini_url_context
          : enableXAINativeWeb
            ? xai_web_search || xai_x_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : web,
        web_search: enableGeminiNativeWeb
          ? gemini_google_search
          : enableXAINativeWeb
            ? xai_web_search
            : enableOpenAINativeWeb
              ? openai_responses_web_search
              : false,
        url_context: enableGeminiNativeWeb ? gemini_url_context : false,
        x_search: enableXAINativeWeb ? xai_x_search : false,
        fetch: enableGeminiNativeWeb ? false : fetch,
        learning_mode,
        gemini_thinking_budget: supportsGeminiThinkingBudgetControl(answerModel)
          ? gemini_thinking_budget
          : undefined,
        deepseek_thinking_enabled: enableDeepSeekThinkingControl
          ? currentDeepSeekThinkingEnabled
          : undefined,
        deepseek_reasoning_effort:
          enableDeepSeekThinkingControl && currentDeepSeekThinkingEnabled
            ? currentDeepSeekReasoningEffort
            : undefined,
        openai_reasoning_effort: enableOpenAIReasoningControl
          ? openAIReasoningEffortForRequest
          : undefined,
        openai_reasoning_summary: openAIReasoningCapabilities.reasoningSummary
          ? "detailed"
          : undefined,
        context: history,
        ignore_context: !context,
        custom_instruction: personalizationInstruction || undefined,
        memory_enabled,
        memory_history_enabled,
        max_tokens: max_tokens > 0 ? max_tokens : undefined,
        temperature,
        top_p,
        top_k,
        presence_penalty,
        frequency_penalty,
        repetition_penalty,
      });
      dispatch(
        answerAskUserMessage({
          id: current,
          toolCallId,
          result,
          model: answerModel,
        }),
      );
      dispatch(
        startGenerationRequest({
          id: current,
          requestId: `tool-result:${Date.now()}:${Math.random().toString(36)}`,
        }),
      );
      logClientEvent("chat.action", "ask-user-answer", {
        current,
        tool_call_id: toolCallId,
        question_count: Object.keys(result.answers).length,
      });
      return true;
    },
    remove: (idx: number) => {
      const conversation = conversations[current];
      if (!conversation || idx < 0 || idx >= conversation.messages.length) {
        logClientEvent(
          "chat.action",
          "remove-ignored-invalid-index",
          {
            current,
            index: idx,
            message_count: conversation?.messages.length,
          },
          "warn",
        );
        return;
      }

      logClientEvent("chat.action", "remove-message", {
        current,
        index: idx,
        message_count: conversation.messages.length,
      });
      dispatch(removeMessage({ id: current, idx }));

      if (!stack.hasConnection(current)) stack.createConnection(current);
      stack.sendRemoveEvent(current, t, idx);
    },
    edit: (idx: number, message: string) => {
      const conversation = conversations[current];
      if (!conversation || idx < 0 || idx >= conversation.messages.length) {
        logClientEvent(
          "chat.action",
          "edit-ignored-invalid-index",
          {
            current,
            index: idx,
            message_count: conversation?.messages.length,
          },
          "warn",
        );
        return;
      }

      logClientEvent("chat.action", "edit-message", {
        current,
        index: idx,
        message_count: conversation.messages.length,
        next_message_length: message.length,
      });
      dispatch(editMessage({ id: current, idx, message }));
      if (!stack.hasConnection(current)) stack.createConnection(current);
      stack.sendEditEvent(current, t, idx, message);
    },
    receive: async (id: number, message: StreamMessage) => {
      if (
        message.end ||
        message.conversation ||
        message.title ||
        message.tool_call ||
        message.search_query ||
        message.search_result ||
        message.search_index
      ) {
        logClientEvent("chat.action", "receive-important", {
          id,
          ...summarizeIncomingStreamMessage(message),
        });
      }
      const conversationModel = conversations[id]?.model;
      if (message.request_id && message.accepted) {
        dispatch(
          startGenerationRequest({
            id,
            requestId: message.request_id,
          }),
        );
      }
      dispatch(updateMessage({ id, message, model: conversationModel }));
      if (
        message.request_id &&
        (message.request_status === "rejected" || message.retryable === false)
      ) {
        dispatch(
          finishGenerationRequest({
            id,
            requestId: message.request_id,
          }),
        );
      }
      if (message.title) {
        dispatch(renameHistory({ id, name: message.title }));
      }

      if (
        id !== -1 &&
        message.request_id &&
        message.request_status === "completed"
      ) {
        await refresh({ useCache: false });
        dispatch(
          finishGenerationRequest({
            id,
            requestId: message.request_id,
          }),
        );
      }

      // Request-state frames carry conversation metadata but no assistant payload.
      // Only an accepted request may promote a newly created conversation.
      if (
        id === -1 &&
        message.conversation &&
        (!message.request_id || message.accepted)
      ) {
        const target: number = message.conversation;
        logClientEvent("chat.action", "raise-conversation", {
          from: id,
          to: target,
        });
        dispatch(raiseConversation(target));
        setNumberMemory("history_conversation", target);
        stack.raiseConnection(target);
        await refresh({ useCache: false });
      }
    },
  };
}

export function useListenMessageEvent() {
  const actions = useMessageActions();

  return (e: ConnectionEvent) => {
    console.debug(`[conversation] receive event: ${e.event} (id: ${e.id})`);

    switch (e.event) {
      case "stop":
        actions.stop();
        break;
      case "restart":
        actions.restart();
        break;
      case "remove":
        actions.remove(e.index ?? -1);
        break;
      case "edit":
        actions.edit(e.index ?? -1, e.message ?? "");
        break;
      case "answer-ask-user":
        if (!e.toolCallId || !e.askUserResult) return false;
        return actions.answerAskUser(e.toolCallId, e.askUserResult);
    }
  };
}

export const listenMessageEvent = useListenMessageEvent;

export function useMessages(): Message[] {
  const conversations = useSelector(selectConversations);
  const current = useSelector(selectCurrent);
  const mask = useSelector(selectMaskItem);
  const generationActive = useSelector(selectCurrentGenerationActive);

  return useMemo(() => {
    const messages = conversations[current]?.messages || [];
    const showMask = current === -1 && mask && messages.length === 0;
    const visibleMessages = !showMask ? messages : mask?.context || [];
    const last = visibleMessages[visibleMessages.length - 1];

    // Request lifecycle is authoritative. A remote refresh can briefly return
    // the persisted user message before the streaming assistant placeholder;
    // keep the loading card mounted until the request actually terminates.
    const hasStreamingAssistant =
      last?.role === AssistantRole &&
      (("end" in last && last.end === false) ||
        ("status" in last && last.status === "streaming"));
    if (generationActive && !hasStreamingAssistant) {
      return [
        ...visibleMessages,
        {
          role: AssistantRole,
          content: "",
          model: conversations[current]?.model,
          status: "streaming",
          end: false,
        },
      ];
    }

    return visibleMessages;
  }, [conversations, current, generationActive, mask]);
}

export function useWorking(): boolean {
  const messages = useMessages();
  const generationActive = useSelector(selectCurrentGenerationActive);

  return useMemo(() => {
    if (generationActive) return true;
    if (messages.length === 0) return false;

    const last = messages[messages.length - 1];
    if (last.role !== AssistantRole) return false;
    if (last.status === "streaming") return true;
    if (last.end === undefined) return false;
    return !last.end;
  }, [generationActive, messages]);
}

export function usePendingAskUser(): MessageToolCall | undefined {
  const messages = useMessages();
  return useMemo(() => findPendingAskUserToolCall(messages), [messages]);
}

export const updateMasks = async (dispatch: AppDispatch) => {
  const resp = await listMasks();
  resp.data && resp.data.length > 0 && dispatch(setCustomMasks(resp.data));

  return resp;
};

export const updateSupportModels = (dispatch: AppDispatch, models: Model[]) => {
  dispatch(setSupportModels(loadPreferenceModels(models)));
};

export default chatSlice.reducer;
