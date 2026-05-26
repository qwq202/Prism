import { createAsyncThunk, createSlice } from "@reduxjs/toolkit";
import { RootState } from "@/store/index.ts";
import {
  getRecordUsageSummary,
  RecordData,
  RecordStats,
  RecordUsageSummary,
} from "@/api/record.ts";
import { toTimeZoneDateInputValue } from "@/utils/record-time.ts";

type RecordProps = {
  data: RecordData;
  stats: RecordStats;
  walletUsageSummary: RecordUsageSummary;
  walletUsageSummaryLoaded: boolean;
  walletUsageSummaryLoading: boolean;
  walletUsageSummaryTimeZone: string;
  page: number;
};

export const refreshWalletUsageSummary = createAsyncThunk(
  "record/refreshWalletUsageSummary",
  async (timeZone: string) => {
    const end = new Date();
    const start = new Date(end);
    start.setDate(start.getDate() - 6);

    const resp = await getRecordUsageSummary({
      self: true,
      start_time: toTimeZoneDateInputValue(start, timeZone),
      end_time: toTimeZoneDateInputValue(end, timeZone),
    });

    if (!resp.status || !resp.data) return null;
    return {
      timeZone,
      data: resp.data,
    };
  },
);

export const recordSlice = createSlice({
  name: "record",
  initialState: {
    data: {
      total: 0,
      records: [],
    },
    stats: {
      billing_today: 0,
      billing_month: 0,
      request_today: 0,
      request_month: 0,
      rpm: 0,
      tpm: 0,
    },
    walletUsageSummary: {
      model_count: 0,
      top_model: "--",
      average_quota: 0,
      max_quota: 0,
      models: [],
    },
    walletUsageSummaryLoaded: false,
    walletUsageSummaryLoading: false,
    walletUsageSummaryTimeZone: "",
    page: 0,
  } as RecordProps,
  reducers: {
    setData: (state, action) => {
      state.data = action.payload;
    },
    setPage: (state, action) => {
      state.page = action.payload;
    },
    setStats: (state, action) => {
      state.stats = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder.addCase(refreshWalletUsageSummary.pending, (state) => {
      state.walletUsageSummaryLoading = true;
    });
    builder.addCase(refreshWalletUsageSummary.fulfilled, (state, action) => {
      state.walletUsageSummaryLoading = false;
      state.walletUsageSummaryLoaded = true;
      state.walletUsageSummaryTimeZone =
        action.payload?.timeZone ?? action.meta.arg;
      if (action.payload) {
        state.walletUsageSummary = action.payload.data;
      }
    });
    builder.addCase(refreshWalletUsageSummary.rejected, (state, action) => {
      state.walletUsageSummaryLoading = false;
      state.walletUsageSummaryLoaded = true;
      state.walletUsageSummaryTimeZone = action.meta.arg;
    });
  },
});

export const { setData, setPage, setStats } = recordSlice.actions;
export default recordSlice.reducer;

export const dataSelector = (state: RootState) => state.record.data;
export const pageSelector = (state: RootState) => state.record.page;
export const statsSelector = (state: RootState) => state.record.stats;
export const walletUsageSummarySelector = (state: RootState) =>
  state.record.walletUsageSummary;
export const walletUsageSummaryLoadedSelector = (state: RootState) =>
  state.record.walletUsageSummaryLoaded;
export const walletUsageSummaryLoadingSelector = (state: RootState) =>
  state.record.walletUsageSummaryLoading;
export const walletUsageSummaryTimeZoneSelector = (state: RootState) =>
  state.record.walletUsageSummaryTimeZone;
