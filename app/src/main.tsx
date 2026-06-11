import ReactDOM from "react-dom/client";
import { dropLegacyBrowserStorageDatabase } from "@/utils/browser-storage.ts";
import { migrateLegacyClientCaches } from "@/utils/client-cache.ts";
import { initializeMemoryStorage } from "@/utils/memory.ts";
import "./assets/main.less";
import "./assets/globals.less";

async function main() {
  const memoryMigrated = await initializeMemoryStorage();
  const clientCachesMigrated = await migrateLegacyClientCaches();
  if (memoryMigrated && clientCachesMigrated) {
    await dropLegacyBrowserStorageDatabase();
  } else {
    console.debug(
      "[storage] kept legacy IndexedDB database because migration was incomplete",
    );
  }
  await import("./conf/bootstrap.ts");
  await import("./i18n.ts");

  const { default: App } = await import("./App.tsx");
  ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
}

void main();
