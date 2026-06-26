import axios from "axios";
import { getErrorMessage } from "@/utils/base.ts";

export type DrawingWorkspaceSyncState<TWorkspace> = {
  active_workspace_id: string;
  workspaces: TWorkspace[];
  updated_at?: string;
};

type DrawingWorkspaceSyncResponse<TWorkspace> = {
  status: boolean;
  data?: DrawingWorkspaceSyncState<TWorkspace>;
  message?: string;
  error?: string;
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
