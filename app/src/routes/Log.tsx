import { useCallback, useEffect, useMemo, useState } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import { motion } from "framer-motion";
import { useTranslation } from "react-i18next";
import {
  Activity,
  BadgeCheck,
  Box,
  CircleDollarSign,
  Clock3,
  Cloud,
  Compass,
  FileText,
  History,
  KeySquare,
  Search,
  Timer,
  Hash,
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
import {
  formatRecordTime,
  toTimeZoneDateInputValue,
} from "@/utils/record-time.ts";
import { useSelector } from "react-redux";
import { infoTimeZoneSelector } from "@/store/info.ts";
import { withNotify } from "@/api/common.ts";
import { getReadableTokenCount } from "@/utils/processor.ts";

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

type SplitMetricCardProps = {
  title: string;
  icon: ReactNode;
  loading?: boolean;
  metrics: {
    label: string;
    value: string | number;
  }[];
};

const emptyStats: RecordStats = {
  billing_today: 0,
  billing_month: 0,
  request_today: 0,
  request_month: 0,
  rpm: 0,
  tpm: 0,
  total_tokens: 0,
};

const logContainerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      duration: 0.1,
      when: "beforeChildren",
      staggerChildren: 0.1,
    },
  },
};

const logItemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.5 },
  },
};

const logRowVariants = {
  hidden: { opacity: 0, y: 12 },
  visible: {
    opacity: 1,
    y: 0,
  },
};

function getInitialFilters(timeZone: string): LogFilters {
  const end = new Date();
  const start = new Date(end);
  start.setDate(start.getDate() - 6);

  return {
    type: RecordType.All,
    model: "",
    token_name: "",
    start_time: toTimeZoneDateInputValue(start, timeZone),
    end_time: toTimeZoneDateInputValue(end, timeZone),
  };
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
    <motion.div
      className="rounded-md border bg-card px-4 py-4 shadow-sm sm:px-5"
      variants={logItemVariants}
    >
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
    </motion.div>
  );
}

function SplitMetricCard({
  title,
  icon,
  loading,
  metrics,
}: SplitMetricCardProps) {
  return (
    <motion.div
      className="rounded-md border bg-card px-4 py-4 shadow-sm sm:px-5"
      variants={logItemVariants}
    >
      <div className="flex items-start justify-between gap-4">
        <p className="min-w-0 text-sm font-medium text-foreground">{title}</p>
        <div className="mt-1 text-foreground [&_svg]:h-6 [&_svg]:w-6 [&_svg]:stroke-[1.8]">
          {icon}
        </div>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-3">
        {metrics.map((metric) => (
          <div key={metric.label} className="min-w-0">
            <p className="truncate text-xs font-medium text-muted-foreground">
              {metric.label}
            </p>
            <div className="mt-1 flex min-h-7 items-center">
              {loading ? (
                <Skeleton className="h-7 w-20" />
              ) : (
                <span className="truncate text-2xl font-semibold leading-none tracking-normal">
                  {metric.value}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>
    </motion.div>
  );
}

function MobileRecordDetail({
  label,
  value,
}: {
  label: ReactNode;
  value: ReactNode;
}) {
  return (
    <div className="min-w-0 rounded-md bg-muted/35 px-3 py-2">
      <p className="truncate text-[11px] font-medium text-muted-foreground">
        {label}
      </p>
      <div className="mt-1 truncate text-sm font-semibold text-foreground">
        {value}
      </div>
    </div>
  );
}

function MobileLogSkeleton() {
  return (
    <>
      {Array.from({ length: 4 }).map((_, index) => (
        <div key={index} className="border-b p-4 last:border-b-0">
          <div className="flex items-center justify-between gap-3">
            <Skeleton className="h-5 w-36" />
            <Skeleton className="h-7 w-16 rounded-full" />
          </div>
          <div className="mt-4 space-y-2">
            <Skeleton className="h-5 w-44" />
            <Skeleton className="h-4 w-32" />
          </div>
          <div className="mt-4 grid grid-cols-2 gap-2">
            <Skeleton className="h-14 rounded-md" />
            <Skeleton className="h-14 rounded-md" />
            <Skeleton className="h-14 rounded-md" />
            <Skeleton className="h-14 rounded-md" />
          </div>
        </div>
      ))}
    </>
  );
}

function MobileLogRecord({
  record,
  timeZone,
}: {
  record: BillingRecord;
  timeZone: string;
}) {
  const { t } = useTranslation();

  return (
    <motion.div
      className="border-b p-4 last:border-b-0"
      variants={logRowVariants}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {formatRecordTime(record.created_at, timeZone)}
          </p>
          <p className="mt-1 truncate text-xs text-muted-foreground">
            {record.token_name || "—"}
          </p>
        </div>
        <RecordTypeLabel type={record.type} />
      </div>

      <div className="mt-4 flex min-w-0 items-center gap-2">
        <Box className="h-4 w-4 shrink-0 text-muted-foreground" />
        <p className="min-w-0 truncate text-base font-semibold">
          {record.model || "—"}
        </p>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2">
        <MobileRecordDetail
          label={t("record.input-tokens")}
          value={record.input_tokens}
        />
        <MobileRecordDetail
          label={t("record.output-tokens")}
          value={record.output_tokens}
        />
        <MobileRecordDetail
          label={t("record.duration")}
          value={`${record.duration.toFixed(1)}s`}
        />
        <MobileRecordDetail
          label={t("record.quota")}
          value={formatQuota(record.quota)}
        />
      </div>
    </motion.div>
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
        </TableRow>
      ))}
    </>
  );
}

function Log() {
  const { t } = useTranslation();
  const timeZone = useSelector(infoTimeZoneSelector);
  const [records, setRecords] = useState<BillingRecord[]>([]);
  const [stats, setStats] = useState<RecordStats>(emptyStats);
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(1);
  const [filters, setFilters] = useState<LogFilters>(() =>
    getInitialFilters(timeZone),
  );
  const [query, setQuery] = useState<LogFilters>(() =>
    getInitialFilters(timeZone),
  );
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);

  const initialLoading = loading && records.length === 0;
  const totalTokenLabel = useMemo(
    () => getReadableTokenCount(stats.total_tokens),
    [stats.total_tokens],
  );

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
      try {
        const resp = await listRecords(targetPage, {
          ...targetQuery,
          self: true,
          show_channel: false,
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
    },
    [page, query, t],
  );

  const syncStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const resp = await getRecordStats({ self: true });
      if (resp.status && resp.data) setStats(resp.data);
      else withNotify(t, resp);
    } finally {
      setStatsLoading(false);
    }
  }, [t]);

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
    <ScrollArea
      className="log-page h-full w-full bg-muted/25"
      classNameViewport="no-scrollbar"
    >
      <motion.div
        className="mx-auto flex w-full max-w-none flex-col gap-3 px-3 py-3 sm:gap-5 sm:px-5 sm:py-5"
        variants={logContainerVariants}
        initial="hidden"
        animate="visible"
      >
        <motion.div
          className="grid grid-cols-1 gap-3 md:grid-cols-2 md:gap-4 xl:grid-cols-4"
          variants={logContainerVariants}
        >
          <SplitMetricCard
            title={t("record.types.consume")}
            icon={<CircleDollarSign />}
            loading={statsLoading}
            metrics={[
              {
                label: t("record.billing-today"),
                value: stats.billing_today.toFixed(2),
              },
              {
                label: t("record.billing-month"),
                value: stats.billing_month.toFixed(2),
              },
            ]}
          />
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
          <MetricCard
            title={t("record.total-tokens")}
            value={totalTokenLabel}
            icon={<Hash />}
            loading={statsLoading}
          />
        </motion.div>

        <motion.div
          className="overflow-hidden rounded-md border bg-card shadow-sm"
          variants={logItemVariants}
        >
          <motion.div
            className="grid gap-x-4 gap-y-3 px-4 py-4 sm:px-5 sm:py-5 lg:grid-cols-[auto_minmax(0,1fr)_auto_minmax(0,1fr)]"
            variants={logItemVariants}
          >
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
                className="w-full px-5 sm:w-auto"
                disabled={loading}
                onClick={handleSearch}
                unClickable
              >
                <Search className="mr-2 h-4 w-4" />
                {t("record.query")}
              </Button>
            </div>
          </motion.div>

          <motion.div
            className="border-t md:hidden"
            variants={logContainerVariants}
          >
            {initialLoading && <MobileLogSkeleton />}
            {records.length === 0 && !loading && (
              <motion.div
                className="px-4 py-10 text-center text-sm text-muted-foreground"
                variants={logItemVariants}
              >
                —
              </motion.div>
            )}
            {records.map((record) => (
              <MobileLogRecord
                key={record.id}
                record={record}
                timeZone={timeZone}
              />
            ))}
          </motion.div>

          <motion.div
            className="no-scrollbar hidden overflow-x-auto border-t px-5 md:block"
            variants={logItemVariants}
          >
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
                </TableRow>
              </TableHeader>
              <TableBody>
                {initialLoading && <LogTableSkeleton />}
                {records.length === 0 && !loading && (
                  <motion.tr
                    className="border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted"
                    variants={logRowVariants}
                    initial="hidden"
                    animate="visible"
                    transition={{ duration: 0.35, ease: "easeOut" }}
                  >
                    <TableCell
                      colSpan={8}
                      className="h-24 text-center text-muted-foreground"
                    >
                      —
                    </TableCell>
                  </motion.tr>
                )}
                {records.map((record, index) => (
                  <motion.tr
                    key={record.id}
                    className="border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted"
                    variants={logRowVariants}
                    initial="hidden"
                    animate="visible"
                    transition={{
                      duration: 0.35,
                      ease: "easeOut",
                      delay: index * 0.035,
                    }}
                  >
                    <TableCell className="whitespace-nowrap text-muted-foreground">
                      {formatRecordTime(record.created_at, timeZone)}
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
                  </motion.tr>
                ))}
              </TableBody>
            </Table>
          </motion.div>

          <motion.div variants={logItemVariants}>
            <PaginationAction
              current={page}
              total={Math.max(total, 1)}
              offset
              onPageChange={setPage}
              className={cn("border-t py-6", records.length === 0 && "pt-7")}
            />
          </motion.div>
        </motion.div>
      </motion.div>
    </ScrollArea>
  );
}

export default Log;
