import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { useState } from "react";
import { useSelector } from "react-redux";
import { selectInit } from "@/store/auth.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import {
  BroadcastInfo,
  createBroadcast,
  getBroadcastList,
  removeBroadcast,
  updateBroadcast,
} from "@/api/broadcast.ts";
import { useTranslation } from "react-i18next";
import { extractMessage } from "@/utils/processor.ts";
import { Button } from "@/components/ui/button.tsx";
import {
  AlertCircle,
  Edit,
  Eye,
  Loader2,
  MoreVertical,
  Plus,
  RotateCcw,
  Trash,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog.tsx";
import { Textarea } from "@/components/ui/textarea.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import EditorProvider from "@/components/EditorProvider.tsx";
import { Alert, AlertDescription } from "@/components/ui/alert.tsx";
import { withNotify } from "@/api/common.ts";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog.tsx";
import { DialogClose } from "@radix-ui/react-dialog";
import { toast } from "sonner";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Switch } from "@/components/ui/switch.tsx";

type BroadcastType = "broadcast" | "popup" | "banner";

const TYPE_COLORS: Record<BroadcastType, string> = {
  broadcast: "secondary",
  popup: "default",
  banner: "gold",
};

type CreateBroadcastDialogProps = {
  onCreated?: () => void;
};

function CreateBroadcastDialog(props: CreateBroadcastDialogProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState<boolean>(false);
  const [content, setContent] = useState<string>("");
  const [notifyAll, setNotifyAll] = useState<boolean>(false);
  const [type, setType] = useState<BroadcastType>("broadcast");
  const [startAt, setStartAt] = useState<string>("");
  const [endAt, setEndAt] = useState<string>("");
  const [isActive, setIsActive] = useState<boolean>(true);

  async function postBroadcast() {
    const broadcast = content.trim();
    if (broadcast.length === 0) return;
    const resp = await createBroadcast(broadcast, notifyAll, {
      type,
      start_at: startAt,
      end_at: endAt,
      is_active: isActive,
    });
    if (resp.status) {
      toast.success(t("admin.post-success"), {
        description: t("admin.post-success-prompt"),
      });
      setContent("");
      setNotifyAll(false);
      setType("broadcast");
      setStartAt("");
      setEndAt("");
      setIsActive(true);
      setOpen(false);
      props.onCreated?.();
    } else {
      toast.error(t("admin.post-failed"), {
        description: t("admin.post-failed-prompt", { reason: resp.error }),
      });
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant={`default`}>
          <Plus className={`w-4 h-4 mr-1`} />
          {t("admin.create-broadcast")}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("admin.create-broadcast")}</DialogTitle>
          <DialogDescription asChild>
            <div className={`pt-4 space-y-4`}>
              <Textarea
                placeholder={t("admin.broadcast-placeholder")}
                value={content}
                rows={4}
                onChange={(e) => setContent(e.target.value)}
              />

              <div className="flex flex-row items-center gap-3">
                <Label className="shrink-0">{t("admin.broadcast-type")}</Label>
                <Select value={type} onValueChange={(v) => setType(v as BroadcastType)}>
                  <SelectTrigger className="w-36">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="broadcast">{t("admin.broadcast-type-broadcast")}</SelectItem>
                    <SelectItem value="popup">{t("admin.broadcast-type-popup")}</SelectItem>
                    <SelectItem value="banner">{t("admin.broadcast-type-banner")}</SelectItem>
                  </SelectContent>
                </Select>
                <div className="flex items-center gap-2 ml-auto">
                  <Switch id="is-active-create" checked={isActive} onCheckedChange={setIsActive} />
                  <Label htmlFor="is-active-create">{t("admin.broadcast-active")}</Label>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <Label className="text-xs text-muted-foreground mb-1 block">{t("admin.broadcast-start-at")}</Label>
                  <Input
                    type="datetime-local"
                    value={startAt}
                    onChange={(e) => setStartAt(e.target.value)}
                  />
                </div>
                <div>
                  <Label className="text-xs text-muted-foreground mb-1 block">{t("admin.broadcast-end-at")}</Label>
                  <Input
                    type="datetime-local"
                    value={endAt}
                    onChange={(e) => setEndAt(e.target.value)}
                  />
                </div>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="notify-all"
                  checked={notifyAll}
                  onCheckedChange={(checked) => setNotifyAll(checked as boolean)}
                />
                <Label htmlFor="notify-all" className="text-sm font-medium text-primary cursor-pointer">
                  {t("admin.notify-all")}
                </Label>
              </div>
            </div>
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant={`outline`}>{t("admin.cancel")}</Button>
          </DialogClose>
          <Button unClickable variant={`default`} onClick={postBroadcast} loading={true}>
            {t("admin.post")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type BroadcastItemProps = {
  item: BroadcastInfo;
  onRefresh?: () => void;
};

function BroadcastItem({ item, onRefresh }: BroadcastItemProps) {
  const { t } = useTranslation();

  const [open, setOpen] = useState<boolean>(false);
  const [dialogOpen, setDialogOpen] = useState<boolean>(false);
  const [value, setValue] = useState<string>("");

  return (
    <TableRow>
      <EditorProvider
        title={t("admin.view")}
        value={value || item.content}
        onChange={setValue}
        open={open}
        setOpen={setOpen}
        submittable
        onSubmit={async (value: string) => {
          const resp = await updateBroadcast(item.index, value, {
            type: item.type,
            start_at: item.start_at,
            end_at: item.end_at,
            is_active: item.is_active,
          });
          withNotify(t, resp, true);
          onRefresh?.();
        }}
      />
      <AlertDialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <AlertDialogTrigger asChild></AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("admin.delete-broadcast")}</AlertDialogTitle>
            <AlertDialogDescription>
              <p>{t("admin.delete-broadcast-desc")}</p>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button unClickable variant={`outline`}>
              {t("cancel")}
            </Button>
            <Button
              unClickable
              variant={`destructive`}
              onClick={async () => {
                const resp = await removeBroadcast(item.index);
                withNotify(t, resp, true);
                onRefresh?.();
                if (resp.status) setDialogOpen(false);
              }}
            >
              {t("delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <TableCell>{item.index}</TableCell>
      <TableCell>
        <Badge variant={(TYPE_COLORS[item.type] ?? "secondary") as "secondary" | "default" | "gold"}>
          {t(`admin.broadcast-type-${item.type}`)}
        </Badge>
      </TableCell>
      <TableCell>
        <Badge variant={item.is_active ? "default" : "outline"}>
          {item.is_active ? t("admin.broadcast-active") : t("admin.broadcast-inactive")}
        </Badge>
      </TableCell>
      <TableCell>{extractMessage(item.content, 25)}</TableCell>
      <TableCell>{item.poster}</TableCell>
      <TableCell>
        {item.start_at || item.end_at ? (
          <span className="text-xs text-muted-foreground">
            {item.start_at || "∞"} → {item.end_at || "∞"}
          </span>
        ) : (
          <span className="text-xs text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell>{item.created_at}</TableCell>
      <TableCell>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant={`outline`} size={`icon`}>
              <MoreVertical className={`w-4 h-4`} />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align={`end`}>
            <DropdownMenuItem onClick={() => setOpen(true)}>
              <Eye className={`w-4 h-4 mr-1.5`} />
              {t("admin.view")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setOpen(true)}>
              <Edit className={`w-4 h-4 mr-1.5`} />
              {t("edit")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setDialogOpen(true)}>
              <Trash className={`w-4 h-4 mr-1.5`} />
              {t("delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableCell>
    </TableRow>
  );
}

function BroadcastTable() {
  const { t } = useTranslation();
  const init = useSelector(selectInit);
  const [data, setData] = useState<BroadcastInfo[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  const doRefresh = async () => {
    if (!init) return;

    setLoading(true);
    setData(await getBroadcastList());
    setLoading(false);
  };

  useEffectAsync(doRefresh, [init]);

  return (
    <div className={`broadcast-table whitespace-nowrap`}>
      <div className={`broadcast-action flex flex-row flex-nowrap w-full mb-4`}>
        <Button
          variant={`outline`}
          size={`icon`}
          className={`select-none`}
          onClick={async () => {
            setData(await getBroadcastList());
          }}
        >
          <RotateCcw className={`w-4 h-4`} />
        </Button>
        <div className={`grow`} />
        <CreateBroadcastDialog onCreated={doRefresh} />
      </div>
      <Alert className={`pb-2 mb-4`}>
        <AlertCircle className={`h-4 w-4`} />
        <AlertDescription className={`break-all whitespace-pre-wrap`}>
          {t("admin.broadcast-tip")}
        </AlertDescription>
      </Alert>

      {data.length ? (
        <Table>
          <TableHeader>
            <TableRow className={`select-none whitespace-nowrap`}>
              <TableHead>ID</TableHead>
              <TableHead>{t("admin.type")}</TableHead>
              <TableHead>{t("admin.state")}</TableHead>
              <TableHead>{t("admin.broadcast-content")}</TableHead>
              <TableHead>{t("admin.poster")}</TableHead>
              <TableHead>{t("admin.broadcast-schedule")}</TableHead>
              <TableHead>{t("admin.post-at")}</TableHead>
              <TableHead>{t("admin.action")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((item, idx) => (
              <BroadcastItem key={idx} item={item} onRefresh={doRefresh} />
            ))}
          </TableBody>
        </Table>
      ) : (
        <div className={`text-center select-none my-8`}>
          {loading ? (
            <Loader2 className={`w-6 h-6 inline-block animate-spin`} />
          ) : (
            <p className={`empty`}>{t("admin.empty")}</p>
          )}
        </div>
      )}
    </div>
  );
}

export default BroadcastTable;
