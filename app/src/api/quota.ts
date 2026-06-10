import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

const noCacheHeaders = {
  "Cache-Control": "no-cache",
  Pragma: "no-cache",
};

export type QuotaPreference = {
  quota: number;
  allow_subscription_quota_fallback: boolean;
};

type UpdateSubscriptionQuotaFallbackResponse = {
  status: boolean;
  error?: string;
  allow_subscription_quota_fallback?: boolean;
};

export async function getQuota(): Promise<QuotaPreference | null> {
  try {
    const response = await axios.get("/quota", {
      headers: noCacheHeaders,
      params: { _: Date.now() },
    });
    if (response.data.status) {
      const quota = Number(response.data.quota);
      return Number.isFinite(quota)
        ? {
            quota,
            allow_subscription_quota_fallback:
              response.data.allow_subscription_quota_fallback === true,
          }
        : null;
    }
  } catch (e) {
    console.debug(e);
  }

  return null;
}

export async function updateSubscriptionQuotaFallback(
  allow: boolean,
): Promise<UpdateSubscriptionQuotaFallbackResponse> {
  try {
    const response = await axios.post("/quota/subscription-fallback", {
      allow,
    });
    return response.data as UpdateSubscriptionQuotaFallbackResponse;
  } catch (e) {
    console.debug(e);
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}
