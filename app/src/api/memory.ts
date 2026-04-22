import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

export type MemoryRecord = {
  id: number;
  content: string;
  source?: string;
  category?: string;
  created_at: string;
  updated_at: string;
  last_used_at?: string;
  pinned?: boolean;
};

type MemoryListResponse = {
  status: boolean;
  data?: MemoryRecord[];
  message?: string;
};

type MemoryMutationResponse = {
  status: boolean;
  data?: MemoryRecord;
  message?: string;
};

export async function listMemories(query?: string): Promise<MemoryRecord[]> {
  try {
    const suffix = query?.trim()
      ? `?q=${encodeURIComponent(query.trim())}`
      : "";
    const resp = await axios.get(`/memory/list${suffix}`);
    const data = resp.data as MemoryListResponse;
    return data.status ? data.data || [] : [];
  } catch (e) {
    console.warn(e);
    return [];
  }
}

export async function createMemory(
  content: string,
  category?: string,
): Promise<MemoryMutationResponse> {
  try {
    const resp = await axios.post("/memory/create", { content, category });
    return resp.data as MemoryMutationResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function updateMemory(
  id: number,
  content: string,
  category?: string,
): Promise<MemoryMutationResponse> {
  try {
    const resp = await axios.post("/memory/update", { id, content, category });
    return resp.data as MemoryMutationResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}

export async function deleteMemory(id: number): Promise<MemoryMutationResponse> {
  try {
    const resp = await axios.get(`/memory/delete?id=${id}`);
    return resp.data as MemoryMutationResponse;
  } catch (e) {
    return { status: false, message: getErrorMessage(e) };
  }
}
