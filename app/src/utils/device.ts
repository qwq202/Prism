import { useEffect, useState } from "react";
import { addEventListeners } from "@/utils/dom.ts";

export let mobile = isMobile();

window.addEventListener("resize", () => {
  mobile = isMobile();
});

export function isMobile(): boolean {
  return (
    (document.documentElement.clientWidth || window.innerWidth) <= 668 ||
    (document.documentElement.clientHeight || window.innerHeight) <= 468 ||
    navigator.userAgent.includes("Mobile") ||
    navigator.userAgent.includes("Android") ||
    navigator.userAgent.includes("iPhone") ||
    navigator.userAgent.includes("iPad") ||
    navigator.userAgent.includes("iPod") ||
    navigator.userAgent.includes("Watch")
  );
}

export function isSafari(): boolean {
  return (
    navigator.userAgent.includes("Safari") &&
    !navigator.userAgent.includes("Chrome") &&
    !navigator.userAgent.includes("Android") &&
    !navigator.userAgent.includes("Edge")
  );
}

export function useMobile(): boolean {
  const [mobile, setMobile] = useState<boolean>(isMobile);

  useEffect(() => {
    const handler = () => setMobile(isMobile);

    return addEventListeners(
      window,
      [
        "resize",
        "orientationchange",
        "touchstart",
        "touchmove",
        "touchend",
        "touchcancel",
        "gesturestart",
        "gesturechange",
        "gestureend",
      ],
      handler,
    );
  }, []);

  return mobile;
}

const safeUrlProtocols = new Set(["http:", "https:", "mailto:", "tel:", "blob:"]);

function getSafeNavigationUrl(url: string): string | null {
  try {
    const parsed = new URL(url, window.location.href);
    return safeUrlProtocols.has(parsed.protocol) ? parsed.href : null;
  } catch {
    return null;
  }
}

export function openWindow(url: string, target?: string): void {
  /**
   * Open a new window with the given URL.
   * If the device does not support opening a new window, the URL will be opened in the current window.
   * @param url The URL to open.
   * @param target The target of the URL.
   */

  const safeUrl = getSafeNavigationUrl(url);
  if (!safeUrl) return;

  if (mobile) {
    window.location.href = safeUrl;
  } else {
    window.open(safeUrl, target, "noopener,noreferrer");
  }
}

export function openPage(url: string): void {
  const safeUrl = getSafeNavigationUrl(url);
  if (!safeUrl) return;

  window.location.href = safeUrl;
}

export function openForm(
  url: string,
  method: string,
  params: Record<string, string>,
): void {
  /**
   * Open a new window with a form that submits the given parameters to the given URL.
   * If the device does not support opening a new window, the form will be submitted in the current window.
   * @param url The URL to open.
   * @param method The method of the form.
   * @param params The parameters of the form.
   */

  const form = document.createElement("form");
  const safeUrl = getSafeNavigationUrl(url);
  if (!safeUrl) return;

  form.style.display = "none";
  form.method = method;
  form.action = safeUrl;

  !isSafari() && form.setAttribute("target", "_blank");

  for (const key in params) {
    const input = document.createElement("input");
    input.type = "hidden";
    input.name = key;
    input.value = params[key];
    form.appendChild(input);
  }

  document.body.appendChild(form);
  form.submit();
  document.body.removeChild(form);
}
