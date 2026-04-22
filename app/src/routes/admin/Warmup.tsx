import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { useTranslation } from "react-i18next";
import { useState } from "react";
import { Button } from "@/components/ui/button.tsx";
import { Textarea } from "@/components/ui/textarea.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Copy, Flame, Loader2 } from "lucide-react";
import { mobile } from "@/utils/device.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import axios from "axios";
import { copyClipboard } from "@/utils/dom.ts";
import { toast } from "sonner";
import Markdown from "@/components/Markdown.tsx";

type WarmupResult = {
  url: string;
  status: number;
  error?: string;
};

async function warmupUrls(urls: string[]): Promise<WarmupResult[]> {
  try {
    const resp = await axios.post("/admin/warmup", { urls });
    return resp.data?.results ?? [];
  } catch (e) {
    return [];
  }
}

function ResultBadge({ status, error }: { status: number; error?: string }) {
  const { t } = useTranslation();

  if (error) {
    return <Badge variant="destructive">{t("admin.cdn.status-error")}</Badge>;
  }
  if (status >= 200 && status < 300) {
    return (
      <Badge variant="default">
        {t("admin.cdn.status-ok", { status })}
      </Badge>
    );
  }
  if (status >= 300 && status < 400) {
    return (
      <Badge variant="secondary">
        {t("admin.cdn.status-redirect", { status })}
      </Badge>
    );
  }
  return <Badge variant="destructive">{status || "—"}</Badge>;
}

function AdminWarmup() {
  const { t } = useTranslation();
  const [urlText, setUrlText] = useState("");
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<WarmupResult[]>([]);

  const getUrls = (): string[] =>
    urlText
      .split("\n")
      .map((u) => u.trim())
      .filter((u) => u.length > 0 && (u.startsWith("http://") || u.startsWith("https://")));

  const handleWarmup = async () => {
    const urls = getUrls();
    if (urls.length === 0) {
      toast.error(t("admin.cdn.invalid-urls"));
      return;
    }
    setLoading(true);
    setResults([]);
    const res = await warmupUrls(urls);
    setResults(res);
    setLoading(false);

    const success = res.filter((r) => !r.error && r.status >= 200 && r.status < 300).length;
    toast.success(
      t("admin.cdn.warmup-complete", {
        success,
        total: res.length,
      }),
    );
  };

  const handleCopyUrls = () => {
    copyClipboard(getUrls().join("\n"));
    toast.success(t("admin.cdn.copy-data"));
  };

  return (
    <div className={cn("user-interface", mobile && "mobile")}>
      <Card className="admin-card">
        <CardHeader className="select-none">
          <CardTitle>{t("admin.cdn.warmup")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-md bg-muted/20 p-4">
            <Markdown className="prose prose-sm max-w-none text-sm leading-relaxed text-muted-foreground">
              {t("admin.cdn.warm-tip")}
            </Markdown>
          </div>

          <div className="space-y-2">
            <Textarea
              placeholder={t("admin.cdn.warmup-placeholder")}
              value={urlText}
              onChange={(e) => setUrlText(e.target.value)}
              rows={8}
              className="font-mono text-sm"
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.cdn.detected-count", { count: getUrls().length })}
            </p>
          </div>

          <div className="flex gap-2">
            <Button onClick={handleWarmup} disabled={loading}>
              {loading ? (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              ) : (
                <Flame className="w-4 h-4 mr-2" />
              )}
              {loading ? t("admin.cdn.warming") : t("admin.cdn.warmup")}
            </Button>
            <Button variant="outline" onClick={handleCopyUrls}>
              <Copy className="w-4 h-4 mr-2" />
              {t("admin.cdn.copy-data")}
            </Button>
          </div>

          {results.length > 0 && (
            <div className="space-y-1">
              <p className="text-sm font-medium">{t("admin.cdn.results")}</p>
              <div className="rounded-md border divide-y">
                {results.map((r, i) => (
                  <div
                    key={i}
                    className="flex items-center justify-between px-3 py-2 text-sm"
                  >
                    <span className="font-mono text-xs text-muted-foreground truncate max-w-[70%]">
                      {r.url}
                    </span>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {r.error && (
                        <span className="text-xs text-destructive">{r.error}</span>
                      )}
                      <ResultBadge status={r.status} error={r.error} />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default AdminWarmup;
