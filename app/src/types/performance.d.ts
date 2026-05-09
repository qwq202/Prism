declare global {
  // see https://developer.mozilla.org/en-US/docs/Web/API/Performance/memory

  interface PerformanceMemory {
    usedJSHeapSize: number;
    totalJSHeapSize: number;
    jsHeapSizeLimit: number;
  }

  interface Performance {
    memory: PerformanceMemory;
  }

  interface Tauri {
    window?: {
      getCurrentWindow?: () => {
        theme?: () => Promise<"dark" | "light" | null>;
      };
      appWindow?: {
        theme?: () => Promise<"dark" | "light" | null>;
      };
    };
  }

  interface Window {
    __TAURI__?: Tauri;
  }
}

export declare function getMemoryPerformance(): number;
