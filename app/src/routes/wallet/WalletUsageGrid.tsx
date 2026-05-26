import { useTranslation } from "react-i18next";
import { cn } from "@/components/ui/lib/utils.ts";
import SubscriptionUsage from "@/components/home/subscription/SubscriptionUsage.tsx";
import { hasPlanPointPool } from "@/conf/subscription.tsx";
import { Plan } from "@/api/types.ts";

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

type WalletUsageGridProps = {
  plan: Plan;
  usage: Record<string, number | SubscriptionUsageValue>;
};

export default function WalletUsageGrid({ plan, usage }: WalletUsageGridProps) {
  const { t } = useTranslation();

  if (!hasPlanPointPool(plan)) {
    const items = plan.items.length === 0 ? [null] : plan.items;
    return (
      <div
        className={cn(
          "grid gap-3",
          items.length > 1
            ? "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3"
            : "grid-cols-1 sm:grid-cols-2",
        )}
      >
        {plan.items.length === 0 && (
          <SubscriptionUsage
            name={t("sub.plan-times")}
            usage={getFallbackTimesUsage(usage)}
            fallbackResetLabel={t("admin.plan.plan-reset-0")}
            variant="flat"
          />
        )}
        {plan.items.map((item, index) => {
          const itemUsage = toSubscriptionUsage(usage?.[item.id], item.value) ?? {
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
              variant="flat"
            />
          );
        })}
      </div>
    );
  }

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
    weeklyQuota > 0 || plan.weekly_quota === -1 || weeklyUsage !== null;
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
        "grid gap-3",
        hasWeekly ? "grid-cols-1 sm:grid-cols-2" : "grid-cols-1",
      )}
    >
      <SubscriptionUsage
        name={t("sub.plan-points")}
        usage={pointUsage}
        blockedBy={weeklyExhausted ? weeklyName : undefined}
        fallbackResetLabel={t("sub.plan-points-reset", {
          period: getPlanResetLabel(t, plan.reset_interval),
        })}
        variant="flat"
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
          fallbackResetLabel={t("sub.plan-points-weekly-reset")}
          variant="flat"
        />
      )}
    </div>
  );
}
