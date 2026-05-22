import "@/assets/pages/settings.less";
import { useTranslation } from "react-i18next";
import { useDispatch, useSelector } from "react-redux";
import * as settings from "@/store/settings.ts";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import { Checkbox } from "@/components/ui/checkbox.tsx";
import { useEffect, useState } from "react";
import { getMemoryPerformance } from "@/utils/app.ts";
import { version } from "@/conf/bootstrap.ts";
import { NumberInput } from "@/components/ui/number-input.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { langsProps, setLanguage } from "@/i18n.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { Slider } from "@/components/ui/slider.tsx";
import Tips from "@/components/Tips.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import Github from "@/components/ui/icons/Github.tsx";
import { useDeeptrain } from "@/conf/env.ts";
import ThemeToggle from "@/components/ThemeProvider.tsx";
import {
  getLatestAvailableRelease,
  isPreviewVersion,
  type GitHubRelease,
} from "@/utils/releases.ts";

function SettingsDialog() {
  const { t, i18n } = useTranslation();
  const dispatch = useDispatch();

  const open = useSelector(settings.dialogSelector);

  const align = useSelector(settings.alignSelector);
  const hideToolbar = useSelector(settings.hideToolbarSelector);
  const hideToolbarText = useSelector(settings.hideToolbarTextSelector);
  const collapseThinking = useSelector(settings.collapseThinkingSelector);
  const context = useSelector(settings.contextSelector);
  const sender = useSelector(settings.senderSelector);
  const history = useSelector(settings.historySelector);

  const temperature = useSelector(settings.temperatureSelector);
  const maxTokens = useSelector(settings.maxTokensSelector);
  const topP = useSelector(settings.topPSelector);
  const topK = useSelector(settings.topKSelector);
  const presencePenalty = useSelector(settings.presencePenaltySelector);
  const frequencyPenalty = useSelector(settings.frequencyPenaltySelector);
  const repetitionPenalty = useSelector(settings.repetitionPenaltySelector);

  const [memorySize, setMemorySize] = useState(getMemoryPerformance());
  const [latestRelease, setLatestRelease] = useState<GitHubRelease | null>(
    null,
  );

  const previewVersion = isPreviewVersion(version);

  useEffect(() => {
    const interval = setInterval(() => {
      setMemorySize(getMemoryPerformance());
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }

    let cancelled = false;

    getLatestAvailableRelease(version)
      .then((release) => {
        if (!cancelled) {
          setLatestRelease(release);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setLatestRelease(null);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [open]);

  return (
    <Dialog
      open={open}
      onOpenChange={(open) => dispatch(settings.setDialog(open))}
    >
      <DialogContent className={`flex-dialog settings-dialog`} couldFullScreen>
        <DialogHeader>
          <DialogTitle>{t("settings.title")}</DialogTitle>
          <DialogDescription asChild>
            <div className={`settings-container`}>
              <div className={`settings-wrapper`}>
                <div className={`settings-segment`}>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.version")}</div>
                    <div className={`grow`} />
                    <div className={`value`}>
                      v{version}
                      <Badge className={`ml-1`}>
                        {t(
                          previewVersion
                            ? "settings.preview"
                            : "settings.stable",
                        )}
                      </Badge>
                      {latestRelease && (
                        <a
                          className={`ml-2 text-xs text-primary underline-offset-2 hover:underline`}
                          href={latestRelease.html_url}
                          rel="noreferrer"
                          target="_blank"
                        >
                          {t("settings.update-available", {
                            version: latestRelease.tag_name,
                          })}
                        </a>
                      )}
                    </div>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.theme")}</div>
                    <div className={`grow`} />
                    <div className={`value`}>
                      <ThemeToggle />
                    </div>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.language")}</div>
                    <div className={`grow`} />
                    <div className={`value`}>
                      <Select
                        value={i18n.language}
                        onValueChange={(value: string) =>
                          setLanguage(i18n, value)
                        }
                      >
                        <SelectTrigger className={`select`}>
                          <SelectValue
                            placeholder={langsProps[i18n.language]}
                          />
                        </SelectTrigger>
                        <SelectContent>
                          {Object.entries(langsProps).map(
                            ([key, value], idx) => (
                              <SelectItem key={idx} value={key}>
                                {value}
                              </SelectItem>
                            ),
                          )}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                </div>
                <div className={`settings-segment`}>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.sender")}</div>
                    <div className={`grow`} />
                    <div className={`value`}>
                      <Select
                        value={sender ? "true" : "false"}
                        onValueChange={(value: string) =>
                          dispatch(settings.setSender(value === "true"))
                        }
                      >
                        <SelectTrigger className={`select`}>
                          <SelectValue
                            placeholder={settings.sendKeys[sender ? 1 : 0]}
                          />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value={"false"}>
                            {settings.sendKeys[0]}
                          </SelectItem>
                          <SelectItem value={"true"}>
                            {settings.sendKeys[1]}
                          </SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.align")}</div>
                    <div className={`grow`} />
                    <Checkbox
                      className={`value`}
                      checked={align}
                      onCheckedChange={(state: boolean) => {
                        dispatch(settings.setAlign(state));
                      }}
                    />
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.hide-toolbar")}</div>
                    <div className={`grow`} />
                    <Checkbox
                      className={`value`}
                      checked={hideToolbar}
                      onCheckedChange={(state: boolean) => {
                        dispatch(settings.setHideToolbar(state));
                      }}
                    />
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.hide-toolbar-text")}
                    </div>
                    <div className={`grow`} />
                    <Checkbox
                      className={`value`}
                      checked={hideToolbarText}
                      onCheckedChange={(state: boolean) => {
                        dispatch(settings.setHideToolbarText(state));
                      }}
                    />
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.collapse-thinking")}
                    </div>
                    <div className={`grow`} />
                    <Checkbox
                      className={`value`}
                      checked={collapseThinking}
                      onCheckedChange={(state: boolean) => {
                        dispatch(settings.setCollapseThinking(state));
                      }}
                    />
                  </div>
                </div>
                <div className={`settings-segment`}>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.context")}</div>
                    <div className={`grow`} />
                    <Checkbox
                      className={`value`}
                      checked={context}
                      onCheckedChange={(state: boolean) => {
                        dispatch(settings.setContext(state));
                      }}
                    />
                  </div>
                  {context && (
                    <div className={`item`}>
                      <div className={`name`}>{t("settings.history")}</div>
                      <div className={`grow`} />
                      <Slider
                        value={[history]}
                        min={settings.minHistoryContext}
                        max={settings.maxHistoryContext}
                        step={1}
                        className={`value ml-2 max-w-[10rem] mr-2`}
                        classNameThumb={`h-4 w-4`}
                        onValueChange={(value: number[]) => {
                          dispatch(settings.setHistory(value[0]));
                        }}
                      />
                      <p className={`slider-value`}>{history}</p>
                    </div>
                  )}
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.max-tokens")}
                      <Tips content={t("settings.max-tokens-tip")} />
                    </div>
                    <div className={`grow`} />
                    <NumberInput
                      className={`value large-value`}
                      value={maxTokens}
                      acceptNaN={false}
                      min={0}
                      max={100000}
                      onValueChange={(value: number) => {
                        dispatch(settings.setMaxTokens(value));
                      }}
                    />
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.temperature")}
                      <Tips content={t("settings.temperature-tip")} />
                    </div>
                    <div className={`grow`} />
                    <Slider
                      value={[temperature * 10]}
                      min={0}
                      max={10}
                      step={1}
                      className={`value ml-2 max-w-[10rem] mr-2`}
                      classNameThumb={`h-4 w-4`}
                      onValueChange={(value: number[]) => {
                        dispatch(settings.setTemperature(value[0] / 10));
                      }}
                    />
                    <p className={`slider-value`}>{temperature.toFixed(1)}</p>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.presence-penalty")}
                      <Tips content={t("settings.presence-penalty-tip")} />
                    </div>
                    <div className={`grow`} />
                    <Slider
                      value={[presencePenalty * 10]}
                      min={-20}
                      max={20}
                      step={1}
                      className={`value ml-2 max-w-[10rem] mr-2`}
                      classNameThumb={`h-4 w-4`}
                      onValueChange={(value: number[]) => {
                        dispatch(settings.setPresencePenalty(value[0] / 10));
                      }}
                    />
                    <p className={`slider-value`}>
                      {presencePenalty.toFixed(1)}
                    </p>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.frequency-penalty")}
                      <Tips content={t("settings.frequency-penalty-tip")} />
                    </div>
                    <div className={`grow`} />
                    <Slider
                      value={[frequencyPenalty * 10]}
                      min={-20}
                      max={20}
                      step={1}
                      className={`value ml-2 max-w-[10rem] mr-2`}
                      classNameThumb={`h-4 w-4`}
                      onValueChange={(value: number[]) => {
                        dispatch(settings.setFrequencyPenalty(value[0] / 10));
                      }}
                    />
                    <p className={`slider-value`}>
                      {frequencyPenalty.toFixed(1)}
                    </p>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.repetition-penalty")}
                      <Tips content={t("settings.repetition-penalty-tip")} />
                    </div>
                    <div className={`grow`} />
                    <Slider
                      value={[repetitionPenalty * 10]}
                      min={0}
                      max={20}
                      step={1}
                      className={`value ml-2 max-w-[10rem] mr-2`}
                      classNameThumb={`h-4 w-4`}
                      onValueChange={(value: number[]) => {
                        dispatch(settings.setRepetitionPenalty(value[0] / 10));
                      }}
                    />
                    <p className={`slider-value`}>
                      {repetitionPenalty.toFixed(1)}
                    </p>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.top-p")}
                      <Tips content={t("settings.top-p-tip")} />
                    </div>
                    <div className={`grow`} />
                    <Slider
                      value={[topP * 10]}
                      min={0}
                      max={10}
                      step={1}
                      className={`value ml-2 max-w-[10rem] mr-2`}
                      classNameThumb={`h-4 w-4`}
                      onValueChange={(value: number[]) => {
                        dispatch(settings.setTopP(value[0] / 10));
                      }}
                    />
                    <p className={`slider-value`}>{topP.toFixed(1)}</p>
                  </div>
                  <div className={`item`}>
                    <div className={`name`}>
                      {t("settings.top-k")}
                      <Tips content={t("settings.top-k-tip")} />
                    </div>
                    <div className={`grow`} />
                    <Slider
                      value={[topK]}
                      min={0}
                      max={20}
                      step={1}
                      className={`value ml-2 max-w-[10rem] mr-2`}
                      classNameThumb={`h-4 w-4`}
                      onValueChange={(value: number[]) => {
                        dispatch(settings.setTopK(value[0]));
                      }}
                    />
                    <p className={`slider-value`}>{topK.toFixed()}</p>
                  </div>
                </div>
                <div className={`settings-segment`}>
                  <div className={`item`}>
                    <div className={`name`}>{t("settings.reset-settings")}</div>
                    <div className={`grow`} />
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          size={`sm`}
                          variant={`destructive`}
                          className={`set-action`}
                        >
                          {t("reset")}
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>
                            {t("settings.reset-settings")}
                          </AlertDialogTitle>
                          <AlertDialogDescription>
                            {t("settings.reset-settings-description")}
                          </AlertDialogDescription>
                          <AlertDialogFooter>
                            <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
                            <AlertDialogAction
                              onClick={() => {
                                dispatch(settings.resetSettings());
                              }}
                            >
                              {t("confirm")}
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogHeader>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
              </div>
              <div className={`grow`} />
              <div className={`info-box`}>
                <p>
                  {t("settings.memory")}
                  &nbsp;
                  {!isNaN(memorySize)
                    ? memorySize.toFixed(2) + " MB"
                    : t("unknown")}
                </p>
                <a
                  className={cn(
                    "flex flex-row items-center",
                    !useDeeptrain && "hidden",
                  )}
                  href={`https://github.com/coaidev/coai`}
                >
                  <Github className={`inline-block h-4 w-4 mr-1.5`} />
                  Prism v{version}
                </a>
              </div>
            </div>
          </DialogDescription>
        </DialogHeader>
      </DialogContent>
    </Dialog>
  );
}

export default SettingsDialog;
