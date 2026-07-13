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

function useEnergyCanvas(
  canvasRef: React.RefObject<HTMLCanvasElement | null>,
  active: boolean,
  disabled: boolean,
) {
  React.useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext("2d");
    if (!context) return;

    if (disabled || !active) {
      context.clearRect(0, 0, canvas.width, canvas.height);
      return;
    }

    let animationFrame = 0;
    let running = false;
    let width = 0;
    let height = 0;
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    const resize = () => {
      const bounds = canvas.getBoundingClientRect();
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      width = Math.max(1, bounds.width);
      height = Math.max(1, bounds.height);
      canvas.width = Math.round(width * ratio);
      canvas.height = Math.round(height * ratio);
      context.setTransform(ratio, 0, 0, ratio, 0, 0);
    };

    const draw = (time: number) => {
      context.clearRect(0, 0, width, height);

      const baseGlow = context.createLinearGradient(0, 0, width, 0);
      baseGlow.addColorStop(0, "rgba(105, 78, 178, 0.08)");
      baseGlow.addColorStop(0.48, "rgba(129, 91, 224, 0.24)");
      baseGlow.addColorStop(0.82, "rgba(160, 116, 246, 0.42)");
      baseGlow.addColorStop(1, "rgba(205, 177, 255, 0.68)");
      context.fillStyle = baseGlow;
      context.fillRect(0, 0, width, height);

      context.save();
      context.globalCompositeOperation = "lighter";

      for (let band = 0; band < 3; band += 1) {
        const bandGlow = context.createLinearGradient(0, 0, width, 0);
        bandGlow.addColorStop(0, "rgba(142, 107, 217, 0)");
        bandGlow.addColorStop(
          0.55,
          `rgba(169, 130, 249, ${0.08 + band * 0.03})`,
        );
        bandGlow.addColorStop(1, `rgba(232, 218, 255, ${0.3 - band * 0.05})`);
        context.beginPath();
        for (let x = 0; x <= width + 4; x += 4) {
          const y =
            height * (0.36 + band * 0.15) +
            Math.sin(x * (0.035 + band * 0.007) - time * (1.25 + band * 0.24)) *
              (2.2 + band * 0.65) +
            Math.cos(x * 0.018 + time * 0.72) * 1.4;
          if (x === 0) context.moveTo(x, y);
          else context.lineTo(x, y);
        }
        context.strokeStyle = bandGlow;
        context.lineWidth = 3.2 + band * 1.3;
        context.stroke();
      }

      for (let particle = 0; particle < 24; particle += 1) {
        const seed =
          Math.sin((particle + 1) * 91.345) * 47453.5453 -
          Math.floor(Math.sin((particle + 1) * 91.345) * 47453.5453);
        const travel = (seed + time * (0.035 + (particle % 6) * 0.006)) % 1;
        const x = width * (0.12 + travel * 0.88);
        const y =
          height * (0.18 + ((particle * 37) % 61) / 95) +
          Math.sin(time * (1.3 + (particle % 4) * 0.16) + particle * 1.7) * 2.5;
        const pulse = (Math.sin(time * 3.1 + particle * 2.31) + 1) / 2;
        const radius = 1.4 + (particle % 4) * 0.42 + pulse * 0.8;
        const alpha = 0.18 + pulse * 0.56;
        const spark = context.createRadialGradient(x, y, 0, x, y, radius * 3.4);
        spark.addColorStop(0, `rgba(252, 249, 255, ${alpha})`);
        spark.addColorStop(0.28, `rgba(191, 160, 255, ${alpha * 0.78})`);
        spark.addColorStop(1, "rgba(142, 107, 217, 0)");
        context.fillStyle = spark;
        context.beginPath();
        context.arc(x, y, radius * 3.4, 0, Math.PI * 2);
        context.fill();
      }

      const edgeGlow = context.createRadialGradient(
        width,
        height / 2,
        0,
        width,
        height / 2,
        Math.max(28, width * 0.22),
      );
      edgeGlow.addColorStop(0, "rgba(255, 252, 255, 0.8)");
      edgeGlow.addColorStop(0.18, "rgba(209, 180, 255, 0.46)");
      edgeGlow.addColorStop(1, "rgba(142, 107, 217, 0)");
      context.fillStyle = edgeGlow;
      context.fillRect(0, 0, width, height);
      context.restore();
    };

    const animate = (timestamp: number) => {
      draw(timestamp / 1000);
      if (!document.hidden) {
        animationFrame = window.requestAnimationFrame(animate);
      } else {
        running = false;
      }
    };

    const start = () => {
      if (running || reducedMotion || document.hidden) return;
      running = true;
      animationFrame = window.requestAnimationFrame(animate);
    };

    const handleVisibilityChange = () => {
      if (document.hidden) {
        window.cancelAnimationFrame(animationFrame);
        running = false;
        return;
      }
      draw(performance.now() / 1000);
      start();
    };

    resize();
    draw(0);
    start();

    const observer = new ResizeObserver(() => {
      resize();
      draw(performance.now() / 1000);
    });
    observer.observe(canvas);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      window.cancelAnimationFrame(animationFrame);
      observer.disconnect();
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      context.clearRect(0, 0, width, height);
    };
  }, [active, canvasRef, disabled]);
}

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
  const canvasRef = React.useRef<HTMLCanvasElement>(null);
  const [dragging, setDragging] = React.useState(false);
  const total = Math.max(levels.length, 1);
  const safeIndex = Math.min(Math.max(index, 0), total - 1);
  const restingProgress = getProgressForIndex(safeIndex, total);
  const [dragProgress, setDragProgress] = React.useState(restingProgress);
  const lastNotifiedIndexRef = React.useRef(safeIndex);
  const progress = dragging ? dragProgress : restingProgress;
  const atTopStop = total > 1 && progress >= 99.5;

  useEnergyCanvas(canvasRef, atTopStop, disabled);

  const notifyIndexChange = (nextIndex: number) => {
    if (lastNotifiedIndexRef.current === nextIndex) return;
    lastNotifiedIndexRef.current = nextIndex;
    onIndexChange(nextIndex);
  };

  const handleValueChange = (value: number[]) => {
    const nextProgress = Math.min(100, Math.max(0, value[0] ?? 0));
    setDragProgress(nextProgress);
    notifyIndexChange(getIndexForProgress(nextProgress, total));
  };

  const handleValueCommit = (value: number[]) => {
    const nextIndex = getIndexForProgress(value[0] ?? dragProgress, total);
    notifyIndexChange(nextIndex);
    setDragProgress(getProgressForIndex(nextIndex, total));
    setDragging(false);
  };

  const cancelDragging = () => {
    lastNotifiedIndexRef.current = safeIndex;
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
    setDragProgress(getProgressForIndex(nextIndex, total));
    onIndexChange(nextIndex);
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
          setDragProgress(restingProgress);
          setDragging(true);
        }}
        onPointerCancel={cancelDragging}
        onKeyDown={handleKeyDown}
      >
        <SliderPrimitive.Track className="claude-range-slider__track">
          <SliderPrimitive.Range className="claude-range-slider__fill">
            <span className="claude-range-slider__energy" aria-hidden="true">
              <canvas ref={canvasRef} />
            </span>
          </SliderPrimitive.Range>
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
