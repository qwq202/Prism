import type { Record as BillingRecord } from "@/api/record.ts";

export type RecordCacheUsage = {
  hitTokens: number;
  missTokens: number;
  writeTokens: number;
  hasCacheUsage: boolean;
  hasCacheHit: boolean;
  hasCacheWrite: boolean;
  promptCache?: RecordPromptCacheStatus;
};

export type RecordPromptCacheStatus = {
  provider?: string;
  mode?: string;
  status?: string;
  reason?: string;
  promptTokens: number;
  thresholdTokens: number;
  eligible: boolean;
  attempted: boolean;
};

function toNumber(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) return 0;
  return Math.max(0, value);
}

function toString(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function toBoolean(value: unknown): boolean {
  return typeof value === "boolean" ? value : false;
}

function parsePromptCacheStatus(value: unknown): RecordPromptCacheStatus | null {
  if (!value || typeof value !== "object") return null;
  const raw = value as {
    provider?: unknown;
    mode?: unknown;
    status?: unknown;
    reason?: unknown;
    prompt_tokens?: unknown;
    threshold_tokens?: unknown;
    eligible?: unknown;
    attempted?: unknown;
  };
  const status = toString(raw.status);
  if (!status || status === "unsupported") return null;

  return {
    provider: toString(raw.provider),
    mode: toString(raw.mode),
    status,
    reason: toString(raw.reason),
    promptTokens: toNumber(raw.prompt_tokens),
    thresholdTokens: toNumber(raw.threshold_tokens),
    eligible: toBoolean(raw.eligible),
    attempted: toBoolean(raw.attempted),
  };
}

export function getRecordCacheUsage(
  record: Pick<BillingRecord, "detail">,
): RecordCacheUsage | null {
  if (!record.detail) return null;

  try {
    const detail = JSON.parse(record.detail) as {
      official_usage?: {
        prompt_cache_hit_tokens?: unknown;
        prompt_cache_miss_tokens?: unknown;
        prompt_cache_write_tokens?: unknown;
      };
      prompt_cache?: unknown;
    };
    const hitTokens = toNumber(detail.official_usage?.prompt_cache_hit_tokens);
    const missTokens = toNumber(
      detail.official_usage?.prompt_cache_miss_tokens,
    );
    const writeTokens = toNumber(
      detail.official_usage?.prompt_cache_write_tokens,
    );
    const promptCache = parsePromptCacheStatus(detail.prompt_cache);
    if (
      hitTokens <= 0 &&
      missTokens <= 0 &&
      writeTokens <= 0 &&
      !promptCache
    )
      return null;

    return {
      hitTokens,
      missTokens,
      writeTokens,
      hasCacheUsage: hitTokens > 0 || missTokens > 0 || writeTokens > 0,
      hasCacheHit: hitTokens > 0,
      hasCacheWrite: writeTokens > 0,
      promptCache: promptCache ?? undefined,
    };
  } catch {
    return null;
  }
}
