import type { ConversationInstance } from "@/api/types.tsx";
import { apiEndpoint, tokenField } from "@/conf/bootstrap.ts";
import {
  getClientCache,
  getClientCacheUpdateStorageKey,
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
  cache_complete?: boolean;
};

type ConversationListSerializedCache =
  | ConversationInstance[]
  | {
      version: 2;
      generation: number;
      conversations: ConversationInstance[];
    };

type SetCachedConversationListOptions = {
  ifGeneration?: number;
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

function getConversationListCacheGenerationKey(): string {
  return `conversation-list-generation:${getCacheScope()}`;
}

function getConversationCacheKey(id: number): string {
  return `conversation:${getCacheScope()}:${id}`;
}

function getConversationCacheKeyPrefix(): string {
  return `conversation:${getCacheScope()}:`;
}

export function isConversationListCacheStorageKey(key: string | null): boolean {
  const listCacheKey = getConversationListCacheKey();
  const generationKey = getConversationListCacheGenerationKey();

  return (
    key === getClientCacheStorageKey(listCacheKey) ||
    key === getClientCacheUpdateStorageKey(listCacheKey) ||
    key === getClientCacheStorageKey(generationKey) ||
    key === getClientCacheUpdateStorageKey(generationKey)
  );
}

function readConversationListFromCachePayload(
  cached: ConversationListSerializedCache | undefined,
  currentGeneration: number,
): ConversationInstance[] | undefined {
  if (!cached) return undefined;

  if (Array.isArray(cached)) {
    return currentGeneration === 0 ? cached : undefined;
  }

  if (
    cached.version !== 2 ||
    cached.generation !== currentGeneration ||
    !Array.isArray(cached.conversations)
  ) {
    return undefined;
  }

  return cached.conversations;
}

export async function getConversationListCacheGeneration(): Promise<number> {
  return (
    (await getClientCache<number>(getConversationListCacheGenerationKey())) ?? 0
  );
}

export async function isCurrentConversationListCacheGeneration(
  generation: number,
): Promise<boolean> {
  return (await getConversationListCacheGeneration()) === generation;
}

async function bumpConversationListCacheGeneration(): Promise<number> {
  const current = await getConversationListCacheGeneration();
  const next = Math.max(current + 1, Date.now());

  await setClientCache(getConversationListCacheGenerationKey(), next);
  return next;
}

export async function getCachedConversationList(): Promise<
  ConversationInstance[] | undefined
> {
  const currentGeneration = await getConversationListCacheGeneration();
  const cached = await getClientCache<ConversationListSerializedCache>(
    getConversationListCacheKey(),
  );

  return readConversationListFromCachePayload(cached, currentGeneration);
}

export async function setCachedConversationList(
  conversations: ConversationInstance[],
  options?: SetCachedConversationListOptions,
): Promise<boolean> {
  const generation =
    options?.ifGeneration ?? (await getConversationListCacheGeneration());

  if (
    options?.ifGeneration !== undefined &&
    !(await isCurrentConversationListCacheGeneration(options.ifGeneration))
  ) {
    return false;
  }

  await setClientCache(getConversationListCacheKey(), {
    version: 2,
    generation,
    conversations,
  });
  return true;
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
  const generation = await bumpConversationListCacheGeneration();

  await setCachedConversationList([], { ifGeneration: generation });
  await removeClientCachesByPrefix(getConversationCacheKeyPrefix());
}
