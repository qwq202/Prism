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
  progress: number,
  disabled: boolean,
) {
  React.useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext("2d");
    if (!context) return;

    let animationFrame = 0;
    let frame = 0;
    let width = 0;
    let height = 0;
    const energy = Math.max(0, Math.min((progress - 25) / 75, 1));
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

    const draw = () => {
      context.clearRect(0, 0, width, height);
      if (disabled || energy <= 0) return;

      const time = frame / 60;
      const fillWidth = width * (progress / 100);
      const glow = context.createLinearGradient(0, 0, fillWidth, 0);
      glow.addColorStop(0, "rgba(142, 107, 217, 0)");
      glow.addColorStop(0.5, `rgba(142, 107, 217, ${0.12 * energy})`);
      glow.addColorStop(1, `rgba(172, 142, 246, ${0.58 * energy})`);
      context.fillStyle = glow;
      context.fillRect(0, 0, fillWidth, height);

      context.save();
      context.globalCompositeOperation = "lighter";
      for (let particle = 0; particle < 18; particle += 1) {
        const phase = particle * 1.813 + time * (0.45 + (particle % 5) * 0.08);
        const travel = (Math.sin(phase * 0.73) + 1) / 2;
        const x = fillWidth * (0.28 + travel * 0.72);
        const y =
          height / 2 +
          Math.sin(phase * 1.9) * height * 0.26 * energy +
          Math.cos(phase * 0.6) * 1.4;
        const radius = 0.45 + ((particle * 7) % 5) * 0.18;
        const alpha = (0.12 + ((Math.sin(phase * 2.4) + 1) / 2) * 0.5) * energy;
        const spark = context.createRadialGradient(x, y, 0, x, y, radius * 4);
        spark.addColorStop(0, `rgba(244, 237, 255, ${alpha})`);
        spark.addColorStop(0.32, `rgba(178, 144, 255, ${alpha * 0.72})`);
        spark.addColorStop(1, "rgba(142, 107, 217, 0)");
        context.fillStyle = spark;
        context.beginPath();
        context.arc(x, y, radius * 4, 0, Math.PI * 2);
        context.fill();
      }
      context.restore();
    };

    const animate = () => {
      draw();
      frame += 1;
      if (!reducedMotion && frame < 180 && energy > 0 && !disabled) {
        animationFrame = window.requestAnimationFrame(animate);
      }
    };

    resize();
    animate();

    const observer = new ResizeObserver(() => {
      resize();
      draw();
    });
    observer.observe(canvas);

    return () => {
      window.cancelAnimationFrame(animationFrame);
      observer.disconnect();
      context.clearRect(0, 0, width, height);
    };
  }, [canvasRef, disabled, progress]);
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
  const progress = total <= 1 ? 0 : (safeIndex / (total - 1)) * 100;
  const atTopStop = total > 1 && safeIndex === total - 1;

  useEnergyCanvas(canvasRef, progress, disabled);

  return (
    <div
      className={cn("claude-range-slider", className)}
      data-disabled={disabled || undefined}
      data-dragging={dragging || undefined}
      data-top-stop={atTopStop || undefined}
      style={
        {
          "--claude-slider-progress": `${progress}%`,
          "--claude-slider-energy-opacity": Math.max(
            0,
            Math.min((progress - 25) / 75, 1),
          ),
        } as React.CSSProperties
      }
    >
      <div className="claude-range-slider__labels" aria-hidden="true">
        <span>{fasterLabel}</span>
        <span>{smarterLabel}</span>
      </div>

      <SliderPrimitive.Root
        className="claude-range-slider__root"
        disabled={disabled}
        value={[safeIndex]}
        min={0}
        max={Math.max(total - 1, 0)}
        step={1}
        aria-label={ariaLabel}
        onValueChange={(value) => onIndexChange(value[0] ?? 0)}
        onValueCommit={() => setDragging(false)}
        onPointerDown={() => setDragging(true)}
        onPointerUp={() => setDragging(false)}
        onPointerCancel={() => setDragging(false)}
        onBlur={() => setDragging(false)}
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
              data-current={step === safeIndex || undefined}
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
