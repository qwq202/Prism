import { tokenField, websocketEndpoint } from "@/conf/bootstrap.ts";
import { getMemory } from "@/utils/memory.ts";
import { getErrorMessage } from "@/utils/base.ts";
import { Mask } from "@/masks/types.ts";
import type { TFunction } from "i18next";
import { logClientEvent } from "@/utils/client-logger.ts";
import {
  acknowledgePendingChatRequest,
  getChatOutboxScope,
  updatePendingChatRequestConversation,
} from "@/utils/chat-outbox.ts";

export const endpoint = `${websocketEndpoint}/chat`;
export const maxRetry = 60; // 30s max websocket retry
export const maxConnection = 5;
const staleConnectionMs = 45_000;
const connectingTimeoutMs = 10_000;
const requestAckRetryMs = 3_000;
const requestCapabilityRetryMs = 5_000;
const requestAckCapability = "chat_request_ack_v1";

export type StreamMessage = {
  conversation?: number;
  request_id?: string;
  request_status?: "reserved" | "accepted" | "completed" | "rejected";
  accepted?: boolean;
  retryable?: boolean;
  keyword?: string;
  quota?: number;
  message: string;
  end: boolean;
  plan?: boolean;
  title?: string;
  search_query?: {
    type: string;
    search_queries: string[];
  };
  search_result?: {
    type: string;
    search_results: Array<{
      url: string;
      title: string;
      snippet: string;
      published_at?: number;
      site_name?: string;
      site_icon?: string;
    }>;
  };
  search_index?: {
    type: string;
    search_indexes: Array<{
      url: string;
      cite_index: number;
    }>;
  };
  tool_call?: {
    id?: string;
    name: string;
    arguments?: string;
    result?: string;
    error?: string;
    status: "start" | "executing" | "pending" | "success" | "error";
  };
  response_type?: string;
  capabilities?: string[];
};

export type ChatProps = {
  type?: string;
  request_id?: string;
  message: string;
  model: string;
  transient?: boolean;
  web?: boolean;
  web_search?: boolean;
  url_context?: boolean;
  x_search?: boolean;
  fetch?: boolean;
  learning_mode?: boolean;
  gemini_thinking_budget?: number;
  deepseek_thinking_enabled?: boolean;
  deepseek_reasoning_effort?: string;
  openai_reasoning_effort?: string;
  openai_reasoning_summary?: string;
  response_format?: unknown;
  thinking?: unknown;
  mask_context?: Array<{ role: string; content: string }>;
  web_search_mode?: "quick" | "detailed";
  web_page_summary?: boolean;
  think?: boolean;
  context?: number;
  ignore_context?: boolean;
  custom_instruction?: string;
  memory_enabled?: boolean;
  memory_history_enabled?: boolean;
  tool_call_id?: string;
  tool_result?: unknown;

  // mcp related fields
  enable_mcp?: boolean;
  mcp_plugin_id?: number;

  max_tokens?: number;
  temperature?: number;
  top_p?: number;
  top_k?: number;
  presence_penalty?: number;
  frequency_penalty?: number;
  repetition_penalty?: number;
};

type StreamCallback = (id: number, message: StreamMessage) => void;

type PendingRequest = {
  t: TFunction | undefined;
  data: ChatProps;
  scope: string;
  pollAfterAccepted: boolean;
  timer?: ReturnType<typeof setTimeout>;
};

type PendingRequestOptions = {
  scope?: string;
  resumed?: boolean;
};

const connectionStacks = new Set<ConnectionStack>();
let lifecycleListenersAttached = false;

function attachConnectionLifecycleListeners(): void {
  if (lifecycleListenersAttached || typeof window === "undefined") return;

  lifecycleListenersAttached = true;
  const reconnectStaleConnections = (reason: string) => {
    connectionStacks.forEach((stack) => stack.reconnectStale(reason));
  };
  const handleVisibilityChange = () => {
    if (typeof document === "undefined") return;
    if (document.visibilityState === "visible") {
      reconnectStaleConnections("visibility");
    }
  };

  window.addEventListener("focus", () => reconnectStaleConnections("focus"));
  window.addEventListener("pageshow", () =>
    reconnectStaleConnections("pageshow"),
  );
  window.addEventListener("online", () => reconnectStaleConnections("online"));
  if (typeof document !== "undefined") {
    document.addEventListener("visibilitychange", handleVisibilityChange);
  }
}

function summarizeChatProps(data: ChatProps): Record<string, unknown> {
  return {
    type: data.type ?? "chat",
    transient: data.transient,
    model: data.model,
    message_length:
      typeof data.message === "string" ? data.message.length : undefined,
    web: data.web,
    web_search: data.web_search,
    url_context: data.url_context,
    x_search: data.x_search,
    fetch: data.fetch,
    learning_mode: data.learning_mode,
    context: data.context,
    ignore_context: data.ignore_context,
    memory_enabled: data.memory_enabled,
    memory_history_enabled: data.memory_history_enabled,
    tool_call_id: data.tool_call_id,
    has_tool_result: data.tool_result !== undefined,
    max_tokens: data.max_tokens,
    temperature: data.temperature,
    top_p: data.top_p,
    top_k: data.top_k,
    presence_penalty: data.presence_penalty,
    frequency_penalty: data.frequency_penalty,
    repetition_penalty: data.repetition_penalty,
    gemini_thinking_budget: data.gemini_thinking_budget,
    deepseek_thinking_enabled: data.deepseek_thinking_enabled,
    deepseek_reasoning_effort: data.deepseek_reasoning_effort,
    openai_reasoning_effort: data.openai_reasoning_effort,
    openai_reasoning_summary: data.openai_reasoning_summary,
    has_response_format: Boolean(data.response_format),
    has_thinking: Boolean(data.thinking),
    mask_context_count: data.mask_context?.length,
    enable_mcp: data.enable_mcp,
    mcp_plugin_id: data.mcp_plugin_id,
  };
}

function summarizeStreamMessage(
  message: StreamMessage,
): Record<string, unknown> {
  return {
    conversation: message.conversation,
    message_length:
      typeof message.message === "string" ? message.message.length : undefined,
    end: message.end,
    keyword: message.keyword,
    quota: message.quota,
    plan: message.plan,
    has_title: Boolean(message.title),
    response_type: message.response_type,
    search_query_count: message.search_query?.search_queries.length,
    search_result_count: message.search_result?.search_results.length,
    search_index_count: message.search_index?.search_indexes.length,
    tool_call: message.tool_call
      ? {
          name: message.tool_call.name,
          status: message.tool_call.status,
          has_arguments: Boolean(message.tool_call.arguments),
          has_result: Boolean(message.tool_call.result),
          has_error: Boolean(message.tool_call.error),
        }
      : undefined,
  };
}

export class Connection {
  protected connection?: WebSocket;
  protected callback?: StreamCallback;
  protected reconnectTimer?: ReturnType<typeof setTimeout>;
  protected stack?: Record<string, unknown>;
  protected lastActivityAt: number;
  protected streamStats: {
    chunks: number;
    totalMessageLength: number;
    startedAt: number;
  };
  protected disposed: boolean;
  protected pendingRequests: Map<string, PendingRequest>;
  protected requestAckSupported?: boolean;
  protected requestCapabilityTimer?: ReturnType<typeof setTimeout>;
  protected incomingQueue: Promise<void>;
  protected pendingStreamMessage?: StreamMessage;
  protected streamFrame?: number;
  public id: number;
  public state: boolean;

  public constructor(id: number, callback?: StreamCallback) {
    this.state = false;
    this.disposed = false;
    this.pendingRequests = new Map();
    this.requestAckSupported = undefined;
    this.incomingQueue = Promise.resolve();
    this.id = id;
    this.lastActivityAt = Date.now();
    this.streamStats = {
      chunks: 0,
      totalMessageLength: 0,
      startedAt: performance.now(),
    };

    callback && this.setCallback(callback);
  }

  public init(): void {
    if (this.disposed) return;
    const reconnecting = this.connection !== undefined;
    if (reconnecting) {
      this.pendingRequests.forEach((pending) => {
        pending.pollAfterAccepted = true;
      });
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
    if (this.requestCapabilityTimer) {
      clearTimeout(this.requestCapabilityTimer);
      this.requestCapabilityTimer = undefined;
    }
    this.requestAckSupported = undefined;
    this.closeCurrentSocket();

    const socket = new WebSocket(endpoint);
    this.connection = socket;
    this.state = false;
    this.lastActivityAt = Date.now();
    logClientEvent("chat.websocket", "init", {
      id: this.id,
      endpoint,
    });
    socket.onopen = () => {
      if (socket !== this.connection || this.disposed) return;
      this.state = true;
      this.lastActivityAt = Date.now();
      logClientEvent("chat.websocket", "open", {
        id: this.id,
      });
      this.send({
        token: getMemory(tokenField) || "anonymous",
        id: this.id,
      });
      this.beginRequestCapabilityNegotiation();
    };
    socket.onclose = (event) => {
      if (socket !== this.connection) return;
      this.state = false;
      this.lastActivityAt = Date.now();
      if (this.requestCapabilityTimer) {
        clearTimeout(this.requestCapabilityTimer);
        this.requestCapabilityTimer = undefined;
      }
      if (this.disposed) return;

      this.stack = {
        error: "websocket connection failed",
        code: event.code,
        reason: event.reason,
        endpoint: endpoint,
      };
      logClientEvent(
        "chat.websocket",
        "close",
        {
          id: this.id,
          code: event.code,
          reason: event.reason,
          was_clean: event.wasClean,
        },
        event.wasClean ? "info" : "warn",
      );

      if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
      this.reconnectTimer = setTimeout(() => {
        if (this.disposed) return;
        console.debug(`[connection] reconnecting... (id: ${this.id})`);
        logClientEvent("chat.websocket", "reconnect", {
          id: this.id,
        });
        this.init();
      }, 3000);
    };
    socket.onerror = () => {
      if (socket !== this.connection || this.disposed) return;
      logClientEvent(
        "chat.websocket",
        "error",
        {
          id: this.id,
          ready_state: socket.readyState,
        },
        "warn",
      );
    };
    socket.onmessage = (event) => {
      this.incomingQueue = this.incomingQueue.then(async () => {
        if (socket !== this.connection || this.disposed) return;
        this.lastActivityAt = Date.now();
        try {
          const message = JSON.parse(event.data) as StreamMessage;
          if (this.handleCapabilities(message)) return;
          await this.handleRequestState(message);
          this.triggerCallback(message);
        } catch (e) {
          console.warn(
            `[connection] failed to parse websocket message: ${getErrorMessage(e)}`,
          );
          logClientEvent(
            "chat.websocket",
            "parse-error",
            {
              id: this.id,
              error: getErrorMessage(e),
              raw_length:
                typeof event.data === "string" ? event.data.length : undefined,
            },
            "warn",
          );
        }
      });
    };
  }

  protected closeCurrentSocket(): void {
    const socket = this.connection;
    if (!socket) return;

    socket.onopen = null;
    socket.onclose = null;
    socket.onerror = null;
    socket.onmessage = null;

    this.flushPendingStreamMessage();

    if (
      socket.readyState === WebSocket.CONNECTING ||
      socket.readyState === WebSocket.OPEN
    ) {
      socket.close();
    }
    if (socket === this.connection) this.connection = undefined;
  }

  public reconnect(reason = "manual"): void {
    this.disposed = false;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    logClientEvent("chat.websocket", "reconnect", {
      id: this.id,
      reason,
      ready_state: this.connection?.readyState,
      idle_ms: Date.now() - this.lastActivityAt,
    });
    this.init();
  }

  public reconnectStale(reason = "stale"): void {
    if (this.disposed) return;

    const readyState = this.connection?.readyState;
    const idleMs = Date.now() - this.lastActivityAt;
    const stale =
      !this.connection ||
      readyState === WebSocket.CLOSED ||
      readyState === WebSocket.CLOSING ||
      (readyState === WebSocket.CONNECTING && idleMs > connectingTimeoutMs) ||
      (readyState === WebSocket.OPEN && idleMs > staleConnectionMs);

    if (!stale) return;

    logClientEvent("chat.websocket", "stale-reconnect", {
      id: this.id,
      reason,
      ready_state: readyState,
      idle_ms: idleMs,
    });
    this.reconnect(reason);
  }

  public send(data: Record<string, unknown>): boolean {
    this.reconnectStale("before-send");
    if (!this.connection || this.connection.readyState !== WebSocket.OPEN) {
      this.state = false;
      if (
        this.connection === undefined ||
        this.connection.readyState === WebSocket.CLOSED
      ) {
        this.init();
      }
      console.debug("[connection] connection not ready, retrying in 500ms...");
      logClientEvent("chat.websocket", "send-not-ready", {
        id: this.id,
        ready_state: this.connection?.readyState,
        type: typeof data.type === "string" ? data.type : undefined,
      });
      return false;
    }
    this.connection.send(JSON.stringify(data));
    this.lastActivityAt = Date.now();
    logClientEvent("chat.websocket", "send", {
      id: this.id,
      type: typeof data.type === "string" ? data.type : "auth",
      model: typeof data.model === "string" ? data.model : undefined,
      message_length:
        typeof data.message === "string" ? data.message.length : undefined,
    });
    return true;
  }

  public sendWithRetry(
    t: TFunction | undefined,
    data: ChatProps,
    times?: number,
    options?: PendingRequestOptions,
  ): void {
    const requestID = data.request_id?.trim();
    if (requestID && (data.type ?? "chat") === "chat") {
      const current = this.pendingRequests.get(requestID);
      this.pendingRequests.set(requestID, {
        t,
        data,
        scope: options?.scope ?? current?.scope ?? getChatOutboxScope(),
        pollAfterAccepted:
          options?.resumed ?? current?.pollAfterAccepted ?? false,
        timer: current?.timer,
      });
      this.flushPendingRequest(requestID);
      return;
    }

    try {
      if (!times || times < maxRetry) {
        if (!this.send(data)) {
          logClientEvent("chat.websocket", "send-retry", {
            id: this.id,
            attempt: (times ?? 0) + 1,
            ...summarizeChatProps(data),
          });
          setTimeout(() => {
            this.sendWithRetry(t, data, (times ?? 0) + 1);
          }, 500);
        }

        return;
      }
    } catch (e) {
      console.warn(
        `[connection] failed to send message: ${getErrorMessage(e)}`,
      );
      logClientEvent(
        "chat.websocket",
        "send-error",
        {
          id: this.id,
          error: getErrorMessage(e),
          ...summarizeChatProps(data),
        },
        "error",
      );
    }

    const trace = JSON.stringify(
      {
        ...(this.stack ?? {
          error: "websocket connection unavailable",
          endpoint: endpoint,
        }),
        type: data.type ?? "chat",
        model: data.model,
        message_length:
          typeof data.message === "string" ? data.message.length : undefined,
      },
      null,
      2,
    );
    this.stack = undefined;
    logClientEvent(
      "chat.websocket",
      "send-failed",
      {
        id: this.id,
        ...summarizeChatProps(data),
      },
      "error",
    );

    t &&
      this.triggerCallback({
        message: `${t("request-failed")}\n\`\`\`json\n${trace}\n\`\`\`\n`,
        end: true,
      });
  }

  protected flushPendingRequest(requestID: string): void {
    const pending = this.pendingRequests.get(requestID);
    if (!pending || this.disposed) return;
    if (pending.timer) clearTimeout(pending.timer);

    if (pending.scope !== getChatOutboxScope()) {
      this.pendingRequests.delete(requestID);
      return;
    }

    if (!this.connection || this.connection.readyState === WebSocket.CLOSED) {
      this.init();
    }

    if (this.requestAckSupported !== true) {
      return;
    }

    let sent = false;
    try {
      sent = this.send(pending.data);
    } catch (error) {
      logClientEvent(
        "chat.request",
        "send-error",
        {
          id: this.id,
          request_id: requestID,
          error: getErrorMessage(error),
        },
        "warn",
      );
    }
    pending.timer = setTimeout(
      () => this.flushPendingRequest(requestID),
      sent ? requestAckRetryMs : 500,
    );
    this.pendingRequests.set(requestID, pending);
  }

  public hasPendingRequests(): boolean {
    return this.pendingRequests.size > 0;
  }

  protected retryPendingRequests(resumed = false): void {
    this.pendingRequests.forEach((_pending, requestID) => {
      if (resumed) {
        const pending = this.pendingRequests.get(requestID);
        if (pending) pending.pollAfterAccepted = true;
      }
      this.flushPendingRequest(requestID);
    });
  }

  protected beginRequestCapabilityNegotiation(): void {
    if (this.requestCapabilityTimer) {
      clearTimeout(this.requestCapabilityTimer);
    }
    this.send({
      type: "capabilities",
      capabilities: [requestAckCapability],
    });
    this.requestCapabilityTimer = setTimeout(() => {
      this.requestCapabilityTimer = undefined;
      if (this.requestAckSupported !== undefined) return;
      logClientEvent("chat.request", "capability-waiting", { id: this.id });
      this.beginRequestCapabilityNegotiation();
    }, requestCapabilityRetryMs);
  }

  protected handleCapabilities(message: StreamMessage): boolean {
    if (message.response_type !== "capabilities") return false;
    if (this.requestCapabilityTimer) {
      clearTimeout(this.requestCapabilityTimer);
      this.requestCapabilityTimer = undefined;
    }
    this.requestAckSupported = Boolean(
      message.capabilities?.includes(requestAckCapability),
    );
    if (this.requestAckSupported) {
      this.retryPendingRequests();
    } else {
      logClientEvent("chat.request", "capability-unsupported", {
        id: this.id,
      });
    }
    return true;
  }

  protected async handleRequestState(message: StreamMessage): Promise<void> {
    const requestID = message.request_id?.trim();
    if (!requestID) return;

    logClientEvent("chat.request", "state", {
      id: this.id,
      request_id: requestID,
      request_status: message.request_status,
      accepted: message.accepted,
      retryable: message.retryable,
      conversation: message.conversation,
    });

    const pending = this.pendingRequests.get(requestID);
    const scope = pending?.scope ?? getChatOutboxScope();

    if (message.request_status === "completed") {
      if (pending?.timer) clearTimeout(pending.timer);
      this.pendingRequests.delete(requestID);
      await acknowledgePendingChatRequest(requestID, scope);
      return;
    }

    if (message.request_status === "rejected" || message.retryable === false) {
      const pending = this.pendingRequests.get(requestID);
      if (pending?.timer) clearTimeout(pending.timer);
      this.pendingRequests.delete(requestID);
      await acknowledgePendingChatRequest(requestID, scope).catch((error) => {
        logClientEvent(
          "chat.request",
          "outbox-ack-failed",
          {
            id: this.id,
            request_id: requestID,
            error: getErrorMessage(error),
          },
          "warn",
        );
      });
      return;
    }

    if (message.accepted && message.conversation && message.conversation > 0) {
      await updatePendingChatRequestConversation(
        requestID,
        message.conversation,
        scope,
      );
    }

    if (!pending) return;
    if (pending.timer) clearTimeout(pending.timer);
    pending.timer =
      !message.accepted || pending.pollAfterAccepted
        ? setTimeout(
            () => this.flushPendingRequest(requestID),
            requestAckRetryMs,
          )
        : undefined;
    this.pendingRequests.set(requestID, pending);
  }

  public sendEvent(
    t: TFunction | undefined,
    event: string,
    data?: string,
    props?: ChatProps,
  ) {
    this.sendWithRetry(t, {
      type: event,
      message: data || "",
      model: "event",
      ...props,
    });
  }

  public sendStopEvent(t: TFunction | undefined) {
    this.sendEvent(t, "stop");
  }

  public sendRestartEvent(t: TFunction | undefined, data?: ChatProps) {
    this.sendEvent(t, "restart", undefined, data);
  }

  public sendToolResultEvent(
    t: TFunction | undefined,
    toolCallId: string,
    toolResult: unknown,
    data: ChatProps,
  ) {
    this.sendWithRetry(t, {
      ...data,
      type: "tool_result",
      message: "",
      tool_call_id: toolCallId,
      tool_result: toolResult,
    });
  }

  public sendMaskEvent(t: TFunction | undefined, mask: Mask) {
    this.sendEvent(t, "mask", JSON.stringify(mask.context));
  }

  public sendEditEvent(t: TFunction | undefined, id: number, message: string) {
    this.sendEvent(t, "edit", `${id}:${message}`);
  }

  public sendRemoveEvent(t: TFunction | undefined, id: number) {
    this.sendEvent(t, "remove", id.toString());
  }

  public sendShareEvent(t: TFunction | undefined, refer: string) {
    this.sendEvent(t, "share", refer);
  }

  public close(): void {
    this.disposed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = undefined;
    }
    if (this.requestCapabilityTimer) {
      clearTimeout(this.requestCapabilityTimer);
      this.requestCapabilityTimer = undefined;
    }
    this.pendingRequests.forEach((pending) => {
      if (pending.timer) clearTimeout(pending.timer);
    });
    this.pendingRequests.clear();
    if (!this.connection) return;
    logClientEvent("chat.websocket", "close-requested", {
      id: this.id,
      ready_state: this.connection.readyState,
    });
    this.closeCurrentSocket();
  }

  public setCallback(callback?: StreamCallback): void {
    this.callback = callback;
  }

  protected triggerCallback(message: StreamMessage): void {
    this.streamStats.chunks += 1;
    this.streamStats.totalMessageLength +=
      typeof message.message === "string" ? message.message.length : 0;

    const shouldLog =
      this.streamStats.chunks === 1 ||
      this.streamStats.chunks % 25 === 0 ||
      message.end ||
      Boolean(message.conversation) ||
      Boolean(message.title) ||
      Boolean(message.tool_call) ||
      Boolean(message.search_query) ||
      Boolean(message.search_result) ||
      Boolean(message.search_index);

    if (shouldLog) {
      logClientEvent("chat.stream", message.end ? "end" : "progress", {
        id: this.id,
        chunks: this.streamStats.chunks,
        total_message_length: this.streamStats.totalMessageLength,
        elapsed_ms: Math.round(performance.now() - this.streamStats.startedAt),
        ...summarizeStreamMessage(message),
      });
    }

    const batchable =
      !message.end &&
      !message.request_id &&
      !message.conversation &&
      !message.title &&
      !message.keyword &&
      !message.tool_call &&
      !message.search_query &&
      !message.search_result &&
      !message.search_index &&
      !message.response_type;

    if (batchable) {
      const pending = this.pendingStreamMessage;
      this.pendingStreamMessage = pending
        ? {
            ...pending,
            ...message,
            message: `${pending.message ?? ""}${message.message ?? ""}`,
            quota: message.quota ?? pending.quota,
            plan: message.plan ?? pending.plan,
          }
        : message;
      if (this.streamFrame === undefined) {
        this.streamFrame = window.requestAnimationFrame(() => {
          this.streamFrame = undefined;
          this.flushPendingStreamMessage();
        });
      }
    } else {
      this.flushPendingStreamMessage();
      this.callback && this.callback(this.id, message);
    }

    if (message.end) {
      this.streamStats = {
        chunks: 0,
        totalMessageLength: 0,
        startedAt: performance.now(),
      };
    }
  }

  protected flushPendingStreamMessage(): void {
    if (!this.pendingStreamMessage) return;
    const pending = this.pendingStreamMessage;
    this.pendingStreamMessage = undefined;
    this.callback && this.callback(this.id, pending);
  }

  public setId(id: number): void {
    this.id = id;
  }

  public isReady(): boolean {
    return this.state;
  }

  public isRunning(): boolean {
    if (!this.connection || !this.state) return false;

    return this.connection.readyState === WebSocket.OPEN;
  }
}

export class ConnectionStack {
  protected connections: Connection[];
  protected callback?: StreamCallback;
  protected authScope: string;

  public constructor(callback?: StreamCallback) {
    this.connections = [];
    this.callback = callback;
    this.authScope = getChatOutboxScope();
    connectionStacks.add(this);
    attachConnectionLifecycleListeners();
  }

  public getConnection(id: number): Connection | undefined {
    return this.connections.find((conn) => conn.id === id);
  }

  public createConnection(id: number): Connection {
    const current = this.getConnection(id);
    if (current) return current;

    const conn = new Connection(id, this.triggerCallback.bind(this));
    this.connections.push(conn);
    logClientEvent("chat.connection-stack", "create", {
      id,
      size: this.connections.length,
    });

    // max connection garbage collection
    if (this.connections.length > maxConnection) {
      const garbageIndex = this.connections.findIndex(
        (item) => item !== conn && !item.hasPendingRequests(),
      );
      const garbage =
        garbageIndex >= 0
          ? this.connections.splice(garbageIndex, 1)[0]
          : undefined;
      if (garbage) {
        logClientEvent("chat.connection-stack", "garbage-collect", {
          id: garbage.id,
          size: this.connections.length,
        });
        garbage.close();
      }
    }
    return conn;
  }

  public bindAuthScope(scope = getChatOutboxScope()): void {
    if (scope === this.authScope) return;
    this.closeAll();
    this.authScope = scope;
  }

  public send(
    id: number,
    t: TFunction | undefined,
    props: ChatProps,
    options?: PendingRequestOptions,
  ) {
    const scope = options?.scope ?? getChatOutboxScope();
    this.bindAuthScope(scope);
    const conn = this.getConnection(id) ?? this.createConnection(id);

    conn.sendWithRetry(t, props, undefined, { ...options, scope });
    return true;
  }

  public hasConnection(id: number): boolean {
    return this.connections.some((conn) => conn.id === id);
  }

  public setCallback(callback?: StreamCallback): void {
    this.callback = callback;
  }

  public sendEvent(
    id: number,
    t: TFunction | undefined,
    event: string,
    data?: string,
  ) {
    const conn = this.getConnection(id);
    conn && conn.sendEvent(t, event, data);
  }

  public sendStopEvent(id: number, t: TFunction | undefined) {
    const conn = this.getConnection(id);
    conn && conn.sendStopEvent(t);
  }

  public sendRestartEvent(
    id: number,
    t: TFunction | undefined,
    data?: ChatProps,
  ) {
    const conn = this.getConnection(id);
    conn && conn.sendRestartEvent(t, data);
  }

  public sendToolResultEvent(
    id: number,
    t: TFunction | undefined,
    toolCallId: string,
    toolResult: unknown,
    data: ChatProps,
  ) {
    const conn = this.getConnection(id);
    conn && conn.sendToolResultEvent(t, toolCallId, toolResult, data);
  }

  public sendMaskEvent(id: number, t: TFunction | undefined, mask: Mask) {
    const conn = this.getConnection(id);
    conn && conn.sendMaskEvent(t, mask);
  }

  public sendEditEvent(
    id: number,
    t: TFunction | undefined,
    messageId: number,
    message: string,
  ) {
    const conn = this.getConnection(id);
    conn && conn.sendEditEvent(t, messageId, message);
  }

  public sendRemoveEvent(
    id: number,
    t: TFunction | undefined,
    messageId: number,
  ) {
    const conn = this.getConnection(id);
    conn && conn.sendRemoveEvent(t, messageId);
  }

  public sendShareEvent(id: number, t: TFunction | undefined, refer: string) {
    const conn = this.getConnection(id);
    conn && conn.sendShareEvent(t, refer);
  }

  public close(id: number): void {
    const conn = this.getConnection(id);
    conn && conn.close();
    this.connections = this.connections.filter((item) => item.id !== id);
    logClientEvent("chat.connection-stack", "close", {
      id,
      size: this.connections.length,
    });
  }

  public closeAll(): void {
    this.connections.forEach((conn) => conn.close());
    this.connections = [];
    logClientEvent("chat.connection-stack", "close-all");
  }

  public reconnect(id: number): void {
    const conn = this.getConnection(id);
    conn && conn.reconnect();
  }

  public reconnectAll(): void {
    this.connections.forEach((conn) => conn.reconnect());
  }

  public reconnectStale(reason = "stale"): void {
    this.connections.forEach((conn) => conn.reconnectStale(reason));
  }

  public raiseConnection(id: number): void {
    const conn = this.getConnection(-1);
    if (!conn) return;

    const existing = this.getConnection(id);
    if (existing && existing !== conn) {
      existing.close();
      this.connections = this.connections.filter((item) => item !== existing);
    }

    conn.setId(id);
    logClientEvent("chat.connection-stack", "raise", {
      from: -1,
      to: id,
      replaced_existing: Boolean(existing && existing !== conn),
    });
  }

  public triggerCallback(id: number, message: StreamMessage): void {
    this.callback && this.callback(id, message);
  }
}
