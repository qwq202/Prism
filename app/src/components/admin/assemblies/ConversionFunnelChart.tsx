import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { ConversionFunnelResponse } from "@/admin/types.ts";
import { getReadableNumber } from "@/utils/processor.ts";

type ConversionFunnelChartProps = {
  data: ConversionFunnelResponse;
};

function FunnelBar({ label, value, max, color }: {
  label: string;
  value: number;
  max: number;
  color: string;
}) {
  const pct = max > 0 ? Math.round((value / max) * 100) : 0;
  return (
    <div className="flex items-center gap-3 mb-3">
      <div className="w-32 shrink-0 text-xs text-right text-muted-foreground truncate">{label}</div>
      <div className="flex-1 bg-muted rounded-full h-4 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <div className="w-20 shrink-0 text-xs font-medium">
        {getReadableNumber(value)} <span className="text-muted-foreground">({pct}%)</span>
      </div>
    </div>
  );
}

function ConversionFunnelChart({ data }: ConversionFunnelChartProps) {
  const { t } = useTranslation();
  const loading = data.registered === 0;

  return (
    <div className={`chart`}>
      <div className={`chart-title mb-4`}>
        <p>{t("admin.conversion-funnel")}</p>
        {loading && <Loader2 className={`h-4 w-4 inline-block animate-spin`} />}
      </div>
      <FunnelBar
        label={t("admin.funnel-registered")}
        value={data.registered}
        max={data.registered}
        color="bg-blue-500"
      />
      <FunnelBar
        label={t("admin.funnel-ever-subscribed")}
        value={data.ever_subscribed}
        max={data.registered}
        color="bg-violet-500"
      />
      <FunnelBar
        label={t("admin.funnel-active-subscribed")}
        value={data.active_subscribed}
        max={data.registered}
        color="bg-emerald-500"
      />
    </div>
  );
}

export default ConversionFunnelChart;
