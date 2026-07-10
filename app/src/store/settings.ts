import { createSlice } from "@reduxjs/toolkit";
import {
  getBooleanMemory,
  getMemory,
  getNumberMemory,
  setBooleanMemory,
  setMemory,
  setNumberMemory,
} from "@/utils/memory.ts";
import { RootState } from "@/store/index.ts";
import { isMobile } from "@/utils/device.ts";
import type {
  PersonalizationRecord,
  PersonalizationSyncStatus,
  SyncedPersonalizationSettings,
} from "@/types/personalization.ts";

export const sendKeys = ["Ctrl + Enter", "Enter"];
export const minHistoryContext = 5;
export const maxHistoryContext = 25;
const personalizationSyncOwnerKey = "personalization_sync_owner";
const personalizationSyncDirtyKey = "personalization_sync_dirty";
const personalizationSyncRevisionKey = "personalization_sync_revision";
const personalizationSyncUpdatedAtKey = "personalization_sync_updated_at";
export const initialSettings = {
  context: true,
  align: false,
  history: minHistoryContext,
  sender: !isMobile(), // default [mobile: Ctrl + Enter, pc: Enter]
  max_tokens: 0,
  temperature: 0.6,
  top_p: 1,
  top_k: 5,
  presence_penalty: 0,
  frequency_penalty: 0,
  repetition_penalty: 1,
  hide_model: false,
  hide_toolbar: false,
  hide_toolbar_text: true,
  show_quota: true,
  collapse_thinking: true,
  persona_style: "default",
  persona_warmth: "default",
  persona_enthusiasm: "default",
  persona_lists: "default",
  persona_emoji: "default",
  persona_custom_instruction: "",
  persona_nickname: "",
  persona_occupation: "",
  persona_about_user: "",
  memory_enabled: false,
  memory_history_enabled: false,
};

const defaultSyncedPersonalizationSettings: SyncedPersonalizationSettings = {
  persona_style: initialSettings.persona_style,
  persona_warmth: initialSettings.persona_warmth,
  persona_enthusiasm: initialSettings.persona_enthusiasm,
  persona_lists: initialSettings.persona_lists,
  persona_emoji: initialSettings.persona_emoji,
  persona_custom_instruction: initialSettings.persona_custom_instruction,
  persona_nickname: initialSettings.persona_nickname,
  persona_occupation: initialSettings.persona_occupation,
  persona_about_user: initialSettings.persona_about_user,
  memory_enabled: initialSettings.memory_enabled,
  memory_history_enabled: initialSettings.memory_history_enabled,
};

const normalizeHistoryCount = (value: number): number => {
  if (!Number.isFinite(value)) return initialSettings.history;

  return Math.min(
    maxHistoryContext,
    Math.max(minHistoryContext, Math.floor(value)),
  );
};

export type PersonalizationSettings = {
  persona_style: string;
  persona_warmth: string;
  persona_enthusiasm: string;
  persona_lists: string;
  persona_emoji: string;
  persona_custom_instruction: string;
  persona_nickname: string;
  persona_occupation: string;
  persona_about_user: string;
};

function readSyncedPersonalizationSettings(
  state: SyncedPersonalizationSettings,
): SyncedPersonalizationSettings {
  return {
    persona_style: state.persona_style,
    persona_warmth: state.persona_warmth,
    persona_enthusiasm: state.persona_enthusiasm,
    persona_lists: state.persona_lists,
    persona_emoji: state.persona_emoji,
    persona_custom_instruction: state.persona_custom_instruction,
    persona_nickname: state.persona_nickname,
    persona_occupation: state.persona_occupation,
    persona_about_user: state.persona_about_user,
    memory_enabled: state.memory_enabled,
    memory_history_enabled: state.memory_history_enabled,
  };
}

function persistSyncedPersonalizationSettings(
  value: SyncedPersonalizationSettings,
) {
  setMemory("persona_style", value.persona_style);
  setMemory("persona_warmth", value.persona_warmth);
  setMemory("persona_enthusiasm", value.persona_enthusiasm);
  setMemory("persona_lists", value.persona_lists);
  setMemory("persona_emoji", value.persona_emoji);
  setMemory("persona_custom_instruction", value.persona_custom_instruction);
  setMemory("persona_nickname", value.persona_nickname);
  setMemory("persona_occupation", value.persona_occupation);
  setMemory("persona_about_user", value.persona_about_user);
  setBooleanMemory("memory_enabled", value.memory_enabled);
  setBooleanMemory("memory_history_enabled", value.memory_history_enabled);
}

function assignSyncedPersonalizationSettings(
  state: SyncedPersonalizationSettings,
  value: SyncedPersonalizationSettings,
) {
  state.persona_style = value.persona_style;
  state.persona_warmth = value.persona_warmth;
  state.persona_enthusiasm = value.persona_enthusiasm;
  state.persona_lists = value.persona_lists;
  state.persona_emoji = value.persona_emoji;
  state.persona_custom_instruction = value.persona_custom_instruction;
  state.persona_nickname = value.persona_nickname;
  state.persona_occupation = value.persona_occupation;
  state.persona_about_user = value.persona_about_user;
  state.memory_enabled = value.memory_enabled;
  state.memory_history_enabled = value.memory_history_enabled;
}

function markPersonalizationDirty(state: {
  personalization_sync_dirty: boolean;
  personalization_sync_change_id: number;
  personalization_sync_status: PersonalizationSyncStatus;
  personalization_sync_error: string;
}) {
  state.personalization_sync_dirty = true;
  state.personalization_sync_change_id += 1;
  state.personalization_sync_status = "saving";
  state.personalization_sync_error = "";
  setBooleanMemory(personalizationSyncDirtyKey, true);
}

const stylePromptMap: Record<string, string> = {
  professional:
    "Keep the response style polished, precise, and professional — refined and detail-oriented.",
  friendly:
    "Keep the response style warm, approachable, and gently supportive.",
  direct:
    "Keep the response style candid, frank, and optimistically straightforward.",
  creative:
    "Keep the response style imaginative, playful, and wonderfully whimsical.",
  efficient:
    "Keep the response style concise, practical, and straight to the point.",
  sarcastic:
    "Feel free to be witty, sharp-tongued, and humorously sarcastic when fitting.",
};

const warmthPromptMap: Record<string, string> = {
  low: "Use a restrained emotional tone.",
  medium: "Use a moderately warm tone.",
  high: "Use a very warm and caring tone without sounding overbearing.",
};

const enthusiasmPromptMap: Record<string, string> = {
  low: "Keep excitement levels subdued.",
  medium: "Show a moderate amount of enthusiasm.",
  high: "Show noticeable enthusiasm and encouragement when appropriate.",
};

const listsPromptMap: Record<string, string> = {
  minimal:
    "Prefer paragraphs over headings and lists unless structure clearly helps.",
  balanced:
    "Use headings and lists when they improve clarity, but avoid over-structuring.",
  structured:
    "Use clear headings and lists more proactively to organize answers.",
};

const emojiPromptMap: Record<string, string> = {
  none: "Do not use emoji.",
  light: "Use emoji sparingly and only when it feels natural.",
  expressive: "Emoji are welcome in light amounts when they support the tone.",
};

export function buildPersonalizationInstruction(
  personalization: PersonalizationSettings,
): string {
  const sections = [
    stylePromptMap[personalization.persona_style],
    warmthPromptMap[personalization.persona_warmth],
    enthusiasmPromptMap[personalization.persona_enthusiasm],
    listsPromptMap[personalization.persona_lists],
    emojiPromptMap[personalization.persona_emoji],
    personalization.persona_nickname
      ? `When it feels natural, address the user as "${personalization.persona_nickname.trim()}".`
      : "",
    personalization.persona_occupation
      ? `The user's occupation is: ${personalization.persona_occupation.trim()}`
      : "",
    personalization.persona_about_user
      ? `User profile and background to keep in mind: ${personalization.persona_about_user.trim()}`
      : "",
    personalization.persona_custom_instruction
      ? `Additional user preference: ${personalization.persona_custom_instruction.trim()}`
      : "",
  ].filter(Boolean);

  if (sections.length === 0) {
    return "";
  }

  return [
    "Follow these user personalization preferences when helpful. They should shape tone and presentation, but must not override the user's current request.",
    ...sections,
  ].join("\n");
}

export const settingsSlice = createSlice({
  name: "settings",
  initialState: {
    dialog: false,
    context: getBooleanMemory("context", true), // keep context
    align: getBooleanMemory("align", false), // chat textarea align center
    history: normalizeHistoryCount(
      getNumberMemory("history_context", initialSettings.history),
    ), // context message count
    sender: getBooleanMemory("sender", !isMobile()), // sender (false: Ctrl + Enter, true: Enter)
    max_tokens: getNumberMemory("max_tokens", 0), // max tokens, 0 means unlimited
    temperature: getNumberMemory("temperature", 0.6), // temperature
    top_p: getNumberMemory("top_p", 1), // top_p
    top_k: getNumberMemory("top_k", 5), // top_k
    presence_penalty: getNumberMemory("presence_penalty", 0), // presence_penalty
    frequency_penalty: getNumberMemory("frequency_penalty", 0), // frequency_penalty
    repetition_penalty: getNumberMemory("repetition_penalty", 1), // repetition_penalty
    hide_model: getBooleanMemory("hide_model", false), // hide model
    hide_toolbar: getBooleanMemory("hide_toolbar", false), // hide toolbar
    hide_toolbar_text: getBooleanMemory("hide_toolbar_text", true), // hide toolbar text
    show_quota: getBooleanMemory("show_quota", true), // show message quota
    collapse_thinking: getBooleanMemory(
      "collapse_thinking",
      initialSettings.collapse_thinking,
    ), // collapse thinking content by default
    persona_style: getMemory("persona_style", initialSettings.persona_style),
    persona_warmth: getMemory("persona_warmth", initialSettings.persona_warmth),
    persona_enthusiasm: getMemory(
      "persona_enthusiasm",
      initialSettings.persona_enthusiasm,
    ),
    persona_lists: getMemory("persona_lists", initialSettings.persona_lists),
    persona_emoji: getMemory("persona_emoji", initialSettings.persona_emoji),
    persona_custom_instruction: getMemory(
      "persona_custom_instruction",
      initialSettings.persona_custom_instruction,
    ),
    persona_nickname: getMemory(
      "persona_nickname",
      initialSettings.persona_nickname,
    ),
    persona_occupation: getMemory(
      "persona_occupation",
      initialSettings.persona_occupation,
    ),
    persona_about_user: getMemory(
      "persona_about_user",
      initialSettings.persona_about_user,
    ),
    memory_enabled: getBooleanMemory(
      "memory_enabled",
      initialSettings.memory_enabled,
    ),
    memory_history_enabled: getBooleanMemory(
      "memory_history_enabled",
      initialSettings.memory_history_enabled,
    ),
    personalization_sync_owner: getMemory(personalizationSyncOwnerKey, ""),
    personalization_sync_dirty: getBooleanMemory(
      personalizationSyncDirtyKey,
      false,
    ),
    personalization_sync_revision: getNumberMemory(
      personalizationSyncRevisionKey,
      0,
    ),
    personalization_sync_updated_at: getMemory(
      personalizationSyncUpdatedAtKey,
      "",
    ),
    personalization_sync_status: "idle" as PersonalizationSyncStatus,
    personalization_sync_error: "",
    personalization_sync_request: 0,
    personalization_sync_change_id: 0,
  },
  reducers: {
    toggleDialog: (state) => {
      state.dialog = !state.dialog;
    },
    setDialog: (state, action) => {
      state.dialog = action.payload as boolean;
    },
    openDialog: (state) => {
      state.dialog = true;
    },
    closeDialog: (state) => {
      state.dialog = false;
    },
    setContext: (state, action) => {
      state.context = action.payload as boolean;
      setBooleanMemory("context", action.payload);
    },
    setAlign: (state, action) => {
      state.align = action.payload as boolean;
      setBooleanMemory("align", action.payload);
    },
    setHistory: (state, action) => {
      const history = normalizeHistoryCount(action.payload as number);
      state.history = history;
      setNumberMemory("history_context", history);
    },
    setSender: (state, action) => {
      state.sender = action.payload as boolean;
      setBooleanMemory("sender", action.payload);
    },
    setMaxTokens: (state, action) => {
      state.max_tokens = action.payload as number;
      setNumberMemory("max_tokens", action.payload);
    },
    setTemperature: (state, action) => {
      state.temperature = action.payload as number;
      setNumberMemory("temperature", action.payload);
    },
    setTopP: (state, action) => {
      state.top_p = action.payload as number;
      setNumberMemory("top_p", action.payload);
    },
    setTopK: (state, action) => {
      state.top_k = action.payload as number;
      setNumberMemory("top_k", action.payload);
    },
    setPresencePenalty: (state, action) => {
      state.presence_penalty = action.payload as number;
      setNumberMemory("presence_penalty", action.payload);
    },
    setFrequencyPenalty: (state, action) => {
      state.frequency_penalty = action.payload as number;
      setNumberMemory("frequency_penalty", action.payload);
    },
    setRepetitionPenalty: (state, action) => {
      state.repetition_penalty = action.payload as number;
      setNumberMemory("repetition_penalty", action.payload);
    },
    setHideModel: (state, action) => {
      state.hide_model = action.payload as boolean;
      setBooleanMemory("hide_model", action.payload);
    },
    setHideToolbar: (state, action) => {
      state.hide_toolbar = action.payload as boolean;
      setBooleanMemory("hide_toolbar", action.payload);
    },
    setHideToolbarText: (state, action) => {
      state.hide_toolbar_text = action.payload as boolean;
      setBooleanMemory("hide_toolbar_text", action.payload);
    },
    setShowQuota: (state, action) => {
      state.show_quota = action.payload as boolean;
      setBooleanMemory("show_quota", action.payload);
    },
    setCollapseThinking: (state, action) => {
      state.collapse_thinking = action.payload as boolean;
      setBooleanMemory("collapse_thinking", action.payload);
    },
    setPersonaStyle: (state, action) => {
      state.persona_style = action.payload as string;
      setMemory("persona_style", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaWarmth: (state, action) => {
      state.persona_warmth = action.payload as string;
      setMemory("persona_warmth", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaEnthusiasm: (state, action) => {
      state.persona_enthusiasm = action.payload as string;
      setMemory("persona_enthusiasm", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaLists: (state, action) => {
      state.persona_lists = action.payload as string;
      setMemory("persona_lists", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaEmoji: (state, action) => {
      state.persona_emoji = action.payload as string;
      setMemory("persona_emoji", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaCustomInstruction: (state, action) => {
      state.persona_custom_instruction = action.payload as string;
      setMemory("persona_custom_instruction", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaNickname: (state, action) => {
      state.persona_nickname = action.payload as string;
      setMemory("persona_nickname", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaOccupation: (state, action) => {
      state.persona_occupation = action.payload as string;
      setMemory("persona_occupation", action.payload);
      markPersonalizationDirty(state);
    },
    setPersonaAboutUser: (state, action) => {
      state.persona_about_user = action.payload as string;
      setMemory("persona_about_user", action.payload);
      markPersonalizationDirty(state);
    },
    setMemoryEnabled: (state, action) => {
      state.memory_enabled = action.payload as boolean;
      setBooleanMemory("memory_enabled", action.payload);
      markPersonalizationDirty(state);
    },
    setMemoryHistoryEnabled: (state, action) => {
      state.memory_history_enabled = action.payload as boolean;
      setBooleanMemory("memory_history_enabled", action.payload);
      markPersonalizationDirty(state);
    },
    preparePersonalizationAccount: (state, action) => {
      const owner = String(action.payload || "").trim();
      if (!owner) return;

      if (
        state.personalization_sync_owner &&
        state.personalization_sync_owner !== owner
      ) {
        assignSyncedPersonalizationSettings(
          state,
          defaultSyncedPersonalizationSettings,
        );
        persistSyncedPersonalizationSettings(
          defaultSyncedPersonalizationSettings,
        );
        state.personalization_sync_dirty = false;
        state.personalization_sync_revision = 0;
        state.personalization_sync_updated_at = "";
        state.personalization_sync_change_id += 1;
        setBooleanMemory(personalizationSyncDirtyKey, false);
        setNumberMemory(personalizationSyncRevisionKey, 0);
        setMemory(personalizationSyncUpdatedAtKey, "");
      }

      state.personalization_sync_owner = owner;
      state.personalization_sync_status = "loading";
      state.personalization_sync_error = "";
      setMemory(personalizationSyncOwnerKey, owner);
    },
    hydratePersonalization: (state, action) => {
      const record = action.payload as PersonalizationRecord;
      assignSyncedPersonalizationSettings(state, record.settings);
      persistSyncedPersonalizationSettings(record.settings);
      state.personalization_sync_dirty = false;
      state.personalization_sync_revision = record.revision;
      state.personalization_sync_updated_at = record.updated_at || "";
      state.personalization_sync_status = "synced";
      state.personalization_sync_error = "";
      setBooleanMemory(personalizationSyncDirtyKey, false);
      setNumberMemory(personalizationSyncRevisionKey, record.revision);
      setMemory(personalizationSyncUpdatedAtKey, record.updated_at || "");
    },
    markPersonalizationSynced: (state, action) => {
      const payload = action.payload as {
        revision: number;
        updated_at?: string;
        change_id: number;
      };
      state.personalization_sync_revision = payload.revision;
      state.personalization_sync_updated_at = payload.updated_at || "";
      state.personalization_sync_error = "";
      if (state.personalization_sync_change_id === payload.change_id) {
        state.personalization_sync_dirty = false;
        state.personalization_sync_status = "synced";
        setBooleanMemory(personalizationSyncDirtyKey, false);
      } else {
        state.personalization_sync_status = "saving";
      }
      setNumberMemory(personalizationSyncRevisionKey, payload.revision);
      setMemory(personalizationSyncUpdatedAtKey, payload.updated_at || "");
    },
    rebasePersonalizationSync: (state, action) => {
      const payload = action.payload as {
        revision: number;
        updated_at?: string;
      };
      state.personalization_sync_revision = payload.revision;
      state.personalization_sync_updated_at = payload.updated_at || "";
      state.personalization_sync_dirty = true;
      state.personalization_sync_status = "saving";
      state.personalization_sync_error = "";
      setNumberMemory(personalizationSyncRevisionKey, payload.revision);
      setMemory(personalizationSyncUpdatedAtKey, payload.updated_at || "");
      setBooleanMemory(personalizationSyncDirtyKey, true);
    },
    setPersonalizationSyncStatus: (state, action) => {
      const payload = action.payload as {
        status: PersonalizationSyncStatus;
        error?: string;
      };
      state.personalization_sync_status = payload.status;
      state.personalization_sync_error = payload.error || "";
    },
    requestPersonalizationSync: (state) => {
      state.personalization_sync_request += 1;
    },
    resetSettings: (state) => {
      state.context = initialSettings.context;
      state.align = initialSettings.align;
      state.history = initialSettings.history;
      state.sender = initialSettings.sender;
      state.max_tokens = initialSettings.max_tokens;
      state.temperature = initialSettings.temperature;
      state.top_p = initialSettings.top_p;
      state.top_k = initialSettings.top_k;
      state.presence_penalty = initialSettings.presence_penalty;
      state.frequency_penalty = initialSettings.frequency_penalty;
      state.repetition_penalty = initialSettings.repetition_penalty;
      state.hide_model = initialSettings.hide_model;
      state.hide_toolbar = initialSettings.hide_toolbar;
      state.hide_toolbar_text = initialSettings.hide_toolbar_text;
      state.show_quota = initialSettings.show_quota;
      state.collapse_thinking = initialSettings.collapse_thinking;
      state.persona_style = initialSettings.persona_style;
      state.persona_warmth = initialSettings.persona_warmth;
      state.persona_enthusiasm = initialSettings.persona_enthusiasm;
      state.persona_lists = initialSettings.persona_lists;
      state.persona_emoji = initialSettings.persona_emoji;
      state.persona_custom_instruction =
        initialSettings.persona_custom_instruction;
      state.persona_nickname = initialSettings.persona_nickname;
      state.persona_occupation = initialSettings.persona_occupation;
      state.persona_about_user = initialSettings.persona_about_user;
      state.memory_enabled = initialSettings.memory_enabled;
      state.memory_history_enabled = initialSettings.memory_history_enabled;

      setBooleanMemory("context", initialSettings.context);
      setBooleanMemory("align", initialSettings.align);
      setNumberMemory("history_context", initialSettings.history);
      setBooleanMemory("sender", initialSettings.sender);
      setNumberMemory("max_tokens", initialSettings.max_tokens);
      setNumberMemory("temperature", initialSettings.temperature);
      setNumberMemory("top_p", initialSettings.top_p);
      setNumberMemory("top_k", initialSettings.top_k);
      setNumberMemory("presence_penalty", initialSettings.presence_penalty);
      setNumberMemory("frequency_penalty", initialSettings.frequency_penalty);
      setNumberMemory("repetition_penalty", initialSettings.repetition_penalty);
      setBooleanMemory("hide_model", initialSettings.hide_model);
      setBooleanMemory("hide_toolbar", initialSettings.hide_toolbar);
      setBooleanMemory("hide_toolbar_text", initialSettings.hide_toolbar_text);
      setBooleanMemory("show_quota", initialSettings.show_quota);
      setBooleanMemory("collapse_thinking", initialSettings.collapse_thinking);
      setMemory("persona_style", initialSettings.persona_style);
      setMemory("persona_warmth", initialSettings.persona_warmth);
      setMemory("persona_enthusiasm", initialSettings.persona_enthusiasm);
      setMemory("persona_lists", initialSettings.persona_lists);
      setMemory("persona_emoji", initialSettings.persona_emoji);
      setMemory(
        "persona_custom_instruction",
        initialSettings.persona_custom_instruction,
      );
      setMemory("persona_nickname", initialSettings.persona_nickname);
      setMemory("persona_occupation", initialSettings.persona_occupation);
      setMemory("persona_about_user", initialSettings.persona_about_user);
      setBooleanMemory("memory_enabled", initialSettings.memory_enabled);
      setBooleanMemory(
        "memory_history_enabled",
        initialSettings.memory_history_enabled,
      );
      markPersonalizationDirty(state);
    },
  },
});

export const {
  toggleDialog,
  setDialog,
  openDialog,
  closeDialog,
  setContext,
  setAlign,
  setHistory,
  setSender,
  setMaxTokens,
  setTemperature,
  setTopP,
  setTopK,
  setPresencePenalty,
  setFrequencyPenalty,
  setRepetitionPenalty,
  resetSettings,
  setHideModel,
  setHideToolbar,
  setHideToolbarText,
  setShowQuota,
  setCollapseThinking,
  setPersonaStyle,
  setPersonaWarmth,
  setPersonaEnthusiasm,
  setPersonaLists,
  setPersonaEmoji,
  setPersonaCustomInstruction,
  setPersonaNickname,
  setPersonaOccupation,
  setPersonaAboutUser,
  setMemoryEnabled,
  setMemoryHistoryEnabled,
  preparePersonalizationAccount,
  hydratePersonalization,
  markPersonalizationSynced,
  rebasePersonalizationSync,
  setPersonalizationSyncStatus,
  requestPersonalizationSync,
} = settingsSlice.actions;
export default settingsSlice.reducer;

export const dialogSelector = (state: RootState): boolean =>
  state.settings.dialog;
export const contextSelector = (state: RootState): boolean =>
  state.settings.context;
export const alignSelector = (state: RootState): boolean =>
  state.settings.align;
export const historySelector = (state: RootState): number =>
  state.settings.history;
export const senderSelector = (state: RootState): boolean =>
  state.settings.sender;
export const maxTokensSelector = (state: RootState): number =>
  state.settings.max_tokens;
export const temperatureSelector = (state: RootState): number =>
  state.settings.temperature;
export const topPSelector = (state: RootState): number => state.settings.top_p;
export const topKSelector = (state: RootState): number => state.settings.top_k;
export const presencePenaltySelector = (state: RootState): number =>
  state.settings.presence_penalty;
export const frequencyPenaltySelector = (state: RootState): number =>
  state.settings.frequency_penalty;
export const repetitionPenaltySelector = (state: RootState): number =>
  state.settings.repetition_penalty;
export const hideModelSelector = (state: RootState): boolean =>
  state.settings.hide_model;
export const hideToolbarSelector = (state: RootState): boolean =>
  state.settings.hide_toolbar;
export const hideToolbarTextSelector = (state: RootState): boolean =>
  state.settings.hide_toolbar_text;
export const showQuotaSelector = (state: RootState): boolean =>
  state.settings.show_quota;
export const collapseThinkingSelector = (state: RootState): boolean =>
  state.settings.collapse_thinking;
export const personaStyleSelector = (state: RootState): string =>
  state.settings.persona_style;
export const personaWarmthSelector = (state: RootState): string =>
  state.settings.persona_warmth;
export const personaEnthusiasmSelector = (state: RootState): string =>
  state.settings.persona_enthusiasm;
export const personaListsSelector = (state: RootState): string =>
  state.settings.persona_lists;
export const personaEmojiSelector = (state: RootState): string =>
  state.settings.persona_emoji;
export const personaCustomInstructionSelector = (state: RootState): string =>
  state.settings.persona_custom_instruction;
export const personaNicknameSelector = (state: RootState): string =>
  state.settings.persona_nickname;
export const personaOccupationSelector = (state: RootState): string =>
  state.settings.persona_occupation;
export const personaAboutUserSelector = (state: RootState): string =>
  state.settings.persona_about_user;
export const memoryEnabledSelector = (state: RootState): boolean =>
  state.settings.memory_enabled;
export const memoryHistoryEnabledSelector = (state: RootState): boolean =>
  state.settings.memory_history_enabled;
export const syncedPersonalizationSettingsSelector = (
  state: RootState,
): SyncedPersonalizationSettings =>
  readSyncedPersonalizationSettings(state.settings);
export const personalizationSyncOwnerSelector = (state: RootState): string =>
  state.settings.personalization_sync_owner;
export const personalizationSyncDirtySelector = (state: RootState): boolean =>
  state.settings.personalization_sync_dirty;
export const personalizationSyncRevisionSelector = (state: RootState): number =>
  state.settings.personalization_sync_revision;
export const personalizationSyncUpdatedAtSelector = (
  state: RootState,
): string => state.settings.personalization_sync_updated_at;
export const personalizationSyncStatusSelector = (
  state: RootState,
): PersonalizationSyncStatus => state.settings.personalization_sync_status;
export const personalizationSyncErrorSelector = (state: RootState): string =>
  state.settings.personalization_sync_error;
export const personalizationSyncRequestSelector = (state: RootState): number =>
  state.settings.personalization_sync_request;
export const personalizationSyncChangeIDSelector = (state: RootState): number =>
  state.settings.personalization_sync_change_id;
