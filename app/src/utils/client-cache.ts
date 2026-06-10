type CacheEnvelope<T> = {
  version: 1;
  updatedAt: number;
  data: T;
};

const cachePrefix = "api-cache:";
const maxCachePayloadLength = 1_500_000;

function getCacheKey(key: string): string {
  return `${cachePrefix}${key}`;
}

export function getClientCacheStorageKey(key: string): string {
  return getCacheKey(key);
}

function getStorageValue<T>(key: string): T | undefined {
  const fallback = localStorage.getItem(key);
  if (!fallback) return undefined;

  try {
    return JSON.parse(fallback) as T;
  } catch {
    return undefined;
  }
}

function isStorageQuotaError(error: unknown): boolean {
  if (error instanceof DOMException) {
    return (
      error.name === "QuotaExceededError" ||
      error.name === "NS_ERROR_DOM_QUOTA_REACHED"
    );
  }

  return String(error).includes("QuotaExceeded");
}

function listCacheStorageKeys(): string[] {
  const keys: string[] = [];

  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key?.startsWith(cachePrefix)) keys.push(key);
  }

  return keys;
}

function removeOldestCacheEntries(exceptKey: string, count: number): void {
  listCacheStorageKeys()
    .filter((key) => key !== exceptKey)
    .map((key) => ({
      key,
      updatedAt:
        getStorageValue<Partial<CacheEnvelope<unknown>>>(key)?.updatedAt ?? 0,
    }))
    .sort((left, right) => left.updatedAt - right.updatedAt)
    .slice(0, count)
    .forEach(({ key }) => localStorage.removeItem(key));
}

function setStorageValue(key: string, value: string): void {
  if (value.length > maxCachePayloadLength) return;

  try {
    localStorage.setItem(key, value);
    return;
  } catch (error) {
    if (!isStorageQuotaError(error)) throw error;
  }

  const cacheKeys = listCacheStorageKeys().filter((item) => item !== key);
  const step = Math.max(1, Math.ceil(cacheKeys.length / 3));

  for (let removed = step; removed <= cacheKeys.length; removed += step) {
    removeOldestCacheEntries(key, step);
    try {
      localStorage.setItem(key, value);
      return;
    } catch (error) {
      if (!isStorageQuotaError(error)) throw error;
    }
  }
}

export async function getClientCache<T>(
  key: string,
  maxAgeMs?: number,
): Promise<T | undefined> {
  const cached = getStorageValue<CacheEnvelope<T>>(getCacheKey(key));
  if (!cached || cached.version !== 1) return undefined;

  if (maxAgeMs && Date.now() - cached.updatedAt > maxAgeMs) {
    return undefined;
  }

  return cached.data;
}

export async function setClientCache<T>(key: string, data: T): Promise<void> {
  setStorageValue(
    getCacheKey(key),
    JSON.stringify({
      version: 1,
      updatedAt: Date.now(),
      data,
    } satisfies CacheEnvelope<T>),
  );
}

export async function removeClientCache(key: string): Promise<void> {
  localStorage.removeItem(getCacheKey(key));
}

export async function removeClientCachesByPrefix(
  keyPrefix: string,
): Promise<void> {
  const storagePrefix = getCacheKey(keyPrefix);
  const keys: string[] = [];

  listCacheStorageKeys().forEach((key) => {
    if (key.startsWith(storagePrefix)) keys.push(key);
  });

  keys.forEach((key) => localStorage.removeItem(key));
}
