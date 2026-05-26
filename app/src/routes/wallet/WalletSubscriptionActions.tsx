import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useSelector } from "react-redux";
import { useNavigate } from "react-router-dom";
import { cn } from "@/components/ui/lib/utils.ts";
import { levelSelector } from "@/store/subscription.ts";
import { subscriptionDataSelector } from "@/store/globals.ts";
import { isPlanSellable } from "@/conf/subscription.tsx";
import { Upgrade } from "@/components/home/subscription/UpgradePlan.tsx";
import { ShoppingCart } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";

type WalletSubscriptionActionBarProps = {
  className?: string;
};

export default function WalletSubscriptionActionBar({
  className,
}: WalletSubscriptionActionBarProps) {
  const { t } = useTranslation();
  const level = useSelector(levelSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);
  const navigate = useNavigate();

  const sellablePlans = useMemo(
    () =>
      subscriptionData.filter((plan) => plan.level > 0 && isPlanSellable(plan)),
    [subscriptionData],
  );

  const isSubscribed = level > 0;
  const otherPlans = useMemo(
    () => sellablePlans.filter((plan) => plan.level !== level),
    [sellablePlans, level],
  );

  const currentPlanSellable = sellablePlans.some((plan) => plan.level === level);
  const singlePlan = sellablePlans.length === 1 ? sellablePlans[0] : null;

  const showRenew = isSubscribed && currentPlanSellable;
  const showChange = isSubscribed && otherPlans.length > 0;
  const showDirectSubscribe = !isSubscribed && !!singlePlan;
  const showSubscribePicker = !isSubscribed && sellablePlans.length > 1;

  if (!showRenew && !showChange && !showDirectSubscribe && !showSubscribePicker) {
    return null;
  }

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {showRenew && <Upgrade level={level} current={level} compact />}

      {showChange && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => navigate("/subscription")}
        >
          {t("sub.change")}
        </Button>
      )}

      {showDirectSubscribe && (
        <Upgrade level={singlePlan!.level} current={level} compact />
      )}

      {showSubscribePicker && (
        <Button size="sm" onClick={() => navigate("/subscription")}>
          <ShoppingCart className="mr-1.5 h-4 w-4" />
          {t("sub.subscribe")}
        </Button>
      )}
    </div>
  );
}
