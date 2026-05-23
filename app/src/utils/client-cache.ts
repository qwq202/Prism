type CacheEnvelope<T> = {
  version: 1;
  updatedAt: number;
  data: T;
};

const cachePrefix = "api-cache:";

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

export async function setClientCache<T>(
  key: string,
  data: T,
): Promise<void> {
  localStorage.setItem(
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

  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (key?.startsWith(storagePrefix)) keys.push(key);
  }

  keys.forEach((key) => localStorage.removeItem(key));
}
