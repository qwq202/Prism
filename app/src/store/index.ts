import { configureStore } from "@reduxjs/toolkit";
import infoReducer from "./info";
import globalReducer from "./globals";
import menuReducer from "./menu";
import authReducer from "./auth";
import chatReducer, { type ConversationSerialized } from "./chat";
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
let lastChatHistoryCacheSignature = "";
let lastChatConversationCacheSignature = "";

function isStreamingConversation(
  conversation: ConversationSerialized,
): boolean {
  const last = conversation.messages[conversation.messages.length - 1];
  return last?.role === "assistant" && last.end === false;
}

function getCacheableConversation(conversation: ConversationSerialized): {
  model: ConversationSerialized["model"];
  messages: ConversationSerialized["messages"];
  updated_at: ConversationSerialized["updated_at"];
} {
  const messages = isStreamingConversation(conversation)
    ? conversation.messages.slice(0, -1)
    : conversation.messages;

  return {
    model: conversation.model,
    messages,
    updated_at: conversation.updated_at,
  };
}

store.subscribe(() => {
  if (chatCacheTimer) clearTimeout(chatCacheTimer);

  chatCacheTimer = setTimeout(() => {
    const { history, current, conversations } = store.getState().chat;
    const cacheableHistory = history.filter((item) => item.id !== -1);
    const currentConversation = current !== -1 ? conversations[current] : null;
    const cacheableConversation = currentConversation
      ? getCacheableConversation(currentConversation)
      : null;
    const historySignature = JSON.stringify(
      cacheableHistory.map((item) => ({
        id: item.id,
        name: item.name,
        model: item.model,
        shared: item.shared,
        favorite: item.favorite,
        updated_at: item.updated_at,
      })),
    );

    if (historySignature !== lastChatHistoryCacheSignature) {
      lastChatHistoryCacheSignature = historySignature;
      void setCachedConversationList(cacheableHistory);
    }

    if (current !== -1 && cacheableConversation) {
      const conversationSignature = JSON.stringify({
        current,
        conversation: cacheableConversation,
      });

      if (conversationSignature !== lastChatConversationCacheSignature) {
        lastChatConversationCacheSignature = conversationSignature;
        void setCachedConversation(current, cacheableConversation);
      }
    } else if (lastChatConversationCacheSignature !== "") {
      lastChatConversationCacheSignature = "";
    }
  }, 500);
});

type RootState = ReturnType<typeof store.getState>;
type AppDispatch = typeof store.dispatch;
type CronJobFactory = () => unknown;

export function createCronJob(
  dispatch: AppDispatch,
  method: CronJobFactory,
  interval: number,
  runWhenInit?: boolean,
) {
  const run = () => dispatch(method() as Parameters<AppDispatch>[0]);
  if (runWhenInit) run();
  return setInterval(run, interval * 1000);
}

export function clearCronJob(job: ReturnType<typeof setInterval>) {
  clearInterval(job);
}

export function clearCronJobs(jobs: ReturnType<typeof setInterval>[]) {
  jobs.forEach((job) => clearInterval(job));
}

export type { RootState, AppDispatch };
export default store;
