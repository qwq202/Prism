import {
  ColumnsVisibilityBar,
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  useColumnsVisibility,
} from "@/components/ui/table.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import {
  Activity,
  ArrowDown10,
  Blocks,
  Check,
  Circle,
  Plus,
  RotateCw,
  Search,
  Settings2,
  Sheet,
  SquareAsterisk,
  Trash,
  Weight,
  Workflow,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import OperationAction from "@/components/OperationAction.tsx";
import { useEffect, useMemo, useState } from "react";
import { Channel, getShortChannelType } from "@/admin/channel.ts";
import { withNotify } from "@/api/common.ts";
import { useTranslation } from "react-i18next";
import { useEffectAsync } from "@/utils/hook.ts";
import {
  activateChannel,
  deactivateChannel,
  deleteChannel,
  listChannel,
  getChannelStats,
  type ChannelStat,
} from "@/admin/api/channel.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { getApiModels } from "@/api/v1.ts";
import { getHostName } from "@/utils/base.ts";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import { Label } from "@/components/ui/label.tsx";
import { Input } from "@/components/ui/input.tsx";
import { DialogClose } from "@radix-ui/react-dialog";
import { ChannelTypeAvatar } from "@/components/ModelAvatar.tsx";
import type { ChannelDispatch } from "@/components/admin/ChannelSettings.tsx";

type ChannelTableProps = {
  display: boolean;
  dispatch: ChannelDispatch;
  setId: (id: number) => void;
  setEnabled: (enabled: boolean) => void;
  data: Channel[];
  setData: (data: Channel[]) => void;
};

type TypeBadgeProps = {
  type: string;
  className?: string;
  variant?:
    | "default"
    | "secondary"
    | "destructive"
    | "outline"
    | "gold"
    | null
    | undefined;
};

export function TypeBadge({ type, className, variant }: TypeBadgeProps) {
  const content = useMemo(() => getShortChannelType(type), [type]);

  return (
    <Badge
      className={cn(`select-none w-max cursor-pointer`, className)}
      variant={variant}
    >
      {content || type}
    </Badge>
  );
}

// ── Health badge ──────────────────────────────────────────────────────────────
function HealthBadge({ stat }: { stat: ChannelStat | undefined }) {
  if (!stat || (stat.requests === 0 && stat.errors === 0)) {
    return (
      <span className="inline-flex items-center gap-1 text-xs text-muted-foreground/50 select-none">
        <Circle className="h-2 w-2 fill-current" />
        <span>—</span>
      </span>
    );
  }

  const rate = stat.error_rate;
  const total = stat.requests + stat.errors;

  let color: string;
  let label: string;
  if (rate === 0) {
    color = "text-green-500";
    label = "正常";
  } else if (rate < 0.1) {
    color = "text-yellow-500";
    label = `${(rate * 100).toFixed(0)}% 错误`;
  } else {
    color = "text-red-500";
    label = `${(rate * 100).toFixed(0)}% 错误`;
  }

  return (
    <span
      className={cn("inline-flex items-center gap-1 text-xs select-none whitespace-nowrap", color)}
      title={`今日 ${total} 次请求，${stat.errors} 次错误`}
    >
      <Circle className="h-2 w-2 fill-current shrink-0" />
      <span>{label}</span>
      <span className="text-muted-foreground/60">({total})</span>
    </span>
  );
}

// ── Summary bar ───────────────────────────────────────────────────────────────
function HealthSummaryBar({
  channels,
  statsMap,
  loading,
}: {
  channels: Channel[];
  statsMap: Map<number, ChannelStat>;
  loading: boolean;
}) {
  const active = channels.filter((c) => c.state);
  const withData = active.filter(
    (c) => statsMap.has(c.id) && (statsMap.get(c.id)!.requests + statsMap.get(c.id)!.errors) > 0,
  );
  const healthy = withData.filter((c) => (statsMap.get(c.id)?.error_rate ?? 0) === 0);
  const warn = withData.filter((c) => {
    const r = statsMap.get(c.id)?.error_rate ?? 0;
    return r > 0 && r < 0.1;
  });
  const critical = withData.filter((c) => (statsMap.get(c.id)?.error_rate ?? 0) >= 0.1);
  const idle = active.filter((c) => !statsMap.has(c.id) || (statsMap.get(c.id)!.requests + statsMap.get(c.id)!.errors) === 0);

  if (active.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 px-1 py-2 text-xs text-muted-foreground select-none">
      {loading ? (
        <span className="animate-pulse">加载健康数据…</span>
      ) : (
        <>
          <span className="font-medium text-foreground/70">今日健康状态</span>
          {healthy.length > 0 && (
            <span className="flex items-center gap-1 text-green-500">
              <Circle className="h-2 w-2 fill-current" />
              正常 {healthy.length}
            </span>
          )}
          {warn.length > 0 && (
            <span className="flex items-center gap-1 text-yellow-500">
              <Circle className="h-2 w-2 fill-current" />
              告警 {warn.length}
            </span>
          )}
          {critical.length > 0 && (
            <span className="flex items-center gap-1 text-red-500">
              <Circle className="h-2 w-2 fill-current" />
              异常 {critical.length}
            </span>
          )}
          {idle.length > 0 && (
            <span className="flex items-center gap-1 opacity-50">
              <Circle className="h-2 w-2 fill-current" />
              无流量 {idle.length}
            </span>
          )}
          <span className="ml-auto opacity-50">共 {active.length} 个启用渠道</span>
        </>
      )}
    </div>
  );
}

// ── Sync dialog ───────────────────────────────────────────────────────────────
type SyncDialogProps = {
  dispatch: ChannelDispatch;
  open: boolean;
  setOpen: (open: boolean) => void;
};

function SyncDialog({ dispatch, open, setOpen }: SyncDialogProps) {
  const { t } = useTranslation();
  const [endpoint, setEndpoint] = useState<string>("https://api.openai.com");
  const [secret, setSecret] = useState<string>("");

  const submit = async (endpoint: string): Promise<boolean> => {
    endpoint = endpoint.trim();
    endpoint.endsWith("/") && (endpoint = endpoint.slice(0, -1));

    const resp = await getApiModels(secret, { endpoint });
    withNotify(t, resp, true);

    if (!resp.status) return false;

    const name = getHostName(endpoint).replace(/\./g, "-");
    const data: Channel = {
      id: -1,
      name,
      type: "openai",
      models: resp.data,
      priority: 0,
      weight: 1,
      retry: 3,
      secret,
      endpoint,
      mapper: "",
      state: true,
      group: [],
      proxy: { proxy: "", proxy_type: 0, username: "", password: "" },
    };

    dispatch({ type: "set", value: data });
    return true;
  };

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.channels.joint")}</DialogTitle>
          </DialogHeader>
          <div className={`pt-2 flex flex-col`}>
            <div className={`flex flex-row items-center mb-4`}>
              <Label className={`mr-2 whitespace-nowrap`}>
                {t("admin.channels.joint-endpoint")}
              </Label>
              <Input
                value={endpoint}
                onChange={(e) => setEndpoint(e.target.value)}
                placeholder={t("admin.channels.upstream-endpoint-placeholder")}
              />
            </div>
            <div className={`flex flex-row items-center`}>
              <Label className={`mr-2 whitespace-nowrap`}>
                {t("admin.channels.secret")}
              </Label>
              <Input
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
                placeholder={t("admin.channels.sync-secret-placeholder")}
              />
            </div>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant={`outline`}>{t("cancel")}</Button>
            </DialogClose>
            <Button
              unClickable
              className={`mb-1`}
              onClick={async () => {
                const status = await submit(endpoint);
                status && setOpen(false);
              }}
            >
              {t("confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ── Main table component ──────────────────────────────────────────────────────
function ChannelTable({
  display,
  dispatch,
  setId,
  setEnabled,
  data,
  setData,
}: ChannelTableProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(false);
  const [open, setOpen] = useState<boolean>(false);
  const [blockDisplayType, setBlockDisplayType] = useState<boolean>(false);

  // ── Health state ──
  const [statsMap, setStatsMap] = useState<Map<number, ChannelStat>>(new Map());
  const [statsLoading, setStatsLoading] = useState<boolean>(false);

  const loadStats = async (channels: Channel[]) => {
    if (channels.length === 0) return;
    setStatsLoading(true);
    const ids = channels.map((c) => c.id);
    const resp = await getChannelStats(ids);
    const m = new Map<number, ChannelStat>();
    for (const s of resp.stats) m.set(s.channel_id, s);
    setStatsMap(m);
    setStatsLoading(false);
  };

  const refresh = async () => {
    setLoading(true);
    const resp = await listChannel();
    setLoading(false);
    if (!resp.status) withNotify(t, resp);
    else {
      setData(resp.data);
      await loadStats(resp.data);
    }
  };

  useEffectAsync(refresh, []);
  useEffectAsync(refresh, [display]);

  useEffect(() => {
    if (display) setId(-1);
  }, [display, setId]);

  const { bar, toggle, merge } = useColumnsVisibility(
    {
      id: true,
      name: true,
      type: true,
      priority: true,
      weight: true,
      ["secret-number"]: true,
      ["retry-name"]: true,
      health: true,
      state: true,
      action: true,
    },
    { translatePrefix: "admin.channels" },
  );

  const channels = useMemo(() => {
    const v = data || [];
    const s = search.trim().toLowerCase();
    if (s.trim() === "") return v;

    return v.filter((x) => {
      return (
        x.name.toLowerCase().includes(s) ||
        x.type.toLowerCase().includes(s) ||
        x.secret.toLowerCase().includes(s) ||
        x.models.some((m) => m.toLowerCase().includes(s))
      );
    });
  }, [search, data]);

  return (
    display && (
      <div>
        <SyncDialog
          open={open}
          setOpen={setOpen}
          dispatch={(action) => {
            dispatch(action);
            setEnabled(true);
            setId(-1);
          }}
        />
        <div className={`flex flex-row w-full h-max`}>
          <Button
            className={`mr-2 shrink-0`}
            onClick={() => {
              setEnabled(true);
              setId(-1);
            }}
          >
            <Plus className={`h-4 w-4 mr-1`} />
            {t("admin.channels.create")}
          </Button>
          <Button
            className={`mr-2 shrink-0`}
            variant={`outline`}
            onClick={() => setOpen(true)}
          >
            <Activity className={`h-4 w-4 mr-1`} />
            {t("admin.channels.joint")}
          </Button>
          <Button
            variant={`outline`}
            size={`icon`}
            className={`ml-auto`}
            onClick={() => setBlockDisplayType(!blockDisplayType)}
          >
            {blockDisplayType ? (
              <Blocks className={`h-4 w-4`} />
            ) : (
              <Sheet className={`h-4 w-4`} />
            )}
          </Button>
          <ColumnsVisibilityBar bar={bar} toggle={toggle} />
        </div>
        <div className={`flex flex-row items-center mt-4`}>
          <Button className={`shrink-0 mr-2`} size={`icon`}>
            <Search className={`h-4 w-4`} />
          </Button>
          <Input
            className={`grow`}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("admin.channels.search-channel")}
          />
          <Button
            variant={`outline`}
            size={`icon`}
            className={`ml-2 shrink-0`}
            onClick={refresh}
          >
            <RotateCw className={cn(`h-4 w-4`, loading && `animate-spin`)} />
          </Button>
        </div>

        {/* Health summary bar */}
        <HealthSummaryBar
          channels={data || []}
          statsMap={statsMap}
          loading={statsLoading}
        />

        {!blockDisplayType ? (
          <Table className={`channel-table mt-2`}>
            <TableHeader>
              <TableRow className={`select-none whitespace-nowrap`}>
                <TableCell className={merge("id")}>
                  {t("admin.channels.id")}
                </TableCell>
                <TableCell className={merge("name")}>
                  {t("admin.channels.name")}
                </TableCell>
                <TableCell className={merge("type")}>
                  {t("admin.channels.type")}
                </TableCell>
                <TableCell className={merge("priority")}>
                  {t("admin.channels.priority")}
                </TableCell>
                <TableCell className={merge("weight")}>
                  {t("admin.channels.weight")}
                </TableCell>
                <TableCell className={merge("secret-number")}>
                  {t("admin.channels.secret-number")}
                </TableCell>
                <TableCell className={merge("retry-name")}>
                  {t("admin.channels.retry-name")}
                </TableCell>
                <TableCell className={merge("health")}>今日状态</TableCell>
                <TableCell className={merge("state")}>
                  {t("admin.channels.state")}
                </TableCell>
                <TableCell className={merge("action")}>
                  {t("admin.channels.action")}
                </TableCell>
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.map((chan, idx) => (
                <TableRow key={idx}>
                  <TableCell className={merge("id", `channel-id select-none`)}>
                    #{chan.id}
                  </TableCell>
                  <TableCell className={merge("name")}>{chan.name}</TableCell>
                  <TableCell className={merge("type")}>
                    <TypeBadge type={chan.type} />
                  </TableCell>
                  <TableCell className={merge("priority")}>
                    {chan.priority}
                  </TableCell>
                  <TableCell className={merge("weight")}>
                    {chan.weight}
                  </TableCell>
                  <TableCell className={merge("secret-number")}>
                    {chan.secret.split("\n").filter((x) => x).length}
                  </TableCell>
                  <TableCell className={merge("retry-name")}>
                    {chan.retry}
                  </TableCell>
                  <TableCell className={merge("health")}>
                    <HealthBadge stat={statsMap.get(chan.id)} />
                  </TableCell>
                  <TableCell className={merge("state")}>
                    {chan.state ? (
                      <Check className={`h-4 w-4 text-green-500`} />
                    ) : (
                      <X className={`h-4 w-4 text-destructive`} />
                    )}
                  </TableCell>
                  <TableCell
                    className={merge("action", `flex flex-row flex-wrap gap-2`)}
                  >
                    <OperationAction
                      tooltip={t("admin.channels.edit")}
                      onClick={() => {
                        setEnabled(true);
                        setId(chan.id);
                      }}
                    >
                      <Settings2 className={`h-4 w-4`} />
                    </OperationAction>
                    {chan.state ? (
                      <OperationAction
                        tooltip={t("admin.channels.disable")}
                        variant={`destructive`}
                        onClick={async () => {
                          const resp = await deactivateChannel(chan.id);
                          withNotify(t, resp, true);
                          await refresh();
                        }}
                      >
                        <X className={`h-4 w-4`} />
                      </OperationAction>
                    ) : (
                      <OperationAction
                        tooltip={t("admin.channels.enable")}
                        onClick={async () => {
                          const resp = await activateChannel(chan.id);
                          withNotify(t, resp, true);
                          await refresh();
                        }}
                      >
                        <Check className={`h-4 w-4`} />
                      </OperationAction>
                    )}
                    <OperationAction
                      tooltip={t("admin.channels.delete")}
                      variant={`destructive`}
                      onClick={async () => {
                        const resp = await deleteChannel(chan.id);
                        withNotify(t, resp, true);
                        await refresh();
                      }}
                    >
                      <Trash className={`h-4 w-4`} />
                    </OperationAction>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <div className={`grid grid-cols-1 md:grid-cols-2 gap-4 mt-4`}>
            {channels.map((chan, idx) => (
              <div
                key={idx}
                onClick={() => {
                  setEnabled(true);
                  setId(chan.id);
                }}
                className={`flex flex-col p-4 border rounded-md cursor-pointer select-none hover:bg-background-hover transition`}
              >
                <div className={`flex flex-row items-center w-full`}>
                  <Circle
                    className={cn(
                      `h-3 w-3 stroke-[3.5] mr-1.5`,
                      chan.state ? `text-green-500` : `text-destructive`,
                    )}
                  />
                  <span className={`mr-1`}>{chan.name}</span>
                  <Badge variant={`outline`} className={`select-none`}>
                    #{chan.id}
                  </Badge>
                  <TypeBadge type={chan.type} className={`ml-auto`} />
                </div>
                <div className={`mt-1 grid grid-cols-2 gap-1`}>
                  <div className={`flex flex-row items-center`}>
                    <ArrowDown10 className={`h-3.5 w-3.5`} />
                    <Label className={`whitespace-nowrap ml-1 mr-2`}>
                      {t("admin.channels.priority")}
                    </Label>
                    <span className={`font-bold`}>{chan.priority}</span>
                  </div>
                  <div className={`flex flex-row items-center`}>
                    <Weight className={`h-3.5 w-3.5`} />
                    <Label className={`whitespace-nowrap ml-1 mr-2`}>
                      {t("admin.channels.weight")}
                    </Label>
                    <span className={`font-bold`}>{chan.weight}</span>
                  </div>
                  <div className={`flex flex-row items-center`}>
                    <SquareAsterisk className={`h-3.5 w-3.5`} />
                    <Label className={`whitespace-nowrap ml-1 mr-2`}>
                      {t("admin.channels.secret-number")}
                    </Label>
                    <span className={`font-bold`}>
                      {chan.secret.split("\n").filter((x) => x).length}
                    </span>
                  </div>
                  <div className={`flex flex-row items-center`}>
                    <Workflow className={`h-3.5 w-3.5`} />
                    <Label className={`whitespace-nowrap ml-1 mr-2`}>
                      {t("admin.channels.retry-name")}
                    </Label>
                    <span className={`font-bold`}>{chan.retry}</span>
                  </div>
                </div>
                {/* Health row in card view */}
                <div className="mt-2 pt-2 border-t">
                  <HealthBadge stat={statsMap.get(chan.id)} />
                </div>
                <div className={`flex flex-row items-center space-x-1 mt-2`}>
                  <OperationAction
                    tooltip={t("admin.channels.edit")}
                    onClick={() => {
                      setEnabled(true);
                      setId(chan.id);
                    }}
                  >
                    <Settings2 className={`h-4 w-4`} />
                  </OperationAction>
                  {chan.state ? (
                    <OperationAction
                      tooltip={t("admin.channels.disable")}
                      variant={`destructive`}
                      onClick={async () => {
                        const resp = await deactivateChannel(chan.id);
                        withNotify(t, resp, true);
                        await refresh();
                      }}
                    >
                      <X className={`h-4 w-4`} />
                    </OperationAction>
                  ) : (
                    <OperationAction
                      tooltip={t("admin.channels.enable")}
                      onClick={async () => {
                        const resp = await activateChannel(chan.id);
                        withNotify(t, resp, true);
                        await refresh();
                      }}
                    >
                      <Check className={`h-4 w-4`} />
                    </OperationAction>
                  )}
                  <OperationAction
                    tooltip={t("admin.channels.delete")}
                    variant={`destructive`}
                    onClick={async () => {
                      const resp = await deleteChannel(chan.id);
                      withNotify(t, resp, true);
                      await refresh();
                    }}
                  >
                    <Trash className={`h-4 w-4`} />
                  </OperationAction>
                  <div className={`grow`} />
                  <ChannelTypeAvatar type={chan.type} size={28} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  );
}

export default ChannelTable;
