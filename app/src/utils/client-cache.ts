import {
  clientCacheStoreName,
  createBrowserStorageStore,
  legacyBrowserStorageDatabaseName,
} from "@/utils/browser-storage.ts";

type CacheEnvelope<T> = {
  version: 1;
  updatedAt: number;
  data: T;
};

const cachePrefix = "api-cache:";
const cacheUpdateSignalSuffix = ":updated";
const maxLegacyCachePayloadLength = 1_500_000;

let cacheStore: LocalForage | null = null;
let legacyCacheStore: LocalForage | null = null;

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function getCacheStore(): LocalForage | null {
  if (!isBrowser()) return null;

  if (!cacheStore) {
    cacheStore = createBrowserStorageStore(clientCacheStoreName);
  }

  return cacheStore;
}

function getLegacyCacheStore(): LocalForage | null {
  if (!isBrowser()) return null;

  if (!legacyCacheStore) {
    legacyCacheStore = createBrowserStorageStore(
      clientCacheStoreName,
      legacyBrowserStorageDatabaseName,
    );
  }

  return legacyCacheStore;
}

function getCacheKey(key: string): string {
  return `${cachePrefix}${key}`;
}

export function getClientCacheStorageKey(key: string): string {
  return getCacheKey(key);
}

export function getClientCacheUpdateStorageKey(key: string): string {
  return `${getCacheKey(key)}${cacheUpdateSignalSuffix}`;
}

function parseEnvelope<T>(raw: unknown): CacheEnvelope<T> | undefined {
  if (!raw || typeof raw !== "object") return undefined;

  const cached = raw as Partial<CacheEnvelope<T>>;
  if (cached.version !== 1 || typeof cached.updatedAt !== "number") {
    return undefined;
  }

  return cached as CacheEnvelope<T>;
}

function isExpired(updatedAt: number, maxAgeMs?: number): boolean {
  return Boolean(maxAgeMs && Date.now() - updatedAt > maxAgeMs);
}

function readLegacyLocalStorage<T>(
  storageKey: string,
): CacheEnvelope<T> | undefined {
  if (!isBrowser()) return undefined;

  try {
    const fallback = localStorage.getItem(storageKey);
    if (!fallback) return undefined;

    return parseEnvelope<T>(JSON.parse(fallback));
  } catch {
    return undefined;
  }
}

function removeLegacyLocalStorage(storageKey: string): void {
  if (!isBrowser()) return;

  try {
    localStorage.removeItem(storageKey);
  } catch {
    // ignore legacy cleanup failures
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

function removeOldestLegacyCacheEntries(
  exceptKey: string,
  count: number,
): void {
  listLegacyCacheStorageKeys()
    .filter((key) => key !== exceptKey)
    .map((key) => ({
      key,
      updatedAt:
        readLegacyLocalStorage<unknown>(key)?.updatedAt ??
        Number.NEGATIVE_INFINITY,
    }))
    .sort((left, right) => left.updatedAt - right.updatedAt)
    .slice(0, count)
    .forEach(({ key }) => removeLegacyLocalStorage(key));
}

function writeLegacyLocalStorage<T>(
  storageKey: string,
  envelope: CacheEnvelope<T>,
): boolean {
  if (!isBrowser()) return false;

  let value: string;
  try {
    value = JSON.stringify(envelope);
  } catch {
    return false;
  }

  if (value.length > maxLegacyCachePayloadLength) return false;

  try {
    localStorage.setItem(storageKey, value);
    return true;
  } catch (error) {
    if (!isStorageQuotaError(error)) {
      console.debug(
        "[client-cache] failed to write to localStorage fallback",
        storageKey,
        error,
      );
      return false;
    }
  }

  const cacheKeys = listLegacyCacheStorageKeys().filter(
    (item) => item !== storageKey,
  );
  const step = Math.max(1, Math.ceil(cacheKeys.length / 3));

  for (let removed = step; removed <= cacheKeys.length; removed += step) {
    removeOldestLegacyCacheEntries(storageKey, step);
    try {
      localStorage.setItem(storageKey, value);
      return true;
    } catch (error) {
      if (!isStorageQuotaError(error)) {
        console.debug(
          "[client-cache] failed to write to localStorage fallback",
          storageKey,
          error,
        );
        return false;
      }
    }
  }

  return false;
}

function writeCacheUpdateSignal(storageKey: string): void {
  if (!isBrowser()) return;

  try {
    localStorage.setItem(
      `${storageKey}${cacheUpdateSignalSuffix}`,
      Date.now().toString(),
    );
  } catch {
    // A cache update signal is only a cross-tab hint.
  }
}

function listLegacyCacheStorageKeys(): string[] {
  if (!isBrowser()) return [];

  const keys: string[] = [];

  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith(cachePrefix)) keys.push(key);
    }
  } catch {
    return [];
  }

  return keys;
}

async function readFromIndexedDB<T>(
  storageKey: string,
): Promise<CacheEnvelope<T> | undefined> {
  const store = getCacheStore();
  if (!store) return undefined;

  try {
    const raw = await store.getItem<CacheEnvelope<T>>(storageKey);
    return parseEnvelope<T>(raw);
  } catch {
    return undefined;
  }
}

async function writeToIndexedDB<T>(
  storageKey: string,
  envelope: CacheEnvelope<T>,
): Promise<boolean> {
  const store = getCacheStore();
  if (!store) return false;

  try {
    await store.setItem(storageKey, envelope);
    return true;
  } catch (error) {
    console.debug(
      "[client-cache] failed to write to IndexedDB",
      storageKey,
      error,
    );
    return false;
  }
}

async function writeToIndexedDBIfNewer<T>(
  storageKey: string,
  envelope: CacheEnvelope<T>,
): Promise<boolean> {
  const store = getCacheStore();
  if (!store) return false;

  try {
    const current = parseEnvelope<T>(await store.getItem<unknown>(storageKey));
    if (current && current.updatedAt >= envelope.updatedAt) return true;

    await store.setItem(storageKey, envelope);
    return true;
  } catch (error) {
    console.debug(
      "[client-cache] failed to migrate to IndexedDB",
      storageKey,
      error,
    );
    return false;
  }
}

type CacheCandidate<T> = {
  source: "indexeddb" | "legacy";
  envelope: CacheEnvelope<T>;
};

function getFreshestCacheCandidate<T>(
  candidates: CacheCandidate<T>[],
): CacheCandidate<T> | undefined {
  return candidates.reduce<CacheCandidate<T> | undefined>((freshest, item) => {
    if (!freshest || item.envelope.updatedAt > freshest.envelope.updatedAt) {
      return item;
    }

    return freshest;
  }, undefined);
}

export async function getClientCache<T>(
  key: string,
  maxAgeMs?: number,
): Promise<T | undefined> {
  if (!isBrowser()) return undefined;

  const storageKey = getCacheKey(key);
  const fromIndexedDB = await readFromIndexedDB<T>(storageKey);
  const fromLegacy = readLegacyLocalStorage<T>(storageKey);

  const candidates: CacheCandidate<T>[] = [];

  if (fromIndexedDB && !isExpired(fromIndexedDB.updatedAt, maxAgeMs)) {
    candidates.push({ source: "indexeddb", envelope: fromIndexedDB });
  }
  if (fromLegacy && !isExpired(fromLegacy.updatedAt, maxAgeMs)) {
    candidates.push({ source: "legacy", envelope: fromLegacy });
  }

  const selected = getFreshestCacheCandidate(candidates);
  if (!selected) return undefined;

  if (selected.source === "legacy") {
    if (await writeToIndexedDBIfNewer(storageKey, selected.envelope)) {
      removeLegacyLocalStorage(storageKey);
    }
  } else if (
    fromLegacy &&
    fromLegacy.updatedAt <= selected.envelope.updatedAt
  ) {
    removeLegacyLocalStorage(storageKey);
  }

  return selected.envelope.data;
}

export async function setClientCache<T>(key: string, data: T): Promise<void> {
  if (!isBrowser()) return;

  const storageKey = getCacheKey(key);
  const envelope: CacheEnvelope<T> = {
    version: 1,
    updatedAt: Date.now(),
    data,
  };

  if (await writeToIndexedDB(storageKey, envelope)) {
    removeLegacyLocalStorage(storageKey);
    writeCacheUpdateSignal(storageKey);
    return;
  }

  if (writeLegacyLocalStorage(storageKey, envelope)) {
    writeCacheUpdateSignal(storageKey);
  }
}

export async function migrateLegacyClientCaches(): Promise<boolean> {
  if (!isBrowser()) return true;

  const indexedDBMigrated = await migrateLegacyIndexedDBClientCaches();

  const localStorageResults = await Promise.all(
    listLegacyCacheStorageKeys().map(async (storageKey) => {
      if (storageKey.endsWith(cacheUpdateSignalSuffix)) {
        removeLegacyLocalStorage(storageKey);
        return true;
      }

      const envelope = readLegacyLocalStorage<unknown>(storageKey);
      if (!envelope) return true;

      if (await writeToIndexedDBIfNewer(storageKey, envelope)) {
        removeLegacyLocalStorage(storageKey);
        return true;
      }

      return false;
    }),
  );

  return indexedDBMigrated && localStorageResults.every(Boolean);
}

async function migrateLegacyIndexedDBClientCaches(): Promise<boolean> {
  const legacyStore = getLegacyCacheStore();
  if (!legacyStore || !getCacheStore()) return false;

  try {
    const keys = await legacyStore.keys();
    const results = await Promise.all(
      keys.map(async (storageKey) => {
        const fromLegacy = parseEnvelope<unknown>(
          await legacyStore.getItem<unknown>(storageKey),
        );
        if (!fromLegacy) return true;

        return await writeToIndexedDBIfNewer(storageKey, fromLegacy);
      }),
    );

    return results.every(Boolean);
  } catch (error) {
    console.debug(
      "[client-cache] failed to migrate legacy IndexedDB cache",
      error,
    );
    return false;
  }
}

export async function removeClientCache(key: string): Promise<void> {
  if (!isBrowser()) return;

  const storageKey = getCacheKey(key);
  const store = getCacheStore();

  if (store) {
    try {
      await store.removeItem(storageKey);
    } catch (error) {
      console.debug(
        "[client-cache] failed to remove from IndexedDB",
        storageKey,
        error,
      );
    }
  }

  removeLegacyLocalStorage(storageKey);
  removeLegacyLocalStorage(`${storageKey}${cacheUpdateSignalSuffix}`);
  writeCacheUpdateSignal(storageKey);
}

export async function removeClientCachesByPrefix(
  keyPrefix: string,
): Promise<void> {
  if (!isBrowser()) return;

  const storagePrefix = getCacheKey(keyPrefix);
  const store = getCacheStore();

  if (store) {
    try {
      const allKeys = await store.keys();
      await Promise.all(
        allKeys
          .filter((key) => key.startsWith(storagePrefix))
          .map((key) => store.removeItem(key)),
      );
    } catch (error) {
      console.debug(
        "[client-cache] failed to remove by prefix from IndexedDB",
        keyPrefix,
        error,
      );
    }
  }

  listLegacyCacheStorageKeys()
    .filter((key) => key.startsWith(storagePrefix))
    .forEach((key) => removeLegacyLocalStorage(key));
}
