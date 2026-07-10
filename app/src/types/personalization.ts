export type SyncedPersonalizationSettings = {
  persona_style: string;
  persona_warmth: string;
  persona_enthusiasm: string;
  persona_lists: string;
  persona_emoji: string;
  persona_custom_instruction: string;
  persona_nickname: string;
  persona_occupation: string;
  persona_about_user: string;
  memory_enabled: boolean;
  memory_history_enabled: boolean;
};

export type PersonalizationRecord = {
  settings: SyncedPersonalizationSettings;
  revision: number;
  updated_at: string;
};

export type PersonalizationSyncStatus =
  | "idle"
  | "loading"
  | "saving"
  | "synced"
  | "offline"
  | "error"
  | "conflict";

export const PERSONALIZATION_TEXT_LIMITS = {
  customInstruction: 10000,
  nickname: 200,
  occupation: 500,
  aboutUser: 10000,
} as const;
