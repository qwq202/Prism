import localforage from "localforage";

export const browserStorageDatabaseName = "Prism";
export const legacyBrowserStorageDatabaseName = "coai";
export const clientCacheStoreName = "client-cache";
export const memoryStoreName = "memory";

export function createBrowserStorageStore(
  storeName: string,
  databaseName = browserStorageDatabaseName,
): LocalForage {
  return localforage.createInstance({
    driver: localforage.INDEXEDDB,
    name: databaseName,
    storeName,
  });
}

export async function dropLegacyBrowserStorageDatabase(): Promise<void> {
  if (typeof window === "undefined") return;

  try {
    await localforage.dropInstance({
      name: legacyBrowserStorageDatabaseName,
    });
  } catch (error) {
    console.debug(
      "[storage] failed to drop legacy IndexedDB database",
      legacyBrowserStorageDatabaseName,
      error,
    );
  }
}
