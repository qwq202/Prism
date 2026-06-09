import type { MessageToolCall } from "@/api/types.ts";

const hiddenToolCallNames = new Set(["web_search"]);

export function isHiddenToolCallName(name?: string): boolean {
  const normalized = name?.trim();
  return Boolean(normalized && hiddenToolCallNames.has(normalized));
}

export function getVisibleToolCalls(
  toolCalls?: MessageToolCall[],
): MessageToolCall[] {
  if (!toolCalls || toolCalls.length === 0) return [];
  return toolCalls.filter(
    (toolCall) => !isHiddenToolCallName(toolCall.function?.name),
  );
}
