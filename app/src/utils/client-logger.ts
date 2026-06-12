import axios, {
  AxiosError,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from "axios";
import { version } from "@/conf/version.json";

type LogLevel = "debug" | "info" | "log" | "warn" | "error";

type ClientLogEntry = {
  id: string;
  time: string;
  offset_ms: number;
  level: LogLevel;
  source: string;
  message: string;
  data?: Record<string, unknown>;
};

type ClientLogExport = {
  metadata: Record<string, unknown>;
  entries: ClientLogEntry[];
};

type RequestMeta = {
  id: string;
  startedAt: number;
};

type CachedResponseSignal = {
  prismCachedResponse?: boolean;
  response?: AxiosResponse;
};

type ConnectionInfo = {
  effectiveType?: string;
  downlink?: number;
  rtt?: number;
  saveData?: boolean;
  type?: string;
};

type NavigatorWithConnection = Navigator & {
  connection?: ConnectionInfo;
};

type PerformanceWithMemory = Performance & {
  memory?: {
    usedJSHeapSize?: number;
    totalJSHeapSize?: number;
    jsHeapSizeLimit?: number;
  };
};

const storageKey = "prism-client-diagnostic-log:v1";
const maxEntries = 1500;
const maxStoredPayloadLength = 1_500_000;
const maxStringLength = 1200;
const maxArrayItems = 20;
const maxObjectKeys = 30;
const sensitiveKeyPattern =
  /authorization|password|passwd|secret|token|api[_-]?key|access[_-]?key|refresh|credential|code|email|phone|content|prompt|response_prompts|prompts/i;
const requestMeta = new WeakMap<InternalAxiosRequestConfig, RequestMeta>();

let installed = false;
let entries: ClientLogEntry[] = loadStoredEntries();
let lastPersistAt = 0;
let persistTimer: ReturnType<typeof setTimeout> | undefined;

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function nowIso(): string {
  return new Date().toISOString();
}

function createId(prefix = "log"): string {
  const random =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2);

  return `${prefix}_${random}`;
}

function elapsedSincePageStart(): number {
  if (!isBrowser() || !performance?.now) return 0;
  return Math.round(performance.now());
}

function clampString(value: string): string {
  const redacted = redactString(value);
  if (redacted.length <= maxStringLength) return redacted;

  return `${redacted.slice(0, maxStringLength)}...<truncated>`;
}

function redactString(value: string): string {
  return value
    .replace(/Bearer\s+[A-Za-z0-9._~+/=-]+/gi, "Bearer <redacted>")
    .replace(/sk-[A-Za-z0-9_-]{10,}/g, "<redacted-api-key>")
    .replace(
      /(token|password|secret|code|api[_-]?key)=([^&\s]+)/gi,
      "$1=<redacted>",
    );
}

function sanitizeUrl(value: string | undefined): string {
  if (!value) return "";

  try {
    const base = isBrowser() ? window.location.origin : "http://localhost";
    const url = new URL(value, base);
    const params = Array.from(url.searchParams.keys());
    const query =
      params.length > 0
        ? `?${params.map((key) => `${key}=...`).join("&")}`
        : "";

    return clampString(`${url.pathname}${query}`);
  } catch {
    return clampString(value.split("?")[0] || value);
  }
}

function describeTarget(target: EventTarget | null): Record<string, unknown> {
  if (!target || !(target instanceof Element)) return {};

  const src =
    target instanceof HTMLImageElement ||
    target instanceof HTMLScriptElement ||
    target instanceof HTMLIFrameElement
      ? target.src
      : target instanceof HTMLLinkElement
        ? target.href
        : undefined;

  return {
    tag: target.tagName.toLowerCase(),
    id: target.id || undefined,
    className:
      typeof target.className === "string"
        ? clampString(target.className)
        : undefined,
    src: src ? sanitizeUrl(src) : undefined,
  };
}

function serializeError(error: unknown): Record<string, unknown> {
  if (error instanceof Error) {
    return {
      name: error.name,
      message: clampString(error.message),
      stack: error.stack ? clampString(error.stack) : undefined,
    };
  }

  return {
    value: serializeValue(error),
  };
}

function serializeValue(value: unknown, depth = 0): unknown {
  if (value === undefined || value === null) return value;
  if (typeof value === "string") return clampString(value);
  if (typeof value === "number" || typeof value === "boolean") return value;
  if (typeof value === "bigint") return value.toString();
  if (typeof value === "function")
    return `[function ${value.name || "anonymous"}]`;
  if (value instanceof Error) return serializeError(value);
  if (value instanceof Event) {
    return {
      type: value.type,
      target: describeTarget(value.target),
    };
  }
  if (value instanceof Element) return describeTarget(value);
  if (Array.isArray(value)) {
    if (depth >= 2) return `[array length=${value.length}]`;

    return value
      .slice(0, maxArrayItems)
      .map((item) => serializeValue(item, depth + 1));
  }
  if (typeof value === "object") {
    if (depth >= 2) return Object.prototype.toString.call(value);

    const result: Record<string, unknown> = {};
    for (const key of Object.keys(value).slice(0, maxObjectKeys)) {
      if (sensitiveKeyPattern.test(key)) {
        result[key] = "<redacted>";
        continue;
      }

      result[key] = serializeValue(
        (value as Record<string, unknown>)[key],
        depth + 1,
      );
    }

    return result;
  }

  return String(value);
}

function readStorage(): ClientLogEntry[] {
  if (!isBrowser()) return [];

  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return [];

    const parsed = JSON.parse(raw) as Partial<ClientLogExport>;
    if (!Array.isArray(parsed.entries)) return [];

    return parsed.entries.filter((entry): entry is ClientLogEntry => {
      return (
        typeof entry === "object" &&
        typeof entry.id === "string" &&
        typeof entry.time === "string" &&
        typeof entry.source === "string" &&
        typeof entry.message === "string"
      );
    });
  } catch {
    return [];
  }
}

function loadStoredEntries(): ClientLogEntry[] {
  return readStorage().slice(-maxEntries);
}

function getMemorySnapshot(): Record<string, unknown> | undefined {
  if (!isBrowser()) return undefined;

  const memory = (performance as PerformanceWithMemory).memory;
  if (!memory) return undefined;

  return {
    used_mb: memory.usedJSHeapSize
      ? Math.round((memory.usedJSHeapSize / 1024 / 1024) * 100) / 100
      : undefined,
    total_mb: memory.totalJSHeapSize
      ? Math.round((memory.totalJSHeapSize / 1024 / 1024) * 100) / 100
      : undefined,
    limit_mb: memory.jsHeapSizeLimit
      ? Math.round((memory.jsHeapSizeLimit / 1024 / 1024) * 100) / 100
      : undefined,
  };
}

function getConnectionSnapshot(): Record<string, unknown> | undefined {
  if (!isBrowser()) return undefined;

  const connection = (navigator as NavigatorWithConnection).connection;
  if (!connection) return undefined;

  return {
    effective_type: connection.effectiveType,
    downlink: connection.downlink,
    rtt: connection.rtt,
    save_data: connection.saveData,
    type: connection.type,
  };
}

function getMetadata(): Record<string, unknown> {
  if (!isBrowser()) {
    return {
      app_version: version,
      exported_at: nowIso(),
      environment: "non-browser",
    };
  }

  return {
    app_version: version,
    exported_at: nowIso(),
    page_loaded_at: new Date(
      Date.now() - Math.round(performance.now?.() ?? 0),
    ).toISOString(),
    url: sanitizeUrl(window.location.href),
    path: window.location.pathname,
    referrer: document.referrer ? sanitizeUrl(document.referrer) : "",
    user_agent: navigator.userAgent,
    language: navigator.language,
    languages: navigator.languages,
    platform: navigator.platform,
    online: navigator.onLine,
    viewport: {
      width: window.innerWidth,
      height: window.innerHeight,
      device_pixel_ratio: window.devicePixelRatio,
    },
    screen: {
      width: window.screen?.width,
      height: window.screen?.height,
      avail_width: window.screen?.availWidth,
      avail_height: window.screen?.availHeight,
    },
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    memory: getMemorySnapshot(),
    connection: getConnectionSnapshot(),
    entry_count: entries.length,
    storage_key: storageKey,
  };
}

function persistEntries(): void {
  if (!isBrowser()) return;

  try {
    let payload = JSON.stringify({
      metadata: getMetadata(),
      entries,
    } satisfies ClientLogExport);

    while (payload.length > maxStoredPayloadLength && entries.length > 100) {
      entries = entries.slice(Math.ceil(entries.length * 0.2));
      payload = JSON.stringify({
        metadata: getMetadata(),
        entries,
      } satisfies ClientLogExport);
    }

    localStorage.setItem(storageKey, payload);
    lastPersistAt = Date.now();
  } catch {
    // Diagnostics must never break the app.
  }
}

function schedulePersist(immediate = false): void {
  if (!isBrowser()) return;

  if (immediate || Date.now() - lastPersistAt > 1500) {
    if (persistTimer) {
      clearTimeout(persistTimer);
      persistTimer = undefined;
    }
    persistEntries();
    return;
  }

  if (persistTimer) return;
  persistTimer = setTimeout(() => {
    persistTimer = undefined;
    persistEntries();
  }, 1500);
}

export function logClientEvent(
  source: string,
  message: string,
  data?: Record<string, unknown>,
  level: LogLevel = "info",
): void {
  const entry: ClientLogEntry = {
    id: createId(),
    time: nowIso(),
    offset_ms: elapsedSincePageStart(),
    level,
    source,
    message: clampString(message),
    data: data ? (serializeValue(data) as Record<string, unknown>) : undefined,
  };

  entries = [...entries, entry].slice(-maxEntries);
  schedulePersist(level === "error" || level === "warn");
}

function summarizeRequest(
  config: InternalAxiosRequestConfig,
): Record<string, unknown> {
  return {
    id: requestMeta.get(config)?.id,
    method: (config.method || "get").toUpperCase(),
    url: sanitizeUrl(config.url),
    base_url: config.baseURL ? sanitizeUrl(config.baseURL) : undefined,
    timeout: config.timeout,
    with_credentials: config.withCredentials,
    cached: Boolean(config.prismCache),
  };
}

function summarizeResponse(response: AxiosResponse): Record<string, unknown> {
  const config = response.config as InternalAxiosRequestConfig;
  const meta = requestMeta.get(config);

  return {
    ...summarizeRequest(config),
    status: response.status,
    status_text: response.statusText,
    duration_ms: meta
      ? Math.round(performance.now() - meta.startedAt)
      : undefined,
  };
}

function summarizeAxiosError(error: AxiosError): Record<string, unknown> {
  const config = error.config as InternalAxiosRequestConfig | undefined;
  const meta = config ? requestMeta.get(config) : undefined;

  return {
    ...(config ? summarizeRequest(config) : {}),
    status: error.response?.status,
    status_text: error.response?.statusText,
    code: error.code,
    message: clampString(error.message),
    duration_ms: meta
      ? Math.round(performance.now() - meta.startedAt)
      : undefined,
  };
}

function isCachedResponseSignal(error: unknown): error is CachedResponseSignal {
  return (
    typeof error === "object" &&
    error !== null &&
    (error as CachedResponseSignal).prismCachedResponse === true
  );
}

function installConsoleCapture(): void {
  (["debug", "info", "log", "warn", "error"] as LogLevel[]).forEach((level) => {
    const original = console[level].bind(console);

    console[level] = (...args: unknown[]) => {
      logClientEvent(
        "console",
        args.map((item) => String(serializeValue(item))).join(" "),
        {
          args: args.map((item) => serializeValue(item)),
        },
        level,
      );
      original(...args);
    };
  });
}

function installWindowCapture(): void {
  window.addEventListener("error", (event) => {
    logClientEvent(
      "window.error",
      event.message || "window error",
      {
        filename: event.filename ? sanitizeUrl(event.filename) : undefined,
        line: event.lineno,
        column: event.colno,
        error: event.error ? serializeError(event.error) : undefined,
        target: describeTarget(event.target),
      },
      "error",
    );
  });

  window.addEventListener("unhandledrejection", (event) => {
    logClientEvent(
      "window.unhandledrejection",
      "unhandled promise rejection",
      {
        reason: serializeValue(event.reason),
      },
      "error",
    );
  });

  document.addEventListener(
    "error",
    (event) => {
      logClientEvent(
        "resource.error",
        "resource failed to load",
        {
          target: describeTarget(event.target),
        },
        "warn",
      );
    },
    true,
  );

  document.addEventListener("visibilitychange", () => {
    logClientEvent("page.visibility", document.visibilityState);
  });
  window.addEventListener("online", () => logClientEvent("network", "online"));
  window.addEventListener("offline", () =>
    logClientEvent("network", "offline", undefined, "warn"),
  );
  window.addEventListener("pagehide", () => persistEntries());
}

function installNavigationCapture(): void {
  const originalPushState = history.pushState;
  const originalReplaceState = history.replaceState;

  history.pushState = function pushState(...args) {
    const result = originalPushState.apply(this, args);
    logClientEvent("navigation", "pushState", {
      url: typeof args[2] === "string" ? sanitizeUrl(args[2]) : undefined,
      path: window.location.pathname,
    });
    return result;
  };

  history.replaceState = function replaceState(...args) {
    const result = originalReplaceState.apply(this, args);
    logClientEvent("navigation", "replaceState", {
      url: typeof args[2] === "string" ? sanitizeUrl(args[2]) : undefined,
      path: window.location.pathname,
    });
    return result;
  };

  window.addEventListener("popstate", () => {
    logClientEvent("navigation", "popstate", {
      path: window.location.pathname,
    });
  });
}

function installAxiosCapture(): void {
  axios.interceptors.request.use((config) => {
    const typedConfig = config as InternalAxiosRequestConfig;
    const meta: RequestMeta = {
      id: createId("req"),
      startedAt: performance.now(),
    };

    requestMeta.set(typedConfig, meta);
    logClientEvent(
      "http.request",
      `${typedConfig.method || "GET"} ${sanitizeUrl(typedConfig.url)}`,
      {
        ...summarizeRequest(typedConfig),
      },
    );
    return config;
  });

  axios.interceptors.response.use(
    (response) => {
      logClientEvent(
        "http.response",
        `${response.status} ${sanitizeUrl(response.config.url)}`,
        summarizeResponse(response),
        response.status >= 400 ? "warn" : "info",
      );
      return response;
    },
    (error: unknown) => {
      if (isCachedResponseSignal(error)) return Promise.reject(error);

      if (!axios.isAxiosError(error)) {
        logClientEvent(
          "http.error",
          "request failed",
          {
            error: serializeValue(error),
          },
          "error",
        );
        return Promise.reject(error);
      }

      logClientEvent(
        "http.error",
        error.message || "request failed",
        summarizeAxiosError(error),
        "error",
      );
      return Promise.reject(error);
    },
  );
}

export function installClientLogger(): void {
  if (!isBrowser() || installed) return;

  installed = true;
  installConsoleCapture();
  installWindowCapture();
  installNavigationCapture();
  installAxiosCapture();
  logClientEvent("client-logger", "installed", {
    path: window.location.pathname,
    previous_entries: entries.length,
  });
}

export function getClientDiagnosticLogFile(): {
  filename: string;
  content: string;
} {
  logClientEvent("client-logger", "export requested", {
    path: isBrowser() ? window.location.pathname : undefined,
  });
  persistEntries();

  const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
  const payload: ClientLogExport = {
    metadata: getMetadata(),
    entries: [...entries],
  };

  return {
    filename: `prism-client-log-${timestamp}.json`,
    content: JSON.stringify(payload, null, 2),
  };
}
