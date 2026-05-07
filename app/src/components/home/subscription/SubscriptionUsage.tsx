import { ValuableProgress } from "@/components/ui/progress.tsx";
import { useTranslation } from "react-i18next";
import { Ban } from "lucide-react";
import { useEffect, useState } from "react";

type UsageProps = {
  name: string;
  usage: {
    used: number;
    total: number;
    unit?: "times" | "points";
    reset_at?: string;
  };
  blockedBy?: string;
  absoluteReset?: boolean;
  fallbackResetLabel?: string;
};

function formatResetIn(
  t: (k: string, opts?: Record<string, unknown>) => string,
  resetAt: string,
  now: number,
): string {
  const diff = new Date(resetAt).getTime() - now;
  if (diff <= 0) return "";
  const totalMinutes = Math.floor(diff / 60000);
  const days = Math.floor(totalMinutes / 1440);
  const hours = Math.floor((totalMinutes % 1440) / 60);
  const minutes = totalMinutes % 60;

  const parts: string[] = [];
  if (days > 0) parts.push(t("sub.reset-days", { days }));
  if (hours > 0) parts.push(t("sub.reset-hours", { hours }));
  if (minutes > 0 || parts.length === 0)
    parts.push(t("sub.reset-minutes", { minutes: minutes || 1 }));
  return t("sub.reset-in", { time: parts.join(" ") });
}

function formatAbsoluteReset(resetAt: string, locale: string): string {
  const d = new Date(resetAt);
  if (isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(locale || undefined, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

function SubscriptionUsage({
  name,
  usage,
  blockedBy,
  absoluteReset,
  fallbackResetLabel,
}: UsageProps) {
  const { t, i18n } = useTranslation();
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!usage?.reset_at || absoluteReset) return;
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, [absoluteReset, usage?.reset_at]);

  if (!usage) return null;

  const isInfinity = usage.total === -1;
  const isPoints = usage.unit === "points";
  const resetLabel = usage.reset_at
    ? absoluteReset
      ? formatAbsoluteReset(usage.reset_at, i18n.language)
      : formatResetIn(t, usage.reset_at, now)
    : fallbackResetLabel ?? "";
  const isBlocked = !!blockedBy;

  if (isPoints) {
    const pct = isInfinity
      ? 100
      : Math.max(0, Math.round((1 - usage.used / usage.total) * 100));
    return (
      <div
        className={`sub-column-wrapper inline-flex flex-col relative ${
          isBlocked ? "opacity-50" : ""
        }`}
      >
        <div className="sub-column">
          <div className="flex items-center text-sm text-secondary gap-1">
            {name}
            {isBlocked && (
              <Ban className="h-3 w-3 text-destructive/70 shrink-0" />
            )}
          </div>
          <div className="grow" />
          <div className="sub-value font-medium text-md">
            {isInfinity ? (
              <p className="text-xs font-semibold">
                {t("sub.points-unlimited")}
              </p>
            ) : (
              <p>{t("sub.points-remaining", { pct })}</p>
            )}
          </div>
        </div>
        {!isInfinity && (
          <ValuableProgress
            className="w-full h-2"
            value={usage.total - usage.used}
            max={usage.total}
          />
        )}
        {isBlocked ? (
          <p className="text-xs text-destructive/70 mt-1">
            {t("sub.blocked-by", { name: blockedBy })}
          </p>
        ) : resetLabel ? (
          <p className="text-xs text-muted-foreground mt-1">
            {absoluteReset
              ? t("sub.reset-at", { time: resetLabel })
              : resetLabel}
          </p>
        ) : null}
      </div>
    );
  }

  const used = usage.used;
  const total = isInfinity ? "∞" : usage.total;
  const hasFiniteTotal = !isInfinity && usage.total > 0;
  const remaining = isInfinity ? 0 : Math.max(0, usage.total - used);

  return (
    <div className="sub-column-wrapper inline-flex flex-col">
      <div className="sub-column">
        <div className="flex items-center text-sm text-secondary">{name}</div>
        <div className="grow" />
        <div className="sub-value font-medium text-md">
          {isInfinity ? (
            <p>{t("sub.times-unlimited")}</p>
          ) : (
            <>
              <p>{t("sub.times-remaining", { remaining })}</p>
              {hasFiniteTotal && (
                <p className="text-secondary !font-normal text-sm">/{total}</p>
              )}
            </>
          )}
        </div>
      </div>
      {hasFiniteTotal && (
        <ValuableProgress
          className="w-full h-2"
          value={remaining}
          max={usage.total}
        />
      )}
      {resetLabel && (
        <p className="text-xs text-muted-foreground mt-1">{resetLabel}</p>
      )}
    </div>
  );
}

export default SubscriptionUsage;
