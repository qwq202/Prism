import { createAsyncThunk, createSlice } from "@reduxjs/toolkit";
import { RootState } from "./index.ts";
import { getQuota } from "@/api/quota.ts";

export const quotaSlice = createSlice({
  name: "quota",
  initialState: {
    quota: 0,
    allow_subscription_quota_fallback: true,
  },
  reducers: {
    setQuota: (state, action) => {
      const quota = Number(action.payload);
      if (Number.isFinite(quota)) {
        state.quota = quota;
      }
    },
    setAllowSubscriptionQuotaFallback: (state, action) => {
      state.allow_subscription_quota_fallback = action.payload !== false;
    },
  },
  extraReducers: (builder) => {
    builder.addCase(refreshQuota.fulfilled, (state, action) => {
      if (action.payload !== null) {
        state.quota = action.payload.quota;
        state.allow_subscription_quota_fallback =
          action.payload.allow_subscription_quota_fallback;
      }
    });
  },
});

export const { setQuota, setAllowSubscriptionQuotaFallback } =
  quotaSlice.actions;
export default quotaSlice.reducer;

export const quotaSelector = (state: RootState): number => state.quota.quota;
export const allowSubscriptionQuotaFallbackSelector = (
  state: RootState,
): boolean => state.quota.allow_subscription_quota_fallback;

export const refreshQuota = createAsyncThunk("quota/refreshQuota", async () => {
  return await getQuota();
});
