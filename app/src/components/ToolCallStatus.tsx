import { MessageToolCall } from "@/api/types.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import { formatToolCallResult } from "@/api/plugin.ts";
import {
  CheckCircle2,
  ChevronDown,
  Loader2,
  Wrench,
  XCircle,
} from "lucide-react";
import { useTranslation } from "react-i18next";

type ToolCallStatusProps = {
  toolCalls: MessageToolCall[];
  className?: string;
};

function getPrettyJson(value: string): string {
  const trimmed = value.trim();
  if (!trimmed) return "";

  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
}

export function ToolCallStatus({ toolCalls, className }: ToolCallStatusProps) {
  const { t } = useTranslation();

  if (toolCalls.length === 0) return null;

  const getStatusMeta = (toolCall: MessageToolCall) => {
    switch (toolCall.status) {
      case "executing":
        return {
          label: t("plugin.mcp.status-executing"),
          icon: <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />,
          tone: "text-blue-600 dark:text-blue-400",
        };
      case "success":
        return {
          label: t("plugin.mcp.status-success"),
          icon: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
          tone: "text-green-600 dark:text-green-400",
        };
      case "error":
        return {
          label: t("plugin.mcp.status-error"),
          icon: <XCircle className="h-3.5 w-3.5 text-red-500" />,
          tone: "text-red-600 dark:text-red-400",
        };
      default:
        return {
          label: t("plugin.mcp.status-prepare"),
          icon: (
            <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
          ),
          tone: "text-muted-foreground",
        };
    }
  };

  return (
    <div className={cn("mt-3 space-y-2", className)}>
      {toolCalls.map((toolCall, index) => {
        const status = getStatusMeta(toolCall);
        const argumentsText = getPrettyJson(toolCall.function.arguments);
        const resultText = toolCall.result
          ? getPrettyJson(formatToolCallResult(toolCall.result))
          : "";
        const errorText = toolCall.error ? toolCall.error.trim() : "";
        const hasDetails = Boolean(argumentsText || resultText || errorText);

        return (
          <details
            key={toolCall.id || `${toolCall.function.name}-${index}`}
            className="overflow-hidden rounded-xl border border-border/70 bg-muted/20"
          >
            <summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-3 py-2.5">
              <div className="flex min-w-0 items-center gap-2.5">
                <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-background/80 border border-border/60">
                  <Wrench className="h-3.5 w-3.5 text-muted-foreground" />
                </div>
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium">
                    {toolCall.function.name}
                  </div>
                  <div
                    className={cn(
                      "flex items-center gap-1.5 text-xs",
                      status.tone,
                    )}
                  >
                    {status.icon}
                    <span>{status.label}</span>
                  </div>
                </div>
              </div>
              {hasDetails && (
                <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground transition-transform details-open:rotate-180" />
              )}
            </summary>

            {hasDetails && (
              <div className="space-y-3 border-t border-border/60 px-3 py-3">
                {argumentsText && (
                  <div className="space-y-1.5">
                    <div className="text-xs font-medium text-muted-foreground">
                      {t("plugin.mcp.tool-arguments")}
                    </div>
                    <pre className="overflow-x-auto rounded-lg bg-background/80 p-2 text-xs leading-relaxed text-foreground whitespace-pre-wrap break-words">
                      {argumentsText}
                    </pre>
                  </div>
                )}
                {resultText && (
                  <div className="space-y-1.5">
                    <div className="text-xs font-medium text-muted-foreground">
                      {t("plugin.mcp.result")}
                    </div>
                    <pre className="overflow-x-auto rounded-lg bg-background/80 p-2 text-xs leading-relaxed text-foreground whitespace-pre-wrap break-words">
                      {resultText}
                    </pre>
                  </div>
                )}
                {errorText && (
                  <div className="space-y-1.5">
                    <div className="text-xs font-medium text-red-500">
                      {t("plugin.mcp.error")}
                    </div>
                    <pre className="overflow-x-auto rounded-lg bg-red-500/10 p-2 text-xs leading-relaxed text-red-600 dark:text-red-400 whitespace-pre-wrap break-words">
                      {errorText}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </details>
        );
      })}
    </div>
  );
}
