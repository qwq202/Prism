import { configureStore } from "@reduxjs/toolkit";
import infoReducer from "./info";
import globalReducer from "./globals";
import menuReducer from "./menu";
import authReducer from "./auth";
import chatReducer from "./chat";
import quotaReducer from "./quota";
import packageReducer from "./package";
import subscriptionReducer from "./subscription";
import sharingReducer from "./sharing";
import settingsReducer from "./settings";
import recordReducer from "./record";
import avatarReducer from "./avatar";
import {
  setCachedConversation,
  setCachedConversationList,
} from "@/utils/conversation-cache.ts";

const store = configureStore({
  reducer: {
    info: infoReducer,
    global: globalReducer,
    menu: menuReducer,
    auth: authReducer,
    chat: chatReducer,
    quota: quotaReducer,
    package: packageReducer,
    subscription: subscriptionReducer,
    sharing: sharingReducer,
    settings: settingsReducer,
    record: recordReducer,
    avatar: avatarReducer,
  },
});

let chatCacheTimer: ReturnType<typeof setTimeout> | undefined;
let lastChatCacheSignature = "";

store.subscribe(() => {
  if (chatCacheTimer) clearTimeout(chatCacheTimer);

  chatCacheTimer = setTimeout(() => {
    const { history, current, conversations } = store.getState().chat;
    const cacheableHistory = history.filter((item) => item.id !== -1);
    const currentConversation = current !== -1 ? conversations[current] : null;
    const signature = JSON.stringify({
      history: cacheableHistory.map((item) => ({
        id: item.id,
        name: item.name,
        model: item.model,
        shared: item.shared,
      })),
      current,
      currentConversation,
    });

    if (signature === lastChatCacheSignature) return;
    lastChatCacheSignature = signature;

    void setCachedConversationList(cacheableHistory);
    if (current !== -1 && currentConversation) {
      void setCachedConversation(current, currentConversation);
    }
  }, 500);
});

type RootState = ReturnType<typeof store.getState>;
type AppDispatch = typeof store.dispatch;

export function createCronJob(
  dispatch: AppDispatch,
  method: () => Parameters<AppDispatch>[0],
  interval: number,
  runWhenInit?: boolean,
) {
  if (runWhenInit) dispatch(method());
  return setInterval(() => dispatch(method()), interval * 1000);
}

export function clearCronJob(job: ReturnType<typeof setInterval>) {
  clearInterval(job);
}

export function clearCronJobs(jobs: ReturnType<typeof setInterval>[]) {
  jobs.forEach((job) => clearInterval(job));
}

export type { RootState, AppDispatch };
export default store;
