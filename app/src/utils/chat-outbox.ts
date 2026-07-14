import type { ChatProps } from "@/api/connection.ts";
import { apiEndpoint, tokenField } from "@/conf/bootstrap.ts";
import {
  getClientCache,
  listClientCachesByPrefix,
  removeClientCache,
  setClientCache,
} from "@/utils/client-cache.ts";
import { getMemory } from "@/utils/memory.ts";

export type PendingChatRequest = {
  requestId: string;
  conversationId: number;
  props: ChatProps;
  createdAt: number;
  scope: string;
};

const legacyOutboxMigrations = new Map<string, Promise<void>>();

function hashScope(value: string): string {
  let first = 0x811c9dc5;
  let second = 0x9e3779b9;
  for (let index = 0; index < value.length; index++) {
    const code = value.charCodeAt(index);
    first = Math.imul(first ^ code, 0x01000193);
    second = Math.imul(second ^ code, 0x85ebca6b);
  }
  return `${(first >>> 0).toString(36)}${(second >>> 0).toString(36)}`;
}

function legacyHashScope(value: string): string {
  let hash = 0;
  for (let index = 0; index < value.length; index++) {
    hash = (hash << 5) - hash + value.charCodeAt(index);
    hash |= 0;
  }
  return Math.abs(hash).toString(36);
}

export function getChatOutboxScope(): string {
  const token = getMemory(tokenField) || "anonymous";
  return hashScope(`${apiEndpoint}\u0000${token}`);
}

function getOutboxPrefix(scope: string): string {
  return `chat-outbox:${scope}:`;
}

function getOutboxKey(scope: string, requestId: string): string {
  return `${getOutboxPrefix(scope)}${requestId}`;
}

function getLegacyOutboxKey(): string {
  const token = getMemory(tokenField) || "anonymous";
  return `chat-outbox:${apiEndpoint}:${legacyHashScope(token)}`;
}

async function migrateLegacyOutbox(scope: string): Promise<void> {
  if (scope !== getChatOutboxScope()) return;

  const current = legacyOutboxMigrations.get(scope);
  if (current) return current;

  const migration = (async () => {
    const legacyKey = getLegacyOutboxKey();
    const legacy = await getClientCache<
      Array<Omit<PendingChatRequest, "scope">>
    >(legacyKey);
    if (!Array.isArray(legacy) || legacy.length === 0) {
      await removeClientCache(legacyKey);
      return;
    }

    for (const request of legacy) {
      if (!request?.requestId || !request.props) continue;
      const scopedRequest: PendingChatRequest = { ...request, scope };
      await setClientCache(
        getOutboxKey(scope, scopedRequest.requestId),
        scopedRequest,
      );
    }
    await removeClientCache(legacyKey);
  })();
  legacyOutboxMigrations.set(scope, migration);
  return migration;
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

export async function getPendingChatRequests(
  scope = getChatOutboxScope(),
): Promise<PendingChatRequest[]> {
  await migrateLegacyOutbox(scope);
  const entries = await listClientCachesByPrefix<PendingChatRequest>(
    getOutboxPrefix(scope),
  );
  return entries
    .map(({ data }) => ({ ...data, scope }))
    .filter(
      (item) =>
        typeof item.requestId === "string" &&
        item.requestId.length > 0 &&
        item.props &&
        typeof item.props === "object",
    )
    .sort((left, right) => left.createdAt - right.createdAt);
}

export async function enqueuePendingChatRequest(
  request: Omit<PendingChatRequest, "scope"> & { scope?: string },
): Promise<PendingChatRequest> {
  const scopedRequest: PendingChatRequest = {
    ...request,
    scope: request.scope ?? getChatOutboxScope(),
  };
  const key = getOutboxKey(scopedRequest.scope, scopedRequest.requestId);
  await setClientCache(key, scopedRequest);

  const persisted = await getClientCache<PendingChatRequest>(key);
  if (persisted?.requestId !== scopedRequest.requestId) {
    throw new Error("failed to persist chat request before sending");
  }
  return scopedRequest;
}

export async function updatePendingChatRequestConversation(
  requestId: string,
  conversationId: number,
  scope = getChatOutboxScope(),
): Promise<void> {
  if (!requestId || conversationId < 0) return;
  const key = getOutboxKey(scope, requestId);
  const pending = await getClientCache<PendingChatRequest>(key);
  if (!pending || pending.conversationId === conversationId) return;
  await setClientCache(key, { ...pending, conversationId, scope });
}

export async function acknowledgePendingChatRequest(
  requestId: string,
  scope = getChatOutboxScope(),
): Promise<void> {
  if (!requestId) return;
  await removeClientCache(getOutboxKey(scope, requestId));
}

export async function clearPendingChatRequestsForConversation(
  conversationId: number,
  scope = getChatOutboxScope(),
): Promise<void> {
  const pending = await getPendingChatRequests(scope);
  await Promise.all(
    pending
      .filter((item) => item.conversationId === conversationId)
      .map((item) =>
        removeClientCache(getOutboxKey(scope, item.requestId)),
      ),
  );
}

export async function clearPendingChatRequests(
  scope = getChatOutboxScope(),
): Promise<void> {
  const pending = await getPendingChatRequests(scope);
  await Promise.all(
    pending.map((item) =>
      removeClientCache(getOutboxKey(scope, item.requestId)),
    ),
  );
}
