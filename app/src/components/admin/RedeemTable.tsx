import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog.tsx";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { RedeemBatch, RedeemForm, RedeemResponse } from "@/admin/types.ts";
import { Button, TemporaryButton } from "@/components/ui/button.tsx";
import {
  Copy,
  Download,
  FileDown,
  History,
  List,
  Loader2,
  RotateCw,
  Trash,
} from "lucide-react";
import {
  deleteRedeem,
  generateRedeem,
  getRedeemBatchCodes,
  getRedeemBatchList,
  getRedeemList,
} from "@/admin/api/chart.ts";
import { Input } from "@/components/ui/input.tsx";
import { Textarea } from "@/components/ui/textarea.tsx";
import { copyClipboard, saveAsFile } from "@/utils/dom.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { PaginationAction } from "@/components/ui/pagination.tsx";
import OperationAction from "@/components/OperationAction.tsx";
import { withNotify } from "@/api/common.ts";
import StateBadge from "@/components/admin/common/StateBadge.tsx";
import { toast } from "sonner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs.tsx";

function GenerateDialog({ update }: { update: () => void }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState<boolean>(false);
  const [quota, setQuota] = useState<string>("5");
  const [number, setNumber] = useState<string>("1");
  const [data, setData] = useState<string>("");

  function getNumber(value: string): string {
    return value.replace(/[^\d.]/g, "");
  }

  async function generateCode() {
    const data = await generateRedeem(Number(quota), Number(number));
    if (data.status) {
      setData(data.data.join("\n"));
      update();
    } else {
      toast.error(t("admin.error"), {
        description: data.message,
      });
    }
  }

  function close() {
    setQuota("5");
    setNumber("1");

    setOpen(false);
    setData("");
  }

  function downloadCode() {
    return saveAsFile("code.txt", data);
  }

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button>{t("admin.generate")}</Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.generate")}</DialogTitle>
            <DialogDescription className={`pt-2`}>
              <div className={`redeem-row`}>
                <p className={`mr-4`}>{t("admin.quota")}</p>
                <Input
                  value={quota}
                  onChange={(e) => setQuota(getNumber(e.target.value))}
                />
              </div>
              <div className={`redeem-row`}>
                <p className={`mr-4`}>{t("admin.number")}</p>
                <Input
                  value={number}
                  onChange={(e) => setNumber(getNumber(e.target.value))}
                />
              </div>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              unClickable
              variant={`outline`}
              onClick={() => setOpen(false)}
            >
              {t("admin.cancel")}
            </Button>
            <Button
              unClickable
              variant={`default`}
              loading={true}
              onClick={generateCode}
            >
              {t("admin.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog
        open={data !== ""}
        onOpenChange={(state: boolean) => {
          if (!state) close();
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("admin.generate-result")}</DialogTitle>
            <DialogDescription className={`pt-4`}>
              <Textarea value={data} rows={12} readOnly />
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button unClickable variant={`outline`} onClick={close}>
              {t("close")}
            </Button>
            <Button unClickable variant={`default`} onClick={downloadCode}>
              <Download className={`h-4 w-4 mr-2`} />
              {t("download")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function exportBatchCsv(batch: RedeemBatch, codes: { code: string; quota: number; used: boolean; created_at: string; updated_at: string }[]) {
  const header = "code,quota,used,created_at,updated_at";
  const rows = codes.map(
    (c) => `${c.code},${c.quota},${c.used},${c.created_at},${c.updated_at}`,
  );
  const content = [header, ...rows].join("\n");
  saveAsFile(`batch-${batch.id}.csv`, content);
}

function BatchHistoryTable() {
  const { t } = useTranslation();
  const [batches, setBatches] = useState<RedeemBatch[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [exporting, setExporting] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    const resp = await getRedeemBatchList();
    setLoading(false);
    if (resp.status) setBatches(resp.data);
  }

  useEffectAsync(load, []);

  async function downloadBatch(batch: RedeemBatch) {
    setExporting(batch.id);
    const resp = await getRedeemBatchCodes(batch.id);
    setExporting(null);
    if (resp.status) {
      exportBatchCsv(batch, resp.data);
    } else {
      toast.error(t("admin.error"), { description: resp.message });
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    );
  }

  if (batches.length === 0) {
    return <p className="empty">{t("admin.empty")}</p>;
  }

  return (
    <div>
      <Table>
        <TableHeader>
          <TableRow className="select-none whitespace-nowrap">
            <TableHead>{t("admin.redeem.batch-id")}</TableHead>
            <TableHead>{t("admin.quota")}</TableHead>
            <TableHead>{t("admin.redeem.count")}</TableHead>
            <TableHead>{t("admin.redeem.used-count")}</TableHead>
            <TableHead>{t("admin.created-at")}</TableHead>
            <TableHead>{t("admin.action")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {batches.map((batch) => (
            <TableRow key={batch.id} className="whitespace-nowrap">
              <TableCell>
                <code className="text-xs bg-muted px-1 py-0.5 rounded">
                  {batch.id}
                </code>
              </TableCell>
              <TableCell>
                <Badge variant="outline">{batch.quota}</Badge>
              </TableCell>
              <TableCell>{batch.count}</TableCell>
              <TableCell>
                <span className={batch.used_count === batch.count ? "text-muted-foreground" : "text-green-500"}>
                  {batch.used_count} / {batch.count}
                </span>
              </TableCell>
              <TableCell>{batch.created_at}</TableCell>
              <TableCell>
                <Button
                  size="icon"
                  variant="outline"
                  onClick={() => downloadBatch(batch)}
                  disabled={exporting === batch.id}
                >
                  {exporting === batch.id ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <FileDown className="h-4 w-4" />
                  )}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="flex justify-end mt-2">
        <Button variant="outline" size="icon" onClick={load}>
          <RotateCw className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

function RedeemTable() {
  const { t } = useTranslation();
  const [data, setData] = useState<RedeemForm>({
    total: 0,
    data: [],
  });
  const [loading, setLoading] = useState<boolean>(false);
  const [page, setPage] = useState<number>(0);

  async function update() {
    setLoading(true);
    const resp = await getRedeemList(page);
    setLoading(false);
    if (resp.status) setData(resp as RedeemResponse);
    else
      toast.error(t("admin.error"), {
        description: resp.message,
      });
  }
  useEffectAsync(update, [page]);

  return (
    <Tabs defaultValue="codes">
      <div className="flex items-center justify-between mb-3">
        <TabsList>
          <TabsTrigger value="codes">
            <List className="h-4 w-4 mr-1.5" />
            {t("admin.redeem.all-codes")}
          </TabsTrigger>
          <TabsTrigger value="batches">
            <History className="h-4 w-4 mr-1.5" />
            {t("admin.redeem.batch-history")}
          </TabsTrigger>
        </TabsList>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={update}>
            <RotateCw className="h-4 w-4" />
          </Button>
          <GenerateDialog update={update} />
        </div>
      </div>

      <TabsContent value="codes">
        <div className={`redeem-table`}>
          {(data.data && data.data.length > 0) || page > 0 ? (
            <>
              <Table>
                <TableHeader>
                  <TableRow className={`select-none whitespace-nowrap`}>
                    <TableHead>{t("admin.redeem.code")}</TableHead>
                    <TableHead>{t("admin.redeem.quota")}</TableHead>
                    <TableHead>{t("admin.used")}</TableHead>
                    <TableHead>{t("admin.created-at")}</TableHead>
                    <TableHead>{t("admin.used-at")}</TableHead>
                    <TableHead>{t("admin.action")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(data.data || []).map((redeem, idx) => (
                    <TableRow key={idx} className={`whitespace-nowrap`}>
                      <TableCell>{redeem.code}</TableCell>
                      <TableCell>
                        <Badge variant={`outline`}>{redeem.quota}</Badge>
                      </TableCell>
                      <TableCell>
                        <StateBadge state={redeem.used} />
                      </TableCell>
                      <TableCell>{redeem.created_at}</TableCell>
                      <TableCell>{redeem.updated_at}</TableCell>
                      <TableCell className={`flex gap-2`}>
                        <TemporaryButton
                          size={`icon`}
                          variant={`outline`}
                          onClick={() => copyClipboard(redeem.code)}
                        >
                          <Copy className={`h-4 w-4`} />
                        </TemporaryButton>
                        <OperationAction
                          native
                          tooltip={t("delete")}
                          variant={`destructive`}
                          onClick={async () => {
                            const resp = await deleteRedeem(redeem.code);
                            withNotify(t, resp, true);

                            resp.status && (await update());
                          }}
                        >
                          <Trash className={`h-4 w-4`} />
                        </OperationAction>
                      </TableCell>
                    </TableRow>
                  ))}
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
            <div className={`flex flex-col my-4 items-center`}>
              <Loader2 className={`w-6 h-6 inline-block animate-spin`} />
            </div>
          ) : (
            <p className={`empty`}>{t("admin.empty")}</p>
          )}
        </div>
      </TabsContent>

      <TabsContent value="batches">
        <BatchHistoryTable />
      </TabsContent>
    </Tabs>
  );
}

export default RedeemTable;
