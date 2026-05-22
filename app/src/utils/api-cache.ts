import axios, {
  AxiosError,
  AxiosHeaders,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from "axios";
import { getClientCache, setClientCache } from "@/utils/client-cache.ts";
import { getMemory } from "@/utils/memory.ts";

type ApiCacheOptions = {
  endpoint: string;
  token: string;
};

type CachedAxiosResponse<T = unknown> = {
  data: T;
  status: number;
  statusText: string;
  headers: Record<string, string>;
};

type CachedResponseSignal<T = unknown> = {
  prismCachedResponse: true;
  response: AxiosResponse<T>;
};

declare module "axios" {
  export interface AxiosRequestConfig {
    prismCache?: false;
    prismCacheRefresh?: boolean;
  }
}

const cacheVersion = 1;
const cachePrefix = "axios";
const servedCacheKeys = new Set<string>();

let installed = false;
let options: ApiCacheOptions | null = null;
let bypassCacheUntil = 0;

function stableStringify(value: unknown): string {
  if (value === undefined) return "";
  if (value === null || typeof value !== "object") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(stableStringify).join(",")}]`;

  return `{${Object.keys(value as Record<string, unknown>)
    .sort()
    .map(
      (key) =>
        `${JSON.stringify(key)}:${stableStringify(
          (value as Record<string, unknown>)[key],
        )}`,
    )
    .join(",")}}`;
}

function hashValue(value: string): string {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash << 5) - hash + value.charCodeAt(i);
    hash |= 0;
  }

  return Math.abs(hash).toString(36);
}

function getHeaderValue(
  headers: InternalAxiosRequestConfig["headers"] | undefined,
  key: string,
): string {
  if (!headers) return "";
  if (headers instanceof AxiosHeaders)
    return headers.get(key)?.toString() ?? "";

  const found = Object.entries(headers).find(
    ([name]) => name.toLowerCase() === key.toLowerCase(),
  );
  return found?.[1]?.toString() ?? "";
}

function getPath(config: InternalAxiosRequestConfig): string {
  try {
    const url = new URL(
      config.url || "",
      config.baseURL || options?.endpoint || window.location.origin,
    );
    return url.pathname;
  } catch {
    return config.url || "";
  }
}

function isSafePostPath(path: string): boolean {
  return ["/record/view", "/record/stats", "/admin/analytics/channel"].some(
    (item) => path === item,
  );
}

function isMutatingGetPath(path: string): boolean {
  return [
    "/conversation/delete",
    "/conversation/clean",
    "/conversation/plugin/test",
    "/conversation/share/delete",
    "/memory/delete",
    "/admin/channel/delete",
    "/admin/channel/activate",
    "/admin/channel/deactivate",
    "/admin/charge/delete",
    "/admin/logger/download",
    "/admin/payment/recheck",
  ].some((item) => path.startsWith(item));
}

function isRealtimePath(path: string): boolean {
  return [
    "/payment/check",
    "/quota",
    "/broadcast/view",
    "/memory/list",
    "/admin/logger/console",
    "/v1/market",
    "/v1/models",
    "/v1/plans",
    "/v1/charge",
  ].some((item) => path.startsWith(item));
}

function isCacheable(config: InternalAxiosRequestConfig): boolean {
  if (config.prismCache === false || config.prismCacheRefresh) return false;
  if (!options) return false;
  if (Date.now() < bypassCacheUntil) return false;

  const method = (config.method || "get").toLowerCase();
  const path = getPath(config);
  if (isRealtimePath(path)) return false;
  if (method === "get") return !isMutatingGetPath(path);
  if (method === "post") return isSafePostPath(path);

  return false;
}

function isMutation(config: InternalAxiosRequestConfig | undefined): boolean {
  if (!config) return false;
  const method = (config.method || "get").toLowerCase();
  if (!["get", "head", "options"].includes(method)) return true;
  return isMutatingGetPath(getPath(config));
}

function getCacheKey(config: InternalAxiosRequestConfig): string {
  const method = (config.method || "get").toLowerCase();
  const auth =
    getHeaderValue(config.headers, "Authorization") ||
    getMemory(options?.token || "") ||
    "anonymous";

  const base = {
    auth: hashValue(auth),
    baseURL: config.baseURL || options?.endpoint || "",
    data: method === "post" ? stableStringify(config.data) : "",
    method,
    params: stableStringify(config.params),
    url: config.url || "",
    version: cacheVersion,
  };

  return `${cachePrefix}:${hashValue(stableStringify(base))}`;
}

function toAxiosResponse<T>(
  config: InternalAxiosRequestConfig,
  cached: CachedAxiosResponse<T>,
): AxiosResponse<T> {
  return {
    config,
    data: cached.data,
    headers: {
      ...cached.headers,
      "x-prism-cache": "hit",
    },
    request: undefined,
    status: cached.status,
    statusText: cached.statusText || "OK",
  };
}

function toCachedResponse<T>(
  response: AxiosResponse<T>,
): CachedAxiosResponse<T> {
  return {
    data: response.data,
    headers: response.headers as Record<string, string>,
    status: response.status,
    statusText: response.statusText,
  };
}

function shouldStoreResponse(response: AxiosResponse): boolean {
  if (!isCacheable(response.config as InternalAxiosRequestConfig)) return false;
  if (response.status < 200 || response.status >= 300) return false;
  if (response.config.responseType && response.config.responseType !== "json") {
    return false;
  }

  return response.data !== undefined;
}

function refreshCache(config: InternalAxiosRequestConfig, key: string): void {
  void axios
    .request({
      ...config,
      prismCacheRefresh: true,
    })
    .then((response) => {
      if (shouldStoreResponse(response)) {
        void setClientCache(key, toCachedResponse(response));
      }
      window.dispatchEvent(
        new CustomEvent("prism-api-cache-updated", {
          detail: { key, url: config.url },
        }),
      );
    })
    .catch((error) => {
      console.debug("[api-cache] background refresh failed:", error);
    });
}

function isCachedResponseSignal<T>(
  error: unknown,
): error is CachedResponseSignal<T> {
  return (
    typeof error === "object" &&
    error !== null &&
    (error as CachedResponseSignal<T>).prismCachedResponse === true
  );
}

export function installApiCache(nextOptions: ApiCacheOptions): void {
  options = nextOptions;
  if (installed) return;
  installed = true;

  axios.interceptors.request.use(async (config) => {
    if (!isCacheable(config)) return config;

    const key = getCacheKey(config);
    if (servedCacheKeys.has(key)) return config;

    const cached = await getClientCache<CachedAxiosResponse>(key);
    if (!cached) return config;

    servedCacheKeys.add(key);
    refreshCache(config, key);

    return Promise.reject({
      prismCachedResponse: true,
      response: toAxiosResponse(config, cached),
    } satisfies CachedResponseSignal);
  });

  axios.interceptors.response.use(
    (response) => {
      if (shouldStoreResponse(response)) {
        void setClientCache(
          getCacheKey(response.config as InternalAxiosRequestConfig),
          toCachedResponse(response),
        );
      }

      if (isMutation(response.config as InternalAxiosRequestConfig)) {
        bypassCacheUntil = Date.now() + 2000;
        servedCacheKeys.clear();
      }

      return response;
    },
    async (error: AxiosError | CachedResponseSignal) => {
      if (isCachedResponseSignal(error)) return error.response;

      const config = (error as AxiosError).config as
        | InternalAxiosRequestConfig
        | undefined;
      if (!config || !isCacheable(config)) return Promise.reject(error);

      const cached = await getClientCache<CachedAxiosResponse>(
        getCacheKey(config),
      );
      if (cached) return toAxiosResponse(config, cached);

      return Promise.reject(error);
    },
  );
}
