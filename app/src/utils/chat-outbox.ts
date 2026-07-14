import type { ChatProps } from "@/api/connection.ts";
import { apiEndpoint, tokenField } from "@/conf/bootstrap.ts";
import { getClientCache, setClientCache } from "@/utils/client-cache.ts";
import { getMemory } from "@/utils/memory.ts";

export type PendingChatRequest = {
  requestId: string;
  conversationId: number;
  props: ChatProps;
  createdAt: number;
};

let mutationQueue: Promise<void> = Promise.resolve();

function hashScope(value: string): string {
  let hash = 0;
  for (let index = 0; index < value.length; index++) {
    hash = (hash << 5) - hash + value.charCodeAt(index);
    hash |= 0;
  }
  return Math.abs(hash).toString(36);
}

function getOutboxKey(): string {
  const token = getMemory(tokenField) || "anonymous";
  return `chat-outbox:${apiEndpoint}:${hashScope(token)}`;
}

export function createChatRequestID(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }
  return `chat-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export async function getPendingChatRequests(): Promise<PendingChatRequest[]> {
  const pending = await getClientCache<PendingChatRequest[]>(getOutboxKey());
  return Array.isArray(pending) ? pending : [];
}

export async function enqueuePendingChatRequest(
  request: PendingChatRequest,
): Promise<void> {
  const key = getOutboxKey();
  const mutation = mutationQueue.then(async () => {
    const pending = (await getClientCache<PendingChatRequest[]>(key)) ?? [];
    const next = [
      ...pending.filter((item) => item.requestId !== request.requestId),
      request,
    ];
    await setClientCache(key, next);

    const persisted = await getClientCache<PendingChatRequest[]>(key);
    if (!persisted?.some((item) => item.requestId === request.requestId)) {
      throw new Error("failed to persist chat request before sending");
    }
  });
  mutationQueue = mutation.catch(() => undefined);
  await mutation;
}

export async function acknowledgePendingChatRequest(
  requestId: string,
): Promise<void> {
  if (!requestId) return;
  const key = getOutboxKey();
  const mutation = mutationQueue.then(async () => {
    const pending = (await getClientCache<PendingChatRequest[]>(key)) ?? [];
    if (!pending.some((item) => item.requestId === requestId)) return;
    await setClientCache(
      key,
      pending.filter((item) => item.requestId !== requestId),
    );
  });
  mutationQueue = mutation.catch(() => undefined);
  await mutation;
}

export async function clearPendingChatRequestsForConversation(
  conversationId: number,
): Promise<void> {
  const key = getOutboxKey();
  const mutation = mutationQueue.then(async () => {
    const pending = (await getClientCache<PendingChatRequest[]>(key)) ?? [];
    await setClientCache(
      key,
      pending.filter((item) => item.conversationId !== conversationId),
    );
  });
  mutationQueue = mutation.catch(() => undefined);
  await mutation;
}

export async function clearPendingChatRequests(): Promise<void> {
  const key = getOutboxKey();
  const mutation = mutationQueue.then(() => setClientCache(key, []));
  mutationQueue = mutation.catch(() => undefined);
  await mutation;
}
