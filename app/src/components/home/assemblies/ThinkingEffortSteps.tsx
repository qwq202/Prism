import React from "react";
import { motion, useReducedMotion } from "framer-motion";
import { Check } from "lucide-react";

import { cn } from "@/components/ui/lib/utils.ts";

type ThinkingEffortStepsProps = {
  levels: string[];
  labels: string[];
  index: number;
  ariaLabel: string;
  disabled?: boolean;
  className?: string;
  onIndexChange: (index: number) => void;
};

const BAR_COUNT = 5;

export function ThinkingEffortSteps({
  levels,
  labels,
  index,
  ariaLabel,
  disabled = false,
  className,
  onIndexChange,
}: ThinkingEffortStepsProps) {
  const reduceMotion = useReducedMotion();
  const total = Math.max(levels.length, 1);
  const safeIndex = Math.min(Math.max(index, 0), total - 1);
  const listRef = React.useRef<HTMLDivElement>(null);

  const moveTo = (nextIndex: number) => {
    if (disabled) return;
    const clamped = Math.min(Math.max(nextIndex, 0), total - 1);
    if (clamped !== safeIndex) onIndexChange(clamped);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (disabled) return;

    let nextIndex = safeIndex;
    switch (event.key) {
      case "ArrowUp":
      case "ArrowLeft":
        nextIndex = Math.max(0, safeIndex - 1);
        break;
      case "ArrowDown":
      case "ArrowRight":
        nextIndex = Math.min(total - 1, safeIndex + 1);
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = total - 1;
        break;
      default:
        return;
    }

    event.preventDefault();
    onIndexChange(nextIndex);
    const option = listRef.current?.querySelector<HTMLElement>(
      `[data-effort-index="${nextIndex}"]`,
    );
    option?.focus();
  };

  return (
    <div
      ref={listRef}
      role="listbox"
      aria-label={ariaLabel}
      aria-disabled={disabled || undefined}
      aria-activedescendant={
        disabled ? undefined : `thinking-effort-option-${safeIndex}`
      }
      tabIndex={disabled ? -1 : 0}
      onKeyDown={handleKeyDown}
      className={cn(
        "thinking-effort-steps relative flex flex-col gap-0.5 outline-none",
        disabled && "pointer-events-none opacity-45",
        className,
      )}
    >
      {levels.map((level, step) => {
        const selected = step === safeIndex;
        const filledBars = Math.max(
          1,
          Math.round(((step + 1) / total) * BAR_COUNT),
        );
        const label = labels[step] ?? level;

        return (
          <button
            key={level}
            id={`thinking-effort-option-${step}`}
            type="button"
            role="option"
            data-effort-index={step}
            aria-selected={selected}
            disabled={disabled}
            tabIndex={selected ? 0 : -1}
            onClick={() => moveTo(step)}
            className={cn(
              "relative flex w-full items-center gap-3 rounded-md px-2.5 py-2 text-left transition-colors duration-150",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1 focus-visible:ring-offset-background",
              selected
                ? "text-foreground"
                : "text-muted-foreground hover:bg-muted/70 hover:text-foreground",
            )}
          >
            {selected && (
              <motion.span
                layoutId={reduceMotion ? undefined : "thinking-effort-active"}
                className="absolute inset-0 rounded-md bg-foreground/[0.07] dark:bg-foreground/[0.12]"
                transition={
                  reduceMotion
                    ? { duration: 0 }
                    : { duration: 0.2, ease: [0.32, 0.72, 0, 1] }
                }
                aria-hidden="true"
              />
            )}

            <span className="relative z-10 min-w-0 flex-1 truncate text-[13px] font-medium leading-none">
              {label}
            </span>

            <span
              className="relative z-10 flex h-4 items-end gap-[3px]"
              aria-hidden="true"
            >
              {Array.from({ length: BAR_COUNT }, (_, bar) => {
                const active = bar < filledBars;
                return (
                  <span
                    key={bar}
                    className={cn(
                      "w-[3px] rounded-full transition-colors duration-150",
                      active
                        ? selected
                          ? "bg-foreground"
                          : "bg-muted-foreground/55"
                        : "bg-border",
                    )}
                    style={{ height: `${7 + bar * 2}px` }}
                  />
                );
              })}
            </span>

            <span
              className={cn(
                "relative z-10 flex h-3.5 w-3.5 shrink-0 items-center justify-center transition-opacity duration-150",
                selected ? "opacity-100" : "opacity-0",
              )}
              aria-hidden="true"
            >
              <Check className="h-3.5 w-3.5 stroke-[2.5]" />
            </span>
          </button>
        );
      })}
    </div>
  );
}
