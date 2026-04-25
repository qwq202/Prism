import {
  getOpenAIResponsesCapabilities,
  isDeepSeekV4ModelId,
  isGeminiModelId,
  isOpenAIResponsesNativeWebModel,
  isXAIModelId,
  selectDeepSeekReasoningEffort,
  selectDeepSeekThinkingEnabled,
  selectOpenAIReasoningEffort,
  selectOpenAIReasoningSummary,
  selectOpenAIResponsesWebSearch,
  selectGeminiThinkingBudget,
  selectGeminiGoogleSearch,
  selectGeminiURLContext,
  selectModel,
  selectSupportModels,
  selectWeb,
  selectXAIWebSearch,
  selectXAIXSearch,
  setOpenAIReasoningEffort,
  setOpenAIReasoningSummary,
  setOpenAIResponsesWebSearch,
  setDeepSeekReasoningEffort,
  setDeepSeekThinkingEnabled,
  setGeminiThinkingBudget,
  setGeminiGoogleSearch,
  setGeminiURLContext,
  setXAIWebSearch,
  setXAIXSearch,
  supportsGeminiThinkingBudgetControl,
  toggleWeb,
  useConversationActions,
  useMessages,
} from "@/store/chat.ts";
import { infoWebSearchSelector } from "@/store/info.ts";
import {
  Brain,
  Globe,
  Info,
  TriangleAlert,
  MessageSquarePlus,
  Wifi,
  WifiOff,
} from "lucide-react";
import { useDispatch, useSelector } from "react-redux";
import { useTranslation } from "react-i18next";
import React from "react";
import { cn } from "@/components/ui/lib/utils.ts";
import { toast } from "sonner";
import Icon from "@/components/utils/Icon.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Checkbox } from "@/components/ui/checkbox.tsx";
import {
  Dialog,
  DialogAction,
  DialogCancel,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip.tsx";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { Label } from "@/components/ui/label.tsx";
import { Slider } from "@/components/ui/slider.tsx";
import { ButtonProps } from "@/components/ui/button.tsx";
import { getBooleanMemory, setMemory } from "@/utils/memory.ts";

const geminiThinkingPresets = [
  { label: "off", budget: 0 },
  { label: "low", budget: 1024 },
  { label: "medium", budget: 4096 },
  { label: "high", budget: 8192 },
];

const openAIReasoningSummaryLevels = ["concise", "auto", "detailed"];
const deepSeekReasoningEfforts = ["high", "max"];
const deepSeekProMaxWarningMemoryKey =
  "deepseek_v4_pro_max_reasoning_warning_dismissed";

function formatModelLabel(model: string): string {
  return model.trim().toUpperCase();
}

function getStepPosition(index: number, total: number): string {
  if (total <= 1) return "0%";
  return `${(index / (total - 1)) * 100}%`;
}

type ChatActionProps = {
  style?: React.CSSProperties;
  className?: string;
  text?: string;
  active?: boolean | number;
  show?: boolean;
  children?: React.ReactElement;
} & Omit<ButtonProps, "children">;

export const ChatAction = React.forwardRef<HTMLButtonElement, ChatActionProps>(
  (
    { className, text, children, active, show = true, onClick, ...props },
    ref,
  ) => {
    return (
      <TooltipProvider>
        <Tooltip delayDuration={250}>
          <TooltipTrigger asChild>
            <Button
              ref={ref}
              size={`icon-sm`}
              variant={`ghost`}
              className={cn(
                "group mr-1 transition-all duration-300 hover:bg-muted-foreground/5",
                active && "bg-muted-foreground/10 hover:bg-muted-foreground/20",
                !show && "pointer-events-none invisible opacity-0",
                className,
              )}
              classNameWrapper="shrink-0"
              tapScale={0.9}
              unClickable
              onClick={onClick}
              {...props}
            >
              <Icon
                icon={children}
                className={cn(
                  "h-[1.125rem] w-[1.125rem] shrink-0 stroke-[2] text-unread transition",
                  active && "text-primary",
                )}
              />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{text}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  },
);
ChatAction.displayName = "ChatAction";

export function WebAction() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const web = useSelector(selectWeb);
  const model = useSelector(selectModel);
  const supportModels = useSelector(selectSupportModels);
  const geminiGoogleSearch = useSelector(selectGeminiGoogleSearch);
  const geminiURLContext = useSelector(selectGeminiURLContext);
  const xaiWebSearch = useSelector(selectXAIWebSearch);
  const xaiXSearch = useSelector(selectXAIXSearch);
  const openAIResponsesWebSearch = useSelector(selectOpenAIResponsesWebSearch);
  const openAIReasoningEffort = useSelector(selectOpenAIReasoningEffort);
  const webSearchEnabled = useSelector(infoWebSearchSelector);

  const isGeminiModel = isGeminiModelId(model);
  const isXAIModel = isXAIModelId(model);
  const isOpenAIWebModel = isOpenAIResponsesNativeWebModel(
    supportModels,
    model,
  );
  const openAIModelLabel = formatModelLabel(model);

  const geminiWebEnabled = geminiGoogleSearch || geminiURLContext;
  const xaiSearchEnabled = xaiWebSearch || xaiXSearch;
  const openAIWebEnabled = openAIResponsesWebSearch;

  if (!webSearchEnabled && !isGeminiModel && !isXAIModel && !isOpenAIWebModel) {
    return null;
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <div>
          <ChatAction
            active={
              isGeminiModel
                ? geminiWebEnabled
                : isXAIModel
                ? xaiSearchEnabled
                : isOpenAIWebModel
                ? openAIWebEnabled
                : web
            }
            text={
              isGeminiModel
                ? t("chat.gemini-web")
                : isXAIModel
                ? t("chat.xai-web")
                : isOpenAIWebModel
                ? t("chat.openai-web", { model: openAIModelLabel })
                : t("chat.web")
            }
          >
            <Globe
              className={cn(
                "h-4 w-4 web",
                (isGeminiModel
                  ? geminiWebEnabled
                  : isXAIModel
                  ? xaiSearchEnabled
                  : isOpenAIWebModel
                  ? openAIWebEnabled
                  : web) && "enable",
              )}
            />
          </ChatAction>
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-64 p-3" side="top" align="start">
        <div className="space-y-4">
          {isGeminiModel ? (
            <>
              <div className="flex items-center justify-between">
                <Label
                  htmlFor="gemini-google-search-toggle"
                  className="text-sm"
                >
                  {t("chat.gemini-google-search")}
                </Label>
                <Switch
                  id="gemini-google-search-toggle"
                  checked={geminiGoogleSearch}
                  onCheckedChange={(state) => {
                    dispatch(setGeminiGoogleSearch(state));
                  }}
                />
              </div>

              <div className="flex items-center justify-between">
                <Label htmlFor="gemini-url-context-toggle" className="text-sm">
                  {t("chat.gemini-url-context")}
                </Label>
                <Switch
                  id="gemini-url-context-toggle"
                  checked={geminiURLContext}
                  onCheckedChange={(state) => {
                    dispatch(setGeminiURLContext(state));
                  }}
                />
              </div>

              <div className="rounded-md bg-muted p-2 text-xs">
                <div className="flex items-start">
                  <Icon
                    icon={<Info />}
                    className="h-3 w-3 mr-1 mt-0.5 shrink-0"
                  />
                  {t("chat.gemini-web-enable-tip")}
                </div>
              </div>
            </>
          ) : isXAIModel ? (
            <>
              <div className="flex items-center justify-between">
                <Label htmlFor="xai-web-search-toggle" className="text-sm">
                  {t("chat.xai-web-search")}
                </Label>
                <Switch
                  id="xai-web-search-toggle"
                  checked={xaiWebSearch}
                  onCheckedChange={(state) => {
                    dispatch(setXAIWebSearch(state));
                  }}
                />
              </div>

              <div className="flex items-center justify-between">
                <Label htmlFor="xai-x-search-toggle" className="text-sm">
                  {t("chat.xai-x-search")}
                </Label>
                <Switch
                  id="xai-x-search-toggle"
                  checked={xaiXSearch}
                  onCheckedChange={(state) => {
                    dispatch(setXAIXSearch(state));
                  }}
                />
              </div>

              <div className="rounded-md bg-muted p-2 text-xs">
                <div className="flex items-start">
                  <Icon
                    icon={<Info />}
                    className="h-3 w-3 mr-1 mt-0.5 shrink-0"
                  />
                  {t("chat.xai-web-enable-tip")}
                </div>
              </div>
            </>
          ) : isOpenAIWebModel ? (
            <>
              <div className="flex items-center justify-between">
                <Label htmlFor="openai-web-search-toggle" className="text-sm">
                  {t("chat.openai-web-search")}
                </Label>
                <Switch
                  id="openai-web-search-toggle"
                  checked={openAIResponsesWebSearch}
                  onCheckedChange={(state) => {
                    const capabilities = getOpenAIResponsesCapabilities(
                      supportModels,
                      model,
                    );
                    if (
                      state &&
                      model.trim().toLowerCase() === "gpt-5" &&
                      openAIReasoningEffort === "minimal" &&
                      capabilities.reasoningEfforts.includes("low")
                    ) {
                      dispatch(setOpenAIReasoningEffort("low"));
                    }
                    dispatch(setOpenAIResponsesWebSearch(state));
                  }}
                />
              </div>

              <div className="rounded-md bg-muted p-2 text-xs">
                <div className="flex items-start">
                  <Icon
                    icon={<Info />}
                    className="h-3 w-3 mr-1 mt-0.5 shrink-0"
                  />
                  {t("chat.openai-web-enable-tip")}
                </div>
              </div>
            </>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <Label htmlFor="web-search-toggle" className="text-sm">
                  {t("chat.web-search")}
                </Label>
                <Switch
                  id="web-search-toggle"
                  checked={web}
                  onCheckedChange={() => {
                    toast(t("chat.web-search"), {
                      description: (
                        <div className={`flex flex-col`}>
                          <div
                            className={`flex flex-row items-center flex-wrap`}
                          >
                            <Icon
                              icon={!web ? <Wifi /> : <WifiOff />}
                              className={`h-4 w-4 mr-1 shrink-0`}
                            />
                            {!web
                              ? t("chat.web-enable-toast")
                              : t("chat.web-disable-toast")}
                          </div>
                          <div
                            className={`mt-1.5 flex flex-row items-center rounded-md border scale-80 py-1 px-2`}
                          >
                            <Icon
                              icon={<Info />}
                              className={`h-3 w-3 mr-1 shrink-0`}
                            />
                            {t("chat.web-enable-tip")}
                          </div>
                        </div>
                      ),
                    });

                    dispatch(toggleWeb());
                  }}
                />
              </div>

              <div className="rounded-md bg-muted p-2 text-xs">
                <div className="flex items-center">
                  <Icon icon={<Info />} className="h-3 w-3 mr-1 shrink-0" />
                  {t("chat.web-enable-tip")}
                </div>
              </div>
            </>
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function GeminiThinkingAction() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const model = useSelector(selectModel);
  const geminiThinkingBudget = useSelector(selectGeminiThinkingBudget);

  if (!supportsGeminiThinkingBudgetControl(model)) {
    return null;
  }

  const enabled = geminiThinkingBudget > 0;
  const levelIndex = Math.max(
    1,
    geminiThinkingPresets.findIndex(
      (item) => item.budget === geminiThinkingBudget,
    ),
  );

  return (
    <Popover>
      <PopoverTrigger asChild>
        <div>
          <ChatAction active={enabled} text={t("chat.gemini-thinking")}>
            <Brain className={cn("h-4 w-4", enabled && "enable")} />
          </ChatAction>
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-3" side="top" align="start">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="gemini-thinking-toggle" className="text-sm">
              {t("chat.gemini-thinking-enable")}
            </Label>
            <Switch
              id="gemini-thinking-toggle"
              checked={enabled}
              onCheckedChange={(state) => {
                dispatch(
                  setGeminiThinkingBudget(
                    state ? geminiThinkingPresets[2].budget : 0,
                  ),
                );
              }}
            />
          </div>

          <div className={cn("space-y-2", !enabled && "opacity-50")}>
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>{t("chat.gemini-thinking-depth")}</span>
              <span>
                {enabled
                  ? t(
                      `chat.gemini-thinking-level-${geminiThinkingPresets[levelIndex].label}`,
                    )
                  : t("chat.gemini-thinking-level-off")}
              </span>
            </div>

            <Slider
              disabled={!enabled}
              value={[levelIndex]}
              min={1}
              max={3}
              step={1}
              onValueChange={(value) => {
                const next = geminiThinkingPresets[value[0]];
                next && dispatch(setGeminiThinkingBudget(next.budget));
              }}
            />

            <div className="flex justify-between text-[11px] text-muted-foreground">
              <span>{t("chat.gemini-thinking-level-low")}</span>
              <span>{t("chat.gemini-thinking-level-medium")}</span>
              <span>{t("chat.gemini-thinking-level-high")}</span>
            </div>
          </div>

          <div className="rounded-md bg-muted p-2 text-xs">
            <div className="flex items-start">
              <Icon icon={<Info />} className="h-3 w-3 mr-1 mt-0.5 shrink-0" />
              {t("chat.gemini-thinking-tip")}
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function DeepSeekThinkingAction() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const model = useSelector(selectModel);
  const deepSeekThinkingEnabled = useSelector(selectDeepSeekThinkingEnabled);
  const deepSeekReasoningEffort = useSelector(selectDeepSeekReasoningEffort);
  const [proMaxWarningOpen, setProMaxWarningOpen] = React.useState(false);
  const [proMaxWarningCountdown, setProMaxWarningCountdown] = React.useState(0);
  const [proMaxWarningDismissed, setProMaxWarningDismissed] = React.useState(
    () => getBooleanMemory(deepSeekProMaxWarningMemoryKey, false),
  );
  const [doNotRemindProMax, setDoNotRemindProMax] = React.useState(false);

  const isDeepSeekProModel = model.trim().toLowerCase() === "deepseek-v4-pro";
  const currentEffort = deepSeekReasoningEfforts.includes(
    deepSeekReasoningEffort,
  )
    ? deepSeekReasoningEffort
    : "high";
  const currentEffortIndex = Math.max(
    0,
    deepSeekReasoningEfforts.indexOf(currentEffort),
  );
  const proMaxWarningLocked = proMaxWarningCountdown > 0;

  React.useEffect(() => {
    if (!proMaxWarningOpen || proMaxWarningCountdown <= 0) return;

    const timer = window.setTimeout(() => {
      setProMaxWarningCountdown((value) => Math.max(0, value - 1));
    }, 1000);

    return () => window.clearTimeout(timer);
  }, [proMaxWarningOpen, proMaxWarningCountdown]);

  const applyDeepSeekReasoningEffort = (effort: string) => {
    dispatch(setDeepSeekReasoningEffort(effort));
  };

  const requestDeepSeekReasoningEffort = (effort: string) => {
    if (
      isDeepSeekProModel &&
      effort === "max" &&
      currentEffort !== "max" &&
      !proMaxWarningDismissed
    ) {
      setDoNotRemindProMax(false);
      setProMaxWarningCountdown(5);
      setProMaxWarningOpen(true);
      return;
    }

    applyDeepSeekReasoningEffort(effort);
  };

  const confirmDeepSeekProMaxReasoning = () => {
    if (proMaxWarningLocked) return;

    if (doNotRemindProMax) {
      setMemory(deepSeekProMaxWarningMemoryKey, "true");
      setProMaxWarningDismissed(true);
    }

    applyDeepSeekReasoningEffort("max");
    setProMaxWarningOpen(false);
  };

  const setDeepSeekProMaxWarningOpen = (open: boolean) => {
    setProMaxWarningOpen(open);
  };

  if (!isDeepSeekV4ModelId(model)) {
    return null;
  }

  return (
    <>
      <Popover>
        <PopoverTrigger asChild>
          <div>
            <ChatAction
              active={deepSeekThinkingEnabled}
              text={t("chat.deepseek-thinking")}
            >
              <Brain
                className={cn("h-4 w-4", deepSeekThinkingEnabled && "enable")}
              />
            </ChatAction>
          </div>
        </PopoverTrigger>
        <PopoverContent className="w-72 p-3" side="top" align="start">
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <Label htmlFor="deepseek-thinking-toggle" className="text-sm">
                {t("chat.deepseek-thinking-enable")}
              </Label>
              <Switch
                id="deepseek-thinking-toggle"
                checked={deepSeekThinkingEnabled}
                onCheckedChange={(state) => {
                  dispatch(setDeepSeekThinkingEnabled(state));
                }}
              />
            </div>

            <div
              className={cn(
                "space-y-2",
                !deepSeekThinkingEnabled && "opacity-50",
              )}
            >
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{t("chat.deepseek-thinking-depth")}</span>
                <span>
                  {deepSeekThinkingEnabled
                    ? t(`chat.deepseek-thinking-level-${currentEffort}`)
                    : t("chat.deepseek-thinking-level-off")}
                </span>
              </div>

              <div className="relative grid grid-cols-2 gap-1 overflow-hidden rounded-md border border-black/10 bg-white p-1 dark:border-white/15 dark:bg-black">
                <span
                  className="absolute inset-y-1 left-1 rounded-sm bg-black transition-transform duration-300 ease-out dark:bg-white"
                  style={{
                    width: "calc(50% - 0.375rem)",
                    transform:
                      currentEffortIndex === 0
                        ? "translateX(0)"
                        : "translateX(calc(100% + 0.25rem))",
                  }}
                />
                {deepSeekReasoningEfforts.map((effort) => (
                  <button
                    key={effort}
                    type="button"
                    disabled={!deepSeekThinkingEnabled}
                    onClick={() => requestDeepSeekReasoningEffort(effort)}
                    className={cn(
                      "relative z-10 h-8 rounded-sm text-xs font-medium transition-colors duration-200 disabled:cursor-not-allowed",
                      currentEffort === effort
                        ? "text-white dark:text-black"
                        : "text-black/70 hover:text-black dark:text-white/70 dark:hover:text-white",
                    )}
                  >
                    {t(`chat.deepseek-thinking-level-${effort}`)}
                  </button>
                ))}
              </div>
            </div>

            <div className="rounded-md bg-muted p-2 text-xs">
              <div className="flex items-start">
                <Icon
                  icon={<Info />}
                  className="h-3 w-3 mr-1 mt-0.5 shrink-0"
                />
                {t("chat.deepseek-thinking-tip")}
              </div>
            </div>
          </div>
        </PopoverContent>
      </Popover>

      <Dialog
        open={proMaxWarningOpen}
        onOpenChange={setDeepSeekProMaxWarningOpen}
      >
        <DialogContent className="max-w-md">
          <DialogHeader notTextCentered>
            <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-md bg-black text-white dark:bg-white dark:text-black">
              <TriangleAlert className="h-5 w-5" />
            </div>
            <DialogTitle>{t("chat.deepseek-pro-max-warning-title")}</DialogTitle>
            <DialogDescription>
              {t("chat.deepseek-pro-max-warning-desc")}
            </DialogDescription>
          </DialogHeader>

          <label className="flex cursor-pointer items-center gap-2 rounded-md border p-3 text-sm">
            <Checkbox
              checked={doNotRemindProMax}
              onCheckedChange={(checked) => {
                setDoNotRemindProMax(checked === true);
              }}
            />
            <span>{t("chat.deepseek-pro-max-warning-dont-remind")}</span>
          </label>

          <DialogFooter>
            <DialogCancel>{t("cancel")}</DialogCancel>
            <DialogAction
              disabled={proMaxWarningLocked}
              onClick={confirmDeepSeekProMaxReasoning}
            >
              {proMaxWarningLocked
                ? t("chat.deepseek-pro-max-warning-wait", {
                    count: proMaxWarningCountdown,
                  })
                : t("chat.deepseek-pro-max-warning-confirm")}
            </DialogAction>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function OpenAIReasoningAction() {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const model = useSelector(selectModel);
  const supportModels = useSelector(selectSupportModels);
  const openAIReasoningEffort = useSelector(selectOpenAIReasoningEffort);
  const openAIReasoningSummary = useSelector(selectOpenAIReasoningSummary);
  const openAIResponsesWebSearch = useSelector(selectOpenAIResponsesWebSearch);
  const capabilities = getOpenAIResponsesCapabilities(supportModels, model);

  if (capabilities.reasoningEfforts.length === 0) {
    return null;
  }

  const availableEfforts =
    model.trim().toLowerCase() === "gpt-5" && openAIResponsesWebSearch
      ? capabilities.reasoningEfforts.filter(
          (item) => item !== "minimal" && item !== "none",
        )
      : capabilities.reasoningEfforts.filter((item) => item !== "none");
  const enabled = openAIReasoningEffort !== "none";
  const summaryEnabled = openAIReasoningSummary !== "none";
  const currentSummary = summaryEnabled ? openAIReasoningSummary : "auto";
  const currentSummaryIndex = Math.max(
    0,
    openAIReasoningSummaryLevels.indexOf(currentSummary),
  );
  const modelLabel = formatModelLabel(model);
  const fallbackEffort = availableEfforts.includes("medium")
    ? "medium"
    : availableEfforts[0];
  const currentEffort = enabled
    ? availableEfforts.includes(openAIReasoningEffort)
      ? openAIReasoningEffort
      : fallbackEffort
    : "none";
  const levelIndex = Math.max(0, availableEfforts.indexOf(currentEffort));

  return (
    <Popover>
      <PopoverTrigger asChild>
        <div>
          <ChatAction
            active={enabled}
            text={t("chat.openai-reasoning", { model: modelLabel })}
          >
            <Brain className={cn("h-4 w-4", enabled && "enable")} />
          </ChatAction>
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-72 p-3" side="top" align="start">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="openai-reasoning-toggle" className="text-sm">
              {t("chat.openai-reasoning-enable", { model: modelLabel })}
            </Label>
            <Switch
              id="openai-reasoning-toggle"
              checked={enabled}
              onCheckedChange={(state) => {
                dispatch(
                  setOpenAIReasoningEffort(state ? fallbackEffort : "none"),
                );
              }}
            />
          </div>

          <div className={cn("space-y-2", !enabled && "opacity-50")}>
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span>{t("chat.openai-reasoning-depth")}</span>
              <span>
                {enabled
                  ? t(`chat.openai-reasoning-level-${currentEffort}`)
                  : t("chat.openai-reasoning-level-none")}
              </span>
            </div>

            <Slider
              disabled={!enabled}
              value={[levelIndex]}
              min={0}
              max={Math.max(availableEfforts.length - 1, 0)}
              step={1}
              onValueChange={(value) => {
                const next = availableEfforts[value[0]];
                next && dispatch(setOpenAIReasoningEffort(next));
              }}
            />

            <div className="relative h-4 text-[11px] text-muted-foreground">
              {availableEfforts.map((effort, index) => (
                <span
                  key={effort}
                  className="absolute top-0 -translate-x-1/2 whitespace-nowrap"
                  style={{
                    left: getStepPosition(index, availableEfforts.length),
                  }}
                >
                  {t(`chat.openai-reasoning-level-${effort}`)}
                </span>
              ))}
            </div>
          </div>

          <div className={cn("space-y-2", !enabled && "opacity-50")}>
            <div className="flex items-center justify-between">
              <Label
                htmlFor="openai-reasoning-summary-toggle"
                className="text-sm"
              >
                {t("chat.openai-reasoning-summary-enable")}
              </Label>
              <Switch
                id="openai-reasoning-summary-toggle"
                disabled={!enabled}
                checked={summaryEnabled}
                onCheckedChange={(state) => {
                  dispatch(
                    setOpenAIReasoningSummary(state ? currentSummary : "none"),
                  );
                }}
              />
            </div>

            <div
              className={cn(
                "space-y-2",
                (!enabled || !summaryEnabled) && "opacity-50",
              )}
            >
              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{t("chat.openai-reasoning-summary-detail")}</span>
                <span>
                  {t(`chat.openai-reasoning-summary-level-${currentSummary}`)}
                </span>
              </div>

              <div className="relative grid grid-cols-3 gap-1 overflow-hidden rounded-md border border-black/10 bg-white p-1 dark:border-white/15 dark:bg-black">
                <span
                  className="absolute inset-y-1 left-1 rounded-sm bg-black transition-transform duration-300 ease-out dark:bg-white"
                  style={{
                    width: "calc((100% - 1rem) / 3)",
                    transform:
                      currentSummaryIndex === 0
                        ? "translateX(0)"
                        : `translateX(calc(${currentSummaryIndex * 100}% + ${
                            currentSummaryIndex * 0.25
                          }rem))`,
                  }}
                />
                {openAIReasoningSummaryLevels.map((summary) => (
                  <button
                    key={summary}
                    type="button"
                    disabled={!enabled || !summaryEnabled}
                    onClick={() => dispatch(setOpenAIReasoningSummary(summary))}
                    className={cn(
                      "relative z-10 h-8 rounded-sm text-xs font-medium transition-colors duration-200 disabled:cursor-not-allowed",
                      currentSummary === summary
                        ? "text-white dark:text-black"
                        : "text-black/70 hover:text-black dark:text-white/70 dark:hover:text-white",
                    )}
                  >
                    {t(`chat.openai-reasoning-summary-level-${summary}`)}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="rounded-md bg-muted p-2 text-xs">
            <div className="flex items-start">
              <Icon icon={<Info />} className="h-3 w-3 mr-1 mt-0.5 shrink-0" />
              {t("chat.openai-reasoning-tip", { model: modelLabel })}
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function NewConversationAction() {
  const { t } = useTranslation();
  const messages = useMessages();
  const { toggle } = useConversationActions();

  return (
    <ChatAction
      text={t("new-chat")}
      onClick={async () => messages.length > 0 && (await toggle(-1))}
    >
      <MessageSquarePlus className={`h-4 w-4`} />
    </ChatAction>
  );
}
