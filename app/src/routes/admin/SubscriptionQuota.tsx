import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Clock, Loader2, RotateCw, Search } from "lucide-react";
import { toast } from "sonner";
import {
  getUserList,
  initialUserFilter,
  releaseAllUsageOperation,
  releaseUsageOperation,
  type ReleaseUsageType,
} from "@/admin/api/chart.ts";
import {
  UserData,
  UserResponse,
  UserSubscriptionWindowData,
} from "@/admin/types.ts";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Progress } from "@/components/ui/progress.tsx";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { PaginationAction } from "@/components/ui/pagination.tsx";
import { isEnter } from "@/utils/base.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import CommonAdminPage from "@/routes/admin/common/CommonAdminPage.tsx";
import type { TFunction } from "i18next";
import { PopupAlertDialog } from "@/components/PopupDialogComponent.tsx";

type QuotaWindowProps = {
  window: UserSubscriptionWindowData;
  now: number;
  onReset?: (type: ReleaseUsageType) => void;
};

type ResetAction =
  | {
      scope: "all";
      type: ReleaseUsageType;
    }
  | {
      scope: "user";
      type: ReleaseUsageType;
      user: UserData;
    };

const userTypeArray = ["normal", "basic_plan", "standard_plan", "pro_plan"];

function clampPercent(value: number): number {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, value));
}

function formatNumber(value: number): string {
  if (!Number.isFinite(value)) return "0";
  return value.toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function getWindowLabel(
  t: TFunction,
  window: UserSubscriptionWindowData,
): string {
  if (window.id === "plan_points") return t("admin.subscription-window-short");
  if (window.id === "plan_points_weekly")
    return t("admin.subscription-window-weekly");
  return window.name || window.id;
}

function getWindowUnit(
  t: TFunction,
  window: UserSubscriptionWindowData,
): string {
  if (window.unit === "points") return t("admin.subscription-window-points");
  if (window.unit === "times") return t("admin.subscription-window-times");
  return window.unit || t("admin.subscription-window-points");
}

function formatResetIn(
  t: TFunction,
  resetAt: string | undefined,
  now: number,
): string {
  if (!resetAt) return "-";
  const diff = new Date(resetAt).getTime() - now;
  if (!Number.isFinite(diff) || diff <= 0)
    return t("admin.subscription-quota-reset-now");

  const totalMinutes = Math.max(1, Math.floor(diff / 60000));
  const days = Math.floor(totalMinutes / 1440);
  const hours = Math.floor((totalMinutes % 1440) / 60);
  const minutes = totalMinutes % 60;
  const parts: string[] = [];

  if (days > 0) parts.push(t("sub.reset-days", { days }));
  if (hours > 0) parts.push(t("sub.reset-hours", { hours }));
  if (minutes > 0 || parts.length === 0)
    parts.push(t("sub.reset-minutes", { minutes }));

  return t("admin.subscription-quota-reset-in", {
    time: parts.join(" "),
  });
}

function getProgressClass(percent: number): string {
  if (percent <= 15) return "bg-destructive";
  if (percent <= 40) return "bg-amber-500";
  return "bg-emerald-500";
}

function getWindowReleaseType(
  window: UserSubscriptionWindowData,
): ReleaseUsageType | null {
  if (window.id === "plan_points") return "hour";
  if (window.id === "plan_points_weekly") return "week";
  return null;
}

function getReleaseWindowLabel(t: TFunction, type: ReleaseUsageType): string {
  return type === "week"
    ? t("admin.subscription-quota-reset-week")
    : t("admin.subscription-quota-reset-hour");
}

function QuotaWindow({ window, now, onReset }: QuotaWindowProps) {
  const { t } = useTranslation();
  const unit = getWindowUnit(t, window);
  const isUnlimited = window.total < 0 || window.remaining < 0;
  const percent = isUnlimited ? 100 : clampPercent(window.remaining_percent);
  const releaseType = getWindowReleaseType(window);
  const remaining = isUnlimited
    ? t("admin.subscription-window-unlimited")
    : `${formatNumber(window.remaining)} ${unit}`;
  const total = isUnlimited
    ? t("admin.subscription-window-unlimited")
    : `${formatNumber(window.total)} ${unit}`;

  return (
    <div className="min-w-[14rem] space-y-2 rounded-md border border-border/70 bg-background px-3 py-2">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0 font-medium">
          {getWindowLabel(t, window)}
        </div>
        <div className="shrink-0 text-sm tabular-nums text-muted-foreground">
          {formatNumber(percent)}%
        </div>
      </div>
      {!isUnlimited && (
        <Progress
          value={percent}
          className="h-1.5"
          classNameIndicator={getProgressClass(percent)}
        />
      )}
      <div className="grid grid-cols-2 gap-3 text-xs text-muted-foreground">
        <div>
          <div>{t("admin.subscription-quota-remaining")}</div>
          <div className="mt-0.5 font-medium text-foreground tabular-nums">
            {remaining}
          </div>
        </div>
        <div>
          <div>{t("admin.subscription-window-used")}</div>
          <div className="mt-0.5 font-medium text-foreground tabular-nums">
            {formatNumber(window.used)} / {total}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Clock className="h-3.5 w-3.5" />
        <span>{formatResetIn(t, window.reset_at, now)}</span>
      </div>
      {releaseType && onReset && (
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs"
          onClick={() => onReset(releaseType)}
        >
          <RotateCw className="mr-1 h-3 w-3" />
          {t("admin.subscription-quota-reset-window")}
        </Button>
      )}
    </div>
  );
}

function SubscriptionQuota() {
  const { t } = useTranslation();
  const [data, setData] = useState<UserResponse>({
    status: true,
    message: "",
    data: [],
    total: 0,
  });
  const [page, setPage] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [now, setNow] = useState(() => Date.now());
  const [resetAction, setResetAction] = useState<ResetAction | null>(null);

  const users = useMemo(
    () => data.data.filter((user) => user.is_subscribed),
    [data.data],
  );

  async function update() {
    setLoading(true);
    const resp = await getUserList(page, search, {
      ...initialUserFilter,
      plan: "yes",
      sort: "plan-desc",
    });
    setLoading(false);

    if (resp.status) {
      setData(resp);
      setNow(Date.now());
    } else {
      toast.error(t("admin.error"), { description: resp.message });
    }
  }

  async function executeReset(action: ResetAction): Promise<boolean> {
    const resp =
      action.scope === "all"
        ? await releaseAllUsageOperation(action.type)
        : await releaseUsageOperation(action.user.id, action.type);

    if (resp.status) {
      toast.success(t("admin.subscription-quota-reset-success"));
      await update();
      return true;
    }

    toast.error(t("admin.error"), { description: resp.message });
    return false;
  }

  const resetTargetLabel = resetAction
    ? resetAction.scope === "all"
      ? t("admin.subscription-quota-reset-target-all")
      : t("admin.subscription-quota-reset-target-user", {
          username: resetAction.user.username,
        })
    : "";
  const resetWindowLabel = resetAction
    ? getReleaseWindowLabel(t, resetAction.type)
    : "";

  useEffectAsync(update, [page]);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  return (
    <CommonAdminPage title="subscription-quota">
      <div className="space-y-5">
        <div className="flex flex-row">
          <Input
            className="search"
            placeholder={t("admin.subscription-quota-search")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={async (e) => {
              if (isEnter(e)) await update();
            }}
          />
          <Button
            size="icon"
            classNameWrapper="flex-shrink-0 ml-2"
            onClick={update}
            aria-label={t("admin.search-username")}
          >
            <Search className="h-4 w-4" />
          </Button>
          <Button
            size="icon"
            classNameWrapper="flex-shrink-0 ml-2"
            onClick={update}
            aria-label={t("admin.subscription-quota-refresh")}
          >
            <RotateCw className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            className="ml-2 whitespace-nowrap"
            onClick={() => setResetAction({ scope: "all", type: "hour" })}
          >
            <RotateCw className="mr-2 h-4 w-4" />
            {t("admin.subscription-quota-reset-all-hour")}
          </Button>
          <Button
            variant="outline"
            className="ml-2 whitespace-nowrap"
            onClick={() => setResetAction({ scope: "all", type: "week" })}
          >
            <RotateCw className="mr-2 h-4 w-4" />
            {t("admin.subscription-quota-reset-all-week")}
          </Button>
        </div>
        {users.length > 0 || page > 0 ? (
          <>
            <Table>
              <TableHeader>
                <TableRow className="select-none whitespace-nowrap">
                  <TableHead>ID</TableHead>
                  <TableHead>{t("admin.username")}</TableHead>
                  <TableHead>{t("admin.level")}</TableHead>
                  <TableHead>{t("admin.expired-at")}</TableHead>
                  <TableHead>{t("admin.subscription-windows")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((user: UserData) => {
                  const windows = user.subscription_windows ?? [];
                  return (
                    <TableRow key={user.id}>
                      <TableCell>{user.id}</TableCell>
                      <TableCell className="whitespace-nowrap">
                        {user.username}
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        <Badge variant="outline">
                          {t(
                            `admin.identity.${
                              userTypeArray[user.level] ?? userTypeArray[0]
                            }`,
                          )}
                        </Badge>
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        {user.expired_at || "-"}
                      </TableCell>
                      <TableCell>
                        {windows.length > 0 ? (
                          <div className="flex flex-wrap gap-2">
                            {windows.map((window) => (
                              <QuotaWindow
                                key={window.id}
                                window={window}
                                now={now}
                                onReset={(type) =>
                                  setResetAction({
                                    scope: "user",
                                    type,
                                    user,
                                  })
                                }
                              />
                            ))}
                          </div>
                        ) : (
                          <span className="text-muted-foreground">
                            {t("admin.subscription-quota-no-window")}
                          </span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
            <PaginationAction
              current={page}
              total={data.total}
              onPageChange={setPage}
              offset
            />
          </>
        ) : loading ? (
          <div className="flex flex-col mb-4 mt-12 items-center">
            <Loader2 className="w-6 h-6 inline-block animate-spin" />
          </div>
        ) : (
          <div className="empty">
            <p>{t("admin.empty")}</p>
          </div>
        )}
      </div>
      <PopupAlertDialog
        open={resetAction !== null}
        setOpen={(open) => {
          if (!open) setResetAction(null);
        }}
        title={t("admin.subscription-quota-reset-confirm-title")}
        description={t("admin.subscription-quota-reset-confirm-desc", {
          target: resetTargetLabel,
          window: resetWindowLabel,
        })}
        confirmLabel={t("admin.subscription-quota-reset-window")}
        destructive
        onSubmit={async () => {
          if (!resetAction) return false;
          return executeReset(resetAction);
        }}
      />
    </CommonAdminPage>
  );
}

export default SubscriptionQuota;
