import axios from "axios";

export function normalizeImageURL(url: string): string {
  if (!url || url.startsWith("data:image/")) return url;

  try {
    const parsed = new URL(url);
    return parsed.href;
  } catch {
    // Relative URL. Continue below.
  }

  const baseURL = (axios.defaults.baseURL || "").replace(/\/+$/, "");
  if (!url.startsWith("/") || !baseURL) return url;

  if (baseURL.startsWith("http://") || baseURL.startsWith("https://")) {
    return new URL(url, baseURL).href;
  }

  if (url === baseURL || url.startsWith(`${baseURL}/`)) return url;
  return `${baseURL}${url}`;
}
