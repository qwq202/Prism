import { useCallback, useEffect, useMemo, useState } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  Activity,
  BadgeCheck,
  Box,
  CalendarDays,
  CircleDollarSign,
  Clock3,
  Cloud,
  Compass,
  FileBox,
  FileText,
  Hash,
  History,
  KeySquare,
  Search,
  Timer,
} from "lucide-react";
import {
  getRecordStats,
  listRecords,
  type Record as BillingRecord,
  RecordStats,
  RecordType,
  RecordTypes,
} from "@/api/record.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import DatePicker from "@/components/ui/date-picker.tsx";
import { Input } from "@/components/ui/input.tsx";
import { PaginationAction } from "@/components/ui/pagination.tsx";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { cn } from "@/components/ui/lib/utils.ts";

type LogFilters = {
  type: RecordType;
  model: string;
  token_name: string;
  start_time: string;
  end_time: string;
};

type MetricCardProps = {
  title: string;
  value: string | number;
  icon: ReactNode;
  loading?: boolean;
  children?: ReactNode;
};

const emptyStats: RecordStats = {
  billing_today: 0,
  billing_month: 0,
  request_today: 0,
  request_month: 0,
  rpm: 0,
  tpm: 0,
};

function toDateInputValue(date: Date) {
  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function getInitialFilters(): LogFilters {
  const end = new Date();
  const start = new Date(end);
  start.setDate(start.getDate() - 6);

  return {
    type: RecordType.All,
    model: "",
    token_name: "",
    start_time: toDateInputValue(start),
    end_time: toDateInputValue(end),
  };
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value || "—";
  return date.toLocaleString();
}

function formatQuota(value: number) {
  return Number.isFinite(value) ? value.toFixed(4).replace(/\.?0+$/, "") : "0";
}

function MetricCard({
  title,
  value,
  icon,
  loading,
  children,
}: MetricCardProps) {
  return (
    <div className="rounded-md border bg-card px-5 py-4 shadow-sm">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-medium text-foreground">{title}</p>
          <div className="mt-2 flex min-h-7 items-center gap-1.5">
            {loading ? (
              <Skeleton className="h-7 w-20" />
            ) : (
              <span className="text-2xl font-semibold leading-none tracking-normal">
                {value}
              </span>
            )}
            {children}
          </div>
        </div>
        <div className="mt-1 text-foreground [&_svg]:h-6 [&_svg]:w-6 [&_svg]:stroke-[1.8]">
          {icon}
        </div>
      </div>
    </div>
  );
}

function FieldLabel({
  icon,
  children,
}: {
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex h-10 items-center gap-2 whitespace-nowrap text-sm font-medium text-foreground">
      <span className="text-foreground [&_svg]:h-4 [&_svg]:w-4 [&_svg]:stroke-[1.8]">
        {icon}
      </span>
      <span>{children}</span>
    </div>
  );
}

function TokenUnit() {
  return (
    <span className="ml-1 rounded bg-muted px-1.5 py-1 text-[10px] font-medium uppercase text-muted-foreground">
      tokens
    </span>
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

function LogTableSkeleton() {
  return (
    <>
      {Array.from({ length: 8 }).map((_, index) => (
        <TableRow
          key={index}
          className="pointer-events-none hover:bg-transparent"
        >
          <TableCell>
            <Skeleton className="h-5 w-32" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-7 w-16 rounded-full" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-24" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-36" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-14" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-14" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-12" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-16" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-28" />
          </TableCell>
          <TableCell>
            <Skeleton className="h-5 w-24" />
          </TableCell>
        </TableRow>
      ))}
    </>
  );
}

function Log() {
  const { t } = useTranslation();
  const [records, setRecords] = useState<BillingRecord[]>([]);
  const [stats, setStats] = useState<RecordStats>(emptyStats);
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(1);
  const [filters, setFilters] = useState<LogFilters>(() => getInitialFilters());
  const [query, setQuery] = useState<LogFilters>(() => getInitialFilters());
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);

  const initialLoading = loading && records.length === 0;

  const requestBadges = useMemo(
    () => (
      <>
        <Badge className="bg-blue-600 text-white hover:bg-blue-600">
          {stats.rpm} RPM
        </Badge>
        <Badge className="bg-violet-600 text-white hover:bg-violet-600">
          {stats.tpm} TPM
        </Badge>
      </>
    ),
    [stats.rpm, stats.tpm],
  );

  const syncRecords = useCallback(
    async (targetPage = page, targetQuery = query) => {
      setLoading(true);
      const resp = await listRecords(targetPage, {
        ...targetQuery,
        self: true,
        show_channel: false,
      });
      if (resp.status && resp.data) {
        setRecords(resp.data.records ?? []);
        setTotal(resp.data.total ?? 1);
      }
      setLoading(false);
    },
    [page, query],
  );

  const syncStats = useCallback(async () => {
    setStatsLoading(true);
    const resp = await getRecordStats({ self: true });
    if (resp.status && resp.data) setStats(resp.data);
    setStatsLoading(false);
  }, []);

  useEffect(() => {
    void syncStats();
  }, [syncStats]);

  useEffect(() => {
    void syncRecords(page, query);
  }, [page, query, syncRecords]);

  async function handleSearch() {
    setQuery(filters);
    if (page !== 0) {
      setPage(0);
      return;
    }
    await syncRecords(0, filters);
  }

  const handleEnterSearch = async (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    await handleSearch();
  };

  return (
    <ScrollArea className="h-full w-full bg-muted/25">
      <div className="mx-auto flex w-full max-w-none flex-col gap-5 px-5 py-5">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            title={t("record.billing-today")}
            value={stats.billing_today.toFixed(2)}
            icon={<CircleDollarSign />}
            loading={statsLoading}
          >
            <Cloud className="h-3.5 w-3.5 stroke-[1.8]" />
          </MetricCard>
          <MetricCard
            title={t("record.billing-month")}
            value={stats.billing_month.toFixed(2)}
            icon={<CalendarDays />}
            loading={statsLoading}
          >
            <Cloud className="h-3.5 w-3.5 stroke-[1.8]" />
          </MetricCard>
          <MetricCard
            title={t("record.request-today")}
            value={stats.request_today}
            icon={<Activity />}
            loading={statsLoading}
          >
            <Clock3 className="h-3.5 w-3.5 stroke-[1.8]" />
            <span className="flex gap-1">{requestBadges}</span>
          </MetricCard>
          <MetricCard
            title={t("record.request-month")}
            value={stats.request_month}
            icon={<BadgeCheck />}
            loading={statsLoading}
          >
            <Clock3 className="h-3.5 w-3.5 stroke-[1.8]" />
          </MetricCard>
        </div>

        <div className="overflow-hidden rounded-md border bg-card shadow-sm">
          <div className="grid gap-x-4 gap-y-3 px-5 py-5 lg:grid-cols-[auto_minmax(0,1fr)_auto_minmax(0,1fr)]">
            <FieldLabel icon={<FileText />}>{t("record.cond.type")}</FieldLabel>
            <Select
              value={filters.type}
              onValueChange={(value) =>
                setFilters({ ...filters, type: value as RecordType })
              }
            >
              <SelectTrigger>
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

            <FieldLabel icon={<Box />}>{t("record.cond.model")}</FieldLabel>
            <Input
              placeholder={t("record.cond.model-placeholder")}
              value={filters.model}
              onChange={(event) =>
                setFilters({ ...filters, model: event.target.value })
              }
              onKeyDown={handleEnterSearch}
            />

            <FieldLabel icon={<Compass />}>
              {t("record.cond.start_time")}
            </FieldLabel>
            <DatePicker
              classNameTrigger="h-10 w-full"
              value={filters.start_time}
              onValueChange={(value) =>
                setFilters({ ...filters, start_time: value })
              }
            />

            <FieldLabel icon={<Compass />}>
              {t("record.cond.end_time")}
            </FieldLabel>
            <DatePicker
              classNameTrigger="h-10 w-full"
              value={filters.end_time}
              onValueChange={(value) =>
                setFilters({ ...filters, end_time: value })
              }
            />

            <FieldLabel icon={<KeySquare />}>
              {t("record.cond.token-name")}
            </FieldLabel>
            <Input
              placeholder={t("record.cond.token-name-placeholder")}
              value={filters.token_name}
              onChange={(event) =>
                setFilters({ ...filters, token_name: event.target.value })
              }
              onKeyDown={handleEnterSearch}
            />
            <div className="hidden lg:block" />
            <div className="hidden lg:block" />

            <div className="lg:col-span-4">
              <Button
                className="px-5"
                disabled={loading}
                onClick={handleSearch}
                unClickable
              >
                <Search className="mr-2 h-4 w-4" />
                {t("record.query")}
              </Button>
            </div>
          </div>

          <div className="overflow-x-auto border-t px-5">
            <Table>
              <TableHeader>
                <TableRow className="select-none whitespace-nowrap">
                  <TableHead className="min-w-[150px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <History className="h-4 w-4" />
                      {t("record.created-at")}
                    </span>
                  </TableHead>
                  <TableHead className="text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Compass className="h-4 w-4" />
                      {t("record.type")}
                    </span>
                  </TableHead>
                  <TableHead className="min-w-[130px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <KeySquare className="h-4 w-4" />
                      {t("record.token")}
                    </span>
                  </TableHead>
                  <TableHead className="min-w-[170px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Box className="h-4 w-4" />
                      {t("record.model")}
                    </span>
                  </TableHead>
                  <TableHead className="text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Cloud className="h-4 w-4" />
                      {t("record.input-tokens")}
                      <TokenUnit />
                    </span>
                  </TableHead>
                  <TableHead className="text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Cloud className="h-4 w-4" />
                      {t("record.output-tokens")}
                      <TokenUnit />
                    </span>
                  </TableHead>
                  <TableHead className="text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Timer className="h-4 w-4" />
                      {t("record.duration")}
                    </span>
                  </TableHead>
                  <TableHead className="text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <CircleDollarSign className="h-4 w-4" />
                      {t("record.quota")}
                    </span>
                  </TableHead>
                  <TableHead className="min-w-[180px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <Hash className="h-4 w-4" />
                      {t("record.prompt")}
                    </span>
                  </TableHead>
                  <TableHead className="min-w-[140px] text-muted-foreground">
                    <span className="inline-flex items-center gap-1.5">
                      <FileBox className="h-4 w-4" />
                      {t("record.detail")}
                    </span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {initialLoading && <LogTableSkeleton />}
                {records.length === 0 && !loading && (
                  <TableRow>
                    <TableCell
                      colSpan={10}
                      className="h-24 text-center text-muted-foreground"
                    >
                      —
                    </TableCell>
                  </TableRow>
                )}
                {records.map((record) => (
                  <TableRow key={record.id}>
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      {formatDateTime(record.created_at)}
                    </TableCell>
                    <TableCell>
                      <RecordTypeLabel type={record.type} />
                    </TableCell>
                    <TableCell className="max-w-[130px] truncate">
                      {record.token_name || "—"}
                    </TableCell>
                    <TableCell className="max-w-[180px] truncate font-medium">
                      {record.model || "—"}
                    </TableCell>
                    <TableCell>{record.input_tokens}</TableCell>
                    <TableCell>{record.output_tokens}</TableCell>
                    <TableCell>{record.duration.toFixed(1)}s</TableCell>
                    <TableCell>{formatQuota(record.quota)}</TableCell>
                    <TableCell className="max-w-[220px] truncate">
                      {record.prompts || "—"}
                    </TableCell>
                    <TableCell className="max-w-[180px] truncate text-muted-foreground">
                      {record.detail || "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <PaginationAction
            current={page}
            total={Math.max(total, 1)}
            offset
            onPageChange={setPage}
            className={cn("border-t py-6", records.length === 0 && "pt-7")}
          />
        </div>
      </div>
    </ScrollArea>
  );
}

export default Log;
