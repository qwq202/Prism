import "@/assets/pages/quota.less";
import "@/assets/pages/subscription.less";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { useTranslation } from "react-i18next";
import { ArrowRight, BadgeCheck, Cloud, Coins, Gift } from "lucide-react";
import { cn } from "@/components/ui/lib/utils.ts";
import { useEffect, useMemo, useState } from "react";
import { useSelector, useDispatch } from "react-redux";
import { Link } from "react-router-dom";
import {
  expiredSelector,
  isSubscribedSelector,
  levelSelector,
  usageSelector,
  refreshSelector,
  refreshSubscription,
} from "@/store/subscription.ts";
import { subscriptionDataSelector } from "@/store/globals.ts";
import {
  allowSubscriptionQuotaFallbackSelector,
  quotaSelector,
  refreshQuota,
  setAllowSubscriptionQuotaFallback,
} from "@/store/quota.ts";
import { AppDispatch } from "@/store";
import { motion } from "framer-motion";
import { getPlan, getPlanName } from "@/conf/subscription.tsx";
import WalletUsageGrid from "@/routes/wallet/WalletUsageGrid.tsx";
import WalletSubscriptionActionBar from "@/routes/wallet/WalletSubscriptionActions.tsx";
import WalletStats from "@/routes/wallet/WalletStats.tsx";
import Icon from "@/components/utils/Icon";
import { Crown, Rocket, Star, Zap } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog.tsx";
import { Input } from "@/components/ui/input.tsx";
import { useRedeem as redeemCode } from "@/api/redeem.ts";
import { Switch } from "@/components/ui/switch.tsx";
import { updateSubscriptionQuotaFallback } from "@/api/quota.ts";

const fadeUp = {
  hidden: { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0, transition: { duration: 0.45 } },
};

const stagger = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { staggerChildren: 0.08 } },
};

function getPlanIcon(level: number) {
  if (level === 1) return <Zap />;
  if (level === 2) return <Rocket />;
  if (level === 3) return <Crown />;
  return <Star />;
}

function getPlanIconClass(level: number) {
  return cn("w-10 h-10 p-2.5 rounded-xl shrink-0", {
    "bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400":
      level === 1,
    "bg-primary/10 text-primary": level === 2,
    "bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400":
      level === 3,
    "bg-muted text-muted-foreground": level === 0,
  });
}

type RedeemDialogProps = { open: boolean; onOpenChange: (v: boolean) => void };
function RedeemDialog({ open, onOpenChange }: RedeemDialogProps) {
  const { t } = useTranslation();
  const [code, setCode] = useState("");
  const dispatch: AppDispatch = useDispatch();

  const doRedeem = async () => {
    if (!code.trim()) return;
    const res = await redeemCode(code.trim());
    if (res.status) {
      toast.success(t("buy.exchange-success"), {
        description: t("buy.exchange-success-prompt", { amount: res.quota }),
      });
      setCode("");
      dispatch(refreshQuota());
      onOpenChange(false);
    } else {
      toast.error(t("buy.exchange-failed"), {
        description: t("buy.exchange-failed-prompt", { reason: res.error }),
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("buy.redeem-title")}</DialogTitle>
          <DialogDescription>{t("buy.redeem-description")}</DialogDescription>
        </DialogHeader>
        <div className="relative">
          <Gift className="h-4 w-4 text-muted-foreground absolute left-3 top-1/2 -translate-y-1/2" />
          <Input
            className="pl-10 text-center"
            placeholder={t("buy.redeem-placeholder")}
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            unClickable
            onClick={() => onOpenChange(false)}
          >
            {t("cancel")}
          </Button>
          <Button
            unClickable
            disabled={!code.trim()}
            loading
            onClick={doRedeem}
          >
            {t("buy.redeem")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function WalletPage() {
  const { t } = useTranslation();
  const dispatch: AppDispatch = useDispatch();
  const quota = useSelector(quotaSelector);
  const allowSubscriptionQuotaFallback = useSelector(
    allowSubscriptionQuotaFallbackSelector,
  );
  const subscription = useSelector(isSubscribedSelector);
  const level = useSelector(levelSelector);
  const expired = useSelector(expiredSelector);
  const refresh = useSelector(refreshSelector);
  const usage = useSelector(usageSelector);
  const subscriptionData = useSelector(subscriptionDataSelector);
  const [redeemOpen, setRedeemOpen] = useState(false);
  const [savingFallback, setSavingFallback] = useState(false);

  const plan = useMemo(
    () => getPlan(subscriptionData, level),
    [subscriptionData, level],
  );
  const planName = useMemo(() => getPlanName(level), [level]);
  const isSubscribed = useMemo(
    () => subscriptionData.length > 0 && level > 0,
    [subscriptionData, level],
  );
  const hasSubscriptionData = subscriptionData.length > 0;

  useEffect(() => {
    void dispatch(refreshQuota());
    void dispatch(refreshSubscription());
  }, [dispatch]);

  const updateFallbackPreference = async (checked: boolean) => {
    const previous = allowSubscriptionQuotaFallback;
    dispatch(setAllowSubscriptionQuotaFallback(checked));
    setSavingFallback(true);

    const res = await updateSubscriptionQuotaFallback(checked);
    setSavingFallback(false);

    if (res.status) {
      dispatch(
        setAllowSubscriptionQuotaFallback(
          res.allow_subscription_quota_fallback ?? checked,
        ),
      );
      toast.success(t("buy.subscription-fallback-save-success"));
      return;
    }

    dispatch(setAllowSubscriptionQuotaFallback(previous));
    toast.error(t("buy.subscription-fallback-save-failed"), {
      description: res.error,
    });
  };

  return (
    <ScrollArea className="w-full h-full bg-muted/25">
      <RedeemDialog open={redeemOpen} onOpenChange={setRedeemOpen} />

      <motion.div
        className="w-full max-w-3xl mx-auto px-4 py-6 md:py-10 space-y-4"
        variants={stagger}
        initial="hidden"
        animate="visible"
      >
        {/* ── 余额卡片 ── */}
        <motion.div
          className="rounded-2xl border bg-background p-5"
          variants={fadeUp}
        >
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-4">
            {t("buy.title")}
          </p>
          <div className="flex items-end justify-between gap-4">
            <div>
              <div className="flex items-baseline gap-1.5">
                <Cloud className="h-5 w-5 text-muted-foreground mb-0.5" />
                <span className="text-3xl font-bold tracking-tight jetbrains-mono">
                  {quota.toFixed(2)}
                </span>
              </div>
              <p className="text-xs text-muted-foreground mt-1.5 max-w-xs">
                {t("buy.quota-info")}
                <Link
                  to="/subscription-guide"
                  className="inline-flex items-center text-sky-500 hover:text-sky-600 ml-1"
                >
                  {t("buy.learn-more")}
                  <ArrowRight className="h-3 w-3 ml-0.5" />
                </Link>
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="shrink-0"
              onClick={() => setRedeemOpen(true)}
            >
              <Gift className="h-4 w-4 mr-1.5" />
              {t("buy.redeem-title")}
            </Button>
          </div>
        </motion.div>

        {/* ── 订阅计划卡片 ── */}
        {hasSubscriptionData && (
          <motion.div
            className="rounded-2xl border bg-background overflow-hidden"
            variants={fadeUp}
          >
            {/* 头部 */}
            <div className="p-5">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-4">
                {t("sub.dialog-title")}
              </p>

              <div className="flex items-start justify-between gap-3">
                <div className="flex items-start gap-3 min-w-0">
                  <Icon
                    icon={getPlanIcon(level)}
                    className={getPlanIconClass(level)}
                  />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="text-xl font-semibold">
                        {t(`sub.${planName}`)}
                      </h2>
                      {isSubscribed && (
                        <span className="inline-flex items-center gap-1 rounded-full border border-emerald-200/70 bg-emerald-50 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-950/30 dark:text-emerald-400">
                          <BadgeCheck className="h-3 w-3" />
                          {t("sub.current")}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {isSubscribed
                        ? t("sub.expired-days", { days: expired })
                        : t("sub.not-subscribed-hint")}
                      {isSubscribed && refresh > 0 && (
                        <span className="ml-1.5">
                          · {t("sub.refresh-days", { refresh_days: refresh })}
                        </span>
                      )}
                    </p>
                  </div>
                </div>
                <WalletSubscriptionActionBar className="shrink-0 pt-0.5" />
              </div>

              <p className="mt-4 text-xs text-muted-foreground">
                {t("buy.plan-info")}
                <Link
                  to="/subscription-guide"
                  className="inline-flex items-center text-sky-500 hover:text-sky-600 ml-1"
                >
                  {t("buy.learn-more")}
                  <ArrowRight className="h-3 w-3 ml-0.5" />
                </Link>
              </p>

              {subscription && (
                <div className="mt-4 flex items-center justify-between gap-4 rounded-lg border bg-muted/20 px-3 py-3">
                  <div className="flex min-w-0 items-start gap-3">
                    <Coins className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-foreground">
                        {t("buy.subscription-fallback-title")}
                      </p>
                      <p className="mt-0.5 text-xs leading-5 text-muted-foreground">
                        {t("buy.subscription-fallback-desc")}
                      </p>
                    </div>
                  </div>
                  <Switch
                    checked={allowSubscriptionQuotaFallback}
                    disabled={savingFallback}
                    onCheckedChange={updateFallbackPreference}
                    aria-label={t("buy.subscription-fallback-title")}
                  />
                </div>
              )}
            </div>

            {/* 额度区块 — 和头部同色，用细线分隔 */}
            {subscription && (
              <div className="px-5 pb-5">
                <div className="border-t pt-4">
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-3">
                    {t("sub.quota-manage")}
                  </p>
                  <WalletUsageGrid plan={plan} usage={usage} />
                </div>
              </div>
            )}
          </motion.div>
        )}

        {/* ── 使用统计卡片 ── */}
        <motion.div variants={fadeUp}>
          <WalletStats />
        </motion.div>
      </motion.div>
    </ScrollArea>
  );
}

export default WalletPage;
