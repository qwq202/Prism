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
    article: getArrayMemory("article"),
    generation: getArrayMemory("generation"),
    footer: getMemory("footer"),
    auth_footer: getBooleanMemory("auth_footer", false),
    relay_plan: getBooleanMemory("relay_plan", false),
    web_search: getBooleanMemory("web_search", false),
    has_task_model: getBooleanMemory("has_task_model", false),
    payment: getArrayMemory("payment"),
    payment_aggregation: getBooleanMemory("payment_aggregation", false),
    title: getMemory("title"),
    logo: getMemory("logo"),
    file: getMemory("file"),
    docs: getMemory("docs"),
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
      state.article = form.article ?? [];
      state.generation = form.generation ?? [];
      state.footer = form.footer ?? "";
      state.auth_footer = form.auth_footer ?? false;
      state.relay_plan = form.relay_plan ?? false;
      state.web_search = form.web_search ?? false;
      state.has_task_model = form.has_task_model ?? false;
      state.payment = form.payment ?? [];
      state.title = form.title ?? "";
      state.logo = form.logo ?? "";
      state.file = form.file ?? "";
      state.docs = form.docs ?? "";
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
      setMemory("file", state.file);
      setMemory("docs", state.docs);
      setMemory("announcement", state.announcement);
      setMemory("buy_link", state.buy_link);
      setBooleanMemory("hide_key_docs", state.hide_key_docs);
      if (state.backend) setMemory("backend", state.backend);
      setMemory("currency", state.currency);
      setBooleanMemory("mail", state.mail);
      setMemory("contact", state.contact);
      setArrayMemory("article", state.article);
      setArrayMemory("generation", state.generation);
      setMemory("footer", state.footer);
      setBooleanMemory("auth_footer", state.auth_footer);
      setBooleanMemory("relay_plan", state.relay_plan);
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
export const infoArticleSelector = (state: RootState): string[] =>
  state.info.article;
export const infoGenerationSelector = (state: RootState): string[] =>
  state.info.generation;
export const infoFooterSelector = (state: RootState): string =>
  state.info.footer;
export const infoAuthFooterSelector = (state: RootState): boolean =>
  state.info.auth_footer;
export const infoRelayPlanSelector = (state: RootState): boolean =>
  state.info.relay_plan;
export const infoWebSearchSelector = (state: RootState): boolean =>
  (state.info as any).web_search ?? false;
export const infoHasTaskModelSelector = (state: RootState): boolean =>
  (state.info as any).has_task_model ?? false;
export const infoPaymentSelector = (state: RootState): string[] =>
  state.info.payment;
export const isPaymentAggregationSelector = (state: RootState): boolean =>
  state.info.payment_aggregation;
export const infoCurrencySelector = (state: RootState): string =>
  state.info.currency;
export const infoAnnouncementSelector = (state: RootState): string =>
  state.info.announcement;
export const infoHideKeyDocsSelector = (state: RootState): boolean =>
  (state.info as any).hide_key_docs ?? false;
export const infoBackendSelector = (state: RootState): string | undefined =>
  (state.info as any).backend;
export const infoBroadcastSelector = (state: RootState): BroadcastEvent =>
  state.info.broadcast;

export const useCurrency = (): Currency => {
  const currency = useSelector(infoCurrencySelector);
  return currencyMap[currency] ?? currencyMap.cny;
};
