import type { MessageToolCall } from "@/api/types.ts";

export const ASK_USER_TOOL_NAME = "ask_user";
export const MAX_ASK_USER_QUESTIONS = 4;

export type AskUserQuestionType = "single" | "multiple";

export type AskUserOption = {
  label: string;
  description?: string;
};

export type AskUserQuestion = {
  id: string;
  header?: string;
  question: string;
  type: AskUserQuestionType;
  options: AskUserOption[];
};

export type AskUserToolInput = {
  questions: AskUserQuestion[];
};

export type AskUserAnswerValue = {
  type: AskUserQuestionType;
  value: string | string[];
  custom: boolean;
  skipped: boolean;
};

export type AskUserResult = {
  type: "ask_user_answer";
  answers: Record<string, AskUserAnswerValue>;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value));
}

function normalizeQuestionType(value: unknown): AskUserQuestionType | null {
  if (value === "single") return "single";
  if (value === "multiple" || value === "multi") return "multiple";
  return null;
}

function normalizeOptions(value: unknown): AskUserOption[] {
  if (!Array.isArray(value)) return [];

  const labels = new Set<string>();
  const options: AskUserOption[] = [];
  for (const raw of value.slice(0, 3)) {
    const option = isRecord(raw)
      ? {
          label: typeof raw.label === "string" ? raw.label.trim() : "",
          description:
            typeof raw.description === "string"
              ? raw.description.trim()
              : undefined,
        }
      : { label: typeof raw === "string" ? raw.trim() : "" };
    if (!option.label || labels.has(option.label)) continue;
    labels.add(option.label);
    options.push(option);
  }
  return options;
}

export function parseAskUserToolInput(
  argumentsText: string,
): AskUserToolInput | null {
  try {
    const parsed = JSON.parse(argumentsText) as unknown;
    if (!isRecord(parsed) || !Array.isArray(parsed.questions)) return null;

    const usedIds = new Set<string>();
    const questions: AskUserQuestion[] = [];
    for (const [index, raw] of parsed.questions
      .slice(0, MAX_ASK_USER_QUESTIONS)
      .entries()) {
      if (!isRecord(raw)) continue;
      const question =
        typeof raw.question === "string" ? raw.question.trim() : "";
      const type = normalizeQuestionType(raw.type);
      const options = normalizeOptions(raw.options);
      if (!question || !type || options.length < 2) continue;

      let id = typeof raw.id === "string" ? raw.id.trim() : "";
      if (!id || usedIds.has(id)) id = `q${index + 1}`;
      while (usedIds.has(id)) id = `${id}_${usedIds.size + 1}`;
      usedIds.add(id);

      questions.push({
        id,
        header: typeof raw.header === "string" ? raw.header.trim() : undefined,
        question,
        type,
        options,
      });
    }

    return questions.length > 0 ? { questions } : null;
  } catch {
    return null;
  }
}

export function parseAskUserResult(value?: string): AskUserResult | null {
  if (!value) return null;
  try {
    const parsed = JSON.parse(value) as unknown;
    if (
      !isRecord(parsed) ||
      parsed.type !== "ask_user_answer" ||
      !isRecord(parsed.answers)
    ) {
      return null;
    }
    return parsed as AskUserResult;
  } catch {
    return null;
  }
}

export function isAskUserToolCallName(name?: string): boolean {
  return name?.trim() === ASK_USER_TOOL_NAME;
}

export function getAskUserToolCalls(
  toolCalls?: MessageToolCall[],
): MessageToolCall[] {
  if (!toolCalls) return [];
  return toolCalls.filter((toolCall) =>
    isAskUserToolCallName(toolCall.function?.name),
  );
}

export function isPendingAskUserToolCall(toolCall: MessageToolCall): boolean {
  return (
    isAskUserToolCallName(toolCall.function?.name) &&
    !toolCall.result &&
    !toolCall.error
  );
}
