import type { Record as BillingRecord } from "@/api/record.ts";

export type RecordCacheUsage = {
  hitTokens: number;
  missTokens: number;
  writeTokens: number;
  hasCacheUsage: boolean;
  hasCacheHit: boolean;
  hasCacheWrite: boolean;
};

function toNumber(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) return 0;
  return Math.max(0, value);
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
    };
    const hitTokens = toNumber(detail.official_usage?.prompt_cache_hit_tokens);
    const missTokens = toNumber(detail.official_usage?.prompt_cache_miss_tokens);
    const writeTokens = toNumber(
      detail.official_usage?.prompt_cache_write_tokens,
    );
    if (hitTokens <= 0 && missTokens <= 0 && writeTokens <= 0) return null;

    return {
      hitTokens,
      missTokens,
      writeTokens,
      hasCacheUsage: true,
      hasCacheHit: hitTokens > 0,
      hasCacheWrite: writeTokens > 0,
    };
  } catch {
    return null;
  }
}
