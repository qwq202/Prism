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
  getPaymentOrders,
  recheckOrderStatus,
  PaymentOrder,
} from "@/payment/request.ts";
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
import { PaginationAction } from "@/components/ui/pagination.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Copy, RotateCw, Search } from "lucide-react";
import { mobile } from "@/utils/device.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { copyClipboard } from "@/utils/dom.ts";
import { toast } from "sonner";
import { withNotify } from "@/api/common.ts";

function PaymentStatusBadge({ state }: { state: boolean }) {
  const { t } = useTranslation();
  return (
    <Badge variant={state ? "default" : "secondary"}>
      {t(state ? "admin.pay.status-true" : "admin.pay.status-false")}
    </Badge>
  );
}

function PaymentTable() {
  const { t } = useTranslation();
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(1);
  const [orders, setOrders] = useState<PaymentOrder[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState("");
  const [recheckingId, setRecheckingId] = useState<string | null>(null);

  const sync = async (p = page, s = search) => {
    setLoading(true);
    try {
      const resp = await getPaymentOrders(p, s);
      if (resp.status && resp.data) {
        setOrders(resp.data ?? []);
        setTotal(resp.total ?? 1);
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
    setPage(0);
    await sync(0, search);
  };

  const handleRecheck = async (order: PaymentOrder) => {
    setRecheckingId(order.order_id);
    try {
      const resp = await recheckOrderStatus(order.order_id, order.service);
      if (!resp.status) {
        withNotify(t, resp);
        return;
      }

      if (resp.is_changed) {
        toast.success(t("admin.pay.check-result-diff"), {
          description: t("admin.pay.check-result-diff-prompt"),
        });
        await sync();
      } else {
        toast.info(t("admin.pay.check-result-same"), {
          description: t("admin.pay.check-result-same-prompt"),
        });
      }
    } finally {
      setRecheckingId(null);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <Input
          placeholder={t("admin.pay.search")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSearch()}
          className="max-w-xs"
        />
        <Button onClick={handleSearch} variant="outline" size="icon">
          <Search className="w-4 h-4" />
        </Button>
        <Button
          onClick={() => sync()}
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
              <TableHead>{t("admin.pay.username")}</TableHead>
              <TableHead>{t("admin.pay.order")}</TableHead>
              <TableHead>{t("admin.pay.amount")}</TableHead>
              <TableHead>{t("admin.pay.type")}</TableHead>
              <TableHead>{t("admin.pay.service")}</TableHead>
              <TableHead>{t("admin.pay.device")}</TableHead>
              <TableHead>{t("admin.pay.status")}</TableHead>
              <TableHead>{t("admin.pay.created-at")}</TableHead>
              <TableHead>{t("admin.pay.action")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {orders.length === 0 && !loading && (
              <TableRow>
                <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                  —
                </TableCell>
              </TableRow>
            )}
            {orders.map((order, i) => (
              <TableRow key={i}>
                <TableCell className="font-medium">{order.username}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-1 max-w-[160px]">
                    <span className="truncate text-xs font-mono">{order.order_id}</span>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-5 w-5 flex-shrink-0"
                      onClick={() => {
                        copyClipboard(order.order_id);
                        toast.success(t("admin.pay.copy-order"));
                      }}
                    >
                      <Copy className="w-3 h-3" />
                    </Button>
                  </div>
                </TableCell>
                <TableCell>{order.amount}</TableCell>
                <TableCell>{t(`admin.pay.${order.type}`) || order.type}</TableCell>
                <TableCell>{order.service}</TableCell>
                <TableCell>{order.device}</TableCell>
                <TableCell>
                  <PaymentStatusBadge state={order.state} />
                </TableCell>
                <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                  {new Date(order.created_at).toLocaleString()}
                </TableCell>
                <TableCell>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={recheckingId === order.order_id}
                    onClick={() => handleRecheck(order)}
                    title={t("admin.pay.check-order")}
                  >
                    <RotateCw
                      className={cn(
                        "w-3 h-3 mr-1",
                        recheckingId === order.order_id && "animate-spin",
                      )}
                    />
                    {t("admin.pay.check-order")}
                  </Button>
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
        onPageChange={(p) => {
          setPage(p);
          sync(p, search);
        }}
      />
    </div>
  );
}

function AdminPayment() {
  const { t } = useTranslation();

  return (
    <div className={cn("user-interface", mobile && "mobile")}>
      <Card className="admin-card">
        <CardHeader className="select-none">
          <CardTitle>{t("admin.payment")}</CardTitle>
        </CardHeader>
        <CardContent>
          <PaymentTable />
        </CardContent>
      </Card>
    </div>
  );
}

export default AdminPayment;
