import localforage from "localforage";

type CacheEnvelope<T> = {
  version: 1;
  updatedAt: number;
  data: T;
};

const cachePrefix = "api-cache:";
const cacheUpdateSignalSuffix = ":updated";

let cacheStore: LocalForage | null = null;

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function getCacheStore(): LocalForage | null {
  if (!isBrowser()) return null;

  if (!cacheStore) {
    cacheStore = localforage.createInstance({
      driver: localforage.INDEXEDDB,
      name: "coai",
      storeName: "client-cache",
    });
  }

  return cacheStore;
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

export async function getClientCache<T>(
  key: string,
  maxAgeMs?: number,
): Promise<T | undefined> {
  if (!isBrowser()) return undefined;

  const storageKey = getCacheKey(key);

  const fromIndexedDB = await readFromIndexedDB<T>(storageKey);
  if (fromIndexedDB) {
    if (isExpired(fromIndexedDB.updatedAt, maxAgeMs)) return undefined;
    return fromIndexedDB.data;
  }

  const fromLegacy = readLegacyLocalStorage<T>(storageKey);
  if (!fromLegacy) return undefined;
  if (isExpired(fromLegacy.updatedAt, maxAgeMs)) return undefined;

  if (await writeToIndexedDB(storageKey, fromLegacy)) {
    removeLegacyLocalStorage(storageKey);
  }

  return fromLegacy.data;
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
