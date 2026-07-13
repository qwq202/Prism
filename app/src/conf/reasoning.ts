export const REASONING_EFFORT_LEVELS = [
  "none",
  "minimal",
  "low",
  "medium",
  "high",
  "xhigh",
  "max",
] as const;

export const REASONING_EFFORT_DISPLAY_LEVELS = [
  "max",
  "xhigh",
  "high",
  "medium",
  "low",
  "minimal",
  "none",
] as const;

export const DEFAULT_REASONING_EFFORTS = ["low", "medium", "high"];

const reasoningEffortSet = new Set<string>(REASONING_EFFORT_LEVELS);

export function normalizeConfiguredReasoningEfforts(
  efforts: readonly string[] | undefined,
): string[] {
  const selected = new Set(
    (efforts || [])
      .map((effort) => effort.trim().toLowerCase())
      .filter((effort) => reasoningEffortSet.has(effort)),
  );
  return REASONING_EFFORT_LEVELS.filter((effort) => selected.has(effort));
}

export function isMaintainedReasoningModel(model: string): boolean {
  const normalized = model
    .trim()
    .toLowerCase()
    .replace(/^xiaomi\//, "");
  if (!normalized) return false;

  if (
    normalized === "deepseek-v4-flash" ||
    normalized === "deepseek-v4-pro" ||
    (normalized.startsWith("mimo-v2") && !normalized.includes("tts"))
  ) {
    return true;
  }

  if (
    normalized === "gpt-5" ||
    normalized.startsWith("gpt-5-") ||
    normalized === "gpt-5.1" ||
    normalized.startsWith("gpt-5.1-") ||
    normalized === "gpt-5.2" ||
    normalized.startsWith("gpt-5.2-") ||
    normalized === "gpt-5.3-chat-latest" ||
    normalized === "gpt-5.4" ||
    normalized.startsWith("gpt-5.4-") ||
    normalized === "gpt-5.5" ||
    normalized.startsWith("gpt-5.5-") ||
    normalized === "gpt-5.6" ||
    normalized.startsWith("gpt-5.6-") ||
    normalized === "o1" ||
    normalized.startsWith("o1-") ||
    normalized === "o3" ||
    normalized.startsWith("o3-") ||
    normalized === "o4-mini" ||
    normalized.startsWith("o4-mini-") ||
    normalized === "gpt-4.5" ||
    normalized.startsWith("gpt-4.5-")
  ) {
    return true;
  }

  return (
    normalized === "gemini-2.5-flash" ||
    normalized.startsWith("gemini-2.5-flash-preview-") ||
    normalized === "gemini-2.5-flash-lite" ||
    normalized.startsWith("gemini-2.5-flash-lite-preview-") ||
    normalized === "gemini-2.5-pro" ||
    normalized.startsWith("gemini-2.5-pro-preview-") ||
    normalized.startsWith("gemini-2.5-pro-exp-") ||
    normalized === "gemini-3.5-flash" ||
    normalized.startsWith("gemini-3.5-flash-") ||
    normalized === "gemini-3-flash-preview" ||
    normalized.startsWith("gemini-3-flash-preview-") ||
    normalized === "gemini-3.1-flash-lite-preview" ||
    normalized.startsWith("gemini-3.1-flash-lite-preview-") ||
    normalized === "gemini-3.1-pro-preview" ||
    normalized.startsWith("gemini-3.1-pro-preview-") ||
    normalized === "gemini-3.1-pro-preview-customtools" ||
    normalized.startsWith("gemini-3.1-pro-preview-customtools-") ||
    normalized === "gemini-3.1-flash-lite-image" ||
    normalized.startsWith("gemini-3.1-flash-lite-image-") ||
    normalized === "gemini-3.1-flash-image" ||
    normalized.startsWith("gemini-3.1-flash-image-") ||
    normalized === "gemini-3-pro-image" ||
    normalized.startsWith("gemini-3-pro-image-") ||
    normalized === "gemini-3-pro-preview" ||
    normalized.startsWith("gemini-3-pro-preview-")
  );
}
