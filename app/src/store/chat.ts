import { createSlice } from "@reduxjs/toolkit";
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
import {
  deleteConversation as doDeleteConversation,
  deleteAllConversations as doDeleteAllConversations,
  renameConversation as doRenameConversation,
  retitleConversation as doRetitleConversation,
  fetchConversation,
  fetchConversationList,
} from "@/api/history.ts";
import {
  clearCachedConversation,
  clearCachedConversations,
  getCachedConversation,
  getCachedConversationList,
  removeCachedConversationFromList,
  setCachedConversation,
} from "@/utils/conversation-cache.ts";
import { CustomMask, Mask } from "@/masks/types.ts";
import { listMasks } from "@/api/mask.ts";
import { useDispatch, useSelector } from "react-redux";
import { useMemo } from "react";
import { ConnectionStack, StreamMessage } from "@/api/connection.ts";
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
  local_pending_until?: number;
  local_revision?: number;
};

export type ConnectionEvent = {
  id: number;
  event: string;
  index?: number;
  message?: string;
};

type initialStateType = {
  history: ConversationInstance[];
  messages: Message[];
  conversations: Record<number, ConversationSerialized>;
  model: string;
  web: boolean;
  gemini_google_search: boolean;
  gemini_url_context: boolean;
  gemini_code_execution: boolean;
  xai_web_search: boolean;
  xai_x_search: boolean;
  openai_responses_web_search: boolean;
  fetch: boolean;
  learning_mode: boolean;
  gemini_thinking_budget: number;
  deepseek_thinking_enabled_by_model: Record<string, boolean>;
  deepseek_reasoning_effort_by_model: Record<string, string>;
  openai_reasoning_effort: string;
  openai_reasoning_summary: string;
  current: number;
  model_list: string[];
  market: boolean;
  mask_item: Mask | null;
  custom_masks: CustomMask[];
  support_models: Model[];
  loadingConversationId: number | null;
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

function markConversationPending(conversation: ConversationSerialized) {
  conversation.local_revision = getConversationLocalRevision(conversation) + 1;
  conversation.local_pending_until = Date.now() + localMutationProtectionMs;
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
  return last?.role === AssistantRole && last.end === false;
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

export function getModel(
  supportModels: Model[],
  model: string | undefined | null,
): string {
  if (supportModels.length === 0) return "";
  return model && inModel(supportModels, model) ? model : supportModels[0].id;
}

export function getModelList(
  supportModels: Model[],
  models: string[],
): string[] {
  return models.filter((item) => inModel(supportModels, item));
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
      ? {
          nativeWeb: false,
          reasoningEfforts: ["none", "high"],
          reasoningSummary: false,
        }
      : emptyOpenAIResponsesCapabilities();
  }

  if (channelType !== "openai-responses") {
    return emptyOpenAIResponsesCapabilities();
  }

  if (normalized === "gpt-5.5" || normalized.startsWith("gpt-5.5-")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    };
  }
  if (normalized.startsWith("gpt-5.4-pro")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["medium", "high", "xhigh"],
      reasoningSummary: true,
    };
  }
  if (normalized.startsWith("gpt-5.4-mini")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    };
  }
  if (normalized.startsWith("gpt-5.4-nano")) {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "gpt-5.2-pro" || normalized.startsWith("gpt-5.2-pro-")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["medium", "high", "xhigh"],
      reasoningSummary: true,
    };
  }
  if (normalized === "gpt-5.2-chat-latest") {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "gpt-5-pro" || normalized.startsWith("gpt-5-pro-")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["high"],
      reasoningSummary: true,
    };
  }
  if (normalized === "gpt-5-mini" || normalized.startsWith("gpt-5-mini-")) {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "gpt-5-nano" || normalized.startsWith("gpt-5-nano-")) {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized.startsWith("gpt-5.4")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    };
  }
  if (normalized.startsWith("gpt-5.2")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
      reasoningSummary: true,
    };
  }
  if (normalized.startsWith("gpt-5.1")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high"],
      reasoningSummary: true,
    };
  }
  if (normalized === "gpt-5") {
    return {
      nativeWeb: true,
      reasoningEfforts: ["minimal", "low", "medium", "high"],
      reasoningSummary: true,
    };
  }
  if (normalized === "gpt-5.3-chat-latest") {
    return { nativeWeb: true, reasoningEfforts: [], reasoningSummary: true };
  }
  if (normalized === "o3" || normalized.startsWith("o3-")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["low", "medium", "high"],
      reasoningSummary: true,
    };
  }
  if (normalized === "o1" || normalized.startsWith("o1-")) {
    return {
      nativeWeb: false,
      reasoningEfforts: ["low", "medium", "high"],
      reasoningSummary: true,
    };
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

export function normalizeOpenAIResponsesReasoningSummary(
  summary: string | undefined | null,
): string {
  const normalized = (summary || "").trim().toLowerCase();
  return ["none", "concise", "auto", "detailed"].includes(normalized)
    ? normalized
    : "auto";
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
    model === "gemini-3.1-flash-image-preview" ||
    model.startsWith("gemini-3.1-flash-image-preview-") ||
    model === "gemini-3-pro-image-preview" ||
    model.startsWith("gemini-3-pro-image-preview-") ||
    model === "gemini-3-pro-preview" ||
    model.startsWith("gemini-3-pro-preview-")
  );
}

const toolStatusPriority: Record<string, number> = {
  start: 0,
  executing: 1,
  success: 2,
  error: 2,
};

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
    gemini_code_execution: getBooleanMemory("gemini_code_execution", false),
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
    openai_reasoning_summary: normalizeOpenAIResponsesReasoningSummary(
      getMemory("openai_reasoning_summary"),
    ),
    current: -1,
    loadingConversationId: null,
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
      if (!inModel(state.support_models, model)) return;

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
    setGeminiCodeExecution: (state, action) => {
      setMemory("gemini_code_execution", action.payload ? "true" : "false");
      state.gemini_code_execution = action.payload as boolean;
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
    setOpenAIReasoningSummary: (state, action) => {
      const summary = normalizeOpenAIResponsesReasoningSummary(
        action.payload as string,
      );
      setMemory("openai_reasoning_summary", summary);
      state.openai_reasoning_summary = summary;
    },
    setCurrent: (state, action) => {
      const current = action.payload as number;
      state.current = current;
      state.loadingConversationId = null;

      const conversation = state.conversations[current];
      if (!conversation) return;
      if (
        conversation.model &&
        inModel(state.support_models, conversation.model)
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
        inModel(state.support_models, conversation.model)
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
        inModel(state.support_models, item),
      );
      setArrayMemory("model_mark_list", state.model_list);
    },
    addModelList: (state, action) => {
      const model = action.payload as string;
      if (
        inModel(state.support_models, model) &&
        !state.model_list.includes(model)
      ) {
        state.model_list.push(model);
        setArrayMemory("model_mark_list", state.model_list);
      }
    },
    removeModelList: (state, action) => {
      const model = action.payload as string;
      if (
        inModel(state.support_models, model) &&
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
        if (inModel(state.support_models, mask.model)) {
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
        maskedModel && inModel(models, maskedModel)
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
  upsertHistory,
  setCurrent,
  setActiveConversation,
  clearConversationLoading,
  setModel,
  setWeb,
  toggleWeb,
  setGeminiGoogleSearch,
  setGeminiURLContext,
  setGeminiCodeExecution,
  setXAIWebSearch,
  setXAIXSearch,
  setOpenAIResponsesWebSearch,
  setFetch,
  setLearningMode,
  setGeminiThinkingBudget,
  setDeepSeekThinkingEnabled,
  setDeepSeekReasoningEffort,
  setOpenAIReasoningEffort,
  setOpenAIReasoningSummary,
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
  editMessage,
  stopMessage,
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
export const selectGeminiCodeExecution = (state: RootState): boolean =>
  state.chat.gemini_code_execution;
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
export const selectOpenAIReasoningSummary = (state: RootState): string =>
  state.chat.openai_reasoning_summary;
export const selectCurrent = (state: RootState): number => state.chat.current;
export const selectConversationLoading = (state: RootState): boolean =>
  state.chat.loadingConversationId === state.chat.current;
export const selectModelList = (state: RootState): string[] =>
  state.chat.model_list;
export const selectCustomMasks = (state: RootState): CustomMask[] =>
  state.chat.custom_masks;
export const selectSupportModels = (state: RootState): Model[] =>
  state.chat.support_models;
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
      dispatch(deleteRemoteConversation({ id, requestedRevision }));
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
        await clearCachedConversation(id);
        await removeCachedConversationFromList(id);
        dispatch(deleteConversation(id));
      }

      return state;
    },
    removeAll: async () => {
      const state = await doDeleteAllConversations();
      if (state) {
        await clearCachedConversations();
        dispatch(deleteAllConversation());
      }

      return state;
    },
    refresh: async (options?: { useCache?: boolean }) => {
      const useCache = options?.useCache ?? true;
      if (useCache) {
        const cached = await getCachedConversationList();
        if (cached) {
          dispatch(setHistory(cached));
        }
      }

      const resp = await fetchConversationList();
      if (!resp.fromCache || resp.conversations.length > 0) {
        dispatch(
          resp.fromCache
            ? setHistory(resp.conversations)
            : setRemoteHistory(resp.conversations),
        );
      }

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
      const cached = useCache ? await getCachedConversationList() : undefined;
      const stored = getNumberMemory("history_conversation", -1);
      if (cached) {
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
      if (!resp.fromCache || resp.conversations.length > 0) {
        dispatch(
          resp.fromCache
            ? setHistory(resp.conversations)
            : setRemoteHistory(resp.conversations),
        );
      }

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
  const gemini_code_execution = useSelector(selectGeminiCodeExecution);
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
  const openai_reasoning_summary = useSelector(selectOpenAIReasoningSummary);
  const support_models = useSelector(selectSupportModels);
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
    send: async (message: string, using_model?: string) => {
      if (conversationLoading) return false;

      const conversationModel =
        current === -1 ? conversations[-1]?.model : undefined;
      const targetModel =
        using_model ||
        (conversationModel && inModel(support_models, conversationModel)
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

      if (!stack.hasConnection(current)) {
        stack.createConnection(current);
      }

      if (current === -1 && mask && mask.context.length > 0) {
        stack.sendMaskEvent(current, t, mask);
        dispatch(fillMaskItem());
      }

      const state = stack.send(current, t, {
        type: "chat",
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
        code_execution: enableGeminiNativeWeb ? gemini_code_execution : false,
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
          ? openai_reasoning_summary
          : undefined,
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
      });
      if (!state) return false;

      if (shouldPreflightHistory) {
        dispatch(
          preflightHistory({
            localKey: `pending:${Date.now()}:${Math.random()
              .toString(36)
              .slice(2)}`,
            name: message,
          }),
        );
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

      return true;
    },
    stop: () => {
      if (!stack.hasConnection(current)) return;
      stack.sendStopEvent(current, t);
      dispatch(stopMessage(current));
    },
    restart: () => {
      if (conversationLoading) return;

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
        code_execution: enableGeminiNativeWeb ? gemini_code_execution : false,
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
          ? openai_reasoning_summary
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
    },
    remove: (idx: number) => {
      const conversation = conversations[current];
      if (!conversation || idx < 0 || idx >= conversation.messages.length)
        return;

      dispatch(removeMessage({ id: current, idx }));

      if (!stack.hasConnection(current)) stack.createConnection(current);
      stack.sendRemoveEvent(current, t, idx);
    },
    edit: (idx: number, message: string) => {
      const conversation = conversations[current];
      if (!conversation || idx < 0 || idx >= conversation.messages.length)
        return;

      dispatch(editMessage({ id: current, idx, message }));
      if (!stack.hasConnection(current)) stack.createConnection(current);
      stack.sendEditEvent(current, t, idx, message);
    },
    receive: async (id: number, message: StreamMessage) => {
      const conversationModel = conversations[id]?.model;
      dispatch(updateMessage({ id, message, model: conversationModel }));
      if (message.title) {
        dispatch(renameHistory({ id, name: message.title }));
      }

      // raise conversation if it is -1
      if (id === -1 && message.conversation) {
        const target: number = message.conversation;
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
    }
  };
}

export const listenMessageEvent = useListenMessageEvent;

export function useMessages(): Message[] {
  const conversations = useSelector(selectConversations);
  const current = useSelector(selectCurrent);
  const mask = useSelector(selectMaskItem);

  return useMemo(() => {
    const messages = conversations[current]?.messages || [];
    const showMask = current === -1 && mask && messages.length === 0;
    return !showMask ? messages : mask?.context;
  }, [conversations, current, mask]);
}

export function useWorking(): boolean {
  const messages = useMessages();

  return useMemo(() => {
    if (messages.length === 0) return false;

    const last = messages[messages.length - 1];
    if (last.role !== AssistantRole || last.end === undefined) return false;
    return !last.end;
  }, [messages]);
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
