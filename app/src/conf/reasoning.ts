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
  return getMaintainedReasoningEfforts(model) !== undefined;
}

export function getMaintainedReasoningEfforts(
  model: string,
): string[] | undefined {
  const normalized = model
    .trim()
    .toLowerCase()
    .replace(/^xiaomi\//, "");
  if (!normalized) return undefined;

  if (normalized === "deepseek-v4-flash" || normalized === "deepseek-v4-pro") {
    return ["high", "max"];
  }

  if (normalized.startsWith("mimo-v2") && !normalized.includes("tts")) {
    return ["none", "high"];
  }

  if (
    normalized === "grok-4.5" ||
    normalized.startsWith("grok-4.5-") ||
    normalized === "grok-build-latest"
  ) {
    return ["low", "medium", "high"];
  }

  if (normalized === "gpt-5.6" || normalized.startsWith("gpt-5.6-")) {
    return ["none", "low", "medium", "high", "xhigh", "max"];
  }
  if (normalized === "gpt-5.5" || normalized.startsWith("gpt-5.5-")) {
    return ["none", "low", "medium", "high", "xhigh"];
  }
  if (normalized.startsWith("gpt-5.4-pro")) {
    return ["medium", "high", "xhigh"];
  }
  if (normalized.startsWith("gpt-5.4-mini")) {
    return ["none", "low", "medium", "high", "xhigh"];
  }
  if (normalized.startsWith("gpt-5.4-nano")) {
    return [];
  }
  if (normalized === "gpt-5.4" || normalized.startsWith("gpt-5.4-")) {
    return ["none", "low", "medium", "high", "xhigh"];
  }
  if (normalized === "gpt-5.2-pro" || normalized.startsWith("gpt-5.2-pro-")) {
    return ["medium", "high", "xhigh"];
  }
  if (normalized === "gpt-5.2-chat-latest") {
    return [];
  }
  if (normalized === "gpt-5.2" || normalized.startsWith("gpt-5.2-")) {
    return ["none", "low", "medium", "high", "xhigh"];
  }
  if (normalized === "gpt-5.1" || normalized.startsWith("gpt-5.1-")) {
    return ["none", "low", "medium", "high"];
  }
  if (normalized === "gpt-5-pro" || normalized.startsWith("gpt-5-pro-")) {
    return ["high"];
  }
  if (
    normalized === "gpt-5-mini" ||
    normalized.startsWith("gpt-5-mini-") ||
    normalized === "gpt-5-nano" ||
    normalized.startsWith("gpt-5-nano-") ||
    normalized === "gpt-5.3-chat-latest"
  ) {
    return [];
  }
  if (normalized === "gpt-5") {
    return ["minimal", "low", "medium", "high"];
  }
  if (normalized.startsWith("gpt-5-")) {
    return [];
  }
  if (normalized === "o1" || normalized.startsWith("o1-")) {
    return ["low", "medium", "high"];
  }
  if (normalized === "o3" || normalized.startsWith("o3-")) {
    return ["low", "medium", "high"];
  }
  if (
    normalized === "o4-mini" ||
    normalized.startsWith("o4-mini-") ||
    normalized === "gpt-4.5" ||
    normalized.startsWith("gpt-4.5-")
  ) {
    return [];
  }

  const isGeminiThinkingModel =
    normalized === "gemini-2.5-flash" ||
    normalized.startsWith("gemini-2.5-flash-preview-") ||
    normalized === "gemini-2.5-flash-lite" ||
    normalized.startsWith("gemini-2.5-flash-lite-preview-") ||
    normalized === "gemini-2.5-pro" ||
    normalized.startsWith("gemini-2.5-pro-preview-") ||
    normalized.startsWith("gemini-2.5-pro-exp-") ||
    normalized === "gemini-3.6-flash" ||
    normalized.startsWith("gemini-3.6-flash-") ||
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
    normalized.startsWith("gemini-3-pro-preview-");

  return isGeminiThinkingModel ? ["low", "medium", "high"] : undefined;
}
