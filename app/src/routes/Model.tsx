import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input.tsx";
import {
  Activity,
  ArrowDownToDot,
  ArrowRightLeft,
  ArrowUpFromDot,
  Award,
  Bolt,
  Brain,
  CheckCircle2,
  Clock3,
  Cloud,
  Cpu,
  DollarSign,
  EyeIcon,
  Gem,
  Gauge,
  Github,
  Globe,
  Image,
  Link,
  Search,
  Snail,
  Sparkles,
  Star,
  Tag,
  X,
  Zap,
} from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { splitList } from "@/utils/base.ts";
import type { Model } from "@/api/types.tsx";
import { useDispatch, useSelector } from "react-redux";
import {
  addModelList,
  removeModelList,
  selectModel,
  selectModelList,
  selectSupportModels,
  setModel,
} from "@/store/chat.ts";
import { levelSelector } from "@/store/subscription.ts";
import { selectAuthenticated } from "@/store/auth.ts";
import { docsEndpoint } from "@/conf/env.ts";
import { goAuth } from "@/utils/app.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { includingModelFromPlan } from "@/conf/subscription.tsx";
import { subscriptionDataSelector } from "@/store/globals.ts";
import { getResolvedModelTags, isDrawingModel } from "@/conf/model.ts";
import {
  ChargeBaseProps,
  nonBilling,
  timesBilling,
  tokenBilling,
} from "@/admin/charge.ts";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import router from "@/router.tsx";
import ModelAvatar from "@/components/ModelAvatar.tsx";
import { ToggleGroup } from "@radix-ui/react-toggle-group";
import { marketTags } from "@/admin/market.ts";
import { ToggleGroupItem } from "@/components/ui/toggle-group.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import { Label } from "@/components/ui/label.tsx";
import { toast } from "sonner";
import { motion, AnimatePresence } from "framer-motion";
import Tips from "@/components/Tips";
import Icon from "@/components/utils/Icon";
import {
  getModelUsageStats,
  getModelsUsageStats,
} from "@/api/model-metrics.ts";
import type {
  ModelUsageStats,
  ModelUsageTrendPoint,
} from "@/api/model-metrics.ts";

const tagIcons: { [key: string]: React.ReactNode } = {
  official: <Award />,
  "multi-modal": <EyeIcon />,
  web: <Globe />,
  "high-quality": <Sparkles />,
  "high-price": <DollarSign />,
  "open-source": <Github />,
  "image-generation": <Image />,
  reasoning: <Brain />,
  fast: <Bolt />,
  unstable: <Snail />,
  "high-context": <Cpu />,
  free: <Zap />,
};

const notDisplayTags = ["fast", "unstable", "free"];

type SearchBarProps = {
  text: string;
  onTextChange: (value: string) => void;
  tags: string[];
  onTagsChange: (value: string[]) => void;
  displayPricing: boolean;
  onDisplayPricingChange: (value: boolean) => void;
  show1mPricing: boolean;
  onShow1mPricingChange: (value: boolean) => void;
};

function getTags(model: Model): string[] {
  return getResolvedModelTags(model);
}

function SearchBar({
  text,
  onTextChange,
  tags,
  onTagsChange,
  displayPricing,
  onDisplayPricingChange,
  show1mPricing,
  onShow1mPricingChange,
}: SearchBarProps) {
  const { t } = useTranslation();

  const supportModels = useSelector(selectSupportModels);
  const availableTags = useMemo(
    () =>
      marketTags.filter((tag) =>
        supportModels.some((model) => getTags(model).includes(tag)),
      ),
    [supportModels],
  );

  return (
    <motion.div
      className={`flex flex-col search-bar-wrapper`}
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <div className={`option-bar flex flex-row mb-2 items-center`}>
        <div className={`grow`} />
        <Label>{t("market.show-pricing")}</Label>
        <Switch
          checked={displayPricing}
          onCheckedChange={onDisplayPricingChange}
          className={`ml-1.5 scale-90`}
        />

        {displayPricing && (
          <>
            <Label className={`ml-2`}>K/M</Label>
            <Switch
              checked={show1mPricing}
              onCheckedChange={onShow1mPricingChange}
              className={`ml-1.5 scale-90`}
            />
          </>
        )}
      </div>
      <div className={`search-bar`}>
        <Search size={16} className={`search-icon`} />
        <Input
          placeholder={t("market.search")}
          className={`input-box`}
          value={text}
          onChange={(e) => onTextChange(e.target.value)}
        />
        <X
          size={16}
          className={cn("clear-icon", text.length > 0 && "active")}
          onClick={() => onTextChange("")}
        />
      </div>
      <motion.div
        className={`tags-search-area`}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.2, duration: 0.3 }}
      >
        <ToggleGroup
          type={`multiple`}
          value={tags}
          onValueChange={onTagsChange}
          className={`flex flex-row flex-wrap justify-center`}
        >
          {availableTags.map((tag, index) => (
            <motion.div
              key={index}
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <ToggleGroupItem value={tag} variant={`outline`} size={`col`}>
                {tagIcons[tag] && (
                  <Icon icon={tagIcons[tag]} className={`w-3.5 h-3.5 mr-1`} />
                )}
                {t(`tag.${tag}`)}
              </ToggleGroupItem>
            </motion.div>
          ))}
        </ToggleGroup>
      </motion.div>
    </motion.div>
  );
}

type ModelProps = Omit<
  React.ComponentPropsWithoutRef<typeof motion.div>,
  "onClick" | "style"
> & {
  model: Model;
  className?: string;
  style?: React.CSSProperties;
  forwardRef?: React.Ref<HTMLDivElement>;
  showPricing?: boolean;
  show1mPricing?: boolean;
  metrics?: ModelUsageStats;
  index: number;
  onModelSelect: (model: Model) => void;
};

type PriceColumnProps = ChargeBaseProps & {
  pro: boolean;
  anonymous?: boolean;
  show1mPricing?: boolean;
};

function PriceColumn({
  type,
  input,
  output,
  pro,
  show1mPricing,
}: PriceColumnProps) {
  const { t } = useTranslation();

  const unitName = !show1mPricing ? "1K TOKENS" : "1M TOKENS";
  const unitValue = !show1mPricing ? 1 : 1000;

  const className = cn(
    "flex flex-row text-sm items-center px-2 pr-1 py-1 w-full rounded-md border transition-all",
    pro && "pro",
  );

  const iconClassName =
    "h-4 w-4 scale-110 mr-2 p-0.5 rounded-full bg-primary/5";

  switch (type) {
    case nonBilling:
      return (
        <motion.div
          className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
          whileHover={{ scale: 1.02 }}
          transition={{ type: "spring", stiffness: 300 }}
        >
          <Cloud className={iconClassName} />
          <span className="min-w-0 flex-grow truncate">
            {t("tag.badges.non-billing")}
          </span>
          <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
            FREE
          </span>
        </motion.div>
      );
    case timesBilling:
      return (
        <motion.div
          className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
          whileHover={{ scale: 1.02 }}
          transition={{ type: "spring", stiffness: 300 }}
        >
          <Cloud className={iconClassName} />
          <span className="min-w-0 flex-grow truncate">
            {t("tag.badges.times-billing", { price: output })}
          </span>
          <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
            TIME
          </span>
        </motion.div>
      );
    case tokenBilling: {
      const inputValue = input * unitValue;
      const outputValue = output * unitValue;

      return (
        <div className="grid grid-cols-2 gap-1">
          <motion.div
            className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
            whileHover={{ scale: 1.02 }}
            transition={{ type: "spring", stiffness: 300 }}
          >
            <ArrowUpFromDot className={iconClassName} />
            <span className="min-w-0 flex-grow truncate">{inputValue}</span>
            <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
              {unitName}
            </span>
          </motion.div>
          <motion.div
            className={cn(className, "bg-secondary/5 hover:bg-secondary/10")}
            whileHover={{ scale: 1.02 }}
            transition={{ type: "spring", stiffness: 300 }}
          >
            <ArrowDownToDot className={iconClassName} />
            <span className="min-w-0 flex-grow truncate">{outputValue}</span>
            <span className="text-2xs ml-1 px-1.5 bg-input/40 select-none rounded-sm">
              {unitName}
            </span>
          </motion.div>
        </div>
      );
    }
  }
}

function formatCardLatency(
  metrics: ModelUsageStats | null | undefined,
): string {
  if (!metrics || !hasMetricSamples(metrics) || metrics.success_count === 0) {
    return "--";
  }
  if (metrics.avg_latency < 1) {
    return `${Math.round(metrics.avg_latency * 1000)}ms`;
  }
  return `${metrics.avg_latency.toFixed(metrics.avg_latency >= 10 ? 1 : 2)}s`;
}

function formatCardThroughput(
  metrics: ModelUsageStats | null | undefined,
): string {
  if (!metrics || !hasMetricSamples(metrics) || metrics.success_count === 0) {
    return "--";
  }
  const value =
    metrics.tps >= 100
      ? metrics.tps.toFixed(0)
      : metrics.tps >= 10
      ? metrics.tps.toFixed(1)
      : metrics.tps.toFixed(2);
  return `${value}tps`;
}

function getCardMetricStatus(metrics: ModelUsageStats | undefined) {
  if (!metrics || !hasMetricSamples(metrics)) return "empty";
  if (metrics.availability >= 0.99 && metrics.success_rate >= 0.98) {
    return "good";
  }
  if (metrics.availability >= 0.95 && metrics.success_rate >= 0.9) {
    return "warn";
  }
  return "bad";
}

function ModelCardStatusBars({ status }: { status: string }) {
  return (
    <div className="flex h-4 items-end justify-center gap-0.5 pt-0.5">
      {[0, 1, 2].map((item) => (
        <span
          key={item}
          className={cn(
            "w-1 rounded-full bg-muted transition-colors",
            item === 0 && "h-2",
            item === 1 && "h-3",
            item === 2 && "h-4",
            status === "good" && "bg-emerald-500",
            status === "warn" && item < 2 && "bg-amber-500",
            status === "bad" && item === 2 && "bg-rose-500",
          )}
        />
      ))}
    </div>
  );
}

function ModelCardMetrics({ metrics }: { metrics?: ModelUsageStats }) {
  const { t } = useTranslation();
  const status = getCardMetricStatus(metrics);

  return (
    <div className="absolute right-3 top-3 z-10 hidden w-[7.25rem] select-none md:block">
      <div className="grid grid-cols-[2.7rem_2.75rem_1.25rem] items-start gap-1 text-center">
        <div className="min-w-0">
          <div className="truncate text-[10px] font-medium leading-none text-muted-foreground/80">
            {t("market.metrics-card-latency")}
          </div>
          <div className="mt-1 whitespace-nowrap font-mono text-xs font-semibold leading-none text-foreground/60">
            {formatCardLatency(metrics)}
          </div>
        </div>
        <div className="min-w-0">
          <div className="truncate text-[10px] font-medium leading-none text-muted-foreground/80">
            {t("market.metrics-card-throughput")}
          </div>
          <div className="mt-1 truncate font-mono text-xs font-semibold leading-none text-foreground/60">
            {formatCardThroughput(metrics)}
          </div>
        </div>
        <div className="min-w-0">
          <div className="truncate text-[10px] font-medium leading-none text-muted-foreground/80">
            {t("market.metrics-card-status")}
          </div>
          <ModelCardStatusBars status={status} />
        </div>
      </div>
    </div>
  );
}

function ModelTagIcons({ tags, pro }: { tags: string[]; pro: boolean }) {
  const { t } = useTranslation();

  return (
    <div className="flex min-w-0 flex-wrap items-center gap-1">
      {pro && (
        <Tips
          content={t("tag.badges.plan-included-tip")}
          trigger={
            <Gem className="h-5 w-5 rounded-sm bg-amber-500/20 p-1 text-amber-600" />
          }
        />
      )}
      {tags
        .filter((tag) => !notDisplayTags.includes(tag))
        .map((tag, index) => (
          <Tips
            key={index}
            content={t(`tag.${tag}`)}
            trigger={
              tagIcons[tag] ? (
                <Icon
                  icon={tagIcons[tag]}
                  className={cn("h-5 w-5 rounded-sm bg-primary/5 p-1", {
                    "bg-amber-500/20 text-amber-600": tag === "official",
                    "bg-blue-500/20 text-blue-600": tag === "multi-modal",
                    "bg-green-500/20 text-green-600": tag === "web",
                    "bg-purple-500/20 text-purple-600": tag === "high-quality",
                    "bg-red-500/20 text-red-600": tag === "high-price",
                    "bg-gray-500/20 text-gray-600": tag === "open-source",
                    "bg-indigo-500/20 text-indigo-600":
                      tag === "image-generation",
                    "bg-cyan-500/20 text-cyan-600": tag === "reasoning",
                    "bg-yellow-500/20 text-yellow-600": tag === "fast",
                    "bg-orange-500/20 text-orange-600": tag === "unstable",
                    "bg-teal-500/20 text-teal-600": tag === "high-context",
                    "bg-emerald-500/20 text-emerald-600": tag === "free",
                  })}
                />
              ) : undefined
            }
          />
        ))}
    </div>
  );
}

function ModelItem({
  model,
  className,
  style,
  forwardRef,
  showPricing,
  show1mPricing,
  metrics,
  index,
  onModelSelect,
  ...props
}: ModelProps) {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const list = useSelector(selectModelList);

  const level = useSelector(levelSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);

  const state = useMemo(() => list.includes(model.id), [model, list]);

  const pro = useMemo(() => {
    return includingModelFromPlan(subscriptionData, level, model.id);
  }, [subscriptionData, model, level]);

  const tags = useMemo(
    (): string[] => getTags(model).filter((tag) => tag !== "free"),
    [model],
  );

  return (
    <motion.div
      className={cn("model-item rounded-md", className)}
      style={style}
      ref={forwardRef}
      {...props}
      onClick={() => onModelSelect(model)}
      initial={{ opacity: 0, x: -50 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      whileHover={{ scale: 1.05 }}
    >
      <ModelCardMetrics metrics={metrics} />
      <motion.div
        className={`model-info-wrapper w-full h-max flex flex-row md:pr-[8rem]`}
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: index * 0.1 + 0.2 }}
      >
        <div
          className={`model-info flex flex-row items-center flex-wrap w-full mt-1 ml-1`}
        >
          <motion.div
            className={`model-avatar-wrapper mr-1.5 -translate-x-2 -translate-y-2 flex w-max h-max border rounded-full`}
            whileHover={{ scale: 1.1, rotate: 360 }}
            whileTap={{ scale: 0.9 }}
            initial={{ opacity: 0, rotate: -180 }}
            animate={{ opacity: 1, rotate: 0 }}
            transition={{ duration: 0.5, delay: index * 0.1 + 0.4 }}
          >
            <ModelAvatar className={`model-avatar`} model={model} size={24} />
          </motion.div>
          <div
            className={"flex flex-row items-center model-name mr-2"}
            title={model.name}
          >
            <span className="model-name-text">{model.name}</span>
          </div>
        </div>
      </motion.div>
      <motion.div
        className={`model-description text-sm my-1.5 ml-1 md:pr-[8rem]`}
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: index * 0.1 + 0.5 }}
        title={[model.id, model.description].filter(Boolean).join(" ")}
      >
        <div className="model-id-badge px-1.5 py-0.5 bg-primary/5 border rounded-md mr-1 text-xs text-muted-foreground">
          <Tag className={`w-3 h-3 scale-90 mr-1 inline`} />
          <span className="model-id-text">{model.id}</span>
        </div>
        <span className="model-description-text">{model.description}</span>
      </motion.div>

      <div className={`flex-grow`} />
      {showPricing && model.price && (
        <motion.div
          className={`mt-2.5`}
          initial={{ opacity: 0, y: 0 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: index * 0.1 + 0.6 }}
        >
          <PriceColumn
            type={model.price.type}
            input={model.price.input}
            output={model.price.output}
            pro={pro}
            show1mPricing={show1mPricing}
            anonymous={true}
          />
        </motion.div>
      )}

      <div className="flex flex-row mt-1.5">
        <ModelTagIcons tags={tags} pro={pro} />
        <div className="flex-grow min-w-2" />
        <motion.span
          className={`clickable w-fit h-fit p-1 border hover:border-hover transition-all rounded-md`}
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();

            dispatch(
              state ? removeModelList(model.id) : addModelList(model.id),
            );

            toast.info(t("market.switch-bookmark"), {
              description: (
                <div
                  className={`inline-flex flex-row items-center flex-wrap space-x-1 space-y-1`}
                >
                  <p className={`translate-y-[1px]`}>
                    {state
                      ? t("market.remove-bookmark")
                      : t("market.add-bookmark")}
                  </p>
                  <ModelAvatar size={20} model={model} />
                  <p>{model.name}</p>
                </div>
              ),
            });
          }}
        >
          {state ? (
            <Star className={`w-4 h-4 shrink-0 fill-current text-amber-500`} />
          ) : (
            <Star className={`w-4 h-4 shrink-0 text-muted-foreground`} />
          )}
        </motion.span>
      </div>
    </motion.div>
  );
}

type MarketPlaceProps = {
  search: string;
  showPricing: boolean;
  show1mPricing: boolean;
  onSelect: (model: Model) => void;
};

function MarketPlace({
  search,
  showPricing,
  show1mPricing,
  onSelect,
}: MarketPlaceProps) {
  const { t } = useTranslation();
  const select = useSelector(selectModel);
  const supportModels = useSelector(selectSupportModels);
  const [metricsMap, setMetricsMap] = useState<Record<string, ModelUsageStats>>(
    {},
  );

  const models = useMemo(() => {
    if (search.length === 0) return supportModels;
    // fuzzy search
    const raw = splitList(search.toLowerCase(), [" ", ",", ";", "-"]);
    return supportModels.filter((model) => {
      const name = model.name.toLowerCase();

      const tag = getTags(model);

      const tag_translated_name = tag
        .map((item) => t(`tag.${item}`))
        .join(" ")
        .toLowerCase();
      const id = model.id.toLowerCase();

      return raw.every(
        (item) =>
          name.includes(item) ||
          tag_translated_name.includes(item) ||
          id.includes(item),
      );
    });
  }, [supportModels, search, t]);
  const visibleModelIds = useMemo(
    () => models.map((model) => model.id),
    [models],
  );
  const visibleModelIdsKey = useMemo(
    () => visibleModelIds.join("\n"),
    [visibleModelIds],
  );

  useEffect(() => {
    let ignore = false;
    if (visibleModelIds.length === 0) {
      setMetricsMap({});
      return () => {
        ignore = true;
      };
    }

    const chunks: string[][] = [];
    for (let index = 0; index < visibleModelIds.length; index += 50) {
      chunks.push(visibleModelIds.slice(index, index + 50));
    }

    Promise.all(chunks.map((chunk) => getModelsUsageStats(chunk))).then(
      (results) => {
        if (ignore) return;
        setMetricsMap(Object.assign({}, ...results));
      },
    );

    return () => {
      ignore = true;
    };
  }, [visibleModelIds, visibleModelIdsKey]);

  return (
    <motion.div
      className={`model-list`}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      <AnimatePresence>
        {models.map((model, index) => (
          <ModelItem
            key={index}
            model={model}
            className={cn(select === model.id && "active")}
            showPricing={showPricing}
            show1mPricing={show1mPricing}
            metrics={metricsMap[model.id]}
            index={index}
            onModelSelect={onSelect}
          />
        ))}
      </AnimatePresence>
    </motion.div>
  );
}

function hasMetricSamples(
  metrics: ModelUsageStats | null,
): metrics is ModelUsageStats {
  return !!metrics && metrics.request_count > 0;
}

function formatMetricTPS(metrics: ModelUsageStats | null): string {
  if (!hasMetricSamples(metrics) || metrics.success_count === 0) return "--";
  if (metrics.tps >= 100) return metrics.tps.toFixed(0);
  if (metrics.tps >= 10) return metrics.tps.toFixed(1);
  return metrics.tps.toFixed(2);
}

function formatMetricLatency(metrics: ModelUsageStats | null): string {
  if (!hasMetricSamples(metrics) || metrics.success_count === 0) return "--";
  if (metrics.avg_latency < 1) {
    return `${Math.round(metrics.avg_latency * 1000)} ms`;
  }
  return `${metrics.avg_latency.toFixed(metrics.avg_latency >= 10 ? 1 : 2)} s`;
}

function formatMetricPercent(
  metrics: ModelUsageStats | null,
  value: number | undefined,
): string {
  if (!hasMetricSamples(metrics) || value === undefined) return "--";
  return `${(value * 100).toFixed(1)}%`;
}

function formatMetricCount(
  metrics: ModelUsageStats | null,
  value: number | undefined,
): string {
  if (!hasMetricSamples(metrics) || value === undefined) return "--";
  return `${value}/${metrics.request_count}`;
}

type ModelMetricCardProps = {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub: string;
  loading: boolean;
  iconClassName?: string;
};

function ModelMetricCard({
  icon,
  label,
  value,
  sub,
  loading,
  iconClassName,
}: ModelMetricCardProps) {
  return (
    <div className="min-h-[88px] rounded-lg border bg-muted/20 p-3">
      <div className="mb-2 flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span className="truncate">{label}</span>
        <span
          className={cn(
            "flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-background",
            iconClassName,
          )}
        >
          {icon}
        </span>
      </div>
      {loading ? (
        <div className="space-y-2">
          <div className="h-6 w-16 animate-pulse rounded bg-muted" />
          <div className="h-3 w-20 animate-pulse rounded bg-muted/70" />
        </div>
      ) : (
        <>
          <div className="text-2xl font-bold leading-none text-foreground">
            {value}
          </div>
          <div className="mt-1 truncate text-xs text-muted-foreground">
            {sub}
          </div>
        </>
      )}
    </div>
  );
}

function TrendSparkline({
  points,
  kind,
}: {
  points: ModelUsageTrendPoint[];
  kind: "latency" | "availability";
}) {
  const width = 100;
  const height = 36;
  const pad = 3;
  const values = points.map((point) =>
    point.requests > 0
      ? kind === "latency"
        ? point.avg_latency
        : point.availability
      : null,
  );
  const active = values.filter((value): value is number => value !== null);

  if (active.length === 0) {
    return (
      <div className="h-12 rounded-md border border-dashed border-border bg-muted/20" />
    );
  }

  const minValue = kind === "availability" ? 0 : Math.min(...active);
  const maxValue = kind === "availability" ? 1 : Math.max(...active);
  const span = maxValue - minValue;
  const normalize = (value: number) =>
    span === 0 ? 0.5 : (value - minValue) / span;
  const pointText = values
    .map((value, index) => {
      const x =
        points.length <= 1 ? width / 2 : (index / (points.length - 1)) * width;
      const y =
        height - pad - normalize(value ?? minValue) * (height - pad * 2);
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(" ");

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      className="h-12 w-full overflow-visible"
    >
      <path
        d={`M 0 ${height - pad} H ${width}`}
        className="stroke-muted"
        strokeWidth="1"
        fill="none"
      />
      <polyline
        points={pointText}
        className="stroke-current"
        strokeWidth="2.4"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  );
}

function ModelMetricsPanel({
  metrics,
  loading,
}: {
  metrics: ModelUsageStats | null;
  loading: boolean;
}) {
  const { t } = useTranslation();
  const noData = t("market.metrics-no-data");
  const hours = metrics?.window_hours ?? 24;
  const availableCount = metrics
    ? metrics.request_count - metrics.availability_failures
    : undefined;

  return (
    <motion.div
      className="space-y-3"
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.14, duration: 0.18 }}
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
          <Activity className="h-4 w-4 text-primary" />
          {t("market.metrics-title")}
        </div>
        <span className="shrink-0 text-xs text-muted-foreground">
          {t("market.metrics-window", { hours })}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-2.5">
        <ModelMetricCard
          icon={<Gauge className="h-3.5 w-3.5" />}
          label={t("market.metrics-tps")}
          value={formatMetricTPS(metrics)}
          sub={
            hasMetricSamples(metrics) ? t("market.metrics-tps-unit") : noData
          }
          loading={loading}
          iconClassName="text-sky-600 dark:text-sky-400"
        />
        <ModelMetricCard
          icon={<Clock3 className="h-3.5 w-3.5" />}
          label={t("market.metrics-avg-latency")}
          value={formatMetricLatency(metrics)}
          sub={
            hasMetricSamples(metrics)
              ? t("market.metrics-window", { hours })
              : noData
          }
          loading={loading}
          iconClassName="text-amber-600 dark:text-amber-400"
        />
        <ModelMetricCard
          icon={<CheckCircle2 className="h-3.5 w-3.5" />}
          label={t("market.metrics-success-rate")}
          value={formatMetricPercent(metrics, metrics?.success_rate)}
          sub={formatMetricCount(metrics, metrics?.success_count)}
          loading={loading}
          iconClassName="text-emerald-600 dark:text-emerald-400"
        />
        <ModelMetricCard
          icon={<Activity className="h-3.5 w-3.5" />}
          label={t("market.metrics-availability")}
          value={formatMetricPercent(metrics, metrics?.availability)}
          sub={formatMetricCount(metrics, availableCount)}
          loading={loading}
          iconClassName="text-indigo-600 dark:text-indigo-400"
        />
      </div>

      <div className="space-y-3 rounded-lg border bg-muted/15 p-3">
        <div>
          <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
            <span>{t("market.metrics-latency-trend")}</span>
            <span>{formatMetricLatency(metrics)}</span>
          </div>
          <div className="text-sky-600 dark:text-sky-400">
            <TrendSparkline
              points={metrics?.latency_trend ?? []}
              kind="latency"
            />
          </div>
        </div>
        <div>
          <div className="mb-1 flex items-center justify-between text-xs text-muted-foreground">
            <span>{t("market.metrics-availability-trend")}</span>
            <span>{formatMetricPercent(metrics, metrics?.availability)}</span>
          </div>
          <div className="text-emerald-600 dark:text-emerald-400">
            <TrendSparkline
              points={metrics?.availability_trend ?? []}
              kind="availability"
            />
          </div>
        </div>
      </div>
    </motion.div>
  );
}

function ModelDetailPanel({
  model,
  onClose,
  show1mPricing,
}: {
  model: Model;
  onClose: () => void;
  show1mPricing: boolean;
}) {
  const { t } = useTranslation();
  const dispatch = useDispatch();
  const list = useSelector(selectModelList);
  const auth = useSelector(selectAuthenticated);
  const level = useSelector(levelSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);
  const [metrics, setMetrics] = useState<ModelUsageStats | null>(null);
  const [metricsLoading, setMetricsLoading] = useState<boolean>(true);

  const state = useMemo(() => list.includes(model.id), [model, list]);
  const pro = useMemo(
    () => includingModelFromPlan(subscriptionData, level, model.id),
    [subscriptionData, model, level],
  );
  const tags = useMemo(() => getTags(model), [model]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [onClose]);

  useEffect(() => {
    let ignore = false;
    setMetricsLoading(true);
    setMetrics(null);

    getModelUsageStats(model.id)
      .then((data) => {
        if (!ignore) setMetrics(data);
      })
      .finally(() => {
        if (!ignore) setMetricsLoading(false);
      });

    return () => {
      ignore = true;
    };
  }, [model.id]);

  const handleUse = () => {
    if (!auth && model.auth) {
      toast(t("login-require"), {
        action: { label: t("login"), onClick: goAuth },
      });
      return;
    }

    if (isDrawingModel(model)) {
      router.navigate(`/drawing?model=${encodeURIComponent(model.id)}`);
      onClose();
      return;
    }

    dispatch(setModel(model.id));
    router.navigate("/");
    onClose();
  };

  const unitLabel = show1mPricing ? "1M" : "1K";
  const unitValue = show1mPricing ? 1000 : 1;
  const quickEnter = 0.18;

  return (
    <motion.div
      className="fixed inset-0 z-40"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.12 }}
    >
      <div
        className="absolute inset-0 bg-black/30 backdrop-blur-[2px]"
        onClick={onClose}
      />
      <motion.div
        className="absolute right-0 top-0 h-full w-[420px] max-w-[92vw] bg-background flex flex-col shadow-2xl overflow-hidden"
        initial={{ x: "100%" }}
        animate={{ x: 0 }}
        exit={{ x: "100%" }}
        transition={{ type: "tween", duration: 0.24, ease: "easeOut" }}
      >
        {/* ── Hero header ── */}
        <div className="relative shrink-0 overflow-hidden">
          {/* gradient mesh background */}
          <div className="absolute inset-0 bg-gradient-to-br from-primary/8 via-primary/4 to-transparent" />
          <div className="absolute -top-8 -right-8 w-40 h-40 rounded-full bg-primary/6 blur-2xl" />
          <div className="absolute -bottom-4 -left-4 w-28 h-28 rounded-full bg-primary/4 blur-xl" />

          {/* close button */}
          <motion.button
            className="absolute top-4 right-4 z-10 p-1.5 rounded-full bg-background/60 backdrop-blur-sm text-muted-foreground hover:text-foreground hover:bg-background/90 transition-colors border border-border/40"
            whileHover={{ scale: 1.08 }}
            whileTap={{ scale: 0.92 }}
            onClick={onClose}
          >
            <X className="w-3.5 h-3.5" />
          </motion.button>

          <div className="relative flex flex-col items-center pt-10 pb-6 px-6">
            {/* avatar with animated ring */}
            <motion.div
              className="relative mb-4"
              initial={{ scale: 0.6, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              transition={{
                type: "spring",
                damping: 20,
                stiffness: 260,
                delay: 0.03,
              }}
            >
              <motion.div
                className="absolute inset-0 rounded-full border-2 border-primary/30"
                animate={{ scale: [1, 1.18, 1], opacity: [0.6, 0, 0.6] }}
                transition={{
                  duration: 2.4,
                  repeat: Infinity,
                  ease: "easeInOut",
                }}
              />
              <div className="relative z-10 rounded-full border-2 border-background shadow-lg overflow-hidden">
                <ModelAvatar model={model} size={64} />
              </div>
              {pro && (
                <div className="absolute -bottom-1 -right-1 z-20 w-5 h-5 rounded-full bg-amber-500 flex items-center justify-center shadow-md">
                  <Gem className="w-2.5 h-2.5 text-white" />
                </div>
              )}
            </motion.div>

            {/* name */}
            <motion.h2
              className="text-xl font-bold text-foreground text-center mb-1"
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.06, duration: quickEnter }}
            >
              {model.name}
            </motion.h2>

            {/* model id */}
            <motion.div
              className="flex items-center gap-1 px-2 py-0.5 rounded-full bg-primary/8 border border-primary/15 text-xs text-muted-foreground"
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.09, duration: quickEnter }}
            >
              <Tag className="w-2.5 h-2.5" />
              <span className="truncate max-w-[240px] font-mono">
                {model.id}
              </span>
            </motion.div>

            {/* tags row */}
            {tags.filter((tag) => tagIcons[tag]).length > 0 && (
              <motion.div
                className="flex flex-wrap justify-center gap-1.5 mt-3"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.12, duration: quickEnter }}
              >
                {tags
                  .filter((tag) => tagIcons[tag])
                  .map((tag, idx) => (
                    <motion.span
                      key={idx}
                      className={cn(
                        "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium",
                        {
                          "text-amber-600 dark:text-amber-400 bg-amber-100/80 dark:bg-amber-900/30":
                            tag === "official",
                          "text-blue-600 dark:text-blue-400 bg-blue-100/80 dark:bg-blue-900/30":
                            tag === "multi-modal",
                          "text-green-600 dark:text-green-400 bg-green-100/80 dark:bg-green-900/30":
                            tag === "web",
                          "text-purple-600 dark:text-purple-400 bg-purple-100/80 dark:bg-purple-900/30":
                            tag === "high-quality",
                          "text-red-600 dark:text-red-400 bg-red-100/80 dark:bg-red-900/30":
                            tag === "high-price",
                          "text-muted-foreground bg-muted/60":
                            tag === "open-source",
                          "text-indigo-600 dark:text-indigo-400 bg-indigo-100/80 dark:bg-indigo-900/30":
                            tag === "image-generation",
                          "text-cyan-600 dark:text-cyan-400 bg-cyan-100/80 dark:bg-cyan-900/30":
                            tag === "reasoning",
                          "text-yellow-600 dark:text-yellow-400 bg-yellow-100/80 dark:bg-yellow-900/30":
                            tag === "fast",
                          "text-orange-600 dark:text-orange-400 bg-orange-100/80 dark:bg-orange-900/30":
                            tag === "unstable",
                          "text-teal-600 dark:text-teal-400 bg-teal-100/80 dark:bg-teal-900/30":
                            tag === "high-context",
                          "text-emerald-600 dark:text-emerald-400 bg-emerald-100/80 dark:bg-emerald-900/30":
                            tag === "free",
                        },
                      )}
                      initial={{ opacity: 0, scale: 0.75 }}
                      animate={{ opacity: 1, scale: 1 }}
                      transition={{ delay: 0.14 + idx * 0.03, duration: 0.16 }}
                    >
                      <Icon icon={tagIcons[tag]} className="w-2.5 h-2.5" />
                      {t(`tag.${tag}`)}
                    </motion.span>
                  ))}
              </motion.div>
            )}
          </div>
        </div>

        {/* ── Scrollable body ── */}
        <ScrollArea className="flex-grow">
          <div className="px-5 py-4 space-y-4">
            {/* description */}
            {model.description && (
              <motion.p
                className="text-sm text-muted-foreground leading-relaxed"
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.12, duration: quickEnter }}
              >
                {model.description}
              </motion.p>
            )}

            <ModelMetricsPanel metrics={metrics} loading={metricsLoading} />

            {/* pricing cards */}
            {model.price && model.price.type === "token-billing" && (
              <motion.div
                className="grid grid-cols-2 gap-3"
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.16, duration: quickEnter }}
              >
                <div className="rounded-xl border bg-gradient-to-br from-blue-50/60 dark:from-blue-950/30 to-transparent p-3.5">
                  <div className="flex items-center gap-1.5 text-blue-600 dark:text-blue-400 mb-2">
                    <ArrowUpFromDot className="w-3.5 h-3.5" />
                    <span className="text-xs font-medium">{t("input")}</span>
                  </div>
                  <p className="text-2xl font-bold text-foreground leading-none">
                    {parseFloat((model.price.input * unitValue).toPrecision(8))}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {t("quota")} / {unitLabel} tokens
                  </p>
                </div>
                <div className="rounded-xl border bg-gradient-to-br from-violet-50/60 dark:from-violet-950/30 to-transparent p-3.5">
                  <div className="flex items-center gap-1.5 text-violet-600 dark:text-violet-400 mb-2">
                    <ArrowDownToDot className="w-3.5 h-3.5" />
                    <span className="text-xs font-medium">{t("output")}</span>
                  </div>
                  <p className="text-2xl font-bold text-foreground leading-none">
                    {parseFloat(
                      (model.price.output * unitValue).toPrecision(8),
                    )}
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    {t("quota")} / {unitLabel} tokens
                  </p>
                </div>
              </motion.div>
            )}

            {model.price && model.price.type !== "token-billing" && (
              <motion.div
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.16, duration: quickEnter }}
              >
                <PriceColumn
                  type={model.price.type}
                  input={model.price.input}
                  output={model.price.output}
                  pro={pro}
                  show1mPricing={show1mPricing}
                  anonymous
                />
              </motion.div>
            )}
          </div>
        </ScrollArea>

        {/* ── Footer ── */}
        <motion.div
          className="shrink-0 flex items-center gap-2 px-5 py-4 bg-background"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.14, duration: quickEnter }}
        >
          <motion.button
            className={cn(
              "shrink-0 w-10 h-10 rounded-xl border flex items-center justify-center transition-colors",
              state
                ? "text-amber-500 bg-amber-50 dark:bg-amber-950/50 border-amber-200 dark:border-amber-800"
                : "text-muted-foreground border-input hover:bg-accent",
            )}
            whileHover={{ scale: 1.08 }}
            whileTap={{ scale: 0.92 }}
            onClick={() =>
              dispatch(
                state ? removeModelList(model.id) : addModelList(model.id),
              )
            }
          >
            <Star className={cn("w-4 h-4", state && "fill-current")} />
          </motion.button>
          <motion.button
            className="flex-grow flex items-center justify-center gap-2 h-10 px-4 rounded-xl bg-primary text-primary-foreground text-sm font-semibold shadow-sm"
            whileHover={{
              scale: 1.02,
              boxShadow: "0 4px 16px rgb(0 0 0 / 0.15)",
            }}
            whileTap={{ scale: 0.98 }}
            onClick={handleUse}
          >
            <ArrowRightLeft className="w-4 h-4" />
            {t("market.switch-model")}
          </motion.button>
        </motion.div>
      </motion.div>
    </motion.div>
  );
}

function MarketHeader() {
  const { t } = useTranslation();

  return (
    <motion.div
      className={`market-header`}
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <div
        className={`title select-none text-center text-primary font-bold flex flex-row items-center justify-center`}
      >
        <motion.div
          className={`header-bar`}
          initial={{ width: 0 }}
          animate={{ width: "0.75rem" }}
          transition={{ duration: 0.5, delay: 0.2 }}
        />
        {t("market.explore")}
        <motion.div
          className={`header-bar reverse`}
          initial={{ width: 0 }}
          animate={{ width: "0.75rem" }}
          transition={{ duration: 0.5, delay: 0.2 }}
        />
      </div>
    </motion.div>
  );
}

function MarketFooter() {
  const { t } = useTranslation();

  return (
    <motion.div
      className={`market-footer`}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <motion.a
        href={docsEndpoint}
        target={`_blank`}
        rel={`noopener noreferrer`}
        whileHover={{ scale: 1.05 }}
        whileTap={{ scale: 0.95 }}
      >
        <Link size={14} className={`mr-1`} />
        {t("pricing")}
      </motion.a>
    </motion.div>
  );
}

function Model() {
  const { t } = useTranslation();
  const [displayPricing, setDisplayPricing] = useState<boolean>(true);
  const [show1mPricing, setShow1mPricing] = useState<boolean>(false);
  const [searchText, setSearchText] = useState<string>("");
  const [searchTags, setSearchTags] = useState<string[]>([]);
  const [selectedModel, setSelectedModel] = useState<Model | null>(null);

  const search = useMemo(() => {
    return [
      searchText,
      ...searchTags.filter((tag) => tag !== "").map((v) => t(`tag.${v}`)),
    ].join(" ");
  }, [searchText, searchTags, t]);

  return (
    <>
      <ScrollArea className={`model-market`}>
        <motion.div
          className={`market-wrapper`}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5 }}
        >
          <motion.div
            className="absolute inset-0 overflow-hidden pointer-events-none"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 1 }}
          >
            {[...Array(50)].map((_, i) => (
              <motion.div
                key={i}
                className="absolute bg-primary/10 rounded-full"
                style={{
                  width: Math.random() * 4 + 1 + "px",
                  height: Math.random() * 4 + 1 + "px",
                  top: Math.random() * 100 + "%",
                  left: Math.random() * 100 + "%",
                }}
                animate={{
                  y: [0, Math.random() * 100 - 50],
                  x: [0, Math.random() * 100 - 50],
                  opacity: [0.7, 0],
                }}
                transition={{
                  duration: Math.random() * 10 + 10,
                  repeat: Infinity,
                  ease: "linear",
                }}
              />
            ))}
          </motion.div>
          <MarketHeader />
          <SearchBar
            text={searchText}
            onTextChange={setSearchText}
            tags={searchTags}
            onTagsChange={setSearchTags}
            displayPricing={displayPricing}
            onDisplayPricingChange={setDisplayPricing}
            show1mPricing={show1mPricing}
            onShow1mPricingChange={setShow1mPricing}
          />
          <MarketPlace
            search={search}
            showPricing={displayPricing}
            show1mPricing={show1mPricing}
            onSelect={setSelectedModel}
          />
          <MarketFooter />
        </motion.div>
      </ScrollArea>
      <AnimatePresence>
        {selectedModel && (
          <ModelDetailPanel
            key="model-detail"
            model={selectedModel}
            onClose={() => setSelectedModel(null)}
            show1mPricing={show1mPricing}
          />
        )}
      </AnimatePresence>
    </>
  );
}

export default Model;
