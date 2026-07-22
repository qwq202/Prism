import {
  Area,
  AreaChart as RechartsAreaChart,
  Bar,
  BarChart as RechartsBarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart as RechartsLineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { TooltipContentProps } from "recharts";
import type { ReactNode } from "react";
import { cn } from "@/components/ui/lib/utils.ts";
import { resolveChartColor } from "./chart-colors.ts";

const chartGridStroke = "hsl(var(--border))";
const chartTickColor = "hsl(var(--muted-foreground))";
const chartCursorStroke = "hsl(var(--foreground) / 0.35)";
const chartInitialDimension = { width: 1, height: 1 };

type ChartDatum = Record<string, string | number | undefined>;

type SeriesChartProps = {
  data: ChartDatum[];
  categories: string[];
  index: string;
  colors: string[];
  className?: string;
  stack?: boolean;
  showLegend?: boolean;
  valueFormatter?: (value: number) => string;
};

type BarChartProps = SeriesChartProps & {
  layout?: "horizontal" | "vertical";
};

type DonutDatum = {
  name: string;
  value: number;
};

type DonutChartProps = {
  data: DonutDatum[];
  colors: string[];
  className?: string;
  valueFormatter?: (value: number) => string;
  tooltipSuffix?: string;
  centerLabel?: ReactNode;
};

function formatValue(
  value: string | number | ReadonlyArray<string | number> | undefined,
  formatter?: (value: number) => string,
) {
  if (typeof value === "number") {
    return formatter ? formatter(value) : String(value);
  }

  return String(value ?? "");
}

function ChartTooltip({
  active,
  payload,
  valueFormatter,
}: TooltipContentProps & {
  valueFormatter?: (value: number) => string;
}) {
  if (!active || !payload?.length) return null;

  return (
    <div className="chart-tooltip min-w-44 w-max rounded-md border border-border bg-popover/95 p-2 text-sm text-popover-foreground shadow-lg backdrop-blur">
      <div className="space-y-1.5">
        {payload.map((item) => (
          <div
            className="flex items-center justify-between gap-8"
            key={`${String(item.dataKey)}-${String(item.name)}`}
          >
            <span className="inline-flex min-w-0 items-center gap-2 text-muted-foreground">
              <span
                className="h-2.5 w-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: item.color ?? item.fill }}
              />
              <span className="truncate">{String(item.name ?? "")}</span>
            </span>
            <span className="whitespace-nowrap font-medium text-foreground">
              {formatValue(item.value, valueFormatter)}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function DonutTooltip({
  active,
  payload,
  valueFormatter,
  tooltipSuffix,
}: TooltipContentProps & {
  valueFormatter?: (value: number) => string;
  tooltipSuffix?: string;
}) {
  if (!active || !payload?.length) return null;
  const item = payload[0];
  const name = String(item.name ?? "");
  const value =
    typeof item.value === "number"
      ? formatValue(item.value, valueFormatter)
      : String(item.value ?? "");

  return (
    <div className="chart-tooltip min-w-44 w-max rounded-md border border-border bg-popover/95 p-2 text-sm text-popover-foreground shadow-lg backdrop-blur">
      <div className="flex items-center justify-between gap-8">
        <span className="inline-flex min-w-0 items-center gap-2 text-muted-foreground">
          <span
            className="h-2.5 w-2.5 shrink-0 rounded-full"
            style={{ backgroundColor: item.color ?? item.fill }}
          />
          <span className="truncate">{name}</span>
        </span>
        <span className="whitespace-nowrap font-medium text-foreground">
          {value}
          {tooltipSuffix}
        </span>
      </div>
    </div>
  );
}

export function AreaChart({
  data,
  categories,
  index,
  colors,
  className,
  stack,
  valueFormatter,
}: SeriesChartProps) {
  return (
    <div className={cn("common-chart", className)}>
      <ResponsiveContainer
        width="100%"
        height="100%"
        minWidth={0}
        initialDimension={chartInitialDimension}
      >
        <RechartsAreaChart data={data} margin={{ left: 0, right: 8, top: 8 }}>
          <CartesianGrid
            stroke={chartGridStroke}
            strokeDasharray="3 3"
            strokeOpacity={0.7}
            vertical={false}
          />
          <XAxis
            dataKey={index}
            tickLine={false}
            axisLine={false}
            tick={{ fill: chartTickColor }}
          />
          <YAxis
            tickLine={false}
            axisLine={false}
            tick={{ fill: chartTickColor }}
            tickFormatter={(v) => formatValue(v, valueFormatter)}
          />
          <Tooltip
            cursor={{ stroke: chartCursorStroke, strokeWidth: 1 }}
            content={(props) => (
              <ChartTooltip {...props} valueFormatter={valueFormatter} />
            )}
          />
          {categories.map((category, idx) => {
            const color = resolveChartColor(colors[idx] ?? "blue");
            return (
              <Area
                key={category}
                type="monotone"
                dataKey={category}
                name={category}
                stackId={stack ? "stack" : undefined}
                stroke={color}
                fill={color}
                fillOpacity={0.2}
                isAnimationActive
              />
            );
          })}
        </RechartsAreaChart>
      </ResponsiveContainer>
    </div>
  );
}

export function BarChart({
  data,
  categories,
  index,
  colors,
  className,
  stack,
  layout = "horizontal",
  valueFormatter,
}: BarChartProps) {
  const vertical = layout === "vertical";

  return (
    <div className={cn("common-chart", className)}>
      <ResponsiveContainer
        width="100%"
        height="100%"
        minWidth={0}
        initialDimension={chartInitialDimension}
      >
        <RechartsBarChart
          data={data}
          layout={vertical ? "vertical" : "horizontal"}
          margin={{ left: vertical ? 8 : 0, right: 8, top: 8 }}
        >
          <CartesianGrid
            stroke={chartGridStroke}
            strokeDasharray="3 3"
            strokeOpacity={0.7}
            horizontal={!vertical}
            vertical={vertical}
          />
          {vertical ? (
            <>
              <XAxis
                type="number"
                tickLine={false}
                axisLine={false}
                tick={{ fill: chartTickColor }}
                tickFormatter={(v) => formatValue(v, valueFormatter)}
              />
              <YAxis
                type="category"
                dataKey={index}
                width={90}
                tickLine={false}
                axisLine={false}
                tick={{ fill: chartTickColor }}
              />
            </>
          ) : (
            <>
              <XAxis
                dataKey={index}
                tickLine={false}
                axisLine={false}
                tick={{ fill: chartTickColor }}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tick={{ fill: chartTickColor }}
                tickFormatter={(v) => formatValue(v, valueFormatter)}
              />
            </>
          )}
          <Tooltip
            cursor={{ fill: "hsl(var(--muted-foreground) / 0.08)" }}
            content={(props) => (
              <ChartTooltip {...props} valueFormatter={valueFormatter} />
            )}
          />
          {categories.map((category, idx) => (
            <Bar
              key={category}
              dataKey={category}
              name={category}
              stackId={stack ? "stack" : undefined}
              fill={resolveChartColor(colors[idx] ?? "blue")}
              isAnimationActive
            />
          ))}
        </RechartsBarChart>
      </ResponsiveContainer>
    </div>
  );
}

export function LineChart({
  data,
  categories,
  index,
  colors,
  className,
  valueFormatter,
}: SeriesChartProps) {
  return (
    <div className={cn("common-chart", className)}>
      <ResponsiveContainer
        width="100%"
        height="100%"
        minWidth={0}
        initialDimension={chartInitialDimension}
      >
        <RechartsLineChart data={data} margin={{ left: 0, right: 8, top: 8 }}>
          <CartesianGrid
            stroke={chartGridStroke}
            strokeDasharray="3 3"
            strokeOpacity={0.7}
            vertical={false}
          />
          <XAxis
            dataKey={index}
            tickLine={false}
            axisLine={false}
            tick={{ fill: chartTickColor }}
          />
          <YAxis
            tickLine={false}
            axisLine={false}
            tick={{ fill: chartTickColor }}
            tickFormatter={(v) => formatValue(v, valueFormatter)}
          />
          <Tooltip
            cursor={{ stroke: chartCursorStroke, strokeWidth: 1 }}
            content={(props) => (
              <ChartTooltip {...props} valueFormatter={valueFormatter} />
            )}
          />
          {categories.map((category, idx) => (
            <Line
              key={category}
              type="linear"
              dataKey={category}
              name={category}
              stroke={resolveChartColor(colors[idx] ?? "blue")}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
              isAnimationActive
            />
          ))}
        </RechartsLineChart>
      </ResponsiveContainer>
    </div>
  );
}

export function DonutChart({
  data,
  colors,
  className,
  valueFormatter,
  tooltipSuffix = "",
  centerLabel,
}: DonutChartProps) {
  return (
    <div className={cn("common-chart relative", className)}>
      <ResponsiveContainer
        width="100%"
        height="100%"
        minWidth={0}
        initialDimension={chartInitialDimension}
      >
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            innerRadius="72%"
            outerRadius="96%"
            paddingAngle={0}
            stroke="hsl(var(--background))"
            strokeWidth={2}
            isAnimationActive={false}
          >
            {data.map((item, idx) => (
              <Cell
                key={item.name}
                fill={resolveChartColor(colors[idx] ?? "blue")}
              />
            ))}
          </Pie>
          <Tooltip
            content={(props) => (
              <DonutTooltip
                {...props}
                valueFormatter={valueFormatter}
                tooltipSuffix={tooltipSuffix}
              />
            )}
          />
        </PieChart>
      </ResponsiveContainer>
      {centerLabel !== undefined && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center text-sm font-medium text-muted-foreground">
          {centerLabel}
        </div>
      )}
    </div>
  );
}

export function ChartLegend({
  categories,
  colors,
  className,
}: {
  categories: string[];
  colors: string[];
  className?: string;
}) {
  return (
    <div className={cn("space-y-2 text-sm text-muted-foreground", className)}>
      {categories.map((category, idx) => (
        <div className="flex min-w-0 items-center gap-2" key={category}>
          <span
            className="h-2.5 w-2.5 shrink-0 rounded-full"
            style={{
              backgroundColor: resolveChartColor(colors[idx] ?? "blue"),
            }}
          />
          <span className="truncate">{category}</span>
        </div>
      ))}
    </div>
  );
}
