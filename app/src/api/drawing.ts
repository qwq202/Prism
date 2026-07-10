import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

export type DrawingWorkspaceSyncState<TWorkspace> = {
  active_workspace_id: string;
  workspaces: TWorkspace[];
  updated_at?: string;
};

export type DrawingTaskStatus =
  | "queued"
  | "running"
  | "canceling"
  | "succeeded"
  | "failed"
  | "canceled";

export type DrawingTask<TImage = unknown> = {
  task_id: string;
  workspace_id: string;
  status: DrawingTaskStatus;
  model: string;
  prompt: string;
  images?: TImage[];
  error?: string;
  quota?: number;
  created_at?: string;
  updated_at?: string;
  started_at?: string;
  completed_at?: string;
};

type DrawingWorkspaceSyncResponse<TWorkspace> = {
  status: boolean;
  data?: DrawingWorkspaceSyncState<TWorkspace>;
  message?: string;
  error?: string;
};

type DrawingTaskResponse<TImage = unknown> = {
  status: boolean;
  data?: DrawingTask<TImage>;
  message?: string;
  error?: string;
};

type DrawingTaskListResponse<TImage = unknown> = {
  status: boolean;
  data?: DrawingTask<TImage>[];
  message?: string;
  error?: string;
};

export type CreateDrawingTaskPayload = {
  workspace_id: string;
  model: string;
  prompt: string;
  message: string;
  response_format?: unknown;
  thinking?: unknown;
};

export async function loadDrawingWorkspaceState<TWorkspace>(): Promise<
  DrawingWorkspaceSyncResponse<TWorkspace>
> {
  try {
    const response = await axios.get("/drawing/workspaces");
    return response.data as DrawingWorkspaceSyncResponse<TWorkspace>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
    };
  }
}

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
    const response = await axios.post("/drawing/tasks", payload, {
      timeout: 30_000,
    });
    return response.data as DrawingTaskResponse<TImage>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
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

export async function saveDrawingWorkspaceState<TWorkspace>(
  state: DrawingWorkspaceSyncState<TWorkspace>,
): Promise<DrawingWorkspaceSyncResponse<TWorkspace>> {
  try {
    const response = await axios.post("/drawing/workspaces", state);
    return response.data as DrawingWorkspaceSyncResponse<TWorkspace>;
  } catch (error) {
    return {
      status: false,
      error: getErrorMessage(error),
    };
  }
}
