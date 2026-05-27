import type { ConversationInstance } from "@/api/types.tsx";
import { apiEndpoint, tokenField } from "@/conf/bootstrap.ts";
import {
  getClientCache,
  getClientCacheStorageKey,
  removeClientCache,
  removeClientCachesByPrefix,
  setClientCache,
} from "@/utils/client-cache.ts";
import { getMemory } from "@/utils/memory.ts";

type ConversationSerializedCache = {
  model?: string;
  messages: ConversationInstance["message"];
  updated_at?: string;
};

function hashCacheScope(value: string): string {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash << 5) - hash + value.charCodeAt(i);
    hash |= 0;
  }

  return Math.abs(hash).toString(36);
}

function getCacheScope(): string {
  const token = getMemory(tokenField) || "anonymous";
  return `${apiEndpoint}:${hashCacheScope(token)}`;
}

function getConversationListCacheKey(): string {
  return `conversation-list:${getCacheScope()}`;
}

function getConversationCacheKey(id: number): string {
  return `conversation:${getCacheScope()}:${id}`;
}

function getConversationCacheKeyPrefix(): string {
  return `conversation:${getCacheScope()}:`;
}

export function isConversationListCacheStorageKey(
  key: string | null,
): boolean {
  return key === getClientCacheStorageKey(getConversationListCacheKey());
}

export async function getCachedConversationList(): Promise<
  ConversationInstance[] | undefined
> {
  return await getClientCache<ConversationInstance[]>(
    getConversationListCacheKey(),
  );
}

export async function setCachedConversationList(
  conversations: ConversationInstance[],
): Promise<void> {
  await setClientCache(getConversationListCacheKey(), conversations);
}

export async function removeCachedConversationFromList(
  id: number,
): Promise<void> {
  const conversations = await getCachedConversationList();
  if (!conversations) return;

  await setCachedConversationList(
    conversations.filter((conversation) => conversation.id !== id),
  );
}

export async function getCachedConversation(
  id: number,
): Promise<ConversationSerializedCache | undefined> {
  if (id === -1) return undefined;
  return await getClientCache<ConversationSerializedCache>(
    getConversationCacheKey(id),
  );
}

export async function setCachedConversation(
  id: number,
  conversation: ConversationSerializedCache,
): Promise<void> {
  if (id === -1) return;
  await setClientCache(getConversationCacheKey(id), conversation);
}

export async function clearCachedConversation(id: number): Promise<void> {
  if (id === -1) return;
  await removeClientCache(getConversationCacheKey(id));
}

export async function clearCachedConversations(): Promise<void> {
  await setCachedConversationList([]);
  await removeClientCachesByPrefix(getConversationCacheKeyPrefix());
}
