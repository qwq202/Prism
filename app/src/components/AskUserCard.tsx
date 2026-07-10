import {
  type AskUserAnswerValue,
  type AskUserQuestion,
  type AskUserResult,
  parseAskUserResult,
  parseAskUserToolInput,
} from "@/api/ask-user.ts";
import type { MessageToolCall } from "@/api/types.tsx";
import { Button } from "@/components/ui/button.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import { Check, CheckCheck, CircleHelp, Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

type AskUserCardProps = {
  toolCall: MessageToolCall;
  onSubmit: (
    toolCallId: string,
    result: AskUserResult,
  ) => boolean | Promise<boolean>;
};

type SelectionState = {
  single: Record<string, string>;
  multiple: Record<string, string[]>;
  custom: Record<string, string>;
  skipped: string[];
};

const emptySelection: SelectionState = {
  single: {},
  multiple: {},
  custom: {},
  skipped: [],
};

function answerText(answer?: AskUserAnswerValue): string {
  if (!answer) return "";
  if (answer.skipped) return "";
  return Array.isArray(answer.value) ? answer.value.join("、") : answer.value;
}

function buildSkippedResult(questions: AskUserQuestion[]): AskUserResult {
  return {
    type: "ask_user_answer",
    answers: Object.fromEntries(
      questions.map((question) => [
        question.id,
        {
          type: question.type,
          value: question.type === "multiple" ? [] : "",
          custom: false,
          skipped: true,
        },
      ]),
    ),
  };
}

function buildResult(
  questions: AskUserQuestion[],
  selection: SelectionState,
): AskUserResult {
  const skipped = new Set(selection.skipped);
  const answers: Record<string, AskUserAnswerValue> = {};

  questions.forEach((question) => {
    if (skipped.has(question.id)) {
      answers[question.id] = {
        type: question.type,
        value: question.type === "multiple" ? [] : "",
        custom: false,
        skipped: true,
      };
      return;
    }

    const custom = selection.custom[question.id]?.trim() ?? "";
    if (question.type === "single") {
      answers[question.id] = {
        type: "single",
        value: custom || selection.single[question.id] || "",
        custom: Boolean(custom),
        skipped: false,
      };
      return;
    }

    const values = [...(selection.multiple[question.id] ?? [])];
    if (custom && !values.includes(custom)) values.push(custom);
    answers[question.id] = {
      type: "multiple",
      value: values,
      custom: Boolean(custom),
      skipped: false,
    };
  });

  return { type: "ask_user_answer", answers };
}

function hasAnswer(question: AskUserQuestion, selection: SelectionState) {
  if (selection.skipped.includes(question.id)) return true;
  if (selection.custom[question.id]?.trim()) return true;
  if (question.type === "single") {
    return Boolean(selection.single[question.id]);
  }
  return (selection.multiple[question.id]?.length ?? 0) > 0;
}

export function AskUserCard({ toolCall, onSubmit }: AskUserCardProps) {
  const { t } = useTranslation();
  const input = useMemo(
    () => parseAskUserToolInput(toolCall.function.arguments),
    [toolCall.function.arguments],
  );
  const answered = useMemo(
    () => parseAskUserResult(toolCall.result),
    [toolCall.result],
  );
  const [selection, setSelection] = useState<SelectionState>(emptySelection);
  const [submitting, setSubmitting] = useState(false);

  if (!input) {
    return (
      <div className="mt-3 border-t border-border/60 pt-3 text-sm text-muted-foreground">
        {t("ask-user.invalid")}
      </div>
    );
  }

  const questions = input.questions;
  const canSubmit = questions.every((question) =>
    hasAnswer(question, selection),
  );

  const submit = async (result: AskUserResult) => {
    if (submitting || answered) return;
    setSubmitting(true);
    try {
      if (!(await onSubmit(toolCall.id, result))) {
        toast.error(t("ask-user.submit-failed"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (answered) {
    return (
      <section
        className="mt-3 border-t border-border/60 pt-3"
        aria-label={t("ask-user.answered")}
      >
        <div className="mb-2 flex items-center gap-2 text-xs font-medium text-muted-foreground">
          <CheckCheck className="h-4 w-4 text-primary" />
          {t("ask-user.answered")}
        </div>
        <div className="space-y-2.5">
          {questions.map((question) => {
            const answer = answered.answers[question.id];
            return (
              <div key={question.id} className="min-w-0">
                <div className="text-xs text-muted-foreground">
                  {question.question}
                </div>
                <div className="mt-0.5 break-words text-sm font-medium text-foreground">
                  {answer?.skipped
                    ? t("ask-user.skipped")
                    : answerText(answer) || t("ask-user.skipped")}
                </div>
              </div>
            );
          })}
        </div>
      </section>
    );
  }

  return (
    <section
      className="mt-3 border-t border-border/60 pt-3"
      aria-label={t("ask-user.title")}
    >
      <div className="mb-4 flex items-center gap-2">
        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <CircleHelp className="h-4 w-4" />
        </span>
        <div className="min-w-0">
          <div className="text-sm font-semibold text-foreground">
            {t("ask-user.title")}
          </div>
          <div className="text-xs text-muted-foreground">
            {t("ask-user.description", { count: questions.length })}
          </div>
        </div>
      </div>

      <div className="space-y-5">
        {questions.map((question, questionIndex) => {
          const skipped = selection.skipped.includes(question.id);
          const selectedSingle = selection.single[question.id];
          const selectedMultiple = new Set(
            selection.multiple[question.id] ?? [],
          );

          return (
            <fieldset key={question.id} disabled={submitting}>
              <legend className="mb-2 flex w-full items-start gap-2 text-sm font-medium text-foreground">
                <span className="mt-0.5 shrink-0 text-xs text-muted-foreground">
                  {questionIndex + 1}/{questions.length}
                </span>
                <span className="min-w-0 flex-1 text-pretty">
                  {question.question}
                </span>
                <span className="max-w-40 shrink-0 truncate rounded-md bg-muted px-1.5 py-0.5 text-[11px] font-normal text-muted-foreground">
                  {question.header ? `${question.header} · ` : ""}
                  {t(
                    question.type === "multiple"
                      ? "ask-user.multiple"
                      : "ask-user.single",
                  )}
                </span>
              </legend>

              <div className={cn("space-y-1.5", skipped && "opacity-45")}>
                {question.options.map((option, optionIndex) => {
                  const selected =
                    question.type === "multiple"
                      ? selectedMultiple.has(option.label)
                      : selectedSingle === option.label;
                  return (
                    <button
                      key={option.label}
                      type="button"
                      disabled={skipped || submitting}
                      aria-pressed={selected}
                      onClick={() => {
                        setSelection((current) => {
                          const next: SelectionState = {
                            single: { ...current.single },
                            multiple: { ...current.multiple },
                            custom: { ...current.custom, [question.id]: "" },
                            skipped: current.skipped.filter(
                              (id) => id !== question.id,
                            ),
                          };
                          if (question.type === "single") {
                            next.single[question.id] = option.label;
                          } else {
                            const values = new Set(
                              next.multiple[question.id] ?? [],
                            );
                            if (values.has(option.label)) {
                              values.delete(option.label);
                            } else {
                              values.add(option.label);
                            }
                            next.multiple[question.id] = [...values];
                          }
                          return next;
                        });
                      }}
                      className={cn(
                        "flex min-h-11 w-full items-center gap-3 rounded-xl px-3 py-2 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                        selected
                          ? "bg-primary/10 text-primary"
                          : "bg-muted/45 text-foreground hover:bg-muted/70",
                      )}
                    >
                      <span
                        className={cn(
                          "flex h-6 w-6 shrink-0 items-center justify-center text-xs",
                          question.type === "multiple"
                            ? "rounded-md border"
                            : "rounded-full border",
                          selected
                            ? "border-primary bg-primary text-primary-foreground"
                            : "border-border bg-background text-muted-foreground",
                        )}
                      >
                        {selected ? (
                          <Check className="h-3.5 w-3.5" />
                        ) : (
                          optionIndex + 1
                        )}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block text-sm font-medium">
                          {option.label}
                        </span>
                        {option.description && (
                          <span className="mt-0.5 block text-xs leading-relaxed text-muted-foreground">
                            {option.description}
                          </span>
                        )}
                      </span>
                    </button>
                  );
                })}

                <label
                  className={cn(
                    "flex min-h-11 items-center gap-3 rounded-xl bg-muted/45 px-3 py-2 transition-colors focus-within:ring-2 focus-within:ring-ring",
                    selection.custom[question.id]?.trim() && "bg-primary/10",
                  )}
                >
                  <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-border bg-background text-xs text-muted-foreground">
                    {question.options.length + 1}
                  </span>
                  <input
                    type="text"
                    maxLength={1000}
                    disabled={skipped || submitting}
                    value={selection.custom[question.id] ?? ""}
                    onChange={(event) => {
                      const value = event.target.value;
                      setSelection((current) => ({
                        single:
                          question.type === "single"
                            ? { ...current.single, [question.id]: "" }
                            : current.single,
                        multiple: current.multiple,
                        custom: { ...current.custom, [question.id]: value },
                        skipped: current.skipped.filter(
                          (id) => id !== question.id,
                        ),
                      }));
                    }}
                    placeholder={t("ask-user.other-placeholder")}
                    className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
                  />
                </label>
              </div>

              <button
                type="button"
                onClick={() => {
                  setSelection((current) => ({
                    single: { ...current.single, [question.id]: "" },
                    multiple: { ...current.multiple, [question.id]: [] },
                    custom: { ...current.custom, [question.id]: "" },
                    skipped: skipped
                      ? current.skipped.filter((id) => id !== question.id)
                      : [...current.skipped, question.id],
                  }));
                }}
                className="mt-1 min-h-8 rounded-md px-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {skipped
                  ? t("ask-user.undo-skip")
                  : t("ask-user.skip-question")}
              </button>
            </fieldset>
          );
        })}
      </div>

      <div className="mt-4 flex items-center justify-end gap-2 border-t border-border/60 pt-3">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={submitting}
          onClick={() => void submit(buildSkippedResult(questions))}
          unClickable
        >
          {t("ask-user.skip-all")}
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={!canSubmit || submitting}
          onClick={() => void submit(buildResult(questions, selection))}
          unClickable
        >
          {submitting && (
            <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
          )}
          {t("ask-user.submit")}
        </Button>
      </div>
    </section>
  );
}
