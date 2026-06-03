import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Loader2 } from "lucide-react";
import { AreaChart } from "./recharts.tsx";
import { ChannelStat } from "@/admin/api/channel.ts";
import { Channel } from "@/admin/channel.ts";
import { getReadableNumber } from "@/utils/processor.ts";

type ChannelHealthChartProps = {
  stats: ChannelStat[];
  channels: Channel[];
};

function ChannelHealthChart({ stats, channels }: ChannelHealthChartProps) {
  const { t } = useTranslation();
  const errorsLabel = t("admin.times");

  const nameMap = useMemo(() => {
    const m = new Map<number, string>();
    for (const ch of channels) m.set(ch.id, ch.name);
    return m;
  }, [channels]);

  const data = useMemo(() => {
    return stats
      .filter((s) => s.requests + s.errors > 0)
      .sort((a, b) => b.requests + b.errors - (a.requests + a.errors))
      .slice(0, 10)
      .map((s) => ({
        name: nameMap.get(s.channel_id) ?? `#${s.channel_id}`,
        [errorsLabel]: s.errors,
      }));
  }, [stats, nameMap, errorsLabel]);

  const loading = stats.length === 0 && channels.length === 0;

  return (
    <div className={`chart`}>
      <div className={`chart-title mb-2`}>
        <p>{t("admin.channel-health-chart")}</p>
        {loading && <Loader2 className={`h-4 w-4 inline-block animate-spin`} />}
        <div className={`ml-auto bg-emerald-500/20 text-emerald-500 px-1 rounded-sm text-xs py-0.5`}>
          {t("admin.today")}
        </div>
      </div>
      {data.length === 0 && !loading ? (
        <div className="flex items-center justify-center h-32 text-muted-foreground text-sm">
          {t("admin.empty")}
        </div>
      ) : (
        <AreaChart
          className={`common-chart`}
          data={data}
          categories={[errorsLabel]}
          index={"name"}
          colors={["red"]}
          valueFormatter={(v) => getReadableNumber(v, 1)}
        />
      )}
    </div>
  );
}

export default ChannelHealthChart;
