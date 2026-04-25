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
  loadConversation,
  getConversationList,
} from "@/api/history.ts";
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
  const normalized = normalizeOpenAIResponsesReasoningEffort(
    supportModels,
    model,
    effort,
  );
  if (!normalized) return undefined;

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
};

export type ConnectionEvent = {
  id: number;
  event: string;
  index?: number;
  message?: string;
};

type initialStateType = {
  history: ConversationInstance[];
  conversations: Record<number, ConversationSerialized>;
  model: string;
  web: boolean;
  gemini_google_search: boolean;
  gemini_url_context: boolean;
  xai_web_search: boolean;
  xai_x_search: boolean;
  openai_responses_web_search: boolean;
  gemini_thinking_budget: number;
  openai_reasoning_effort: string;
  openai_reasoning_summary: string;
  current: number;
  model_list: string[];
  market: boolean;
  mask_item: Mask | null;
  custom_masks: CustomMask[];
  support_models: Model[];
};

const defaultConversation: ConversationSerialized = { messages: [] };

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

export type OpenAIResponsesCapabilities = {
  nativeWeb: boolean;
  reasoningEfforts: string[];
};

export function getOpenAIResponsesCapabilities(
  supportModels: Model[],
  model: string | undefined | null,
): OpenAIResponsesCapabilities {
  if (!model) {
    return { nativeWeb: false, reasoningEfforts: [] };
  }
  const current = supportModels.find((item) => item.id === model);
  if (
    !current ||
    (current.channel_type || "").toLowerCase() !== "openai-responses"
  ) {
    return { nativeWeb: false, reasoningEfforts: [] };
  }

  const normalized = model.trim().toLowerCase();
  if (normalized.startsWith("gpt-5.4-pro")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["medium", "high", "xhigh"],
    };
  }
  if (normalized.startsWith("gpt-5.4-mini")) {
    return { nativeWeb: true, reasoningEfforts: [] };
  }
  if (normalized.startsWith("gpt-5.4-nano")) {
    return { nativeWeb: true, reasoningEfforts: [] };
  }
  if (normalized === "gpt-5.2-pro" || normalized.startsWith("gpt-5.2-pro-")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["medium", "high", "xhigh"],
    };
  }
  if (normalized === "gpt-5.2-chat-latest") {
    return { nativeWeb: true, reasoningEfforts: [] };
  }
  if (normalized === "gpt-5-pro" || normalized.startsWith("gpt-5-pro-")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["high"],
    };
  }
  if (normalized === "gpt-5-mini" || normalized.startsWith("gpt-5-mini-")) {
    return { nativeWeb: true, reasoningEfforts: [] };
  }
  if (normalized === "gpt-5-nano" || normalized.startsWith("gpt-5-nano-")) {
    return { nativeWeb: true, reasoningEfforts: [] };
  }
  if (normalized.startsWith("gpt-5.4")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
    };
  }
  if (normalized.startsWith("gpt-5.2")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high", "xhigh"],
    };
  }
  if (normalized.startsWith("gpt-5.1")) {
    return {
      nativeWeb: true,
      reasoningEfforts: ["none", "low", "medium", "high"],
    };
  }
  if (normalized === "gpt-5") {
    return {
      nativeWeb: true,
      reasoningEfforts: ["minimal", "low", "medium", "high"],
    };
  }
  if (normalized === "gpt-5.3-chat-latest") {
    return { nativeWeb: true, reasoningEfforts: [] };
  }
  if (normalized === "o3" || normalized.startsWith("o3-")) {
    return { nativeWeb: true, reasoningEfforts: ["low", "medium", "high"] };
  }
  if (normalized === "o1" || normalized.startsWith("o1-")) {
    return { nativeWeb: false, reasoningEfforts: ["low", "medium", "high"] };
  }
  if (normalized.startsWith("gpt-4.5")) {
    return { nativeWeb: false, reasoningEfforts: [] };
  }

  return { nativeWeb: false, reasoningEfforts: [] };
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
const offline = loadPreferenceModels(getOfflineModels());
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
    gemini_thinking_budget: getNumberMemory("gemini_thinking_budget", 0),
    openai_reasoning_effort: getMemory("openai_reasoning_effort") || "none",
    openai_reasoning_summary: normalizeOpenAIResponsesReasoningSummary(
      getMemory("openai_reasoning_summary"),
    ),
    current: -1,
    model: getModel(offline, getMemory("model")),
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

      if (conversation.messages.length === 0) {
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
      if (message.message.length > 0) instance.content += message.message;
      if (!instance.model && model) instance.model = model;
      if (message.keyword) instance.keyword = message.keyword;
      if (message.quota) instance.quota = message.quota;
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
      instance.plan = message.plan;
    },
    removeMessage: (state, action) => {
      const { id, idx } = action.payload as { id: number; idx: number };
      const conversation = state.conversations[id];
      if (!conversation) return;

      conversation.messages.splice(idx, 1);
    },
    restartMessage: (state, action) => {
      const { id, model } = action.payload as { id: number; model?: string };
      const conversation = state.conversations[id];
      if (!conversation || conversation.messages.length === 0) return;

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

      conversation.messages[idx].content = message;
    },
    stopMessage: (state, action) => {
      const { id } = action.payload as { id: number };
      const conversation = state.conversations[id];
      if (!conversation || conversation.messages.length === 0) return;

      conversation.messages[conversation.messages.length - 1].end = true;
    },
    raiseConversation: (state, action) => {
      // raise conversation `-1` to target id
      const id = action.payload as number;
      const conversation = state.conversations[-1];
      if (!conversation || id === -1) return;

      state.conversations[id] = conversation;
      if (state.current === -1) state.current = id;

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
    deleteConversation: (state, action) => {
      const id = action.payload as number;

      if (id === -1) return;

      state.history = state.history.filter((item) => item.id !== id);

      if (!state.conversations[id]) return;

      if (state.current === id) state.current = -1;
      delete state.conversations[id];
    },
    deleteAllConversation: (state) => {
      state.history = [];

      state.conversations = { [-1]: { ...defaultConversation } };
      state.current = -1;
    },
    setHistory: (state, action) => {
      state.history = action.payload as ConversationInstance[];
    },
    preflightHistory: (state, action) => {
      const name = action.payload as string;

      // add a new history at the beginning
      state.history = [{ id: -1, name, message: [] }, ...state.history];
    },
    renameHistory: (state, action) => {
      const { id, name } = action.payload as { id: number; name: string };
      const conversation = state.history.find((item) => item.id === id);
      if (conversation) conversation.name = name;
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
    setGeminiThinkingBudget: (state, action) => {
      setNumberMemory("gemini_thinking_budget", action.payload as number);
      state.gemini_thinking_budget = action.payload as number;
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

      const conversation = state.conversations[current];
      if (!conversation) return;
      if (
        conversation.model &&
        inModel(state.support_models, conversation.model)
      ) {
        state.model = conversation.model;
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
    clearMaskItem: (state) => {
      state.mask_item = null;
    },
    setCustomMasks: (state, action) => {
      state.custom_masks = action.payload as CustomMask[];
    },
    setSupportModels: (state, action) => {
      const models = action.payload as Model[];

      state.support_models = models;
      state.model = getModel(models, getMemory("model"));
      state.model_list = getModelList(
        models,
        getArrayMemory("model_mark_list"),
      );

      setOfflineModels(models);
    },
  },
});

export const {
  setHistory,
  renameHistory,
  setCurrent,
  setModel,
  setWeb,
  toggleWeb,
  setGeminiGoogleSearch,
  setGeminiURLContext,
  setXAIWebSearch,
  setXAIXSearch,
  setOpenAIResponsesWebSearch,
  setGeminiThinkingBudget,
  setOpenAIReasoningEffort,
  setOpenAIReasoningSummary,
  setModelList,
  addModelList,
  removeModelList,
  setCustomMasks,
  setSupportModels,
  setMaskItem,
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
export const selectGeminiThinkingBudget = (state: RootState): number =>
  state.chat.gemini_thinking_budget;
export const selectOpenAIReasoningEffort = (state: RootState): string =>
  state.chat.openai_reasoning_effort;
export const selectOpenAIReasoningSummary = (state: RootState): string =>
  state.chat.openai_reasoning_summary;
export const selectCurrent = (state: RootState): number => state.chat.current;
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

  return {
    toggle: async (id: number) => {
      const conversation = conversations[id];
      setNumberMemory("history_conversation", id);
      if (!conversation) {
        const data = await loadConversation(id);
        const props: ConversationSerialized = {
          model: data.model,
          messages: data.message,
        };
        dispatch(
          importConversation({
            conversation: props,
            id,
          }),
        );
      }

      if (current === -1 && conversations[-1].messages.length === 0) {
        // current is mask, clear mask
        mask && dispatch(clearMaskItem());
      }

      dispatch(setCurrent(id));
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
      state && dispatch(deleteConversation(id));

      return state;
    },
    removeAll: async () => {
      const state = await doDeleteAllConversations();
      state && dispatch(deleteAllConversation());

      return state;
    },
    refresh: async () => {
      const resp = await getConversationList();
      dispatch(setHistory(resp));

      return resp;
    },
    mask: (mask: Mask) => {
      dispatch(setMaskItem(mask));

      if (current !== -1) {
        dispatch(setCurrent(-1));
      }
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
  const gemini_thinking_budget = useSelector(selectGeminiThinkingBudget);
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
      const targetModel = using_model || model;
      const enableGeminiNativeWeb = isGeminiModelId(targetModel);
      const enableXAINativeWeb = isXAIModelId(targetModel);
      const enableOpenAINativeWeb = isOpenAIResponsesNativeWebModel(
        support_models,
        targetModel,
      );
      const enableOpenAIReasoningControl =
        supportsOpenAIResponsesReasoningControl(support_models, targetModel);
      const openAIReasoningEffortForRequest =
        resolveOpenAIReasoningEffortForRequest(
          support_models,
          targetModel,
          openai_reasoning_effort,
          enableOpenAINativeWeb && openai_responses_web_search,
        );

      if (current === -1 && conversations[-1].messages.length === 0) {
        // preflight history if it's a new conversation
        dispatch(preflightHistory(message));
      }

      if (!stack.hasConnection(current)) {
        const conn = stack.createConnection(current);

        if (current === -1 && mask && mask.context.length > 0) {
          conn.sendMaskEvent(t, mask);
          dispatch(fillMaskItem());
        }
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
        x_search: enableXAINativeWeb ? xai_x_search : false,
        gemini_thinking_budget: supportsGeminiThinkingBudgetControl(targetModel)
          ? gemini_thinking_budget
          : undefined,
        openai_reasoning_effort: enableOpenAIReasoningControl
          ? openAIReasoningEffortForRequest
          : undefined,
        openai_reasoning_summary: enableOpenAIReasoningControl
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
      const enableGeminiNativeWeb = isGeminiModelId(model);
      const enableXAINativeWeb = isXAIModelId(model);
      const enableOpenAINativeWeb = isOpenAIResponsesNativeWebModel(
        support_models,
        model,
      );
      const enableOpenAIReasoningControl =
        supportsOpenAIResponsesReasoningControl(support_models, model);
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
        x_search: enableXAINativeWeb ? xai_x_search : false,
        gemini_thinking_budget: supportsGeminiThinkingBudgetControl(model)
          ? gemini_thinking_budget
          : undefined,
        openai_reasoning_effort: enableOpenAIReasoningControl
          ? openAIReasoningEffortForRequest
          : undefined,
        openai_reasoning_summary: enableOpenAIReasoningControl
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
      if (idx < 0 || idx >= conversations[current].messages.length) return;

      dispatch(removeMessage({ id: current, idx }));

      if (!stack.hasConnection(current)) stack.createConnection(current);
      stack.sendRemoveEvent(current, t, idx);
    },
    edit: (idx: number, message: string) => {
      if (idx < 0 || idx >= conversations[current].messages.length) return;

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
        await refresh();
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
