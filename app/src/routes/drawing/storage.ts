import type { DrawingWorkspace } from "@/routes/drawing/domain.ts";

const DRAWING_DATABASE_NAME = "prism-drawing";
const DRAWING_DATABASE_VERSION = 1;
const DRAWING_STATE_STORE = "workspace-state";
const DRAWING_STATE_KEY = "current";
const DRAWING_BOOTSTRAP_KEY = "drawing-workspace-bootstrap-v1";

export type DrawingLocalState = {
  activeWorkspaceId: string;
  workspaces: DrawingWorkspace[];
};

export function loadDrawingLocalBootstrap(): DrawingLocalState | null {
  if (typeof window === "undefined") return null;

  try {
    const raw = window.localStorage.getItem(DRAWING_BOOTSTRAP_KEY);
    if (!raw) return null;
    const value = JSON.parse(raw) as Partial<DrawingLocalState>;
    if (!Array.isArray(value.workspaces)) return null;
    return {
      activeWorkspaceId:
        typeof value.activeWorkspaceId === "string"
          ? value.activeWorkspaceId
          : "",
      workspaces: value.workspaces,
    };
  } catch {
    return null;
  }
}

export function saveDrawingLocalBootstrap(state: DrawingLocalState): void {
  if (typeof window === "undefined") return;

  const workspaces = state.workspaces.map((workspace) => ({
    ...workspace,
    references: [],
    images: [],
  }));
  window.localStorage.setItem(
    DRAWING_BOOTSTRAP_KEY,
    JSON.stringify({ ...state, workspaces }),
  );
}

function openDrawingDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(
      DRAWING_DATABASE_NAME,
      DRAWING_DATABASE_VERSION,
    );

    request.onupgradeneeded = () => {
      const database = request.result;
      if (!database.objectStoreNames.contains(DRAWING_STATE_STORE)) {
        database.createObjectStore(DRAWING_STATE_STORE);
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

export async function loadDrawingLocalState(): Promise<DrawingLocalState | null> {
  if (typeof indexedDB === "undefined") return null;

  const database = await openDrawingDatabase();
  try {
    return await new Promise((resolve, reject) => {
      const transaction = database.transaction(
        DRAWING_STATE_STORE,
        "readonly",
      );
      const request = transaction
        .objectStore(DRAWING_STATE_STORE)
        .get(DRAWING_STATE_KEY);
      request.onsuccess = () =>
        resolve((request.result as DrawingLocalState | undefined) ?? null);
      request.onerror = () => reject(request.error);
    });
  } finally {
    database.close();
  }
}

export async function saveDrawingLocalState(
  state: DrawingLocalState,
): Promise<void> {
  if (typeof indexedDB === "undefined") return;

  const database = await openDrawingDatabase();
  try {
    await new Promise<void>((resolve, reject) => {
      const transaction = database.transaction(
        DRAWING_STATE_STORE,
        "readwrite",
      );
      transaction.objectStore(DRAWING_STATE_STORE).put(state, DRAWING_STATE_KEY);
      transaction.oncomplete = () => resolve();
      transaction.onerror = () => reject(transaction.error);
      transaction.onabort = () => reject(transaction.error);
    });
  } finally {
    database.close();
  }
}
