import "@/assets/pages/subscription.less";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { useTranslation } from "react-i18next";
import {
  ArrowRight,
  BadgeCheck,
  CheckIcon,
  Coins,
  Crown,
  Rocket,
  Star,
  Zap,
} from "lucide-react";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/components/ui/lib/utils.ts";
import { useMemo, useState } from "react";
import { useSelector } from "react-redux";
import { Link } from "react-router-dom";
import { useCurrency } from "@/store/info.ts";
import { levelSelector } from "@/store/subscription.ts";
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
import ModelAvatar from "@/components/ModelAvatar";
import Icon from "@/components/utils/Icon";
import Tips from "@/components/Tips";

function getPlanIcon(level: number) {
  if (level === 1) return <Zap />;
  if (level === 2) return <Rocket />;
  if (level === 3) return <Crown />;
  return <Star />;
}

function getPlanIconBg(level: number) {
  return cn("w-11 h-11 p-2.5 rounded-2xl shrink-0", {
    "bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400":
      level === 1,
    "bg-primary/10 text-primary": level === 2,
    "bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400":
      level === 3,
    "bg-muted text-muted-foreground": level === 0,
  });
}

function getPlanResetLabel(
  t: (key: string) => string,
  resetInterval?: number,
): string {
  const s = resetInterval ?? 0;
  if (s === 0 || s === 18000) return t("admin.plan.plan-reset-18000");
  if (s === 86400) return t("admin.plan.plan-reset-86400");
  if (s === 604800) return t("admin.plan.plan-reset-604800");
  return `${Math.round(s / 3600)}h`;
}

type FeatureLineProps = {
  children: React.ReactNode;
  highlight?: boolean;
};

function FeatureLine({ children, highlight }: FeatureLineProps) {
  return (
    <li className="flex items-start gap-2.5 text-sm">
      <CheckIcon
        className={cn(
          "mt-0.5 h-4 w-4 shrink-0",
          highlight ? "text-primary" : "text-muted-foreground/60",
        )}
      />
      <span className={highlight ? "text-foreground" : "text-muted-foreground"}>
        {children}
      </span>
    </li>
  );
}

type PlanCardProps = {
  level: number;
  isYearly: boolean;
};

function PlanCard({ level, isYearly }: PlanCardProps) {
  const { t } = useTranslation();
  const current = useSelector(levelSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);
  const { symbol } = useCurrency();
  const [ref, inView] = useInView({ triggerOnce: true, threshold: 0.05 });

  const plan = useMemo(
    () => getPlan(subscriptionData, level),
    [subscriptionData, level],
  );
  const name = useMemo(() => getPlanName(level), [level]);

  const pricing = useMemo(() => {
    let discount = 1.0;
    if (isYearly) {
      discount =
        plan.discounts?.["12"] !== undefined ? plan.discounts["12"] : 0.8;
    }
    const result = plan.price * discount;
    return result % 1 !== 0 ? result.toFixed(1) : result;
  }, [plan, isYearly]);

  const discountPercent = useMemo(() => {
    if (!isYearly) return 0;
    if (plan.discounts?.["12"] !== undefined) {
      return Math.round((1 - plan.discounts["12"]) * 100);
    }
    return 20;
  }, [plan, isYearly]);

  const yearlyBillingTotal = useMemo(() => Number(pricing) * 12, [pricing]);

  const isCurrent = current === level;
  const isHighlight = level === 2;

  return (
    <motion.div
      ref={ref}
      className={cn(
        "relative flex flex-col rounded-2xl border bg-background transition-shadow",
        isCurrent
          ? "border-primary/50 shadow-lg shadow-primary/5 ring-2 ring-primary/20"
          : isHighlight
            ? "border-primary/30 shadow-md shadow-primary/5"
            : "border-border/60 shadow-sm",
      )}
      initial={{ opacity: 0, y: 24 }}
      animate={inView ? { opacity: 1, y: 0 } : { opacity: 0, y: 24 }}
      transition={{ duration: 0.45, ease: "easeOut" }}
    >
      {(isCurrent || isHighlight) && (
        <div
          className={cn(
            "absolute -top-3.5 left-1/2 -translate-x-1/2 rounded-full px-3 py-0.5 text-[11px] font-semibold whitespace-nowrap",
            isCurrent
              ? "bg-primary text-primary-foreground"
              : "bg-primary text-primary-foreground",
          )}
        >
          {isCurrent ? t("sub.current") : t("sub.best-choice")}
        </div>
      )}

      <div className={cn("px-6 pt-7 pb-5", (isCurrent || isHighlight) && "pt-8")}>
        <div className="flex items-center gap-3 mb-5">
          <Icon icon={getPlanIcon(level)} className={getPlanIconBg(level)} />
          <span className="text-lg font-bold">{t(`sub.${name}`)}</span>
        </div>

        <div className="flex items-end gap-1 mb-1">
          <span className="text-sm font-medium text-muted-foreground leading-none mb-1">
            {symbol}
          </span>
          <motion.span
            className="text-4xl font-bold tracking-tight leading-none"
            key={`${pricing}-${isYearly}`}
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25 }}
          >
            {pricing}
          </motion.span>
          <span className="text-sm text-muted-foreground mb-1 ml-0.5">
            /{t("sub.month")}
          </span>
          {discountPercent > 0 && (
            <span className="ml-auto mb-1 text-[11px] font-medium text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200/60 dark:border-emerald-800/40 px-1.5 py-0.5 rounded">
              {t("sub.year-earn-tip", { percent: `${discountPercent}%` })}
            </span>
          )}
        </div>

        {isYearly && plan.price > 0 && (
          <p className="text-xs text-muted-foreground mb-5">
            {t("sub.billed-yearly", {
              total: yearlyBillingTotal.toFixed(0),
              symbol,
            })}
          </p>
        )}
        {(!isYearly || plan.price === 0) && <div className="mb-5" />}

        <Upgrade level={level} current={current} isYearly={isYearly} />
      </div>

      <div className="mx-6 border-t border-border/50" />

      <div className="px-6 py-5 flex-1">
        <ul className="space-y-3">
          {hasPlanPointPool(plan) ? (
            <>
              <FeatureLine highlight>
                <span className="flex items-center gap-1.5 flex-wrap">
                  <Coins className="h-3.5 w-3.5 text-amber-500 shrink-0" />
                  <span className="font-medium">{t("sub.plan-points-pool")}</span>
                  <span className="text-muted-foreground/70">
                    · {t("sub.plan-points-reset", {
                      period: getPlanResetLabel(t, plan.reset_interval),
                    })}
                  </span>
                </span>
              </FeatureLine>
              <FeatureLine highlight>
                <span className="flex items-center gap-1 flex-wrap">
                  <span className="font-medium flex items-center gap-1">
                    {t("sub.including-model")}
                    <Tips content={t("sub.including-model-tip")} />
                  </span>
                </span>
              </FeatureLine>
              {plan.items.length === 0 ? (
                <FeatureLine>{t("sub.all-models")}</FeatureLine>
              ) : (
                plan.items.map((item, i) => (
                  <FeatureLine key={i}>
                    <span className="flex items-center gap-1.5">
                      <ModelAvatar
                        model={{ id: item.id, name: item.name, avatar: item.icon }}
                        size={15}
                      />
                      {item.name}
                    </span>
                  </FeatureLine>
                ))
              )}
            </>
          ) : plan.items.length === 0 ? (
            <FeatureLine highlight>{t("sub.plan-times")}</FeatureLine>
          ) : (
            plan.items.map((item, i) => (
              <FeatureLine key={i} highlight>
                <span className="flex items-center justify-between gap-2 w-full">
                  <span className="flex items-center gap-1.5">
                    <ModelAvatar
                      model={{ id: item.id, name: item.name, avatar: item.icon }}
                      size={15}
                    />
                    {item.name}
                  </span>
                  <span className="tabular-nums font-medium text-foreground shrink-0 text-xs">
                    {item.value !== -1
                      ? t("sub.plan-item-usage", { times: item.value })
                      : t("sub.plan-item-unlimited-usage")}
                    {item.value !== -1 && (
                      <span className="text-muted-foreground font-normal">
                        /{t("sub.month")}
                      </span>
                    )}
                  </span>
                </span>
              </FeatureLine>
            ))
          )}
        </ul>
      </div>
    </motion.div>
  );
}

function SubscriptionPage() {
  const { t } = useTranslation();
  const subscriptionData = useSelector(subscriptionDataSelector);
  const [isYearly, setIsYearly] = useState(false);

  const sellablePlans = useMemo(
    () =>
      subscriptionData.filter((plan) => plan.level > 0 && isPlanSellable(plan)),
    [subscriptionData],
  );

  const yearlyDiscountPercent = useMemo(() => {
    const first = sellablePlans[0];
    if (first?.discounts?.["12"] !== undefined) {
      return Math.round((1 - first.discounts["12"]) * 100);
    }
    return 20;
  }, [sellablePlans]);

  const gridClass = useMemo(() => {
    if (sellablePlans.length === 1) return "grid grid-cols-1 max-w-sm mx-auto";
    if (sellablePlans.length === 2) return "grid grid-cols-1 sm:grid-cols-2 max-w-2xl mx-auto";
    return "grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3";
  }, [sellablePlans.length]);

  if (subscriptionData.length === 0) return null;

  return (
    <ScrollArea className="w-full h-full bg-muted/25">
      <div className="w-full max-w-5xl mx-auto px-4 py-10 md:py-16">

        <div className="text-center mb-10">
          <motion.h1
            className="text-3xl font-bold tracking-tight mb-2"
            initial={{ opacity: 0, y: -12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4 }}
          >
            {t("sub.choose-plan")}
          </motion.h1>
          <motion.p
            className="text-muted-foreground text-sm"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.4, delay: 0.1 }}
          >
            {t("buy.plan-info")}
            <Link
              to="/subscription-guide"
              className="inline-flex items-center text-sky-500 hover:text-sky-600 ml-1"
            >
              {t("buy.learn-more")}
              <ArrowRight className="h-3 w-3 ml-0.5" />
            </Link>
          </motion.p>
        </div>

        <motion.div
          className="flex justify-center mb-10"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.35, delay: 0.15 }}
        >
          <Tabs
            value={isYearly ? "yearly" : "monthly"}
            onValueChange={(v) => setIsYearly(v === "yearly")}
          >
            <TabsList className="h-10 p-1">
              <TabsTrigger value="monthly" className="w-28 text-sm">
                {t("sub.month-plan")}
              </TabsTrigger>
              <TabsTrigger value="yearly" className="relative w-28 text-sm">
                {t("sub.year-plan")}
                {yearlyDiscountPercent > 0 && (
                  <span className="absolute -top-2.5 -right-2 rounded-full border border-emerald-200/60 bg-emerald-50 px-1.5 py-0 text-[10px] font-semibold leading-5 text-emerald-600 dark:border-emerald-800/40 dark:bg-emerald-950/30 dark:text-emerald-400">
                    -{yearlyDiscountPercent}%
                  </span>
                )}
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </motion.div>

        {sellablePlans.length > 0 ? (
          <div className={cn(gridClass, "gap-5 mt-8")}>
            {sellablePlans.map((plan) => (
              <PlanCard key={plan.level} level={plan.level} isYearly={isYearly} />
            ))}
          </div>
        ) : (
          <div className="rounded-xl border border-dashed py-16 text-center text-sm text-muted-foreground">
            {t("sub.no-sellable-plans")}
          </div>
        )}

        <motion.div
          className="mt-10 flex justify-center"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.4, delay: 0.3 }}
        >
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <BadgeCheck className="h-3.5 w-3.5 text-emerald-500" />
            {t("sub.secure-payment")}
          </div>
        </motion.div>
      </div>
    </ScrollArea>
  );
}

export default SubscriptionPage;
