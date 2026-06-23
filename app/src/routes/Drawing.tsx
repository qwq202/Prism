import { useState } from "react";
import {
  Wand2,
  Settings,
  Plus,
  SlidersHorizontal,
  Palette,
  Ratio,
  Upload,
  ArrowUp,
} from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/components/ui/lib/utils";
import { useTranslation } from "react-i18next";

type Mode = "generate" | "edit";

function Drawing() {
  const { t } = useTranslation();
  const [mode, setMode] = useState<Mode>("generate");
  const [prompt, setPrompt] = useState("");
  const [focused, setFocused] = useState(false);

  return (
    <div className="flex h-full min-h-0 w-full bg-background text-foreground overflow-hidden">
      {/* Left Sidebar */}
      <aside className="w-72 min-h-0 flex flex-col z-10 shrink-0 border-r border-border/60 bg-card/50 backdrop-blur-sm">
        <div className="p-5 flex-1 flex flex-col gap-5 overflow-y-auto">
          {/* Provider */}
          <div className="space-y-2.5">
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold tracking-widest text-muted-foreground uppercase">
                {t("drawing.model")}
              </label>
              <button className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors group">
                <Settings className="w-3 h-3 group-hover:rotate-45 transition-transform duration-300" />
                {t("drawing.manage")}
              </button>
            </div>
            <Select defaultValue="openai">
              <SelectTrigger className="w-full h-10 text-sm border-border/60 bg-background/60">
                <SelectValue placeholder={t("drawing.selectModel")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI (DALL-E 3)</SelectItem>
                <SelectItem value="midjourney">Midjourney</SelectItem>
                <SelectItem value="sd">Stable Diffusion</SelectItem>
              </SelectContent>
            </Select>
          </div>

        </div>
      </aside>

      {/* Main Area */}
      <main className="relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_50%_at_50%_-20%,hsl(var(--primary)/0.06),transparent)]" />
        <div className="absolute inset-0 bg-[radial-gradient(hsl(var(--muted-foreground))_1px,transparent_1px)] [background-size:28px_28px] opacity-[0.035]" />

        {/* Mode Toggle */}
        <div className="absolute top-6 left-1/2 -translate-x-1/2 z-20">
          <div className="relative grid grid-cols-2 items-center rounded-full border border-border/70 bg-background/80 p-1 shadow-sm backdrop-blur-xl">
            <div
              className={cn(
                "pointer-events-none absolute inset-y-1 left-1 w-[calc(50%-0.25rem)] rounded-full bg-foreground shadow-sm transition-all duration-300 ease-out",
                mode === "edit" && "translate-x-full"
              )}
            />
            {(["generate", "edit"] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={cn(
                  "relative z-10 min-w-[76px] px-6 py-2 rounded-full text-sm font-medium transition-colors duration-300",
                  mode === m ? "text-background" : "text-muted-foreground hover:text-foreground"
                )}
                aria-pressed={mode === m}
              >
                {t(`drawing.mode.${m}`)}
              </button>
            ))}
          </div>
        </div>

        {/* Empty Canvas */}
        <div className="flex-1 flex flex-col items-center justify-center pb-44 relative z-10">
          <p className="text-base font-semibold text-foreground/70 tracking-wide">
            {t("drawing.emptyTitle")}
          </p>
          <p className="text-sm text-muted-foreground/50 mt-3">
            {t("drawing.emptyPrompt")}
          </p>
        </div>

        {/* Floating Input */}
        <div className="absolute bottom-6 left-0 right-0 px-6 sm:bottom-8 sm:px-10 flex justify-center z-20 pointer-events-none">
          <div
            className={cn(
              "pointer-events-auto w-full max-w-2xl rounded-2xl transition-all duration-200",
              "border bg-background/96 backdrop-blur-2xl",
              focused
                ? "border-border shadow-[0_24px_64px_-12px_rgba(0,0,0,0.16),0_0_0_1px_rgba(0,0,0,0.02)] dark:shadow-[0_24px_64px_-12px_rgba(0,0,0,0.5)]"
                : "border-border/55 shadow-[0_8px_32px_-8px_rgba(0,0,0,0.1)] dark:shadow-[0_8px_32px_-8px_rgba(0,0,0,0.4)]"
            )}
          >
            {/* Meta row */}
            <div className="flex items-center justify-between px-4 pt-3.5 pb-0">
              <div className="flex items-center gap-2">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-foreground text-background">
                  <Wand2 className="h-3 w-3" />
                </div>
                <span className="text-[13px] font-semibold text-foreground">
                  {t("drawing.promptLabel")}
                </span>
                <span className="text-[11px] text-muted-foreground/50 hidden sm:inline">
                  {t("drawing.promptHint")}
                </span>
              </div>
              <button
                className="rounded-md p-1.5 text-muted-foreground/50 hover:text-muted-foreground hover:bg-muted/50 transition-all duration-150"
                title={t("drawing.uploadReference")}
              >
                <Upload className="h-3.5 w-3.5" />
              </button>
            </div>

            {/* Textarea */}
            <Textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              onFocus={() => setFocused(true)}
              onBlur={() => setFocused(false)}
              className="min-h-[84px] w-full resize-none border-0 bg-transparent px-4 py-3 text-sm leading-relaxed text-foreground shadow-none placeholder:text-muted-foreground/35 focus-visible:ring-0 focus-visible:ring-offset-0"
              placeholder={t("drawing.promptPlaceholder")}
            />

            {/* Toolbar */}
            <div className="flex items-center justify-between px-2.5 pb-2.5 gap-2">
              <div className="flex items-center">
                {[
                  { icon: Palette, key: "style" },
                  { icon: Ratio, key: "ratio" },
                ].map(({ icon: Icon, key }) => (
                  <button
                    key={key}
                    className="flex items-center gap-1.5 h-8 px-2.5 rounded-lg text-xs text-muted-foreground/70 hover:text-foreground hover:bg-muted/50 transition-all duration-150 font-medium"
                  >
                    <Icon className="h-3.5 w-3.5" />
                    {t(`drawing.tools.${key}`)}
                  </button>
                ))}
                <div className="mx-1 h-4 w-px bg-border/60" />
                <button
                  className="h-8 w-8 rounded-lg flex items-center justify-center text-muted-foreground/50 hover:text-muted-foreground hover:bg-muted/50 transition-all duration-150"
                  title={t("drawing.advanced")}
                >
                  <SlidersHorizontal className="h-3.5 w-3.5" />
                </button>
              </div>

              <button
                disabled={!prompt.trim()}
                className={cn(
                  "flex h-9 w-9 items-center justify-center rounded-full transition-all duration-150 shrink-0 select-none",
                  prompt.trim()
                    ? "bg-foreground text-background hover:opacity-85 active:scale-[0.96] shadow-sm"
                    : "bg-muted/60 text-muted-foreground/40 cursor-not-allowed"
                )}
                aria-label={t("drawing.generateImage")}
                title={t("drawing.generateImage")}
              >
                <ArrowUp className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      </main>

      {/* Right Sidebar - History */}
      <aside className="w-[72px] min-h-0 bg-card/50 border-l border-border/60 flex flex-col z-10 shrink-0 backdrop-blur-sm">
        <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-3 items-center no-scrollbar pt-4">
          <button className="w-12 h-12 border-2 border-dashed border-border/60 rounded-2xl flex items-center justify-center text-muted-foreground/60 hover:border-primary/40 hover:text-primary/60 hover:bg-primary/5 transition-all duration-200 group">
            <Plus className="w-4 h-4 group-hover:rotate-90 transition-transform duration-300" />
          </button>
          <div className="w-12 h-12 border-2 border-primary/60 rounded-2xl bg-gradient-to-br from-violet-500/10 to-blue-500/10 shadow-sm relative cursor-pointer overflow-hidden group ring-2 ring-primary/10">
            <div className="absolute inset-0 bg-gradient-to-br from-violet-500/5 to-blue-500/5 group-hover:from-violet-500/10 group-hover:to-blue-500/10 transition-all" />
          </div>
          {[0.5, 0.3].map((opacity, i) => (
            <div
              key={i}
              className="w-12 h-12 border border-border/50 rounded-2xl bg-muted/40 cursor-pointer hover:border-border hover:bg-muted/60 transition-all duration-200 overflow-hidden"
              style={{ opacity }}
            />
          ))}
        </div>
      </aside>
    </div>
  );
}

export default Drawing;
