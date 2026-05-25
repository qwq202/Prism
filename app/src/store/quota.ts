import { createAsyncThunk, createSlice } from "@reduxjs/toolkit";
import { RootState } from "./index.ts";
import { getQuota } from "@/api/quota.ts";

export const quotaSlice = createSlice({
  name: "quota",
  initialState: {
    quota: 0,
  },
  reducers: {
    setQuota: (state, action) => {
      const quota = Number(action.payload);
      if (Number.isFinite(quota)) {
        state.quota = quota;
      }
    },
  },
  extraReducers: (builder) => {
    builder.addCase(refreshQuota.fulfilled, (state, action) => {
      if (action.payload !== null) {
        state.quota = action.payload;
      }
    });
  },
});

export const { setQuota } = quotaSlice.actions;
export default quotaSlice.reducer;

export const quotaSelector = (state: RootState): number => state.quota.quota;

export const refreshQuota = createAsyncThunk("quota/refreshQuota", async () => {
  return await getQuota();
});
