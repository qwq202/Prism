import { useState } from "react";
import {
  Wand2,
  Settings,
  Sparkles,
  Plus,
  Image as ImageIcon,
  Languages,
  SlidersHorizontal,
  Palette,
  Ratio,
  Upload,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

type Mode = "generate" | "edit";

function Drawing() {
  const [mode, setMode] = useState<Mode>("generate");

  return (
    <div className="flex h-full min-h-0 w-full bg-background text-foreground font-sans overflow-hidden">
      {/* Left Sidebar - Configuration */}
      <aside className="w-72 min-h-0 bg-card border-r border-border flex flex-col z-10 shrink-0">
        {/* Config Content */}
        <div className="p-6 flex-1 flex flex-col gap-6 overflow-y-auto">
          {/* Provider Selection */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <label className="text-sm font-semibold text-foreground">模型提供商</label>
              <Button variant="ghost" size="sm" className="h-8 px-2 text-muted-foreground hover:text-primary">
                <Settings className="w-3.5 h-3.5 mr-1" />
                管理
              </Button>
            </div>
            <Select defaultValue="openai">
              <SelectTrigger className="w-full">
                <SelectValue placeholder="选择模型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI (DALL-E 3)</SelectItem>
                <SelectItem value="midjourney">Midjourney</SelectItem>
                <SelectItem value="sd">Stable Diffusion</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Empty State / Notice */}
          <div className="mt-8 flex-1 flex flex-col items-center justify-center text-center">
            <div className="w-20 h-20 bg-muted rounded-2xl flex items-center justify-center mb-5 border border-border">
              <ImageIcon className="w-8 h-8 text-muted-foreground" />
            </div>
            <h3 className="text-sm font-semibold text-foreground mb-2">暂无可用模型</h3>
            <p className="text-muted-foreground text-xs leading-relaxed mb-6 px-2">
              请先新增模型并设置端点类型为图像生成以开始创作。
            </p>
            <Button className="w-full">
              <Settings className="w-4 h-4 mr-2" />
              去设置
            </Button>
          </div>
        </div>
      </aside>

      {/* Main Canvas Area */}
      <main className="relative flex min-h-0 min-w-0 flex-1 flex-col bg-background overflow-hidden">
        {/* Subtle grid background for the entire main area */}
        <div className="absolute inset-0 bg-[radial-gradient(hsl(var(--muted-foreground))_1px,transparent_1px)] [background-size:24px_24px] opacity-[0.05]"></div>

        {/* Top Controls - Floating */}
        <div className="absolute top-6 left-1/2 -translate-x-1/2 z-20">
          <div className="relative grid grid-cols-2 items-center rounded-full border border-border/80 bg-background/90 p-1.5 shadow-sm backdrop-blur-xl">
            <div
              className={`pointer-events-none absolute inset-y-1.5 left-1.5 w-[calc(50%-0.375rem)] rounded-full bg-foreground shadow-sm transition-transform duration-300 ease-out ${
                mode === "edit" ? "translate-x-full" : "translate-x-0"
              }`}
            />
            <button
              onClick={() => setMode("generate")}
              className={`relative z-10 min-w-[72px] px-7 py-2.5 rounded-full text-sm font-medium transition-colors duration-300 ${
                mode === "generate"
                  ? "text-background"
                  : "text-muted-foreground hover:text-foreground"
              }`}
              aria-pressed={mode === "generate"}
            >
              绘图
            </button>
            <button
              onClick={() => setMode("edit")}
              className={`relative z-10 min-w-[72px] px-7 py-2.5 rounded-full text-sm font-medium transition-colors duration-300 ${
                mode === "edit"
                  ? "text-background"
                  : "text-muted-foreground hover:text-foreground"
              }`}
              aria-pressed={mode === "edit"}
            >
              编辑
            </button>
          </div>
        </div>

        {/* Canvas (No square box) */}
        <div className="flex-1 flex flex-col items-center justify-center pb-32 relative z-10">
          <div className="flex flex-col items-center justify-center gap-5">
            <div className="flex flex-col items-center gap-1.5">
              <span className="text-foreground/80 font-medium tracking-wide text-base">开始你的创作</span>
              <span className="text-muted-foreground/60 text-sm">在下方输入描述，AI 将为你生成精美图片</span>
            </div>
          </div>
        </div>

        {/* Input Area - Floating Bottom */}
        <div className="absolute bottom-5 left-0 right-0 px-4 sm:bottom-8 sm:px-8 flex justify-center z-20 pointer-events-none">
          <div className="pointer-events-auto w-full max-w-3xl overflow-hidden rounded-[26px] border border-white/80 bg-white/[0.88] shadow-[0_24px_80px_rgba(15,23,42,0.16),inset_0_1px_0_rgba(255,255,255,0.9)] backdrop-blur-2xl transition-all duration-300 focus-within:border-sky-300/80 focus-within:shadow-[0_28px_90px_rgba(14,165,233,0.20),inset_0_1px_0_rgba(255,255,255,0.95)] dark:border-border/80 dark:bg-background/[0.92] dark:focus-within:border-primary/50">
            <div className="flex items-center justify-between gap-3 border-b border-slate-200/70 px-5 py-3 dark:border-border/50">
              <div className="flex min-w-0 items-center gap-2.5">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-slate-950 text-white shadow-[0_10px_24px_rgba(15,23,42,0.18)] dark:bg-foreground dark:text-background">
                  <Wand2 className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-foreground">
                    图像提示词
                  </div>
                  <div className="truncate text-xs text-muted-foreground">
                    支持中文描述，文本内容用双引号包裹
                  </div>
                </div>
              </div>
              <Button
                variant="ghost"
                size="icon"
                className="h-11 w-11 shrink-0 rounded-2xl text-muted-foreground hover:bg-sky-50 hover:text-sky-700 dark:hover:bg-primary/10 dark:hover:text-primary"
                aria-label="上传参考图"
                title="上传参考图"
              >
                <Upload className="h-[18px] w-[18px]" />
              </Button>
            </div>
            <Textarea
              className="min-h-[104px] w-full resize-none border-0 bg-transparent px-5 py-4 text-[15px] leading-relaxed text-foreground shadow-none outline-none placeholder:text-muted-foreground/60 focus-visible:ring-0 focus-visible:ring-offset-0"
              placeholder='输入你的图片描述，文本绘制用 "双引号" 包裹'
            />
            <div className="flex flex-col gap-3 border-t border-slate-200/70 bg-slate-50/75 px-4 py-3 dark:border-border/50 dark:bg-muted/15 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-10 rounded-full border border-slate-200/80 bg-white/80 px-3 text-muted-foreground shadow-sm hover:bg-sky-50 hover:text-sky-700 dark:border-border/70 dark:bg-background/70 dark:hover:bg-primary/10 dark:hover:text-primary"
                >
                  <Languages className="mr-2 h-4 w-4" />
                  翻译
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-10 rounded-full border border-slate-200/80 bg-white/80 px-3 text-muted-foreground shadow-sm hover:bg-rose-50 hover:text-rose-700 dark:border-border/70 dark:bg-background/70 dark:hover:bg-primary/10 dark:hover:text-primary"
                >
                  <Palette className="mr-2 h-4 w-4" />
                  风格
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-10 rounded-full border border-slate-200/80 bg-white/80 px-3 text-muted-foreground shadow-sm hover:bg-amber-50 hover:text-amber-700 dark:border-border/70 dark:bg-background/70 dark:hover:bg-primary/10 dark:hover:text-primary"
                >
                  <Ratio className="mr-2 h-4 w-4" />
                  比例
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-10 w-10 rounded-full border border-slate-200/80 bg-white/80 text-muted-foreground shadow-sm hover:bg-slate-100 hover:text-foreground dark:border-border/70 dark:bg-background/70"
                  aria-label="高级参数"
                  title="高级参数"
                >
                  <SlidersHorizontal className="h-4 w-4" />
                </Button>
              </div>
              <Button className="h-11 shrink-0 rounded-2xl bg-slate-950 px-5 font-semibold text-white shadow-[0_12px_28px_rgba(15,23,42,0.22)] transition-all hover:-translate-y-0.5 hover:bg-slate-900 hover:shadow-[0_16px_34px_rgba(15,23,42,0.26)] dark:bg-primary dark:text-primary-foreground dark:hover:bg-primary/90">
                <Sparkles className="mr-2 h-4 w-4" />
                生成图片
              </Button>
            </div>
          </div>
        </div>
      </main>

      {/* Right Sidebar - History */}
      <aside className="w-24 min-h-0 bg-card border-l border-border flex flex-col z-10 shrink-0">
        <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-4 items-center no-scrollbar">
          {/* Empty slot */}
          <button className="w-14 h-14 border-2 border-dashed border-border rounded-xl flex items-center justify-center text-muted-foreground hover:border-primary/50 hover:text-primary hover:bg-primary/5 transition-all">
            <Plus className="w-5 h-5" />
          </button>
          {/* Active slot dummy */}
          <div className="w-14 h-14 border-2 border-primary rounded-xl bg-muted shadow-sm relative cursor-pointer overflow-hidden group">
            <div className="absolute inset-0 bg-black/0 group-hover:bg-black/5 transition-colors"></div>
          </div>
          {/* History slots */}
          <div className="w-14 h-14 border border-border rounded-xl bg-muted cursor-pointer hover:border-primary/50 transition-colors overflow-hidden opacity-60 hover:opacity-100"></div>
          <div className="w-14 h-14 border border-border rounded-xl bg-muted cursor-pointer hover:border-primary/50 transition-colors overflow-hidden opacity-60 hover:opacity-100"></div>
        </div>
      </aside>
    </div>
  );
}

export default Drawing;
