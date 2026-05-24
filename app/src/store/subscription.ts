import { createAsyncThunk, createSlice } from "@reduxjs/toolkit";
import { getSubscription } from "@/api/addition.ts";
import { RootState } from "@/store/index.ts";

export type SubscriptionUsageValue = {
  used: number;
  total: number;
  unit?: "times" | "points";
  reset_interval?: number;
  reset_at?: string;
};

export const subscriptionSlice = createSlice({
  name: "subscription",
  initialState: {
    is_subscribed: false,
    level: 0,
    enterprise: false,
    expired: 0,
    expired_at: "",
    refresh: 0,
    refresh_at: "",
    usage: {},
  },
  reducers: {},
  extraReducers: (builder) => {
    builder.addCase(refreshSubscription.fulfilled, (state, action) => {
      if (!action.payload.status) return;
      state.is_subscribed = action.payload.is_subscribed;
      state.expired = action.payload.expired;
      state.usage = action.payload.usage || {};
      state.enterprise = action.payload.enterprise || false;
      state.level = action.payload.level;
      state.expired_at = action.payload.expired_at || "";
      state.refresh = action.payload.refresh || 0;
      state.refresh_at = action.payload.refresh_at || "";
    });
  },
});

export default subscriptionSlice.reducer;

export const isSubscribedSelector = (state: RootState): boolean =>
  state.subscription.is_subscribed;
export const levelSelector = (state: RootState): number =>
  state.subscription.level;
export const expiredSelector = (state: RootState): number =>
  state.subscription.expired;
export const expiredAtSelector = (state: RootState): string =>
  state.subscription.expired_at;
export const refreshSelector = (state: RootState): number =>
  state.subscription.refresh;
export const refreshAtSelector = (state: RootState): string =>
  state.subscription.refresh_at;
export const usageSelector = (
  state: RootState,
): Record<string, number | SubscriptionUsageValue> => state.subscription.usage;

export const refreshSubscription = createAsyncThunk(
  "subscription/refreshSubscription",
  async () => {
    return await getSubscription();
  },
);
