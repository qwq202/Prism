import { createSlice } from "@reduxjs/toolkit";
import { RootState } from "@/store/index.ts";
import {
  getArrayMemory,
  getBooleanMemory,
  getMemory,
  setArrayMemory,
  setBooleanMemory,
  setMemory,
} from "@/utils/memory.ts";
import { SiteInfo } from "@/admin/api/info.ts";
import { useSelector } from "react-redux";
import { BroadcastEvent } from "@/api/broadcast";

type Currency = {
  symbol: string;
  name: string;
  id: string;
};

export const currencyMap: Record<string, Currency> = {
  cny: { symbol: "￥", name: "CNY", id: "cny" },
  jpy: { symbol: "JP¥", name: "JPY", id: "jpy" },
  hkd: { symbol: "HK$", name: "HKD", id: "hkd" },
  usd: { symbol: "$", name: "USD", id: "usd" },
  eur: { symbol: "€", name: "EUR", id: "eur" },
  gbp: { symbol: "£", name: "GBP", id: "gbp" },
};

export const infoSlice = createSlice({
  name: "info",
  initialState: {
    mail: getBooleanMemory("mail", false),
    currency: getMemory("currency"),
    contact: getMemory("contact"),
    footer: getMemory("footer"),
    auth_footer: getBooleanMemory("auth_footer", false),
    web_search: getBooleanMemory("web_search", false),
    has_task_model: getBooleanMemory("has_task_model", false),
    payment: getArrayMemory("payment"),
    payment_aggregation: getBooleanMemory("payment_aggregation", false),
    title: getMemory("title"),
    logo: getMemory("logo"),
    docs: getMemory("docs"),
    timezone: getMemory("timezone") || "Asia/Shanghai",
    announcement: getMemory("announcement"),
    buy_link: getMemory("buy_link"),
    hide_key_docs: getBooleanMemory("hide_key_docs", false),
    backend: getMemory("backend"),
    group_pricing: {},

    broadcast: getMemory("broadcast_data")
      ? (JSON.parse(getMemory("broadcast_data")) as BroadcastEvent)
      : {
          message: "",
          firstReceived: false,
        },
    oauth_providers: {},
  } as SiteInfo & {
    broadcast: BroadcastEvent;
  },
  reducers: {
    setForm: (state, action) => {
      const form = action.payload as SiteInfo;
      state.mail = form.mail ?? false;
      state.currency = form.currency ?? "cny";
      state.contact = form.contact ?? "";
      state.footer = form.footer ?? "";
      state.auth_footer = form.auth_footer ?? false;
      state.web_search = form.web_search ?? false;
      state.has_task_model = form.has_task_model ?? false;
      state.payment = form.payment ?? [];
      state.title = form.title ?? "";
      state.logo = form.logo ?? "";
      state.docs = form.docs ?? "";
      state.timezone = form.timezone || "Asia/Shanghai";
      state.announcement = form.announcement ?? "";
      state.buy_link = form.buy_link ?? "";
      state.hide_key_docs = form.hide_key_docs ?? false;
      state.backend = form.backend ?? state.backend;
      state.payment_aggregation = form.payment_aggregation ?? false;
      state.broadcast = form.broadcast ?? {
        message: "",
        firstReceived: false,
      };
      setMemory("title", state.title);
      setMemory("logo", state.logo);
      setMemory("docs", state.docs);
      setMemory("timezone", state.timezone);
      setMemory("announcement", state.announcement);
      setMemory("buy_link", state.buy_link);
      setBooleanMemory("hide_key_docs", state.hide_key_docs);
      if (state.backend) setMemory("backend", state.backend);
      setMemory("currency", state.currency);
      setBooleanMemory("mail", state.mail);
      setMemory("contact", state.contact);
      setMemory("footer", state.footer);
      setBooleanMemory("auth_footer", state.auth_footer);
      setBooleanMemory("web_search", state.web_search);
      setBooleanMemory("has_task_model", state.has_task_model);
      setArrayMemory("payment", state.payment);
      setBooleanMemory("payment_aggregation", state.payment_aggregation);
      setMemory("broadcast_data", JSON.stringify(state.broadcast));
    },
  },
});

export const { setForm } = infoSlice.actions;

export default infoSlice.reducer;

export const infoMailSelector = (state: RootState): boolean => state.info.mail;
export const infoContactSelector = (state: RootState): string =>
  state.info.contact;
export const infoFooterSelector = (state: RootState): string =>
  state.info.footer;
export const infoAuthFooterSelector = (state: RootState): boolean =>
  state.info.auth_footer;
export const infoWebSearchSelector = (state: RootState): boolean =>
  state.info.web_search ?? false;
export const infoHasTaskModelSelector = (state: RootState): boolean =>
  state.info.has_task_model ?? false;
export const infoPaymentSelector = (state: RootState): string[] =>
  state.info.payment;
export const isPaymentAggregationSelector = (state: RootState): boolean =>
  state.info.payment_aggregation;
export const infoCurrencySelector = (state: RootState): string =>
  state.info.currency;
export const infoAnnouncementSelector = (state: RootState): string =>
  state.info.announcement;
export const infoHideKeyDocsSelector = (state: RootState): boolean =>
  state.info.hide_key_docs ?? false;
export const infoBackendSelector = (state: RootState): string | undefined =>
  state.info.backend;
export const infoTimeZoneSelector = (state: RootState): string =>
  state.info.timezone || "Asia/Shanghai";
export const infoBroadcastSelector = (state: RootState): BroadcastEvent =>
  state.info.broadcast;

export const useCurrency = (): Currency => {
  const currency = useSelector(infoCurrencySelector);
  return currencyMap[currency] ?? currencyMap.cny;
};
