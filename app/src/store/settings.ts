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

export const sendKeys = ["Ctrl + Enter", "Enter"];
export const initialSettings = {
  context: true,
  align: false,
  history: 8,
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
  collapse_thinking: false,
  persona_style: "friendly",
  persona_warmth: "default",
  persona_enthusiasm: "default",
  persona_lists: "default",
  persona_emoji: "default",
  persona_custom_instruction: "",
  persona_nickname: "",
  persona_about_user: "",
};

export type PersonalizationSettings = {
  persona_style: string;
  persona_warmth: string;
  persona_enthusiasm: string;
  persona_lists: string;
  persona_emoji: string;
  persona_custom_instruction: string;
  persona_nickname: string;
  persona_about_user: string;
};

const stylePromptMap: Record<string, string> = {
  friendly: "Keep the default response style warm, approachable, and collaborative.",
  concise: "Keep the default response style concise, crisp, and efficient.",
  professional: "Keep the default response style polished, calm, and professional.",
  playful: "Keep the default response style lively, witty, and playful when appropriate.",
  direct: "Keep the default response style straightforward, candid, and low-frills.",
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
  minimal: "Prefer paragraphs over headings and lists unless structure clearly helps.",
  balanced: "Use headings and lists when they improve clarity, but avoid over-structuring.",
  structured: "Use clear headings and lists more proactively to organize answers.",
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
    history: getNumberMemory("history_context", 8), // max history context length
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
    collapse_thinking: getBooleanMemory("collapse_thinking", false), // collapse thinking content by default
    persona_style: getMemory("persona_style", initialSettings.persona_style),
    persona_warmth: getMemory(
      "persona_warmth",
      initialSettings.persona_warmth,
    ),
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
    persona_about_user: getMemory(
      "persona_about_user",
      initialSettings.persona_about_user,
    ),
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
      state.history = action.payload as number;
      setNumberMemory("history_context", action.payload);
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
    setCollapseThinking: (state, action) => {
      state.collapse_thinking = action.payload as boolean;
      setBooleanMemory("collapse_thinking", action.payload);
    },
    setPersonaStyle: (state, action) => {
      state.persona_style = action.payload as string;
      setMemory("persona_style", action.payload);
    },
    setPersonaWarmth: (state, action) => {
      state.persona_warmth = action.payload as string;
      setMemory("persona_warmth", action.payload);
    },
    setPersonaEnthusiasm: (state, action) => {
      state.persona_enthusiasm = action.payload as string;
      setMemory("persona_enthusiasm", action.payload);
    },
    setPersonaLists: (state, action) => {
      state.persona_lists = action.payload as string;
      setMemory("persona_lists", action.payload);
    },
    setPersonaEmoji: (state, action) => {
      state.persona_emoji = action.payload as string;
      setMemory("persona_emoji", action.payload);
    },
    setPersonaCustomInstruction: (state, action) => {
      state.persona_custom_instruction = action.payload as string;
      setMemory("persona_custom_instruction", action.payload);
    },
    setPersonaNickname: (state, action) => {
      state.persona_nickname = action.payload as string;
      setMemory("persona_nickname", action.payload);
    },
    setPersonaAboutUser: (state, action) => {
      state.persona_about_user = action.payload as string;
      setMemory("persona_about_user", action.payload);
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
      state.collapse_thinking = initialSettings.collapse_thinking;
      state.persona_style = initialSettings.persona_style;
      state.persona_warmth = initialSettings.persona_warmth;
      state.persona_enthusiasm = initialSettings.persona_enthusiasm;
      state.persona_lists = initialSettings.persona_lists;
      state.persona_emoji = initialSettings.persona_emoji;
      state.persona_custom_instruction =
        initialSettings.persona_custom_instruction;
      state.persona_nickname = initialSettings.persona_nickname;
      state.persona_about_user = initialSettings.persona_about_user;

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
      setMemory("persona_about_user", initialSettings.persona_about_user);
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
  setCollapseThinking,
  setPersonaStyle,
  setPersonaWarmth,
  setPersonaEnthusiasm,
  setPersonaLists,
  setPersonaEmoji,
  setPersonaCustomInstruction,
  setPersonaNickname,
  setPersonaAboutUser,
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
export const personaAboutUserSelector = (state: RootState): string =>
  state.settings.persona_about_user;
