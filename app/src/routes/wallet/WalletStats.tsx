import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useDispatch, useSelector } from "react-redux";
import { BarList, DonutChart } from "@tremor/react";
import type { BarListProps } from "@tremor/react";
import { getModelColor } from "@/admin/colors.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { infoTimeZoneSelector } from "@/store/info.ts";
import {
  refreshWalletUsageSummary,
  walletUsageSummaryLoadedSelector,
  walletUsageSummaryLoadingSelector,
  walletUsageSummarySelector,
  walletUsageSummaryTimeZoneSelector,
} from "@/store/record.ts";
import { AppDispatch } from "@/store";
import {
  ArrowUpRight,
  BarChart3,
  Cloud,
  Gauge,
  Layers,
  Loader2,
  PieChart,
  Zap,
} from "lucide-react";

type ModelUsage = { name: string; value: number; count: number };
type ModelUsageBar = ModelUsage & { color: string };

function formatQuotaValue(value: number): string {
  return Number.isFinite(value)
    ? value.toLocaleString(undefined, { maximumFractionDigits: 4 })
    : "0";
}

function QuotaValue({ value }: { value: number }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5">
      <span>{formatQuotaValue(value)}</span>
      <Cloud className="h-4 w-4 shrink-0 text-muted-foreground" />
    </span>
  );
}

type StatCardProps = {
  label: string;
  value: React.ReactNode;
  icon: React.ReactNode;
  iconClass?: string;
  loading?: boolean;
  title?: string;
};

function StatCard({
  label,
  value,
  icon,
  iconClass,
  loading,
  title,
}: StatCardProps) {
  return (
    <div className="flex min-h-[5.75rem] items-center justify-start gap-3 rounded-xl border bg-background p-4">
      <div
        className={cn(
          "w-8 h-8 rounded-lg flex items-center justify-center shrink-0",
          iconClass ?? "bg-muted text-muted-foreground",
        )}
      >
        {icon}
      </div>
      <div className="min-w-0 text-left">
        <p className="text-xs text-muted-foreground leading-none mb-1.5">
          {label}
        </p>
        {loading ? (
          <Skeleton className="h-7 w-20" />
        ) : (
          <p
            className="break-words text-lg font-semibold tracking-tight leading-tight"
            title={title}
          >
            {value}
          </p>
        )}
      </div>
    </div>
  );
}

function ModelBreakdownSkeleton() {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
      <div className="flex justify-center sm:w-52 shrink-0">
        <Skeleton className="h-44 w-44 rounded-full" />
      </div>
      <div className="flex-1 min-w-0 space-y-3">
        {[72, 56, 44, 38, 32].map((width, index) => (
          <div key={index} className="flex items-center gap-3">
            <Skeleton
              className="h-9 rounded-lg"
              style={{ width: `${width}%` }}
            />
            <Skeleton className="ml-auto h-5 w-14 shrink-0" />
          </div>
        ))}
      </div>
    </div>
  );
}

export default function WalletStats() {
  const { t } = useTranslation();
  const quotaUnit = t("quota");
  const dispatch: AppDispatch = useDispatch();
  const timeZone = useSelector(infoTimeZoneSelector);
  const usageSummary = useSelector(walletUsageSummarySelector);
  const loaded = useSelector(walletUsageSummaryLoadedSelector);
  const refreshing = useSelector(walletUsageSummaryLoadingSelector);
  const loadedTimeZone = useSelector(walletUsageSummaryTimeZoneSelector);
  const loading = !loaded || loadedTimeZone !== timeZone;
  const modelUsage = useMemo(
    () => usageSummary.models ?? [],
    [usageSummary.models],
  );

  useEffect(() => {
    if (!refreshing && (!loaded || loadedTimeZone !== timeZone)) {
      void dispatch(refreshWalletUsageSummary(timeZone));
    }
  }, [dispatch, loaded, loadedTimeZone, refreshing, timeZone]);

  useEffect(() => {
    function handleCacheUpdate(event: Event) {
      const detail = (event as CustomEvent<{ url?: string }>).detail;
      if (detail?.url === "/record/usage-summary") {
        void dispatch(refreshWalletUsageSummary(timeZone));
      }
    }

    window.addEventListener("prism-api-cache-updated", handleCacheUpdate);
    return () =>
      window.removeEventListener("prism-api-cache-updated", handleCacheUpdate);
  }, [dispatch, timeZone]);

  const donutColors = useMemo(
    () => modelUsage.map((m) => getModelColor(m.name)),
    [modelUsage],
  );

  const barListData = useMemo(
    () =>
      modelUsage.slice(0, 8).map((m) => ({
        name: m.name,
        value: m.value,
        count: m.count,
        color: getModelColor(m.name),
      })),
    [modelUsage],
  );

  const hasModelData = modelUsage.length > 0;

  return (
    <div className="rounded-2xl border bg-background overflow-hidden">
      <div className="p-5">
        <div className="flex items-center justify-between mb-4">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {t("bar.wallet-usage-stats")}
          </p>
          {refreshing && (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
          )}
        </div>

        {/* 四项统计 */}
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <StatCard
            label={t("bar.wallet-model-count")}
            value={String(usageSummary.model_count)}
            icon={<Layers className="h-4 w-4" />}
            iconClass="bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400"
            loading={loading}
          />
          <StatCard
            label={t("bar.wallet-top-model")}
            value={usageSummary.top_model || "--"}
            title={usageSummary.top_model || "--"}
            icon={<PieChart className="h-4 w-4" />}
            iconClass="bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"
            loading={loading}
          />
          <StatCard
            label={t("bar.wallet-average-cost")}
            value={<QuotaValue value={usageSummary.average_quota} />}
            title={`${formatQuotaValue(usageSummary.average_quota)} ${quotaUnit}`}
            icon={<Gauge className="h-4 w-4" />}
            iconClass="bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400"
            loading={loading}
          />
          <StatCard
            label={t("bar.wallet-max-cost")}
            value={<QuotaValue value={usageSummary.max_quota} />}
            title={`${formatQuotaValue(usageSummary.max_quota)} ${quotaUnit}`}
            icon={<Zap className="h-4 w-4" />}
            iconClass="bg-violet-100 text-violet-600 dark:bg-violet-900/30 dark:text-violet-400"
            loading={loading}
          />
        </div>
      </div>

      {/* 模型消费分布 */}
      <div className="px-5 pb-5 pt-4">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-4">
          {t("bar.wallet-model-breakdown")}
        </p>

        {loading ? (
          <ModelBreakdownSkeleton />
        ) : hasModelData ? (
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
            <div className="flex justify-center sm:w-52 shrink-0">
              <DonutChart
                className="w-44 h-44"
                variant="donut"
                data={modelUsage}
                showAnimation
                showLabel={false}
                showTooltip
                valueFormatter={(v: number) => formatQuotaValue(v)}
                colors={donutColors}
              />
            </div>

            <div className="flex-1 min-w-0">
              <BarList
                data={
                  barListData as unknown as BarListProps<ModelUsageBar>["data"]
                }
                valueFormatter={(v: number) => formatQuotaValue(v)}
                showAnimation
                className="text-sm"
              />
              {modelUsage.length > 8 && (
                <p className="text-xs text-muted-foreground mt-2 text-right">
                  {t("bar.wallet-more-models", {
                    count: modelUsage.length - 8,
                  })}
                </p>
              )}
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground gap-2">
            <BarChart3 className="h-8 w-8 opacity-30" />
            <p className="text-sm">{t("bar.wallet-no-usage")}</p>
          </div>
        )}
      </div>

      {/* 查看完整记录 */}
      <div className="px-5 py-3 flex justify-end">
        <Link
          to="/log"
          className="inline-flex items-center gap-1 text-xs text-sky-500 hover:text-sky-600"
        >
          {t("bar.wallet-view-records")}
          <ArrowUpRight className="h-3 w-3" />
        </Link>
      </div>
    </div>
  );
}
