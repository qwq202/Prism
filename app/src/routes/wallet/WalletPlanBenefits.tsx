import { useTranslation } from "react-i18next";
import { Coins } from "lucide-react";
import { Plan } from "@/api/types.ts";
import { hasPlanPointPool } from "@/conf/subscription.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import ModelAvatar from "@/components/ModelAvatar";
import Tips from "@/components/Tips";

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

type WalletPlanBenefitsProps = {
  plan: Plan;
  compact?: boolean;
};

export default function WalletPlanBenefits({
  plan,
  compact = false,
}: WalletPlanBenefitsProps) {
  const { t } = useTranslation();

  if (hasPlanPointPool(plan)) {
    return (
      <div className={cn("space-y-2", compact && "space-y-1.5")}>
        <div
          className={cn(
            "flex items-center gap-1.5 rounded-md border border-amber-200/50 bg-amber-50/80 px-2.5 py-1.5 dark:border-amber-800/40 dark:bg-amber-950/20",
            compact && "px-2 py-1",
          )}
        >
          <Coins
            className={cn(
              "h-3.5 w-3.5 shrink-0 text-amber-500",
              compact && "h-3 w-3",
            )}
          />
          <span
            className={cn(
              "text-xs font-medium text-amber-700 dark:text-amber-400",
              compact && "text-[11px]",
            )}
          >
            {t("sub.plan-points-pool")}
          </span>
          <span className="mx-0.5 text-xs text-amber-300 dark:text-amber-700">
            /
          </span>
          <span className="text-[11px] text-amber-600/70 dark:text-amber-500/70">
            {t("sub.plan-points-reset", {
              period: getPlanResetLabel(t, plan.reset_interval),
            })}
          </span>
        </div>
        {!compact && (plan.quota ?? 0) > 0 && (
          <p className="text-[11px] text-muted-foreground">
            {t("sub.plan-points-amount", { amount: plan.quota })}
          </p>
        )}
        {!compact && plan.quota === -1 && (
          <p className="text-[11px] text-muted-foreground">
            {t("sub.plan-points-unlimited")}
          </p>
        )}
        {!compact && (plan.weekly_quota ?? 0) > 0 && (
          <p className="text-[11px] text-muted-foreground">
            {t("sub.plan-points-weekly")}:{" "}
            {t("sub.plan-points-amount", { amount: plan.weekly_quota })}
          </p>
        )}
        <div>
          {!compact && (
            <p className="mb-2 flex items-center gap-1 text-xs text-muted-foreground">
              {t("sub.including-model")}
              <Tips content={t("sub.including-model-tip")} />
            </p>
          )}
          {compact && (
            <p className="mb-1.5 text-[11px] text-muted-foreground">
              {t("sub.including-model")}
            </p>
          )}
          <div className="flex flex-wrap gap-1.5">
            {plan.items.length === 0 ? (
              <div className="flex items-center gap-1.5 rounded-full border border-border/60 bg-muted/30 px-2.5 py-1 text-xs">
                {t("sub.all-models")}
              </div>
            ) : (
              plan.items.map((item, index) => (
                <div
                  key={index}
                  className="flex items-center gap-1.5 rounded-full border border-border/60 bg-muted/30 py-1 pl-1 pr-2.5 text-xs"
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
    );
  }

  if (plan.items.length === 0) {
    return null;
  }

  return (
    <div className={cn("space-y-1", compact && "space-y-0.5")}>
      {!compact ? (
        <p className="mb-2 flex items-center gap-1 text-xs text-muted-foreground">
          {t("sub.including-model")}
          <Tips content={t("sub.including-model-tip")} />
        </p>
      ) : (
        <p className="mb-1.5 text-[11px] text-muted-foreground">
          {t("sub.including-model")}
        </p>
      )}
      {plan.items.map((item, index) => (
        <div
          key={index}
          className={cn("flex items-center", compact ? "py-1" : "py-1.5")}
        >
          <ModelAvatar
            model={{ id: item.id, name: item.name, avatar: item.icon }}
            size={compact ? 16 : 20}
          />
          <span
            className={cn(
              "ml-2 mr-auto truncate",
              compact ? "text-xs" : "text-sm",
            )}
          >
            {item.name}
          </span>
          <span
            className={cn(
              "shrink-0 font-medium tabular-nums",
              compact ? "text-xs" : "text-sm",
            )}
          >
            {item.value !== -1
              ? t("sub.plan-item-usage", { times: item.value })
              : t("sub.plan-item-unlimited-usage")}
            {item.value !== -1 && (
              <span className="text-xs font-normal text-muted-foreground">
                /{t("sub.month")}
              </span>
            )}
          </span>
        </div>
      ))}
    </div>
  );
}
