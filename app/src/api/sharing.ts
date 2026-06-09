import axios from "axios";
import { Message } from "./types.tsx";
import { getErrorMessage } from "@/utils/base.ts";

const noCacheHeaders = {
  "Cache-Control": "no-cache",
  Pragma: "no-cache",
};

export type SharingForm = {
  status: boolean;
  message: string;
  data: string;
};

export type SharingPreviewForm = {
  name: string;
  conversation_id: number;
  hash: string;
  time: string;
};

export type ViewData = {
  name: string;
  username: string;
  time: string;
  model?: string;
  messages: Message[];
};

export type ViewForm = {
  status: boolean;
  message: string;
  data: ViewData | null;
};

export type ListSharingResponse = {
  status: boolean;
  message: string;
  data?: SharingPreviewForm[];
};

export type DeleteSharingResponse = {
  status: boolean;
  message: string;
};

export async function shareConversation(
  id: number,
  refs: number[] = [-1],
): Promise<SharingForm> {
  try {
    const resp = await axios.post("/conversation/share", { id, refs });
    return resp.data;
  } catch (e) {
    return { status: false, message: getErrorMessage(e), data: "" };
  }
}

export async function viewConversation(hash: string): Promise<ViewForm> {
  try {
    const resp = await axios.get(`/conversation/view?hash=${hash}`);
    return resp.data as ViewForm;
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
      data: null,
    };
  }
}

export async function listSharing(): Promise<ListSharingResponse> {
  try {
    const resp = await axios.get("/conversation/share/list", {
      headers: noCacheHeaders,
      params: { _: Date.now() },
    });
    return resp.data as ListSharingResponse;
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
    };
  }
}

export async function deleteSharing(
  hash: string,
): Promise<DeleteSharingResponse> {
  try {
    const resp = await axios.post("/conversation/share/delete", { hash });
    return resp.data as DeleteSharingResponse;
  } catch (e) {
    return {
      status: false,
      message: getErrorMessage(e),
    };
  }
}

export function getSharedLink(hash: string): string {
  return `${location.origin}/share/${hash}`;
}
