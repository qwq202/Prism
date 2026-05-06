import { useTranslation } from "react-i18next";
import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import { AreaChart } from "@tremor/react";
import { getReadableNumber } from "@/utils/processor.ts";

type ActiveUserChartProps = {
  labels: string[];
  datasets: number[];
};

const SERIES = "DAU";

function ActiveUserChart({ labels, datasets }: ActiveUserChartProps) {
  const { t } = useTranslation();
  const data = useMemo(() => {
    return datasets.map((v, i) => ({
      date: labels[i],
      [SERIES]: v,
    }));
  }, [labels, datasets]);

  const total = useMemo(() => datasets.reduce((a, b) => a + b, 0), [datasets]);

  return (
    <div className={`chart`}>
      <div className={`chart-title mb-2`}>
        <p>{t("admin.active-user-chart")}</p>
        {labels.length === 0 && (
          <Loader2 className={`h-4 w-4 inline-block animate-spin`} />
        )}
        <div className={`ml-auto bg-violet-500/20 text-violet-500 px-1 rounded-sm text-xs py-0.5`}>
          {getReadableNumber(total)} {t("admin.times")}
        </div>
      </div>
      <AreaChart
        className={`common-chart`}
        data={data}
        categories={[SERIES]}
        index={"date"}
        colors={["violet"]}
        showAnimation={true}
        valueFormatter={(v) => getReadableNumber(v, 1)}
      />
    </div>
  );
}

export default ActiveUserChart;
