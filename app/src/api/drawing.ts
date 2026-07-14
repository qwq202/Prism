import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

export type DrawingTaskStatus =
  | "queued"
  | "running"
  | "canceling"
  | "succeeded"
  | "failed"
  | "canceled";

export type DrawingTaskOptions = {
  response_format?: {
    aspect_ratio?: string;
    image_size?: string;
    mime_type?: string;
  };
  thinking?: {
    thinking_level?: string;
  };
};

export type DrawingTask<TImage = unknown> = {
  task_id: string;
  workspace_id: string;
  status: DrawingTaskStatus;
  model: string;
  prompt: string;
  options?: DrawingTaskOptions;
  images?: TImage[];
  error?: string;
  quota?: number;
  created_at?: string;
  updated_at?: string;
  started_at?: string;
  completed_at?: string;
};

type DrawingTaskResponse<TImage = unknown> = {
  status: boolean;
  data?: DrawingTask<TImage>;
  message?: string;
  error?: string;
  uncertain?: boolean;
};

type DrawingTaskListResponse<TImage = unknown> = {
  status: boolean;
  data?: DrawingTask<TImage>[];
  message?: string;
  error?: string;
};

export type CreateDrawingTaskPayload = {
  request_id: string;
  workspace_id: string;
  model: string;
  prompt: string;
  message: string;
  response_format?: unknown;
  thinking?: unknown;
};

export async function listDrawingTasks<TImage>(): Promise<
  DrawingTaskListResponse<TImage>
> {
  try {
    const response = await axios.get("/drawing/tasks");
    return response.data as DrawingTaskListResponse<TImage>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
    };
  }
}

export async function createDrawingTask<TImage>(
  payload: CreateDrawingTaskPayload,
): Promise<DrawingTaskResponse<TImage>> {
  try {
    const response = await axios.post("/drawing/tasks", payload);
    return response.data as DrawingTaskResponse<TImage>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
      uncertain: true,
    };
  }
}

export async function getDrawingTask<TImage>(
  taskId: string,
  signal?: AbortSignal,
): Promise<DrawingTaskResponse<TImage>> {
  try {
    const response = await axios.get(`/drawing/tasks/${taskId}`, {
      signal,
      timeout: 15_000,
    });
    return response.data as DrawingTaskResponse<TImage>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
    };
  }
}

export async function cancelDrawingTask<TImage>(
  taskId: string,
): Promise<DrawingTaskResponse<TImage>> {
  try {
    const response = await axios.post(`/drawing/tasks/${taskId}/cancel`);
    return response.data as DrawingTaskResponse<TImage>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
    };
  }
}

export async function acknowledgeDrawingTask(taskId: string): Promise<boolean> {
  try {
    const response = await axios.delete(`/drawing/tasks/${taskId}`);
    return Boolean(response.data?.status);
  } catch (error) {
    console.debug("[drawing] failed to acknowledge local task", getErrorMessage(error));
    return false;
  }
}
