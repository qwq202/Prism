import { MessageToolCall } from "@/api/types.tsx";
import { getVisibleToolCalls } from "@/api/tool-calls.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { formatToolCallResult } from "@/api/plugin.ts";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import {
  CheckCircle2,
  Loader2,
  Wrench,
  XCircle,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip.tsx";

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

type ToolCallRowProps = {
  toolCall: MessageToolCall;
};

function ToolCallRow({ toolCall }: ToolCallRowProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

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

  const status = getStatusMeta(toolCall);
  const argumentsText = getPrettyJson(toolCall.function.arguments);
  const resultText = toolCall.result
    ? getPrettyJson(formatToolCallResult(toolCall.result))
    : "";
  const errorText = toolCall.error ? toolCall.error.trim() : "";
  const hasDetails = Boolean(argumentsText || resultText || errorText);

  return (
    <>
      <TooltipProvider delayDuration={200}>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              aria-label={`${toolCall.function.name} ${status.label}`}
              aria-disabled={!hasDetails}
              onClick={() => hasDetails && setOpen(true)}
              className={cn(
                "relative inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border/60 bg-muted/5 text-muted-foreground transition-colors",
                hasDetails
                  ? "cursor-pointer hover:bg-muted/10 hover:text-foreground"
                  : "cursor-default opacity-95",
              )}
            >
              <Wrench className="h-3.5 w-3.5" />
              <span className="absolute -bottom-0.5 -right-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-background">
                {status.icon}
              </span>
            </button>
          </TooltipTrigger>
          <TooltipContent>
            {toolCall.function.name} · {status.label}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      {hasDetails && (
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogContent className="max-w-2xl overflow-hidden">
            <DialogHeader className="min-w-0">
              <DialogTitle className="flex min-w-0 items-center gap-2 pr-8 text-sm">
                <span className="min-w-0 break-words [overflow-wrap:anywhere]">
                  {toolCall.function.name}
                </span>
                <span
                  className={cn(
                    "flex shrink-0 items-center gap-1 text-xs font-normal",
                    status.tone,
                  )}
                >
                  {status.icon}
                  {status.label}
                </span>
              </DialogTitle>
              <DialogDescription className="min-w-0 break-words text-left [overflow-wrap:anywhere]">
                {toolCall.id || t("plugin.mcp.tool-arguments")}
              </DialogDescription>
            </DialogHeader>

            <div className="min-w-0 space-y-3 overflow-hidden">
              {argumentsText && (
                <div className="min-w-0 space-y-1">
                  <div className="text-xs font-medium text-muted-foreground">
                    {t("plugin.mcp.tool-arguments")}
                  </div>
                  <pre className="max-h-72 max-w-full overflow-y-auto overflow-x-hidden rounded-md bg-background/80 p-3 text-xs leading-relaxed text-foreground whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                    {argumentsText}
                  </pre>
                </div>
              )}
              {resultText && (
                <div className="min-w-0 space-y-1">
                  <div className="text-xs font-medium text-muted-foreground">
                    {t("plugin.mcp.result")}
                  </div>
                  <pre className="max-h-72 max-w-full overflow-y-auto overflow-x-hidden rounded-md bg-background/80 p-3 text-xs leading-relaxed text-foreground whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                    {resultText}
                  </pre>
                </div>
              )}
              {errorText && (
                <div className="min-w-0 space-y-1">
                  <div className="text-xs font-medium text-red-500">
                    {t("plugin.mcp.error")}
                  </div>
                  <pre className="max-h-72 max-w-full overflow-y-auto overflow-x-hidden rounded-md bg-red-500/10 p-3 text-xs leading-relaxed text-red-600 dark:text-red-400 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                    {errorText}
                  </pre>
                </div>
              )}
            </div>
          </DialogContent>
        </Dialog>
      )}
    </>
  );
}

export function ToolCallStatus({ toolCalls, className }: ToolCallStatusProps) {
  const visibleToolCalls = getVisibleToolCalls(toolCalls);
  if (visibleToolCalls.length === 0) return null;

  return (
    <div className={cn("mt-1.5 flex flex-wrap gap-1", className)}>
      {visibleToolCalls.map((toolCall, index) => (
        <ToolCallRow
          key={toolCall.id || `${toolCall.function.name}-${index}`}
          toolCall={toolCall}
        />
      ))}
    </div>
  );
}
