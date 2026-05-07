import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { DonutChart, Legend } from "@tremor/react";
import { getReadableNumber } from "@/utils/processor.ts";
import { ConversionFunnelResponse } from "@/admin/types.ts";

type ConversionFunnelChartProps = {
  data: ConversionFunnelResponse;
};

type CustomTooltipType = {
  payload?: { color?: string; name: string; value: number }[];
  active: boolean | undefined;
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

  const colors = ["emerald", "violet", "slate"];

  const customTooltip = ({ payload, active }: CustomTooltipType) => {
    if (!active || !payload) return null;
    const item = payload[0];
    if (!item) return null;
    return (
      <div className="chart-tooltip min-w-44 w-max z-10 rounded-tremor-default border border-tremor-border bg-tremor-background p-2 text-tremor-default shadow-tremor-dropdown">
        <div className="flex flex-1 space-x-2.5">
          <div className={`flex w-1.5 flex-col bg-${item.color} rounded`} />
          <div className="w-full">
            <div className="flex items-center justify-between space-x-8">
              <p className="whitespace-nowrap text-tremor-content">{item.name}</p>
              <p className="whitespace-nowrap font-medium text-tremor-content-emphasis">
                {getReadableNumber(item.value)}
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="chart">
      <div className="chart-title mb-2">
        <p>{t("admin.conversion-funnel")}</p>
        {loading && <Loader2 className="h-4 w-4 inline-block animate-spin" />}
      </div>
      <div className="flex flex-row">
        <DonutChart
          className="common-chart p-4 w-[50%]"
          variant="donut"
          data={chartData}
          showAnimation={true}
          valueFormatter={(v) => getReadableNumber(v)}
          customTooltip={customTooltip}
          colors={colors}
        />
        <Legend
          className="common-chart p-2 w-[50%] z-0"
          categories={categories}
          colors={colors}
        />
      </div>
    </div>
  );
}

export default ConversionFunnelChart;
