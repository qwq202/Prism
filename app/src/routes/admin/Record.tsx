import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { useEffectAsync } from "@/utils/hook.ts";
import {
  listRecords,
  getRecordStats,
  type Record as BillingRecord,
  RecordQuery,
  RecordStats,
  RecordType,
  RecordTypes,
} from "@/api/record.ts";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { PaginationAction } from "@/components/ui/pagination.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { RotateCw, Search, DollarSign, Hash, Zap, Clock } from "lucide-react";
import { Switch } from "@/components/ui/switch.tsx";
import { mobile } from "@/utils/device.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { formatRecordTime } from "@/utils/record-time.ts";
import { useSelector } from "react-redux";
import { infoTimeZoneSelector } from "@/store/info.ts";
import { withNotify } from "@/api/common.ts";
import { getReadableTokenCount } from "@/utils/processor.ts";
import { getRecordCacheUsage } from "@/utils/record-cache.ts";

const defaultRecordQuery: RecordQuery = {
  type: RecordType.All,
  show_channel: true,
  cache_hit: false,
  cache_write: false,
};

const defaultRecordInput = {
  username: "",
  model: "",
  token_name: "",
  start_time: "",
  end_time: "",
  type: RecordType.All as RecordType,
  cache_hit: false,
  cache_write: false,
};

function StatCard({
  title,
  value,
  icon,
  className,
}: {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  className?: string;
}) {
  return (
    <Card className={cn("flex-1", className)}>
      <CardContent className="pt-4 pb-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="text-sm text-muted-foreground">{title}</p>
            <p
              className="invisible mt-2 truncate text-xs text-muted-foreground"
              aria-hidden="true"
            >
              —
            </p>
            <p className="mt-1 truncate text-2xl font-bold">{value}</p>
          </div>
          <div className="text-muted-foreground">{icon}</div>
        </div>
      </CardContent>
    </Card>
  );
}

function SplitStatCard({
  title,
  icon,
  metrics,
  className,
}: {
  title: string;
  icon: React.ReactNode;
  metrics: {
    label: string;
    value: string | number;
  }[];
  className?: string;
}) {
  return (
    <Card className={cn("flex-1", className)}>
      <CardContent className="pt-4 pb-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0 flex-1">
            <p className="text-sm text-muted-foreground">{title}</p>
            <div className="mt-2 grid grid-cols-2 gap-3">
              {metrics.map((metric) => (
                <div key={metric.label} className="min-w-0">
                  <p className="truncate text-xs text-muted-foreground">
                    {metric.label}
                  </p>
                  <p className="mt-1 truncate text-2xl font-bold">
                    {metric.value}
                  </p>
                </div>
              ))}
            </div>
          </div>
          <div className="text-muted-foreground">{icon}</div>
        </div>
      </CardContent>
    </Card>
  );
}

function RecordTypeLabel({ type }: { type: string }) {
  const { t } = useTranslation();
  const variants: {
    [key: string]: "default" | "secondary" | "destructive" | "outline";
  } = {
    consume: "destructive",
    topup: "default",
    system: "secondary",
  };
  return (
    <Badge variant={variants[type] ?? "outline"}>
      {t(`record.types.${type}`) || type}
    </Badge>
  );
}

function CacheUsageBadges({ record }: { record: BillingRecord }) {
  const { t } = useTranslation();
  const usage = getRecordCacheUsage(record);
  if (!usage) return null;

  return (
    <div className="mt-1 flex flex-wrap gap-1">
      {usage.hitTokens > 0 && (
        <Badge variant="secondary" className="text-[10px] font-medium">
          {t("record.cache-hit-short", { tokens: usage.hitTokens })}
        </Badge>
      )}
      {usage.missTokens > 0 && (
        <Badge variant="outline" className="text-[10px] font-medium">
          {t("record.cache-miss-short", { tokens: usage.missTokens })}
        </Badge>
      )}
      {usage.writeTokens > 0 && (
        <Badge variant="outline" className="text-[10px] font-medium">
          {t("record.cache-write-short", { tokens: usage.writeTokens })}
        </Badge>
      )}
    </div>
  );
}

function RecordTableSkeleton() {
  return (
    <>
      {Array.from({ length: 12 }).map((_, index) => (
        <TableRow
          key={index}
          className="pointer-events-none hover:bg-transparent"
        >
          <TableCell>
            <Skeleton className="h-5 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-7 w-16 rounded-full" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-36" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-14" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-14" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-12" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-36" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

function RecordTable() {
  const { t } = useTranslation();
  const timeZone = useSelector(infoTimeZoneSelector);
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(1);
  const [records, setRecords] = useState<BillingRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<RecordQuery>(defaultRecordQuery);
  const [input, setInput] = useState(defaultRecordInput);

  const sync = async (p = page, q = query) => {
    setLoading(true);
    try {
      const resp = await listRecords(p, {
        ...q,
        show_channel: true,
      });
      if (resp.status && resp.data) {
        setRecords(resp.data.records ?? []);
        setTotal(resp.data.total ?? 1);
      } else {
        withNotify(t, resp);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffectAsync(async () => {
    await sync();
  }, [page]);

  const handleSearch = async () => {
    const q: RecordQuery = {
      type: input.type,
      username: input.username || undefined,
      model: input.model || undefined,
      token_name: input.token_name || undefined,
      start_time: input.start_time || undefined,
      end_time: input.end_time || undefined,
      cache_hit: input.cache_hit || undefined,
      cache_write: input.cache_write || undefined,
      show_channel: true,
    };
    setQuery(q);
    setPage(0);
    await sync(0, q);
  };

  const handleReset = async () => {
    setInput(defaultRecordInput);
    setQuery(defaultRecordQuery);
    setPage(0);
    await sync(0, defaultRecordQuery);
  };

  const handleEnterSearch = async (
    event: React.KeyboardEvent<HTMLInputElement>,
  ) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    await handleSearch();
  };

  const initialLoading = loading && records.length === 0;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2 items-end">
        <Input
          placeholder={t("record.cond.username-placeholder")}
          value={input.username}
          onChange={(e) => setInput({ ...input, username: e.target.value })}
          onKeyDown={handleEnterSearch}
          className="w-36"
        />
        <Input
          placeholder={t("record.cond.model-placeholder")}
          value={input.model}
          onChange={(e) => setInput({ ...input, model: e.target.value })}
          onKeyDown={handleEnterSearch}
          className="w-36"
        />
        <Input
          placeholder={t("record.cond.token-name-placeholder")}
          value={input.token_name}
          onChange={(e) => setInput({ ...input, token_name: e.target.value })}
          onKeyDown={handleEnterSearch}
          className="w-36"
        />
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground px-1">
            {t("record.cond.start_time")}
          </span>
          <Input
            type="date"
            value={input.start_time}
            onChange={(e) => setInput({ ...input, start_time: e.target.value })}
            onKeyDown={handleEnterSearch}
            className="w-36"
          />
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="text-xs text-muted-foreground px-1">
            {t("record.cond.end_time")}
          </span>
          <Input
            type="date"
            value={input.end_time}
            onChange={(e) => setInput({ ...input, end_time: e.target.value })}
            onKeyDown={handleEnterSearch}
            className="w-36"
          />
        </div>
        <Select
          value={input.type}
          onValueChange={(v) => setInput({ ...input, type: v as RecordType })}
        >
          <SelectTrigger className="w-28">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {RecordTypes.map((type) => (
              <SelectItem key={type} value={type}>
                {t(`record.types.${type}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="flex h-10 items-center gap-2 rounded-md border px-3">
          <Switch
            checked={input.cache_hit}
            onCheckedChange={(cache_hit) => setInput({ ...input, cache_hit })}
          />
          <span className="whitespace-nowrap text-sm">
            {t("record.cache-hit-only")}
          </span>
        </div>
        <div className="flex h-10 items-center gap-2 rounded-md border px-3">
          <Switch
            checked={input.cache_write}
            onCheckedChange={(cache_write) =>
              setInput({ ...input, cache_write })
            }
          />
          <span className="whitespace-nowrap text-sm">
            {t("record.cache-write-only")}
          </span>
        </div>
        <Button onClick={handleSearch} variant="outline" size="icon">
          <Search className="w-4 h-4" />
        </Button>
        <Button
          onClick={handleReset}
          variant="outline"
          size="icon"
          disabled={loading}
        >
          <RotateCw className={cn("w-4 h-4", loading && "animate-spin")} />
        </Button>
      </div>

      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("record.user")}</TableHead>
              <TableHead>{t("record.type")}</TableHead>
              <TableHead>{t("record.model")}</TableHead>
              <TableHead>{t("record.token")}</TableHead>
              <TableHead>{t("record.input-tokens")}</TableHead>
              <TableHead>{t("record.output-tokens")}</TableHead>
              <TableHead>{t("record.quota")}</TableHead>
              <TableHead>{t("record.duration")}</TableHead>
              <TableHead>{t("record.channel")}</TableHead>
              <TableHead>{t("record.created-at")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {initialLoading && <RecordTableSkeleton />}
            {records.length === 0 && !loading && (
              <TableRow>
                <TableCell
                  colSpan={10}
                  className="text-center text-muted-foreground py-8"
                >
                  —
                </TableCell>
              </TableRow>
            )}
            {records.map((r, i) => (
              <TableRow key={i}>
                <TableCell className="font-medium">{r.username}</TableCell>
                <TableCell>
                  <RecordTypeLabel type={r.type} />
                </TableCell>
                <TableCell className="max-w-[120px] truncate">
                  {r.model}
                </TableCell>
                <TableCell className="max-w-[80px] truncate">
                  {r.token_name}
                </TableCell>
                <TableCell>
                  <div>{r.input_tokens}</div>
                  <CacheUsageBadges record={r} />
                </TableCell>
                <TableCell>{r.output_tokens}</TableCell>
                <TableCell>{r.quota.toFixed(4)}</TableCell>
                <TableCell>{r.duration.toFixed(1)}s</TableCell>
                <TableCell>{r.channel_name || r.channel || "—"}</TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {formatRecordTime(r.created_at, timeZone)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <PaginationAction
        current={page}
        total={total}
        offset
        onPageChange={setPage}
      />
    </div>
  );
}

function AdminRecord() {
  const { t } = useTranslation();
  const [stats, setStats] = useState<RecordStats | null>(null);

  useEffectAsync(async () => {
    const resp = await getRecordStats();
    if (resp.status && resp.data) setStats(resp.data);
    else withNotify(t, resp);
  }, []);

  return (
    <div className={cn("user-interface", mobile && "mobile")}>
      <div className="flex flex-wrap gap-3">
        <SplitStatCard
          title={t("record.types.consume")}
          icon={<DollarSign className="w-6 h-6" />}
          metrics={[
            {
              label: t("record.billing-today"),
              value: stats ? stats.billing_today.toFixed(2) : "—",
            },
            {
              label: t("record.billing-month"),
              value: stats ? stats.billing_month.toFixed(2) : "—",
            },
          ]}
        />
        <StatCard
          title={t("record.request-today")}
          value={stats ? stats.request_today : "—"}
          icon={<Zap className="w-6 h-6" />}
        />
        <StatCard
          title={t("record.rpm-tips")}
          value={stats ? stats.rpm : "—"}
          icon={<Clock className="w-6 h-6" />}
        />
        <StatCard
          title={t("record.tpm-tips")}
          value={stats ? stats.tpm : "—"}
          icon={<Clock className="w-6 h-6" />}
        />
        <StatCard
          title={t("record.total-tokens")}
          value={stats ? getReadableTokenCount(stats.total_tokens) : "—"}
          icon={<Hash className="w-6 h-6" />}
        />
      </div>

      <Card className="admin-card">
        <CardHeader className="select-none">
          <CardTitle>{t("record.title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <RecordTable />
        </CardContent>
      </Card>
    </div>
  );
}

export default AdminRecord;
