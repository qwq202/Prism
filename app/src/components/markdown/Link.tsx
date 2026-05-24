import React from "react";
import { Codepen, Codesandbox, Github, Twitter, Youtube } from "lucide-react";
import { VirtualMessage } from "./VirtualMessage";

const safeLinkProtocols = new Set([
  "http:",
  "https:",
  "mailto:",
  "tel:",
  "blob:",
]);

function getSafeLink(url: string): string | null {
  const value = url.trim();
  if (!value) return null;

  try {
    const parsed = new URL(value, window.location.href);
    if (!safeLinkProtocols.has(parsed.protocol)) return null;
    return value;
  } catch {
    return null;
  }
}

function getSocialIcon(url: string) {
  try {
    const { hostname } = new URL(url, window.location.href);

    if (hostname.includes("github.com"))
      return <Github className="h-4 w-4 inline-block mr-0.5" />;
    if (hostname.includes("twitter.com"))
      return <Twitter className="h-4 w-4 inline-block mr-0.5" />;
    if (hostname.includes("youtube.com"))
      return <Youtube className="h-4 w-4 inline-block mr-0.5" />;
    if (hostname.includes("codepen.io"))
      return <Codepen className="h-4 w-4 inline-block mr-0.5" />;
    if (hostname.includes("codesandbox.io"))
      return <Codesandbox className="h-4 w-4 inline-block mr-0.5" />;
  } catch (e) {
    return;
  }
}

type LinkProps = {
  href?: string;
  children: React.ReactNode;
};

function Link({ href, children }: LinkProps) {
  const url: string = href?.toString() || "";
  const safeUrl = getSafeLink(url);

  if (url.startsWith("https://coai.virtual/reference::")) {
    const referenceUrl = url.slice("https://coai.virtual/reference::".length);
    return (
      <VirtualMessage message={`reference::${referenceUrl}`}>
        {children}
      </VirtualMessage>
    );
  }

  if (url.startsWith("https://coai.virtual")) {
    const message = url.slice(20);

    return <VirtualMessage message={message}>{children}</VirtualMessage>;
  }

  if (!safeUrl) {
    return <span>{children}</span>;
  }

  return (
    <a href={safeUrl} target={`_blank`} rel={`noopener noreferrer`}>
      {getSocialIcon(safeUrl)}
      {children}
    </a>
  );
}

export default Link;
