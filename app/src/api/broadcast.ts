import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";
import { getMemory, setMemory } from "@/utils/memory.ts";

export type Broadcast = {
  content: string;
  index: number;
};

export type BroadcastInfo = Broadcast & {
  poster: string;
  type: "broadcast" | "popup" | "banner";
  start_at: string;
  end_at: string;
  is_active: boolean;
  created_at: string;
};

export type BroadcastListResponse = {
  data: BroadcastInfo[];
};

export type CommonBroadcastResponse = {
  status: boolean;
  error: string;
};

export async function getRawBroadcast(): Promise<Broadcast> {
  try {
    const data = await axios.get("/broadcast/view");
    if (data.data) return data.data as Broadcast;
  } catch (e) {
    console.warn(e);
  }

  return {
    content: "",
    index: 0,
  };
}

export type BroadcastEvent = {
  message: string;
  firstReceived: boolean;
};

export async function getBroadcast(): Promise<BroadcastEvent> {
  const data = await getRawBroadcast();
  const content = (data?.content ?? "").trim();

  if (content.length === 0)
    return {
      message: "",
      firstReceived: false,
    };

  const memory = getMemory("broadcast");
  if (memory === content)
    return {
      message: content,
      firstReceived: false,
    };

  setMemory("broadcast", content);
  return {
    message: content,
    firstReceived: true,
  };
}

export async function getBroadcastList(): Promise<BroadcastInfo[]> {
  try {
    const resp = await axios.get("/broadcast/list");
    const data = resp.data as BroadcastListResponse;
    return data.data || [];
  } catch (e) {
    console.warn(e);
    return [];
  }
}

export type CreateBroadcastParams = {
  content: string;
  notify_all?: boolean;
  type?: "broadcast" | "popup" | "banner";
  start_at?: string;
  end_at?: string;
  is_active?: boolean;
};

export async function createBroadcast(
  content: string,
  notify_all?: boolean,
  extra?: Omit<CreateBroadcastParams, "content" | "notify_all">,
): Promise<CommonBroadcastResponse> {
  try {
    const resp = await axios.post("/broadcast/create", {
      content,
      notify_all,
      ...extra,
    });
    return resp.data as CommonBroadcastResponse;
  } catch (e) {
    console.warn(e);
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export async function removeBroadcast(
  index: number,
): Promise<CommonBroadcastResponse> {
  try {
    const resp = await axios.post(`/broadcast/remove/${index}`);
    return resp.data as CommonBroadcastResponse;
  } catch (e) {
    console.warn(e);
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}

export type UpdateBroadcastParams = {
  id: number;
  content: string;
  type?: "broadcast" | "popup" | "banner";
  start_at?: string;
  end_at?: string;
  is_active?: boolean;
};

export async function updateBroadcast(
  id: number,
  content: string,
  extra?: Omit<UpdateBroadcastParams, "id" | "content">,
): Promise<CommonBroadcastResponse> {
  try {
    const resp = await axios.post("/broadcast/update", { id, content, ...extra });
    return resp.data as CommonBroadcastResponse;
  } catch (e) {
    console.warn(e);
    return {
      status: false,
      error: getErrorMessage(e),
    };
  }
}
