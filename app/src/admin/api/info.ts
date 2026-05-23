import axios from "axios";
import { setAppLogo, setAppName, setBuyLink, setDocsUrl } from "@/conf/env.ts";
import { infoEvent } from "@/events/info.ts";
import { initGoogleAnalytics } from "@/utils/analytics.ts";
import { BroadcastEvent, getBroadcast } from "@/api/broadcast";
import { getClientCache, setClientCache } from "@/utils/client-cache.ts";

const siteInfoCacheKey = "site-info";

function getSiteInfoCacheKey(): string {
  return `${siteInfoCacheKey}:${axios.defaults.baseURL || "default"}`;
}

export type SiteInfo = {
  title: string;
  logo: string;
  docs: string;
  timezone: string;
  backend?: string;
  currency: string;
  announcement: string;
  buy_link: string;
  mail: boolean;
  contact: string;
  footer: string;
  auth_footer: boolean;
  hide_key_docs?: boolean;
  web_search?: boolean;
  has_task_model?: boolean;
  payment: string[];
  payment_aggregation: boolean;
  ga_tracking_id?: string;
  broadcast?: BroadcastEvent;
};

export async function getSiteInfo(): Promise<SiteInfo> {
  try {
    const response = await axios.get("/info");
    const info = response.data as SiteInfo;
    void setClientCache(getSiteInfoCacheKey(), info);
    return info;
  } catch (e) {
    console.warn(e);
    const cached = await getCachedSiteInfo();
    if (cached) return cached;

    return {
      title: "",
      logo: "",
      docs: "",
      timezone: "Asia/Shanghai",
      backend: undefined,
      currency: "cny",
      announcement: "",
      buy_link: "",
      contact: "",
      footer: "",
      auth_footer: false,
      hide_key_docs: false,
      mail: false,
      web_search: false,
      has_task_model: false,
      payment: [],
      payment_aggregation: false,

      broadcast: {
        message: "",
        firstReceived: false,
      },
    };
  }
}

export async function getCachedSiteInfo(): Promise<SiteInfo | undefined> {
  return await getClientCache<SiteInfo>(getSiteInfoCacheKey());
}

function applySiteInfo(info: SiteInfo) {
  setAppName(info.title);
  setAppLogo(info.logo);
  setDocsUrl(info.docs);
  setBuyLink(info.buy_link);
  initGoogleAnalytics(info.ga_tracking_id);

  infoEvent.emit(info);
}

export function syncSiteInfo() {
  void getCachedSiteInfo().then((info) => {
    if (info) applySiteInfo(info);
  });

  setTimeout(async () => {
    const info = await getSiteInfo();
    info.broadcast = await getBroadcast();
    void setClientCache(getSiteInfoCacheKey(), info);

    applySiteInfo(info);
  }, 25);
}
