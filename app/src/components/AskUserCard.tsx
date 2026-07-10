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
import {
  ArrowLeft,
  Check,
  CheckCheck,
  CornerDownLeft,
  Loader2,
  X,
} from "lucide-react";
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
  const [questionIndex, setQuestionIndex] = useState(0);
  const [submitting, setSubmitting] = useState(false);

  if (!input) {
    return (
      <div className="mt-3 border-t border-border/60 pt-3 text-sm text-muted-foreground">
        {t("ask-user.invalid")}
      </div>
    );
  }

  const questions = input.questions;
  const currentQuestion = questions[questionIndex];
  const isLastQuestion = questionIndex === questions.length - 1;

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

  const moveNext = () => {
    if (!hasAnswer(currentQuestion, selection)) return;
    if (isLastQuestion) {
      void submit(buildResult(questions, selection));
      return;
    }
    setQuestionIndex((current) => current + 1);
  };

  const skipCurrentQuestion = () => {
    const nextSelection: SelectionState = {
      single: { ...selection.single, [currentQuestion.id]: "" },
      multiple: { ...selection.multiple, [currentQuestion.id]: [] },
      custom: { ...selection.custom, [currentQuestion.id]: "" },
      skipped: selection.skipped.includes(currentQuestion.id)
        ? selection.skipped
        : [...selection.skipped, currentQuestion.id],
    };
    setSelection(nextSelection);
    if (isLastQuestion) {
      void submit(buildResult(questions, nextSelection));
      return;
    }
    setQuestionIndex((current) => current + 1);
  };

  const skipped = selection.skipped.includes(currentQuestion.id);
  const selectedSingle = selection.single[currentQuestion.id];
  const selectedMultiple = new Set(
    selection.multiple[currentQuestion.id] ?? [],
  );
  const canMoveNext = hasAnswer(currentQuestion, selection);

  return (
    <section
      className="mb-2 mt-3 flex h-[min(24.5rem,58dvh)] min-h-[20rem] w-full min-w-0 max-w-[calc(100vw-2rem)] self-stretch flex-col overflow-hidden rounded-2xl border border-border/70 bg-background shadow-sm sm:h-[24.5rem] sm:min-h-0 md:max-w-[44rem]"
      aria-label={t("ask-user.title")}
    >
      <header className="flex min-h-20 shrink-0 items-start gap-2 px-3 py-3 sm:h-20 sm:min-h-0 sm:gap-3 sm:px-5 sm:py-3.5">
        <span className="mt-0.5 shrink-0 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium tabular-nums text-primary">
          {questionIndex + 1}/{questions.length}
        </span>
        <div className="min-w-0 flex-1">
          <h3
            className="max-h-10 overflow-hidden break-words text-sm font-semibold leading-5 text-foreground sm:truncate sm:text-base sm:leading-6"
            title={currentQuestion.question}
          >
            {currentQuestion.question}
          </h3>
          <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground sm:gap-2">
            {currentQuestion.header && (
              <span className="max-w-full truncate sm:max-w-48">
                {currentQuestion.header}
              </span>
            )}
            <span className="rounded-md bg-muted px-1.5 py-0.5">
              {t(
                currentQuestion.type === "multiple"
                  ? "ask-user.multiple"
                  : "ask-user.single",
              )}
            </span>
          </div>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          disabled={submitting}
          onClick={() => void submit(buildSkippedResult(questions))}
          aria-label={t("ask-user.skip-all")}
          title={t("ask-user.skip-all")}
          className="-mr-1 shrink-0 text-muted-foreground hover:text-foreground"
          unClickable
        >
          <X className="h-4 w-4" />
        </Button>
      </header>

      <fieldset
        key={currentQuestion.id}
        disabled={submitting}
        className="thin-scrollbar min-h-0 min-w-0 flex-1 animate-fade-in overflow-x-hidden overflow-y-auto border-0 px-3 pb-3 sm:px-5 sm:pb-4"
      >
        <legend className="sr-only">{currentQuestion.question}</legend>
        <div className={cn("space-y-1.5", skipped && "opacity-45")}>
          {currentQuestion.options.map((option, optionIndex) => {
            const selected =
              currentQuestion.type === "multiple"
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
                      custom: {
                        ...current.custom,
                        [currentQuestion.id]: "",
                      },
                      skipped: current.skipped.filter(
                        (id) => id !== currentQuestion.id,
                      ),
                    };
                    if (currentQuestion.type === "single") {
                      next.single[currentQuestion.id] = option.label;
                    } else {
                      const values = new Set(
                        next.multiple[currentQuestion.id] ?? [],
                      );
                      if (values.has(option.label)) {
                        values.delete(option.label);
                      } else {
                        values.add(option.label);
                      }
                      next.multiple[currentQuestion.id] = [...values];
                    }
                    return next;
                  });
                }}
                className={cn(
                  "flex min-h-14 w-full min-w-0 items-center gap-2.5 rounded-xl border px-3 py-2.5 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:gap-3 sm:px-3.5 sm:py-2",
                  selected
                    ? "border-primary/40 bg-primary/10 text-foreground"
                    : "border-transparent bg-muted/45 text-foreground hover:bg-muted/70",
                )}
              >
                <span className="min-w-0 flex-1">
                  <span className="block break-words text-sm font-medium leading-5 sm:truncate">
                    {option.label}
                  </span>
                  {option.description && (
                    <span className="mt-0.5 block break-words text-xs leading-relaxed text-muted-foreground sm:truncate">
                      {option.description}
                    </span>
                  )}
                </span>
                <span
                  className={cn(
                    "flex h-6 min-w-6 shrink-0 items-center justify-center rounded-md border px-1 text-xs tabular-nums",
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
              </button>
            );
          })}

          <label
            className={cn(
              "flex min-h-14 cursor-text items-center gap-2.5 rounded-xl border px-3 py-2 transition-colors focus-within:ring-2 focus-within:ring-ring sm:gap-3 sm:px-3.5",
              selection.custom[currentQuestion.id]?.trim()
                ? "border-primary/40 bg-primary/10"
                : "border-transparent bg-muted/45 hover:bg-muted/70",
            )}
          >
            <input
              type="text"
              maxLength={1000}
              disabled={skipped || submitting}
              value={selection.custom[currentQuestion.id] ?? ""}
              onChange={(event) => {
                const value = event.target.value;
                setSelection((current) => ({
                  single:
                    currentQuestion.type === "single"
                      ? { ...current.single, [currentQuestion.id]: "" }
                      : current.single,
                  multiple: current.multiple,
                  custom: {
                    ...current.custom,
                    [currentQuestion.id]: value,
                  },
                  skipped: current.skipped.filter(
                    (id) => id !== currentQuestion.id,
                  ),
                }));
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter" && canMoveNext) {
                  event.preventDefault();
                  moveNext();
                }
              }}
              placeholder={t("ask-user.other-placeholder")}
              className="min-w-0 flex-1 appearance-none border-0 bg-transparent p-0 text-sm text-foreground shadow-none outline-none ring-0 placeholder:text-muted-foreground focus:border-0 focus:outline-none focus:ring-0 focus-visible:outline-none disabled:cursor-not-allowed"
            />
            <span className="flex h-6 min-w-6 shrink-0 items-center justify-center rounded-md border border-border bg-background px-1 text-xs tabular-nums text-muted-foreground">
              {currentQuestion.options.length + 1}
            </span>
          </label>
        </div>
      </fieldset>

      <footer className="flex min-w-0 shrink-0 items-center justify-between gap-2 px-3 pb-3 sm:gap-3 sm:px-5 sm:pb-4">
        <div className="shrink-0">
          {questionIndex > 0 && (
            <Button
              type="button"
              variant="outline"
              size="xs"
              disabled={submitting}
              onClick={() => setQuestionIndex((current) => current - 1)}
              className="gap-1.5"
              unClickable
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              {t("ask-user.back")}
            </Button>
          )}
        </div>
        <div className="ml-auto flex min-w-0 items-center gap-1.5 sm:gap-2">
          <Button
            type="button"
            variant="outline"
            size="xs"
            disabled={submitting}
            onClick={skipCurrentQuestion}
            unClickable
          >
            {t("ask-user.skip")}
          </Button>
          <Button
            type="button"
            size="xs"
            disabled={!canMoveNext || submitting}
            onClick={moveNext}
            className="gap-1.5"
            unClickable
          >
            {submitting && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            {t(isLastQuestion ? "ask-user.submit" : "ask-user.next")}
            {!submitting && <CornerDownLeft className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </footer>
    </section>
  );
}
