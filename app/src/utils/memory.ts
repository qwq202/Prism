import {
  createBrowserStorageStore,
  legacyBrowserStorageDatabaseName,
  memoryStoreName,
} from "@/utils/browser-storage.ts";

const volatileKeys = new Set<string>();
const memorySnapshot = new Map<string, string>();
const clientCacheStoragePrefix = "api-cache:";

let memoryStore: LocalForage | null = null;
let legacyMemoryStore: LocalForage | null = null;
let memoryInitialization: Promise<boolean> | null = null;

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function getMemoryStore(): LocalForage | null {
  if (!isBrowser()) return null;

  if (!memoryStore) {
    memoryStore = createBrowserStorageStore(memoryStoreName);
  }

  return memoryStore;
}

function getLegacyMemoryStore(): LocalForage | null {
  if (!isBrowser()) return null;

  if (!legacyMemoryStore) {
    legacyMemoryStore = createBrowserStorageStore(
      memoryStoreName,
      legacyBrowserStorageDatabaseName,
    );
  }

  return legacyMemoryStore;
}

function normalizeMemoryValue(value: string): string {
  return value.trim();
}

function getSessionMemory(key: string): string {
  if (!isBrowser()) return "";

  try {
    return sessionStorage.getItem(key) || "";
  } catch {
    return "";
  }
}

function setSessionMemory(key: string, value: string): void {
  if (!isBrowser()) return;

  try {
    sessionStorage.setItem(key, value);
  } catch {
    // Session storage is only a volatile fallback.
  }
}

function removeSessionMemory(key: string): void {
  if (!isBrowser()) return;

  try {
    sessionStorage.removeItem(key);
  } catch {
    // ignore
  }
}

function clearSessionMemory(): void {
  if (!isBrowser()) return;

  try {
    sessionStorage.clear();
  } catch {
    // ignore
  }
}

function getLegacyMemory(key: string): string {
  if (!isBrowser()) return "";

  try {
    return localStorage.getItem(key) || "";
  } catch {
    return "";
  }
}

function setLegacyMemory(key: string, value: string): void {
  if (!isBrowser()) return;

  try {
    localStorage.setItem(key, value);
  } catch {
    // If both IndexedDB and localStorage are unavailable, the in-memory
    // snapshot still keeps the value for the current tab.
  }
}

function removeLegacyMemory(key: string): void {
  if (!isBrowser()) return;

  try {
    localStorage.removeItem(key);
  } catch {
    // ignore
  }
}

function clearLegacyMemory(): void {
  if (!isBrowser()) return;

  try {
    localStorage.clear();
  } catch {
    // ignore
  }
}

function listLegacyMemoryKeys(): string[] {
  if (!isBrowser()) return [];

  const keys: string[] = [];

  try {
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key) keys.push(key);
    }
  } catch {
    return [];
  }

  return keys;
}

async function setPersistentMemory(
  key: string,
  value: string,
): Promise<boolean> {
  const store = getMemoryStore();
  if (!store) return false;

  try {
    await store.setItem(key, value);
    removeLegacyMemory(key);
    return true;
  } catch (error) {
    console.debug("[memory] failed to write to IndexedDB", key, error);
    return false;
  }
}

async function removePersistentMemory(key: string): Promise<void> {
  const store = getMemoryStore();

  if (store) {
    try {
      await store.removeItem(key);
    } catch (error) {
      console.debug("[memory] failed to remove from IndexedDB", key, error);
    }
  }

  removeLegacyMemory(key);
}

async function clearPersistentMemory(): Promise<void> {
  const store = getMemoryStore();

  if (store) {
    try {
      await store.clear();
    } catch (error) {
      console.debug("[memory] failed to clear IndexedDB", error);
    }
  }

  clearLegacyMemory();
}

async function loadPersistentMemorySnapshot(): Promise<boolean> {
  const store = getMemoryStore();
  if (!store) return false;

  try {
    const keys = await store.keys();
    await Promise.all(
      keys.map(async (key) => {
        const value = await store.getItem<unknown>(key);
        if (typeof value === "string") {
          memorySnapshot.set(key, normalizeMemoryValue(value));
        }
      }),
    );
    return true;
  } catch (error) {
    console.debug("[memory] failed to read from IndexedDB", error);
    return false;
  }
}

async function migrateLegacyMemory(): Promise<boolean> {
  if (!isBrowser()) return true;

  const keys = listLegacyMemoryKeys().filter(
    (key) => !key.startsWith(clientCacheStoragePrefix),
  );

  const results = await Promise.all(
    keys.map(async (key) => {
      const value = getLegacyMemory(key);
      if (!value) {
        removeLegacyMemory(key);
        return true;
      }

      const normalized = normalizeMemoryValue(value);
      if (volatileKeys.has(key)) {
        if (!getSessionMemory(key)) {
          setSessionMemory(key, normalized);
        }
        memorySnapshot.delete(key);
        removeLegacyMemory(key);
        return true;
      }

      if (await setPersistentMemory(key, normalized)) {
        memorySnapshot.set(key, normalized);
        return true;
      }

      return false;
    }),
  );

  return results.every(Boolean);
}

export async function initializeMemoryStorage(): Promise<boolean> {
  if (!isBrowser()) return true;

  if (!memoryInitialization) {
    memoryInitialization = (async () => {
      const legacyIndexedDBMigrated = await migrateLegacyIndexedDBMemory();
      const snapshotLoaded = await loadPersistentMemorySnapshot();
      const legacyLocalStorageMigrated = await migrateLegacyMemory();

      return (
        legacyIndexedDBMigrated && snapshotLoaded && legacyLocalStorageMigrated
      );
    })();
  }

  return await memoryInitialization;
}

async function migrateLegacyIndexedDBMemory(): Promise<boolean> {
  const legacyStore = getLegacyMemoryStore();
  const store = getMemoryStore();
  if (!legacyStore || !store) return false;

  try {
    const keys = await legacyStore.keys();
    const results = await Promise.all(
      keys.map(async (key) => {
        try {
          const value = await legacyStore.getItem<unknown>(key);
          if (typeof value !== "string") return true;

          const current = await store.getItem<unknown>(key);
          if (current === null) {
            await store.setItem(key, normalizeMemoryValue(value));
          }
          return true;
        } catch (error) {
          console.debug(
            "[memory] failed to migrate legacy IndexedDB memory key",
            key,
            error,
          );
          return false;
        }
      }),
    );

    return results.every(Boolean);
  } catch (error) {
    console.debug("[memory] failed to migrate legacy IndexedDB memory", error);
    return false;
  }
}

export function markVolatileMemoryKey(key: string) {
  volatileKeys.add(key);

  const persistedValue = memorySnapshot.get(key) || getLegacyMemory(key);
  if (persistedValue && !getSessionMemory(key)) {
    setSessionMemory(key, normalizeMemoryValue(persistedValue));
  }

  memorySnapshot.delete(key);
  void removePersistentMemory(key);
}

export function setMemory(key: string, value: string) {
  const data = normalizeMemoryValue(value);
  if (volatileKeys.has(key)) {
    memorySnapshot.delete(key);
    setSessionMemory(key, data);
    void removePersistentMemory(key);
    return;
  }

  memorySnapshot.set(key, data);
  void setPersistentMemory(key, data).then((success) => {
    if (!success) {
      setLegacyMemory(key, data);
    }
  });
}

export function setBooleanMemory(key: string, value: boolean) {
  setMemory(key, String(value));
}

export function setNumberMemory(key: string, value: number) {
  setMemory(key, value.toString());
}

export function setArrayMemory(key: string, value: string[]) {
  setMemory(key, value.join(","));
}

export function getMemory(key: string, defaultValue?: string): string {
  if (volatileKeys.has(key)) {
    return normalizeMemoryValue(getSessionMemory(key) || (defaultValue ?? ""));
  }

  const snapshotValue = memorySnapshot.get(key);
  if (snapshotValue !== undefined) return normalizeMemoryValue(snapshotValue);

  const legacyValue = getLegacyMemory(key);
  if (legacyValue) {
    const normalized = normalizeMemoryValue(legacyValue);
    memorySnapshot.set(key, normalized);
    void setPersistentMemory(key, normalized);
    return normalized;
  }

  return normalizeMemoryValue(getSessionMemory(key) || (defaultValue ?? ""));
}

export function getBooleanMemory(key: string, defaultValue: boolean): boolean {
  const value = getMemory(key);
  return value ? value === "true" : defaultValue;
}

export function getNumberMemory(key: string, defaultValue: number): number {
  const value = getMemory(key);
  return value ? Number(value) : defaultValue;
}

export function getArrayMemory(key: string): string[] {
  const value = getMemory(key);
  return value ? value.split(",") : [];
}

export function forgetMemory(key: string) {
  memorySnapshot.delete(key);
  removeSessionMemory(key);
  void removePersistentMemory(key);
}

export function clearMemory() {
  memorySnapshot.clear();
  clearSessionMemory();
  void clearPersistentMemory();
}

export function popMemory(key: string): string {
  const value = getMemory(key);
  forgetMemory(key);
  return value;
}
