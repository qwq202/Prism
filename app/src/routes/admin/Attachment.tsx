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
  Attachment,
  deleteAttachment,
  listAttachments,
} from "@/admin/api/attachment.ts";
import { getSizeUnit } from "@/utils/base.ts";
import { withNotify } from "@/api/common.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import { ExternalLink, RotateCw, Trash2 } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import { mobile } from "@/utils/device.ts";

function AdminAttachment() {
  const { t } = useTranslation();
  const [data, setData] = useState<Attachment[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  const sync = async () => {
    if (loading) return;
    setLoading(true);
    const res = await listAttachments();
    if (res.status) {
      setData(res.data);
    } else {
      withNotify(t, res);
    }
    setLoading(false);
  };

  useEffectAsync(async () => {
    await sync();
  }, []);

  const onDelete = async (item: Attachment) => {
    const confirmed = window.confirm(
      item.referenced
        ? t("admin.attachment.delete-referenced-confirm", {
            name: item.name,
            count: item.reference_count,
          })
        : t("admin.attachment.delete-confirm", { name: item.name }),
    );
    if (!confirmed) return;

    const res = await deleteAttachment(item.name, item.referenced);
    withNotify(t, res, true);
    if (res.status) await sync();
  };

  return (
    <div className={cn("user-interface", mobile && "mobile")}>
      <Card className="admin-card">
        <CardHeader className="select-none">
          <div className="flex items-center gap-2">
            <CardTitle>{t("admin.attachment.title")}</CardTitle>
            <div className="grow" />
            <Button onClick={sync} variant="outline" size="icon">
              <RotateCw className={cn("w-4 h-4", loading && "animate-spin")} />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-sm text-muted-foreground mb-4">
            {t("admin.attachment.description")}
          </div>

          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("admin.attachment.file-name")}</TableHead>
                  <TableHead>{t("admin.attachment.storage-mode")}</TableHead>
                  <TableHead>{t("admin.attachment.file-size")}</TableHead>
                  <TableHead>{t("admin.attachment.reference-status")}</TableHead>
                  <TableHead>{t("admin.attachment.updated-at")}</TableHead>
                  <TableHead>{t("admin.attachment.action")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.length === 0 && !loading && (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className="text-center text-muted-foreground py-8"
                    >
                      {t("admin.attachment.empty")}
                    </TableCell>
                  </TableRow>
                )}
                {data.map((item) => (
                  <TableRow key={item.name}>
                    <TableCell className="font-medium max-w-[280px]">
                      <div className="break-all whitespace-pre-wrap">
                        {item.name}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="uppercase">
                        {item.storage_mode}
                      </Badge>
                    </TableCell>
                    <TableCell>{getSizeUnit(item.size)}</TableCell>
                    <TableCell>
                      <Badge
                        variant={item.referenced ? "default" : "secondary"}
                        className={cn(
                          !item.referenced && "text-muted-foreground",
                        )}
                      >
                        {item.referenced
                          ? t("admin.attachment.referenced-count", {
                              count: item.reference_count,
                            })
                          : t("admin.attachment.orphan")}
                      </Badge>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                      {new Date(item.updated_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            window.open(
                              item.public_url,
                              "_blank",
                              "noopener,noreferrer",
                            )
                          }
                          title={t("admin.attachment.open")}
                        >
                          <ExternalLink className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => onDelete(item)}
                          title={t("delete")}
                        >
                          <Trash2 className="w-4 h-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default AdminAttachment;
