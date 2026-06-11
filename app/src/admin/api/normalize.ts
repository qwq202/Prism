import {
  type UserSubscriptionWindowData,
} from "@/admin/types.ts";

export type AdminRecord = Record<string, unknown>;

export function asRecord(value: unknown): AdminRecord {
  return value !== null && typeof value === "object"
    ? (value as AdminRecord)
    : {};
}

export function asArray<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}

export function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

export function asNumber(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

export type AdminUserRecord = AdminRecord & {
  subscription_windows?: unknown;
};

export function asUserRecord(value: unknown): AdminUserRecord | null {
  if (value === null || typeof value !== "object") return null;
  return value as AdminUserRecord;
}

export function normalizeSubscriptionWindows(
  value: unknown,
): UserSubscriptionWindowData[] {
  return asArray<unknown>(value).filter(
    (window): window is UserSubscriptionWindowData =>
      window !== null && typeof window === "object",
  );
}
