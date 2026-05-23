import { tokenField, websocketEndpoint } from "@/conf/bootstrap.ts";
import { getMemory } from "@/utils/memory.ts";
import { getErrorMessage } from "@/utils/base.ts";
import { Mask } from "@/masks/types.ts";
import type { TFunction } from "i18next";

export const endpoint = `${websocketEndpoint}/chat`;
export const maxRetry = 60; // 30s max websocket retry
export const maxConnection = 5;

export type StreamMessage = {
  conversation?: number;
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
    status: "start" | "executing" | "success" | "error";
  };
  response_type?: string;
};

export type ChatProps = {
  type?: string;
  message: string;
  model: string;
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
  web_search_mode?: "quick" | "detailed";
  web_page_summary?: boolean;
  think?: boolean;
  context?: number;
  ignore_context?: boolean;
  custom_instruction?: string;
  memory_enabled?: boolean;
  memory_history_enabled?: boolean;

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

export class Connection {
  protected connection?: WebSocket;
  protected callback?: StreamCallback;
  protected reconnectTimer?: ReturnType<typeof setTimeout>;
  protected stack?: Record<string, unknown>;
  protected disposed: boolean;
  public id: number;
  public state: boolean;

  public constructor(id: number, callback?: StreamCallback) {
    this.state = false;
    this.disposed = false;
    this.id = id;

    callback && this.setCallback(callback);
  }

  public init(): void {
    if (this.disposed) return;
    this.connection = new WebSocket(endpoint);
    this.state = false;
    this.connection.onopen = () => {
      this.state = true;
      this.send({
        token: getMemory(tokenField) || "anonymous",
        id: this.id,
      });
    };
    this.connection.onclose = (event) => {
      this.state = false;
      if (this.disposed) return;

      this.stack = {
        error: "websocket connection failed",
        code: event.code,
        reason: event.reason,
        endpoint: endpoint,
      };

      if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
      this.reconnectTimer = setTimeout(() => {
        if (this.disposed) return;
        console.debug(`[connection] reconnecting... (id: ${this.id})`);
        this.init();
      }, 3000);
    };
    this.connection.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        this.triggerCallback(message as StreamMessage);
      } catch (e) {
        console.warn(
          `[connection] failed to parse websocket message: ${getErrorMessage(e)}`,
        );
      }
    };
  }

  public reconnect(): void {
    this.disposed = false;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.init();
  }

  public send(data: Record<string, string | boolean | number | undefined>): boolean {
    if (!this.state || !this.connection) {
      if (this.connection === undefined) this.init();
      console.debug("[connection] connection not ready, retrying in 500ms...");
      return false;
    }
    this.connection.send(JSON.stringify(data));
    return true;
  }

  public sendWithRetry(t: TFunction | undefined, data: ChatProps, times?: number): void {
    try {
      if (!times || times < maxRetry) {
        if (!this.send(data)) {
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
    }

    const trace = JSON.stringify(
      this.stack ?? {
        message: data.message,
        endpoint: endpoint,
      },
      null,
      2,
    );
    this.stack = undefined;

    t &&
      this.triggerCallback({
        message: `${t("request-failed")}\n\`\`\`json\n${trace}\n\`\`\`\n`,
        end: true,
      });
  }

  public sendEvent(t: TFunction | undefined, event: string, data?: string, props?: ChatProps) {
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
    if (!this.connection) return;
    this.connection.close();
  }

  public setCallback(callback?: StreamCallback): void {
    this.callback = callback;
  }

  protected triggerCallback(message: StreamMessage): void {
    this.callback && this.callback(this.id, message);
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

  public constructor(callback?: StreamCallback) {
    this.connections = [];
    this.callback = callback;
  }

  public getConnection(id: number): Connection | undefined {
    return this.connections.find((conn) => conn.id === id);
  }

  public createConnection(id: number): Connection {
    const current = this.getConnection(id);
    if (current) return current;

    const conn = new Connection(id, this.triggerCallback.bind(this));
    this.connections.push(conn);

    // max connection garbage collection
    if (this.connections.length > maxConnection) {
      const garbage = this.connections.shift();
      garbage && garbage.close();
    }
    return conn;
  }

  public send(id: number, t: TFunction | undefined, props: ChatProps) {
    const conn = this.getConnection(id);
    if (!conn) return false;

    conn.sendWithRetry(t, props);
    return true;
  }

  public hasConnection(id: number): boolean {
    return this.connections.some((conn) => conn.id === id);
  }

  public setCallback(callback?: StreamCallback): void {
    this.callback = callback;
  }

  public sendEvent(id: number, t: TFunction | undefined, event: string, data?: string) {
    const conn = this.getConnection(id);
    conn && conn.sendEvent(t, event, data);
  }

  public sendStopEvent(id: number, t: TFunction | undefined) {
    const conn = this.getConnection(id);
    conn && conn.sendStopEvent(t);
  }

  public sendRestartEvent(id: number, t: TFunction | undefined, data?: ChatProps) {
    const conn = this.getConnection(id);
    conn && conn.sendRestartEvent(t, data);
  }

  public sendMaskEvent(id: number, t: TFunction | undefined, mask: Mask) {
    const conn = this.getConnection(id);
    conn && conn.sendMaskEvent(t, mask);
  }

  public sendEditEvent(id: number, t: TFunction | undefined, messageId: number, message: string) {
    const conn = this.getConnection(id);
    conn && conn.sendEditEvent(t, messageId, message);
  }

  public sendRemoveEvent(id: number, t: TFunction | undefined, messageId: number) {
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
  }

  public closeAll(): void {
    this.connections.forEach((conn) => conn.close());
    this.connections = [];
  }

  public reconnect(id: number): void {
    const conn = this.getConnection(id);
    conn && conn.reconnect();
  }

  public reconnectAll(): void {
    this.connections.forEach((conn) => conn.reconnect());
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
  }

  public triggerCallback(id: number, message: StreamMessage): void {
    this.callback && this.callback(id, message);
  }
}
