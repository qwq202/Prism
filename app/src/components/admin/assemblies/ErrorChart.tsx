import { useTranslation } from "react-i18next";
import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import { AreaChart } from "./recharts.tsx";
import { getReadableNumber } from "@/utils/processor.ts";

type ErrorChartProps = {
  labels: string[];
  datasets: number[];
};
function ErrorChart({ labels, datasets }: ErrorChartProps) {
  const { t } = useTranslation();
  const timesLabel = t("admin.times");
  const data = useMemo(() => {
    return datasets.map((data, index) => ({
      date: labels[index],
      [timesLabel]: data,
    }));
  }, [labels, datasets, timesLabel]);

  return (
    <div className={`chart`}>
      <div className={`chart-title mb-2`}>
        <p>{t("admin.error-chart")}</p>
        {labels.length === 0 && (
          <Loader2 className={`h-4 w-4 inline-block animate-spin`} />
        )}
      </div>
      <AreaChart
        className={`common-chart`}
        data={data}
        categories={[timesLabel]}
        index={"date"}
        colors={["red"]}
        valueFormatter={(value) => getReadableNumber(value, 1)}
      />
    </div>
  );
}

export default ErrorChart;
