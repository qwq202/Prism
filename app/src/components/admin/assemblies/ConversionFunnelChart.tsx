import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { ChartLegend, DonutChart } from "./recharts.tsx";
import { getReadableNumber } from "@/utils/processor.ts";
import { ConversionFunnelResponse } from "@/admin/types.ts";

type ConversionFunnelChartProps = {
  data: ConversionFunnelResponse;
};

function ConversionFunnelChart({ data }: ConversionFunnelChartProps) {
  const { t } = useTranslation();
  const loading = data.registered === 0;

  const chartData = useMemo(() => {
    const neverSubscribed = Math.max(0, data.registered - data.ever_subscribed);
    const churned = Math.max(0, data.ever_subscribed - data.active_subscribed);
    const active = Math.max(0, data.active_subscribed);
    return [
      { name: t("admin.funnel-active-subscribed"), value: active },
      { name: t("admin.funnel-churned"), value: churned },
      { name: t("admin.funnel-never-subscribed"), value: neverSubscribed },
    ].filter((d) => d.value > 0);
  }, [data, t]);

  const categories = useMemo(
    () => chartData.map((d) => `${d.name} (${getReadableNumber(d.value)})`),
    [chartData],
  );

  const total = useMemo(
    () => chartData.reduce((sum, item) => sum + item.value, 0),
    [chartData],
  );

  const colors = ["emerald", "violet", "slate"];

  return (
    <div className="chart">
      <div className="chart-title mb-2">
        <p>{t("admin.conversion-funnel")}</p>
        {loading && <Loader2 className="h-4 w-4 inline-block animate-spin" />}
      </div>
      <div className="flex flex-row">
        <DonutChart
          className="common-chart p-4 w-[50%]"
          data={chartData}
          valueFormatter={(v) => getReadableNumber(v)}
          colors={colors}
          centerLabel={getReadableNumber(total)}
        />
        <ChartLegend
          className="common-chart p-2 w-[50%] z-0"
          categories={categories}
          colors={colors}
        />
      </div>
    </div>
  );
}

export default ConversionFunnelChart;
