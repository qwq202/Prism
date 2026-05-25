import axios from "axios";
import { CommonResponse } from "@/api/common.ts";
import { getErrorMessage } from "@/utils/base.ts";

export type Attachment = {
  name: string;
  size: number;
  updated_at: string;
  storage_mode: string;
  public_url: string;
  referenced: boolean;
  reference_count: number;
};

export type AttachmentListResponse = CommonResponse & {
  data: Attachment[];
};

export async function listAttachments(): Promise<AttachmentListResponse> {
  try {
    const response = await axios.get("/admin/attachment/list");
    if (Array.isArray(response.data)) {
      return { status: true, data: response.data as Attachment[] };
    }

    const data = response.data as CommonResponse;
    return {
      status: data.status,
      error: data.error,
      reason: data.reason,
      message: data.message,
      data: [],
    };
  } catch (e) {
    console.warn(e);
    return { status: false, error: getErrorMessage(e), data: [] };
  }
}

export async function deleteAttachment(
  name: string,
  force = false,
): Promise<CommonResponse> {
  try {
    const response = await axios.post("/admin/attachment/delete", null, {
      params: { name, force },
    });
    return response.data as CommonResponse;
  } catch (e) {
    console.warn(e);
    return { status: false, error: getErrorMessage(e) };
  }
}
