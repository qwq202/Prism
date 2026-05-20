import "@/assets/pages/quota.less";
import "@/assets/pages/subscription.less";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { useTranslation } from "react-i18next";
import {
  BadgeCheck,
  BadgeMinus,
  CalendarClock,
  Coins,
  Crown,
  ExternalLink,
  Star,
  Rocket,
  Zap,
} from "lucide-react";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { docsEndpoint } from "@/conf/env.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { useMemo, useState } from "react";
import { useSelector } from "react-redux";
import { useCurrency } from "@/store/info.ts";
import {
  expiredSelector,
  isSubscribedSelector,
  levelSelector,
  usageSelector,
  refreshSelector,
} from "@/store/subscription.ts";
import { subscriptionDataSelector } from "@/store/globals.ts";
import { motion } from "framer-motion";
import { useInView } from "react-intersection-observer";
import {
  getPlan,
  getPlanName,
  hasPlanPointPool,
  isPlanSellable,
} from "@/conf/subscription.tsx";
import { Upgrade } from "@/components/home/subscription/UpgradePlan.tsx";
import SubscriptionUsage from "@/components/home/subscription/SubscriptionUsage.tsx";
import WalletQuotaBox from "@/routes/wallet/WalletQuotaBox.tsx";
import ModelAvatar from "@/components/ModelAvatar";
import Icon from "@/components/utils/Icon";
import Tips from "@/components/Tips";

type SubscriptionUsageValue = {
  used: number;
  total: number;
  unit?: "times" | "points";
  reset_interval?: number;
  reset_at?: string;
};

const pointResetInterval = 5 * 60 * 60;

function toSubscriptionUsage(
  value: unknown,
  fallbackTotal: number,
): SubscriptionUsageValue | null {
  if (typeof value === "number") {
    return {
      used: value,
      total: fallbackTotal,
    };
  }

  if (
    value &&
    typeof value === "object" &&
    "used" in value &&
    "total" in value
  ) {
    const usage = value as Record<string, unknown>;
    if (typeof usage.used === "number" && typeof usage.total === "number") {
      return {
        used: usage.used,
        total: usage.total,
        unit: usage.unit === "points" ? "points" : "times",
        reset_interval:
          typeof usage.reset_interval === "number"
            ? usage.reset_interval
            : undefined,
        reset_at:
          typeof usage.reset_at === "string" ? usage.reset_at : undefined,
      };
    }
  }

  return null;
}

function getPlanResetLabel(
  t: (key: string) => string,
  resetInterval?: number,
): string {
  const s = resetInterval ?? 0;
  if (s === 0) return t("admin.plan.plan-reset-18000");
  if (s === 18000) return t("admin.plan.plan-reset-18000");
  if (s === 86400) return t("admin.plan.plan-reset-86400");
  if (s === 604800) return t("admin.plan.plan-reset-604800");
  const hours = Math.round(s / 3600);
  return `${hours}h`;
}

function normalizePointWindowUsage(
  usage: SubscriptionUsageValue | null,
  total: number,
  resetInterval = pointResetInterval,
): SubscriptionUsageValue {
  const resetAt = new Date(
    Date.now() +
      (resetInterval > 0 ? resetInterval : pointResetInterval) * 1000,
  ).toISOString();
  if (!usage) {
    return {
      used: 0,
      total,
      unit: "points",
      reset_interval: resetInterval,
      reset_at: resetAt,
    };
  }

  if (!usage.reset_at) {
    return {
      ...usage,
      reset_interval: usage.reset_interval ?? resetInterval,
      reset_at: resetAt,
    };
  }

  return usage;
}

function getFallbackTimesUsage(
  usage: Record<string, number | SubscriptionUsageValue>,
): SubscriptionUsageValue {
  const entries = Object.entries(usage)
    .filter(([id]) => id !== "plan_points" && id !== "plan_points_weekly")
    .map(([, value]) => toSubscriptionUsage(value, 0))
    .filter(
      (value): value is SubscriptionUsageValue =>
        value !== null && value.unit !== "points",
    );

  if (entries.some((value) => value.total === -1)) {
    return {
      used: 0,
      total: -1,
      unit: "times",
    };
  }

  return entries.reduce<SubscriptionUsageValue>(
    (total, value) => ({
      used: total.used + value.used,
      total: total.total + value.total,
      unit: "times",
    }),
    {
      used: 0,
      total: 0,
      unit: "times",
    },
  );
}

type PlanItemProps = {
  level: number;
  isYearly: boolean;
};

function PlanItem({ level, isYearly }: PlanItemProps) {
  const { t } = useTranslation();
  const current = useSelector(levelSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);
  const { symbol } = useCurrency();
  const [ref, inView] = useInView({
    triggerOnce: true,
    threshold: 0.1,
  });

  const plan = useMemo(
    () => getPlan(subscriptionData, level),
    [subscriptionData, level],
  );
  const name = useMemo(() => getPlanName(level), [level]);
  const isHighlight = level === 2;

  const pricing = useMemo(() => {
    let discount = 1.0;
    if (isYearly) {
      if (plan.discounts && plan.discounts["12"] !== undefined) {
        discount = plan.discounts["12"];
      } else {
        discount = 0.8;
      }
    }

    const result = plan.price * discount;
    if (result % 1 !== 0) {
      return result.toFixed(1);
    }
    return result;
  }, [plan, isYearly]);

  const discountPercent = useMemo(() => {
    if (!isYearly) return 0;
    const p = subscriptionData.find((p) => p.level === level);
    if (p && p.discounts && p.discounts["12"] !== undefined) {
      return Math.round((1 - p.discounts["12"]) * 100);
    }
    return 20;
  }, [subscriptionData, level, isYearly]);

  const iconEl =
    level === 1 ? (
      <Zap />
    ) : level === 2 ? (
      <Rocket />
    ) : level === 3 ? (
      <Crown />
    ) : (
      <Star />
    );

  return (
    <motion.div
      ref={ref}
      className={cn(
        "relative flex flex-col rounded-xl border bg-background transition-shadow",
        isHighlight
          ? "border-primary/50 shadow-md shadow-primary/5"
          : "border-border/60 shadow-sm",
      )}
      initial={{ opacity: 0, y: 30 }}
      animate={inView ? { opacity: 1, y: 0 } : { opacity: 0, y: 30 }}
      transition={{ duration: 0.4, ease: "easeOut", delay: level * 0.08 }}
    >
      {isHighlight && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2 bg-primary text-primary-foreground text-[11px] font-semibold px-3 py-0.5 rounded-full whitespace-nowrap">
          {t("sub.best-choice")}
        </div>
      )}

      {/* Header */}
      <div className={cn("px-5 pt-5 pb-4", isHighlight && "pt-6")}>
        <div className="flex items-center gap-2.5 mb-3">
          <Icon
            icon={iconEl}
            className={cn("w-8 h-8 p-1.5 rounded-lg", {
              "bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400":
                level === 1,
              "bg-primary/10 text-primary": level === 2,
              "bg-amber-50 text-amber-500 dark:bg-amber-900/20 dark:text-amber-400":
                level === 3,
              "bg-muted text-muted-foreground": level === 0,
            })}
          />
          <span className="text-base font-semibold">{t(`sub.${name}`)}</span>
        </div>

        {/* Price */}
        <div className="flex items-baseline gap-0.5">
          <span className="text-sm font-medium text-muted-foreground">
            {symbol}
          </span>
          <motion.span
            className="text-3xl font-bold tracking-tight"
            key={`${pricing}-${isYearly}`}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            {pricing}
          </motion.span>
          <span className="text-sm text-muted-foreground ml-0.5">
            /{t("sub.month")}
          </span>
          {discountPercent > 0 && (
            <span className="ml-auto text-[11px] font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200/60 dark:border-emerald-800/40 px-1.5 py-0.5 rounded">
              {t("sub.year-earn-tip", { percent: `${discountPercent}%` })}
            </span>
          )}
        </div>
      </div>

      {/* Action */}
      <div className="px-5 pb-4">
        <Upgrade level={level} current={current} isYearly={isYearly} />
      </div>

      {/* Divider */}
      <div className="mx-5 border-t border-border/50" />

      {/* Features */}
      <div className="px-5 py-4 flex-1">
        {hasPlanPointPool(plan) ? (
          <div className="space-y-3">
            <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md bg-amber-50/80 dark:bg-amber-950/20 border border-amber-200/50 dark:border-amber-800/40">
              <Coins className="h-3.5 w-3.5 text-amber-500 shrink-0" />
              <span className="text-xs font-medium text-amber-700 dark:text-amber-400">
                {t("sub.plan-points-pool")}
              </span>
              <span className="text-amber-300 dark:text-amber-700 text-xs mx-0.5">
                /
              </span>
              <span className="text-[11px] text-amber-600/70 dark:text-amber-500/70">
                {t("sub.plan-points-reset", {
                  period: getPlanResetLabel(t, plan.reset_interval),
                })}
              </span>
            </div>
            <div>
              <p className="text-xs text-muted-foreground flex items-center gap-1 mb-2">
                {t("sub.including-model")}
                <Tips content={t("sub.including-model-tip")} />
              </p>
              <div className="flex flex-wrap gap-1.5">
                {plan.items.length === 0 ? (
                  <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full border border-border/60 bg-muted/30 text-xs">
                    {t("sub.all-models")}
                  </div>
                ) : (
                  plan.items.map((item, index) => (
                    <div
                      key={index}
                      className="flex items-center gap-1.5 pl-1 pr-2.5 py-1 rounded-full border border-border/60 bg-muted/30 text-xs"
                    >
                      <ModelAvatar
                        model={{
                          id: item.id,
                          name: item.name,
                          avatar: item.icon,
                        }}
                        size={16}
                      />
                      <span>{item.name}</span>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-1">
            <p className="text-xs text-muted-foreground flex items-center gap-1 mb-2">
              {t("sub.including-model")}
              <Tips content={t("sub.including-model-tip")} />
            </p>
            {plan.items.map((item, index) => (
              <div key={index} className="flex items-center py-1.5">
                <ModelAvatar
                  model={{ id: item.id, name: item.name, avatar: item.icon }}
                  size={20}
                />
                <span className="text-sm ml-2 mr-auto truncate">
                  {item.name}
                </span>
                <span className="text-sm font-medium tabular-nums shrink-0">
                  {item.value !== -1
                    ? t("sub.plan-item-usage", { times: item.value })
                    : t("sub.plan-item-unlimited-usage")}
                  {item.value !== -1 && (
                    <span className="text-xs text-muted-foreground font-normal">
                      /{t("sub.month")}
                    </span>
                  )}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </motion.div>
  );
}

function WalletPlanBox() {
  const { t } = useTranslation();
  const subscription = useSelector(isSubscribedSelector);
  const level = useSelector(levelSelector);
  const expired = useSelector(expiredSelector);
  const refresh = useSelector(refreshSelector);
  const usage = useSelector(usageSelector);
  const [isYearly, setIsYearly] = useState(false);
  const subscriptionData = useSelector(subscriptionDataSelector);

  const plan = useMemo(
    () => getPlan(subscriptionData, level),
    [subscriptionData, level],
  );

  const planName = useMemo(() => getPlanName(level), [level]);
  const isSubscribed = useMemo(
    () => subscriptionData.length > 0 && level > 0,
    [subscriptionData, level],
  );

  const enablePlanFlag = subscriptionData.length > 0;
  const sellablePlans = useMemo(
    () =>
      subscriptionData.filter((plan) => plan.level > 0 && isPlanSellable(plan)),
    [subscriptionData],
  );

  if (!enablePlanFlag) {
    return null;
  }

  return (
    <motion.div
      className="w-full mt-0 rounded-xl border bg-background overflow-hidden"
      id="plan"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3, delay: 0.2 }}
    >
      {/* Header section */}
      <div className="px-5 pt-5 pb-4 space-y-3">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs text-muted-foreground mb-1">
              {t("sub.dialog-title")}
            </p>
            <div className="flex items-center gap-2">
              <Icon
                icon={isSubscribed ? <BadgeCheck /> : <BadgeMinus />}
                className={cn(
                  "h-5 w-5",
                  isSubscribed
                    ? "text-green-500 fill-green-500/20"
                    : "text-muted-foreground fill-muted-foreground/20",
                )}
              />
              <span className="text-xl font-semibold">
                {t(`sub.${planName}`)}
              </span>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-1 text-xs text-muted-foreground">
          <p>
            {t("buy.plan-info")}
            <a
              href={docsEndpoint}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center text-sky-500 hover:text-sky-600 ml-1"
            >
              <ExternalLink className="h-3 w-3 mr-0.5" />
              {t("buy.learn-more")}
            </a>
          </p>
        </div>
      </div>

      {/* Usage accordion */}
      {subscription && (
        <div className="px-5 pb-3">
          <Accordion
            type="single"
            collapsible
            defaultValue="sub-items"
            className="w-full border rounded-lg overflow-hidden"
          >
            <AccordionItem value="sub-items" className="border-none">
              <AccordionTrigger className="px-4 py-3 bg-muted/25 border-b hover:no-underline">
                <div className="flex items-center gap-3 mr-auto">
                  <CalendarClock className="h-5 w-5 stroke-[1.5] text-muted-foreground shrink-0 !rotate-0" />
                  <div className="text-left">
                    <h3 className="text-sm font-medium">
                      {t("sub.quota-manage")}
                    </h3>
                    <p className="text-xs text-muted-foreground">
                      {t("sub.expired-days", { days: expired })}
                      {refresh > 0 && (
                        <span className="ml-2">
                          {t("sub.refresh-days", { refresh_days: refresh })}
                        </span>
                      )}
                    </p>
                  </div>
                </div>
              </AccordionTrigger>
              <AccordionContent className="p-0">
                <div className="p-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
                  {!hasPlanPointPool(plan) && plan.items.length === 0 && (
                    <SubscriptionUsage
                      name={t("sub.plan-times")}
                      usage={getFallbackTimesUsage(usage)}
                      fallbackResetLabel={t("admin.plan.plan-reset-0")}
                    />
                  )}
                  {!hasPlanPointPool(plan) &&
                    plan.items.map((item, index) => {
                      const itemUsage = toSubscriptionUsage(
                        usage?.[item.id],
                        item.value,
                      ) ?? {
                        used: 0,
                        total: item.value,
                        unit: "times" as const,
                      };
                      return (
                        <SubscriptionUsage
                          name={item.name}
                          usage={itemUsage}
                          key={index}
                          fallbackResetLabel={t("admin.plan.plan-reset-0")}
                        />
                      );
                    })}
                  {(() => {
                    if (!hasPlanPointPool(plan)) return null;
                    const planQuota = plan.quota ?? 0;
                    const weeklyQuota = plan.weekly_quota ?? 0;
                    const pointUsage = normalizePointWindowUsage(
                      toSubscriptionUsage(usage?.plan_points, planQuota),
                      planQuota,
                      plan.reset_interval,
                    );
                    const weeklyUsage = toSubscriptionUsage(
                      usage?.plan_points_weekly,
                      weeklyQuota,
                    );
                    const hasWeekly =
                      weeklyQuota > 0 ||
                      plan.weekly_quota === -1 ||
                      weeklyUsage !== null;
                    const weeklyTotal = weeklyUsage?.total ?? weeklyQuota;
                    const weeklyName = t("sub.plan-points-weekly");
                    const weeklyExhausted =
                      hasWeekly &&
                      weeklyTotal !== -1 &&
                      weeklyUsage !== null &&
                      weeklyUsage.used >= weeklyTotal;
                    return (
                      <div
                        className={cn(
                          "col-span-full grid gap-3",
                          hasWeekly
                            ? "grid-cols-1 md:grid-cols-2"
                            : "grid-cols-1",
                        )}
                      >
                        <SubscriptionUsage
                          name={t("sub.plan-points")}
                          usage={pointUsage}
                          blockedBy={weeklyExhausted ? weeklyName : undefined}
                          fallbackResetLabel={t("sub.plan-points-reset", {
                            period: getPlanResetLabel(t, plan.reset_interval),
                          })}
                        />
                        {hasWeekly && (
                          <SubscriptionUsage
                            name={weeklyName}
                            usage={
                              weeklyUsage ?? {
                                used: 0,
                                total: weeklyTotal,
                                unit: "points",
                              }
                            }
                            absoluteReset
                            fallbackResetLabel={t(
                              "sub.plan-points-weekly-reset",
                            )}
                          />
                        )}
                      </div>
                    );
                  })()}
                </div>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      )}

      {/* Period toggle + Plans grid */}
      <div className="px-5 pb-5">
        {sellablePlans.length > 0 && (
          <div className="flex justify-center mb-4">
            <Tabs
              value={isYearly ? "yearly" : "monthly"}
              onValueChange={(value) => setIsYearly(value === "yearly")}
            >
              <TabsList>
                <TabsTrigger
                  value="monthly"
                  className="w-[7rem] justify-center"
                >
                  {t("sub.month-plan")}
                </TabsTrigger>
                <TabsTrigger
                  value="yearly"
                  className="relative w-[7rem] justify-center"
                >
                  {t("sub.year-plan")}
                  {(() => {
                    const firstPlan = sellablePlans[0];
                    let discountPercent = 20;
                    if (
                      firstPlan &&
                      firstPlan.discounts &&
                      firstPlan.discounts["12"] !== undefined
                    ) {
                      discountPercent = Math.round(
                        (1 - firstPlan.discounts["12"]) * 100,
                      );
                    }
                    return discountPercent > 0 ? (
                      <span className="absolute -top-2 -right-2 text-[10px] font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200/60 dark:border-emerald-800/40 px-1 py-0 rounded-full leading-5">
                        -{discountPercent}%
                      </span>
                    ) : null;
                  })()}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
        )}

        {sellablePlans.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {sellablePlans.map((item, index) => (
              <PlanItem key={index} level={item.level} isYearly={isYearly} />
            ))}
          </div>
        ) : (
          <div className="rounded-lg border border-dashed py-8 text-center text-sm text-muted-foreground">
            {t("sub.no-sellable-plans")}
          </div>
        )}
      </div>
    </motion.div>
  );
}

function Wallet() {
  return (
    <ScrollArea className={`w-full h-full flex flex-col p-2 pr-4 bg-muted/25`}>
      <div className={`w-full h-fit max-w-5xl mx-auto py-2 md:py-6`}>
        <WalletQuotaBox />
        <WalletPlanBox />
      </div>
    </ScrollArea>
  );
}

export default Wallet;
