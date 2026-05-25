import axios from "axios";

export async function getQuota(): Promise<number | null> {
  try {
    const response = await axios.get("/quota");
    if (response.data.status) {
      const quota = Number(response.data.quota);
      return Number.isFinite(quota) ? quota : null;
    }
  } catch (e) {
    console.debug(e);
  }

  return null;
}
