import axios from "axios";
import type { ConversationInstance } from "./types.tsx";
import { setHistory, setRemoteHistory } from "@/store/chat.ts";
import { AppDispatch } from "@/store";
import { CommonResponse } from "@/api/common.ts";
import { getErrorMessage } from "@/utils/base.ts";
import { VirtualWebSearchRole, VirtualRolePrefix, Message } from "./types.tsx";
import { formatToolCallResult } from "@/api/plugin.ts";
import {
  getCachedConversationList,
  setCachedConversationList,
} from "@/utils/conversation-cache.ts";
import { getVisibleToolCalls } from "@/api/tool-calls.ts";

type ConversationListResult = {
  conversations: ConversationInstance[];
  fromCache: boolean;
};

type ConversationLoadResult =
  | {
      status: "ok";
      conversation: ConversationInstance;
    }
  | {
      status: "not_found" | "error";
      conversation?: undefined;
    };

const noCacheHeaders = {
  "Cache-Control": "no-cache",
  Pragma: "no-cache",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isConversationList(value: unknown): value is ConversationInstance[] {
  return (
    Array.isArray(value) &&
    value.every(
      (item) =>
        isRecord(item) &&
        typeof item.id === "number" &&
        typeof item.name === "string",
    )
  );
}

function readConversationListPayload(
  value: unknown,
  visited = new WeakSet<object>(),
): ConversationInstance[] | undefined {
  if (isConversationList(value)) return value;

  if (typeof value === "string") {
    const text = value.trim();
    if (!text || (!text.startsWith("{") && !text.startsWith("["))) {
      return undefined;
    }

    try {
      return readConversationListPayload(JSON.parse(text), visited);
    } catch {
      return undefined;
    }
  }

  if (!isRecord(value)) return undefined;
  if (visited.has(value)) return undefined;
  visited.add(value);

  const candidates = [
    value.data,
    value.conversations,
    value.list,
    value.items,
    value.history,
    value.rows,
    value.results,
  ];

  for (const candidate of candidates) {
    if (isConversationList(candidate)) return candidate;
    if (isRecord(candidate) || typeof candidate === "string") {
      const nested = readConversationListPayload(candidate, visited);
      if (nested) return nested;
    }
  }

  for (const candidate of Object.values(value)) {
    if (candidates.includes(candidate)) continue;
    if (isConversationList(candidate)) return candidate;
    if (isRecord(candidate) || typeof candidate === "string") {
      const nested = readConversationListPayload(candidate, visited);
      if (nested) return nested;
    }
  }

  return undefined;
}

export async function getConversationList(): Promise<ConversationInstance[]> {
  return (await fetchConversationList()).conversations;
}

export async function fetchConversationList(): Promise<ConversationListResult> {
  try {
    const resp = await axios.get("/conversation/list", {
      headers: noCacheHeaders,
      params: { _: Date.now() },
    });
    if (resp.data?.status === false) {
      throw new Error(resp.data.message || "failed to fetch conversations");
    }

    const conversations = readConversationListPayload(resp.data);
    if (!conversations) {
      throw new Error("invalid conversation list response");
    }

    void setCachedConversationList(
      conversations.filter((item) => item.id !== -1),
    );
    return { conversations, fromCache: false };
  } catch (e) {
    const message = getErrorMessage(e);
    const conversations = (await getCachedConversationList()) ?? [];
    if (message === "invalid conversation list response") {
      console.debug("[conversation] failed to refresh list:", message);
    } else {
      console.warn("[conversation] failed to refresh list:", message);
    }
    return {
      conversations,
      fromCache: true,
    };
  }
}

export async function updateConversationList(
  dispatch: AppDispatch,
): Promise<void> {
  const resp = await fetchConversationList();
  if (resp.fromCache && resp.conversations.length === 0) return;

  dispatch(
    resp.fromCache
      ? setHistory(resp.conversations)
      : setRemoteHistory(resp.conversations),
  );
}

export async function fetchConversation(
  id: number,
): Promise<ConversationLoadResult> {
  try {
    const resp = await axios.get("/conversation/load", {
      headers: noCacheHeaders,
      params: { id, _: Date.now() },
    });

    if (resp.data.status) {
      const conversation = resp.data.data as ConversationInstance;

      if (conversation.message && conversation.message.length > 0) {
        const processedMessages: Message[] = [];

        for (let i = 0; i < conversation.message.length; i++) {
          const currentMsg = conversation.message[i];

          if (
            currentMsg.role === "assistant" &&
            !currentMsg.model &&
            conversation.model
          ) {
            currentMsg.model = conversation.model;
          }

          if (currentMsg.role === VirtualWebSearchRole) {
            let nextMsgIndex = i + 1;
            while (
              nextMsgIndex < conversation.message.length &&
              conversation.message[nextMsgIndex].role.startsWith(
                VirtualRolePrefix,
              )
            ) {
              nextMsgIndex++;
            }

            if (nextMsgIndex < conversation.message.length) {
              conversation.message[nextMsgIndex].search_query =
                currentMsg.search_query;
              conversation.message[nextMsgIndex].search_result =
                currentMsg.search_result;
              conversation.message[nextMsgIndex].search_index =
                currentMsg.search_index;
            }

            continue;
          }

          if (currentMsg.role === "assistant" && currentMsg.tool_calls) {
            const visibleToolCalls = getVisibleToolCalls(
              currentMsg.tool_calls,
            ).map((toolCall) => ({
              ...toolCall,
              status: toolCall.status ?? "success",
            }));
            if (visibleToolCalls.length > 0) {
              currentMsg.tool_calls = visibleToolCalls;
            } else {
              delete currentMsg.tool_calls;
            }
            processedMessages.push(currentMsg);
          } else if (currentMsg.role === "tool" && currentMsg.tool_call_id) {
            const toolCallId = currentMsg.tool_call_id;
            for (let j = processedMessages.length - 1; j >= 0; j--) {
              const prevMsg = processedMessages[j];
              if (prevMsg.role === "assistant" && prevMsg.tool_calls) {
                const toolCall = prevMsg.tool_calls.find(
                  (tc) => tc.id === toolCallId,
                );
                if (toolCall) {
                  try {
                    const result = JSON.parse(currentMsg.content);
                    if (result.error) {
                      toolCall.error = result.error;
                      toolCall.status = "error";
                    } else {
                      const formattedResult = formatToolCallResult(
                        currentMsg.content,
                      );
                      toolCall.result = formattedResult;
                      toolCall.status = "success";
                    }
                  } catch {
                    const formattedResult = formatToolCallResult(
                      currentMsg.content,
                    );
                    toolCall.result = formattedResult;
                    toolCall.status = "success";
                  }
                }
                break;
              }
            }
            processedMessages.push(currentMsg);
          } else {
            processedMessages.push(currentMsg);
          }
        }

        conversation.message = processedMessages;
      }

      return { status: "ok", conversation };
    }
    return { status: "not_found" };
  } catch (e) {
    console.warn(e);
    return { status: "error" };
  }
}

export async function loadConversation(
  id: number,
): Promise<ConversationInstance> {
  const resp = await fetchConversation(id);
  if (resp.status === "ok") return resp.conversation;
  return { id, name: "", message: [] };
}

export async function deleteConversation(id: number): Promise<boolean> {
  try {
    const resp = await axios.post(
      "/conversation/delete",
      { id },
      {
        headers: noCacheHeaders,
      },
    );
    return resp.data.status;
  } catch (e) {
    console.warn(e);
    return false;
  }
}

export async function renameConversation(
  id: number,
  name: string,
): Promise<CommonResponse> {
  try {
    const resp = await axios.post("/conversation/rename", { id, name });
    return resp.data as CommonResponse;
  } catch (e) {
    console.warn(e);
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function updateConversationModel(
  id: number,
  model: string,
): Promise<CommonResponse> {
  try {
    const resp = await axios.post("/conversation/model", { id, model });
    return resp.data as CommonResponse;
  } catch (e) {
    console.warn(e);
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function retitleConversation(id: number): Promise<CommonResponse> {
  try {
    const resp = await axios.post("/conversation/retitle", { id });
    return resp.data as CommonResponse;
  } catch (e) {
    console.warn(e);
    return { status: false, error: getErrorMessage(e) };
  }
}

export async function deleteAllConversations(): Promise<boolean> {
  try {
    const resp = await axios.post(
      "/conversation/clean",
      {},
      {
        headers: noCacheHeaders,
      },
    );
    return resp.data.status;
  } catch (e) {
    console.warn(e);
    return false;
  }
}
