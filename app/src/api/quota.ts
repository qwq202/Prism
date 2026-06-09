import axios from "axios";

const noCacheHeaders = {
  "Cache-Control": "no-cache",
  Pragma: "no-cache",
};

export async function getQuota(): Promise<number | null> {
  try {
    const response = await axios.get("/quota", {
      headers: noCacheHeaders,
      params: { _: Date.now() },
    });
    if (response.data.status) {
      const quota = Number(response.data.quota);
      return Number.isFinite(quota) ? quota : null;
    }
  } catch (e) {
    console.debug(e);
  }

  return null;
}
