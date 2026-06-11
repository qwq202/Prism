import ReactDOM from "react-dom/client";
import { dropLegacyBrowserStorageDatabase } from "@/utils/browser-storage.ts";
import { migrateLegacyClientCaches } from "@/utils/client-cache.ts";
import { initializeMemoryStorage } from "@/utils/memory.ts";
import "./assets/main.less";
import "./assets/globals.less";

async function main() {
  await initializeMemoryStorage();
  await migrateLegacyClientCaches();
  await dropLegacyBrowserStorageDatabase();
  await import("./conf/bootstrap.ts");
  await import("./i18n.ts");

  const { default: App } = await import("./App.tsx");
  ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
}

void main();
