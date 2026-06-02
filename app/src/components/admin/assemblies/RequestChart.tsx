import { useTranslation } from "react-i18next";
import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import { AreaChart } from "./recharts.tsx";
import { getReadableNumber } from "@/utils/processor.ts";

type RequestChartProps = {
  labels: string[];
  datasets: number[];
};

function RequestChart({ labels, datasets }: RequestChartProps) {
  const { t } = useTranslation();
  const requestsLabel = t("admin.requests");
  const data = useMemo(() => {
    return datasets.map((data, index) => ({
      date: labels[index],
      [requestsLabel]: data,
    }));
  }, [labels, datasets, requestsLabel]);

  const rpm = useMemo(() => {
    // request per month, sum of datasets
    return datasets.reduce((acc, curr) => acc + curr, 0);
  }, [datasets]);

  return (
    <div className={`chart`}>
      <div className={`chart-title mb-2`}>
        <p>{t("admin.request-chart")}</p>
        {labels.length === 0 && (
          <Loader2 className={`h-4 w-4 inline-block animate-spin`} />
        )}
        <div
          className={`ml-auto bg-blue-500/20 text-blue-500 px-1 rounded-sm text-xs py-0.5`}
        >
          RPM {getReadableNumber(rpm)}
        </div>
      </div>
      <AreaChart
        className={`common-chart`}
        data={data}
        categories={[requestsLabel]}
        index={"date"}
        colors={["blue"]}
        valueFormatter={(value) => getReadableNumber(value, 1)}
      />
    </div>
  );
}

export default RequestChart;
