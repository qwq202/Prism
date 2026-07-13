import * as SliderPrimitive from "@radix-ui/react-slider";
import React from "react";

import { cn } from "@/components/ui/lib/utils.ts";

import "./ClaudeRangeSlider.less";

type ClaudeRangeSliderProps = {
  levels: string[];
  index: number;
  fasterLabel: string;
  smarterLabel: string;
  ariaLabel: string;
  ariaValueText?: string;
  disabled?: boolean;
  className?: string;
  onIndexChange: (index: number) => void;
};

function getProgressForIndex(index: number, total: number): number {
  return total <= 1 ? 0 : (index / (total - 1)) * 100;
}

function getIndexForProgress(progress: number, total: number): number {
  if (total <= 1) return 0;
  return Math.min(
    total - 1,
    Math.max(0, Math.round((progress / 100) * (total - 1))),
  );
}

export function ClaudeRangeSlider({
  levels,
  index,
  fasterLabel,
  smarterLabel,
  ariaLabel,
  ariaValueText,
  disabled = false,
  className,
  onIndexChange,
}: ClaudeRangeSliderProps) {
  const [dragging, setDragging] = React.useState(false);
  const total = Math.max(levels.length, 1);
  const safeIndex = Math.min(Math.max(index, 0), total - 1);
  const restingProgress = getProgressForIndex(safeIndex, total);
  const [dragProgress, setDragProgress] = React.useState(restingProgress);
  const lastNotifiedIndexRef = React.useRef(safeIndex);
  const pendingProgressRef = React.useRef(restingProgress);
  const animationFrameRef = React.useRef<number | null>(null);
  const onIndexChangeRef = React.useRef(onIndexChange);
  const progress = dragging ? dragProgress : restingProgress;
  const atTopStop = total > 1 && progress >= 99.5;

  React.useEffect(() => {
    onIndexChangeRef.current = onIndexChange;
  }, [onIndexChange]);

  const notifyIndexChange = React.useCallback((nextIndex: number) => {
    if (lastNotifiedIndexRef.current === nextIndex) return;
    lastNotifiedIndexRef.current = nextIndex;
    onIndexChangeRef.current(nextIndex);
  }, []);

  const cancelScheduledProgress = React.useCallback(() => {
    if (animationFrameRef.current === null) return;
    cancelAnimationFrame(animationFrameRef.current);
    animationFrameRef.current = null;
  }, []);

  React.useEffect(
    () => () => {
      cancelScheduledProgress();
    },
    [cancelScheduledProgress],
  );

  const handleValueChange = (value: number[]) => {
    const nextProgress = Math.min(100, Math.max(0, value[0] ?? 0));
    pendingProgressRef.current = nextProgress;

    if (animationFrameRef.current !== null) return;
    animationFrameRef.current = requestAnimationFrame(() => {
      animationFrameRef.current = null;
      const latestProgress = pendingProgressRef.current;
      setDragProgress(latestProgress);
      notifyIndexChange(getIndexForProgress(latestProgress, total));
    });
  };

  const handleValueCommit = () => {
    cancelScheduledProgress();
    const nextIndex = getIndexForProgress(pendingProgressRef.current, total);
    notifyIndexChange(nextIndex);
    const nextProgress = getProgressForIndex(nextIndex, total);
    pendingProgressRef.current = nextProgress;
    setDragProgress(nextProgress);
    setDragging(false);
  };

  const cancelDragging = () => {
    cancelScheduledProgress();
    lastNotifiedIndexRef.current = safeIndex;
    pendingProgressRef.current = restingProgress;
    setDragProgress(restingProgress);
    setDragging(false);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLSpanElement>) => {
    let nextIndex = safeIndex;

    switch (event.key) {
      case "ArrowLeft":
      case "ArrowDown":
        nextIndex = Math.max(0, safeIndex - 1);
        break;
      case "ArrowRight":
      case "ArrowUp":
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
    lastNotifiedIndexRef.current = nextIndex;
    const nextProgress = getProgressForIndex(nextIndex, total);
    pendingProgressRef.current = nextProgress;
    setDragProgress(nextProgress);
    onIndexChangeRef.current(nextIndex);
  };

  return (
    <div
      className={cn("claude-range-slider", className)}
      data-disabled={disabled || undefined}
      data-dragging={dragging || undefined}
      data-top-stop={atTopStop || undefined}
    >
      <div className="claude-range-slider__labels" aria-hidden="true">
        <span>{fasterLabel}</span>
        <span>{smarterLabel}</span>
      </div>

      <SliderPrimitive.Root
        className="claude-range-slider__root"
        disabled={disabled}
        value={[progress]}
        min={0}
        max={100}
        step={0.1}
        aria-label={ariaLabel}
        onValueChange={handleValueChange}
        onValueCommit={handleValueCommit}
        onPointerDown={() => {
          lastNotifiedIndexRef.current = safeIndex;
          pendingProgressRef.current = restingProgress;
          setDragProgress(restingProgress);
          setDragging(true);
        }}
        onPointerCancel={cancelDragging}
        onKeyDown={handleKeyDown}
      >
        <SliderPrimitive.Track className="claude-range-slider__track">
          <SliderPrimitive.Range className="claude-range-slider__fill" />
        </SliderPrimitive.Track>

        <span className="claude-range-slider__dots" aria-hidden="true">
          {levels.map((level, step) => (
            <span
              key={level}
              className="claude-range-slider__dot"
              data-last={step === total - 1 || undefined}
              style={
                {
                  "--claude-slider-dot-delay": `${Math.max(
                    0,
                    (total - 1 - step) * 35,
                  )}ms`,
                } as React.CSSProperties
              }
            />
          ))}
        </span>

        <SliderPrimitive.Thumb
          className="claude-range-slider__thumb"
          aria-valuetext={ariaValueText}
        />
      </SliderPrimitive.Root>
    </div>
  );
}
