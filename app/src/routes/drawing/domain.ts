import type { FileArray } from "@/api/file.ts";
import type { DrawingTask, DrawingTaskOptions } from "@/api/drawing.ts";
import type { Model } from "@/api/types.ts";
import { normalizeImageURL } from "@/utils/image-url.ts";

export type Mode = "generate" | "edit";

export type GeminiImageAspectRatio =
  | "1:1"
  | "1:4"
  | "1:8"
  | "2:3"
  | "3:2"
  | "3:4"
  | "4:1"
  | "4:3"
  | "4:5"
  | "5:4"
  | "8:1"
  | "9:16"
  | "16:9"
  | "21:9";

export type GeminiImageSize = "512px" | "1K" | "2K" | "4K";
export type GeminiImageMimeType = "image/png" | "image/jpeg";
export type GeminiImageThinkingLevel = "minimal" | "high";

export type DrawingOptions = {
  aspectRatio: GeminiImageAspectRatio;
  imageSize: GeminiImageSize;
  mimeType: GeminiImageMimeType;
  thinkingLevel: GeminiImageThinkingLevel;
};

export type DrawingModelCapabilities = {
  aspectRatios: readonly GeminiImageAspectRatio[];
  imageSizes: readonly GeminiImageSize[];
  mimeTypes: readonly GeminiImageMimeType[];
  thinkingLevels: readonly GeminiImageThinkingLevel[];
  maxReferenceImages: number;
  supportsEditing: boolean;
};

export type DrawingModel = Pick<
  Model,
  "id" | "name" | "channel_type" | "drawing_model"
>;

export type DrawingGeneratedImage = {
  id: string;
  src: string;
  prompt: string;
  createdAt: number;
  model?: string;
  options?: DrawingTaskOptions;
};

export type DrawingWorkspace = {
  id: string;
  model: string;
  mode: Mode;
  prompt: string;
  options: DrawingOptions;
  references: FileArray;
  images: DrawingGeneratedImage[];
  pending: boolean;
  taskId?: string;
  taskStatus?: DrawingTask["status"];
  taskError?: string;
  lastPrompt: string;
  createdAt: number;
  accent: number;
};

type DrawingFileAction =
  | { type: "add"; payload: FileArray[number] }
  | { type: "remove"; payload: number }
  | { type: "clear" };

export const DRAWING_TASK_POLL_INTERVAL_MS = 2500;
export const MAX_DRAWING_WORKSPACES = 64;

const DRAWING_WORKSPACES_KEY = "drawing.workspaces.v1";
const DRAWING_ACTIVE_WORKSPACE_KEY = "drawing.activeWorkspaceId.v1";
const DRAWING_MODEL_QUERY_PARAM = "model";
const DRAWING_WORKSPACE_ACCENT_COUNT = 4;

const DEFAULT_DRAWING_OPTIONS: DrawingOptions = {
  aspectRatio: "1:1",
  imageSize: "1K",
  mimeType: "image/png",
  thinkingLevel: "minimal",
};

const GEMINI_25_FLASH_IMAGE_ASPECT_RATIOS: readonly GeminiImageAspectRatio[] = [
  "1:1",
  "2:3",
  "3:2",
  "3:4",
  "4:3",
  "4:5",
  "5:4",
  "9:16",
  "16:9",
  "21:9",
];
const GEMINI_3_PRO_IMAGE_ASPECT_RATIOS = GEMINI_25_FLASH_IMAGE_ASPECT_RATIOS;
const GEMINI_31_FLASH_IMAGE_ASPECT_RATIOS: readonly GeminiImageAspectRatio[] = [
  ...GEMINI_3_PRO_IMAGE_ASPECT_RATIOS,
  "1:4",
  "4:1",
  "1:8",
  "8:1",
];
const GEMINI_31_FLASH_IMAGE_SIZES: readonly GeminiImageSize[] = [
  "512px",
  "1K",
  "2K",
  "4K",
];
const GEMINI_3_PRO_IMAGE_SIZES: readonly GeminiImageSize[] = ["1K", "2K", "4K"];
const MIME_TYPE_OPTIONS: readonly GeminiImageMimeType[] = [
  "image/png",
  "image/jpeg",
];
const GEMINI_31_FLASH_THINKING_LEVELS: readonly GeminiImageThinkingLevel[] = [
  "minimal",
  "high",
];
const GEMINI_25_FLASH_IMAGE_REFERENCE_LIMIT = 3;
const GEMINI_31_FLASH_IMAGE_REFERENCE_LIMIT = 10;
const GEMINI_3_PRO_IMAGE_REFERENCE_LIMIT = 14;
const DEFAULT_DRAWING_MODEL_CAPABILITIES: DrawingModelCapabilities = {
  aspectRatios: GEMINI_25_FLASH_IMAGE_ASPECT_RATIOS,
  imageSizes: [],
  mimeTypes: MIME_TYPE_OPTIONS,
  thinkingLevels: [],
  maxReferenceImages: GEMINI_25_FLASH_IMAGE_REFERENCE_LIMIT,
  supportsEditing: true,
};
const BASIC_DRAWING_MODEL_CAPABILITIES: DrawingModelCapabilities = {
  aspectRatios: [],
  imageSizes: [],
  mimeTypes: [],
  thinkingLevels: [],
  maxReferenceImages: 0,
  supportsEditing: false,
};

function createWorkspaceId() {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `workspace-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function createDrawingWorkspace(
  index = 0,
  model = "",
): DrawingWorkspace {
  return {
    id: createWorkspaceId(),
    model,
    mode: "generate",
    prompt: "",
    options: { ...DEFAULT_DRAWING_OPTIONS },
    references: [],
    images: [],
    pending: false,
    taskId: undefined,
    taskStatus: undefined,
    taskError: undefined,
    lastPrompt: "",
    createdAt: Date.now(),
    accent: index % DRAWING_WORKSPACE_ACCENT_COUNT,
  };
}

export function drawingFileReducer(
  state: FileArray,
  action: DrawingFileAction,
): FileArray {
  switch (action.type) {
    case "add":
      return [...state, action.payload];
    case "remove":
      return state.filter((_, index) => index !== action.payload);
    case "clear":
      return [];
    default:
      return state;
  }
}

export function hasDraggedFiles(dataTransfer: DataTransfer | null): boolean {
  return Boolean(
    dataTransfer && Array.from(dataTransfer.types).includes("Files"),
  );
}

export function getDroppedFiles(dataTransfer: DataTransfer | null): File[] {
  if (!dataTransfer) return [];

  const filesFromItems = Array.from(dataTransfer.items ?? [])
    .filter((item) => item.kind === "file")
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file));

  return filesFromItems.length
    ? filesFromItems
    : Array.from(dataTransfer.files ?? []);
}

function hasStringOption<T extends string>(
  options: readonly T[],
  value: unknown,
): value is T {
  return typeof value === "string" && options.includes(value as T);
}

export function getDrawingModelCapabilities(
  model: DrawingModel | string,
): DrawingModelCapabilities {
  const modelId = typeof model === "string" ? model : model.id;
  const normalizedModelId = modelId.trim().toLowerCase();
  const channelType =
    typeof model === "string"
      ? ""
      : (model.channel_type ?? "").trim().toLowerCase();
  const usesGeminiImageAPI =
    channelType === "" ||
    channelType === "palm" ||
    channelType === "gemini-enterprise-agent-platform";

  if (
    usesGeminiImageAPI &&
    normalizedModelId.includes("gemini-3.1-flash-image")
  ) {
    return {
      aspectRatios: GEMINI_31_FLASH_IMAGE_ASPECT_RATIOS,
      imageSizes: GEMINI_31_FLASH_IMAGE_SIZES,
      mimeTypes: MIME_TYPE_OPTIONS,
      thinkingLevels: GEMINI_31_FLASH_THINKING_LEVELS,
      maxReferenceImages: GEMINI_31_FLASH_IMAGE_REFERENCE_LIMIT,
      supportsEditing: true,
    };
  }

  if (usesGeminiImageAPI && normalizedModelId.includes("gemini-3-pro-image")) {
    return {
      aspectRatios: GEMINI_3_PRO_IMAGE_ASPECT_RATIOS,
      imageSizes: GEMINI_3_PRO_IMAGE_SIZES,
      mimeTypes: MIME_TYPE_OPTIONS,
      thinkingLevels: [],
      maxReferenceImages: GEMINI_3_PRO_IMAGE_REFERENCE_LIMIT,
      supportsEditing: true,
    };
  }

  if (
    usesGeminiImageAPI &&
    normalizedModelId.includes("gemini-2.5-flash-image")
  ) {
    return DEFAULT_DRAWING_MODEL_CAPABILITIES;
  }

  return BASIC_DRAWING_MODEL_CAPABILITIES;
}

export function normalizeDrawingOptions(
  options?: Partial<DrawingOptions>,
  capabilities: DrawingModelCapabilities = DEFAULT_DRAWING_MODEL_CAPABILITIES,
): DrawingOptions {
  const aspectRatio = options?.aspectRatio;
  const imageSize = options?.imageSize;
  const mimeType = options?.mimeType;
  const thinkingLevel = options?.thinkingLevel;

  return {
    aspectRatio: hasStringOption(capabilities.aspectRatios, aspectRatio)
      ? aspectRatio
      : DEFAULT_DRAWING_OPTIONS.aspectRatio,
    imageSize: hasStringOption(capabilities.imageSizes, imageSize)
      ? imageSize
      : (capabilities.imageSizes[0] ?? DEFAULT_DRAWING_OPTIONS.imageSize),
    mimeType: hasStringOption(capabilities.mimeTypes, mimeType)
      ? mimeType
      : DEFAULT_DRAWING_OPTIONS.mimeType,
    thinkingLevel: hasStringOption(capabilities.thinkingLevels, thinkingLevel)
      ? thinkingLevel
      : DEFAULT_DRAWING_OPTIONS.thinkingLevel,
  };
}

export function buildDrawingRequestOptions(
  options: DrawingOptions,
  capabilities: DrawingModelCapabilities,
) {
  const responseFormat: {
    type: "image";
    mime_type?: GeminiImageMimeType;
    aspect_ratio?: GeminiImageAspectRatio;
    image_size?: GeminiImageSize;
  } = {
    type: "image",
  };

  if (capabilities.mimeTypes.length > 0) {
    responseFormat.mime_type = options.mimeType;
  }

  if (capabilities.aspectRatios.length > 0) {
    responseFormat.aspect_ratio = options.aspectRatio;
  }

  if (capabilities.imageSizes.length > 0) {
    responseFormat.image_size = options.imageSize;
  }

  const requestOptions: {
    response_format: typeof responseFormat;
    thinking?: { thinking_level: GeminiImageThinkingLevel };
  } = {
    response_format: responseFormat,
  };

  if (capabilities.thinkingLevels.length > 0) {
    requestOptions.thinking = {
      thinking_level: options.thinkingLevel,
    };
  }

  return requestOptions;
}

function normalizeDrawingImages(value: unknown): DrawingGeneratedImage[] {
  if (!Array.isArray(value)) return [];

  return value
    .map((item): DrawingGeneratedImage | null => {
      if (!item || typeof item !== "object") return null;
      const image = item as Partial<DrawingGeneratedImage>;
      if (typeof image.src !== "string" || image.src.trim() === "") {
        return null;
      }

      return {
        id:
          typeof image.id === "string" && image.id
            ? image.id
            : createWorkspaceId(),
        src: normalizeImageURL(image.src),
        prompt: typeof image.prompt === "string" ? image.prompt : "",
        createdAt:
          typeof image.createdAt === "number" &&
          Number.isFinite(image.createdAt)
            ? image.createdAt
            : Date.now(),
        model:
          typeof image.model === "string" && image.model.trim()
            ? image.model.trim()
            : undefined,
        options: normalizeDrawingImageOptions(image.options),
      };
    })
    .filter((image): image is DrawingGeneratedImage => Boolean(image));
}

function normalizeDrawingImageOptions(
  value: unknown,
): DrawingTaskOptions | undefined {
  if (!value || typeof value !== "object") return undefined;

  const raw = value as DrawingTaskOptions;
  const rawResponseFormat = raw.response_format;
  const responseFormat =
    rawResponseFormat && typeof rawResponseFormat === "object"
      ? {
          aspect_ratio:
            typeof rawResponseFormat.aspect_ratio === "string"
              ? rawResponseFormat.aspect_ratio
              : undefined,
          image_size:
            typeof rawResponseFormat.image_size === "string"
              ? rawResponseFormat.image_size
              : undefined,
          mime_type:
            typeof rawResponseFormat.mime_type === "string"
              ? rawResponseFormat.mime_type
              : undefined,
        }
      : undefined;
  const rawThinking = raw.thinking;
  const thinking =
    rawThinking && typeof rawThinking === "object"
      ? {
          thinking_level:
            typeof rawThinking.thinking_level === "string"
              ? rawThinking.thinking_level
              : undefined,
        }
      : undefined;

  if (
    !responseFormat?.aspect_ratio &&
    !responseFormat?.image_size &&
    !responseFormat?.mime_type &&
    !thinking?.thinking_level
  ) {
    return undefined;
  }

  return {
    response_format: responseFormat,
    thinking,
  };
}

export function normalizeDrawingReferences(value: unknown): FileArray {
  if (!Array.isArray(value)) return [];

  return value
    .map((item): FileArray[number] | null => {
      if (!item || typeof item !== "object") return null;
      const file = item as Partial<FileArray[number]>;
      if (typeof file.content !== "string" || file.content.trim() === "") {
        return null;
      }

      return {
        name:
          typeof file.name === "string" && file.name.trim()
            ? file.name
            : "reference.png",
        content: normalizeImageURL(file.content),
        size:
          typeof file.size === "number" && Number.isFinite(file.size)
            ? file.size
            : undefined,
      };
    })
    .filter((file): file is FileArray[number] => Boolean(file));
}

export function getDrawingImageExtension(source: string): string {
  const normalized = source.trim().toLowerCase();
  if (normalized.startsWith("data:image/jpeg")) return "jpg";
  if (normalized.startsWith("data:image/webp")) return "webp";
  if (normalized.startsWith("data:image/gif")) return "gif";
  if (normalized.startsWith("data:image/png")) return "png";

  const pathname = normalized.split(/[?#]/, 1)[0] ?? "";
  const match = pathname.match(/\.([a-z0-9]+)$/);
  if (match?.[1] === "jpeg") return "jpg";
  if (["jpg", "png", "webp", "gif"].includes(match?.[1] ?? "")) {
    return match?.[1] ?? "png";
  }
  return "png";
}

function mergeGeneratedImages(
  current: DrawingGeneratedImage[],
  incoming: DrawingGeneratedImage[],
): DrawingGeneratedImage[] {
  if (incoming.length === 0) return current;

  const indexes = new Map(
    current.map((image, index) => [image.src, index] as const),
  );
  const next = [...current];
  const additions: DrawingGeneratedImage[] = [];
  let changed = false;
  incoming.forEach((image) => {
    const existingIndex = indexes.get(image.src);
    if (existingIndex !== undefined) {
      if (existingIndex < 0) return;
      const existing = next[existingIndex];
      const model = existing.model || image.model;
      const options = existing.options || image.options;
      if (model !== existing.model || options !== existing.options) {
        next[existingIndex] = { ...existing, model, options };
        changed = true;
      }
      return;
    }
    indexes.set(image.src, -1);
    additions.push(image);
    changed = true;
  });
  return changed ? [...additions.reverse(), ...next] : current;
}

function reconcileGeneratedImagesForTask(
  current: DrawingGeneratedImage[],
  taskId: string,
  incoming: DrawingGeneratedImage[],
): DrawingGeneratedImage[] {
  const taskImagePrefix = `${taskId}-`;
  const incomingIds = new Set(incoming.map((image) => image.id));
  const taskImages = current.filter((image) =>
    image.id.startsWith(taskImagePrefix),
  );
  const hasLegacyImages =
    taskImages.length !== incomingIds.size ||
    taskImages.some((image) => !incomingIds.has(image.id));

  if (!hasLegacyImages) {
    return mergeGeneratedImages(current, incoming);
  }

  return mergeGeneratedImages(
    current.filter((image) => !image.id.startsWith(taskImagePrefix)),
    incoming,
  );
}

export function isActiveDrawingTask(task?: Pick<DrawingTask, "status"> | null) {
  return (
    task?.status === "queued" ||
    task?.status === "running" ||
    task?.status === "canceling"
  );
}

export function applyDrawingTaskToWorkspaces(
  current: DrawingWorkspace[],
  task: DrawingTask<DrawingGeneratedImage>,
): DrawingWorkspace[] {
  if (!current.some((workspace) => workspace.id === task.workspace_id)) {
    const workspace = {
      ...createDrawingWorkspace(current.length, task.model),
      id: task.workspace_id,
      lastPrompt: task.prompt,
    };
    return applyDrawingTaskToWorkspaces([...current, workspace], task);
  }

  let changed = false;
  const next = current.map((workspace) => {
    if (workspace.id !== task.workspace_id) {
      return workspace;
    }

    if (task.status === "succeeded") {
      const taskImages = (task.images ?? []).map((image) => ({
        ...image,
        model: image.model || task.model,
        options: image.options || task.options,
      }));
      const images = reconcileGeneratedImagesForTask(
        workspace.images,
        task.task_id,
        taskImages,
      );
      if (
        !workspace.pending &&
        workspace.taskId === undefined &&
        workspace.taskStatus === undefined &&
        workspace.taskError === undefined &&
        images === workspace.images
      ) {
        return workspace;
      }
      changed = true;
      return {
        ...workspace,
        pending: false,
        taskId: undefined,
        taskStatus: undefined,
        taskError: undefined,
        images,
      };
    }

    if (task.status === "failed" || task.status === "canceled") {
      const taskError = task.error || undefined;
      if (
        !workspace.pending &&
        workspace.taskId === task.task_id &&
        workspace.taskStatus === task.status &&
        workspace.taskError === taskError
      ) {
        return workspace;
      }
      changed = true;
      return {
        ...workspace,
        pending: false,
        taskId: task.task_id,
        taskStatus: task.status,
        taskError,
      };
    }

    const lastPrompt = task.prompt || workspace.lastPrompt;
    if (
      workspace.pending &&
      workspace.taskId === task.task_id &&
      workspace.taskStatus === task.status &&
      workspace.taskError === undefined &&
      workspace.lastPrompt === lastPrompt
    ) {
      return workspace;
    }
    changed = true;
    return {
      ...workspace,
      pending: true,
      taskId: task.task_id,
      taskStatus: task.status,
      taskError: undefined,
      lastPrompt,
    };
  });

  return changed ? next : current;
}

export function preserveLocalActiveTaskState(
  next: DrawingWorkspace[],
  current: DrawingWorkspace[],
): DrawingWorkspace[] {
  const currentById = new Map(
    current.map((workspace) => [workspace.id, workspace]),
  );

  return next.map((workspace) => {
    const currentWorkspace = currentById.get(workspace.id);
    if (!currentWorkspace?.taskId || !currentWorkspace.taskStatus) {
      return workspace;
    }

    return {
      ...workspace,
      pending: currentWorkspace.pending,
      taskId: currentWorkspace.taskId,
      taskStatus: currentWorkspace.taskStatus,
      taskError: currentWorkspace.taskError,
      lastPrompt: currentWorkspace.lastPrompt || workspace.lastPrompt,
    };
  });
}

export function getClipboardImageFiles(
  clipboardData: DataTransfer | null,
): File[] {
  if (!clipboardData) return [];

  return Array.from(clipboardData.items)
    .filter((item) => item.kind === "file" && item.type.startsWith("image/"))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file));
}

export function normalizeDrawingWorkspaces(value: unknown): DrawingWorkspace[] {
  if (!Array.isArray(value)) {
    return [createDrawingWorkspace()];
  }

  const workspaces = value
    .map((item, index): DrawingWorkspace | null => {
      if (!item || typeof item !== "object") {
        return null;
      }

      const workspace = item as Partial<DrawingWorkspace>;
      const model = typeof workspace.model === "string" ? workspace.model : "";

      return {
        id:
          typeof workspace.id === "string" && workspace.id
            ? workspace.id
            : createWorkspaceId(),
        model,
        mode: workspace.mode === "edit" ? "edit" : "generate",
        prompt: typeof workspace.prompt === "string" ? workspace.prompt : "",
        options: normalizeDrawingOptions(
          workspace.options,
          getDrawingModelCapabilities(model),
        ),
        references: normalizeDrawingReferences(workspace.references),
        images: normalizeDrawingImages(workspace.images),
        pending:
          workspace.taskStatus === "queued" ||
          workspace.taskStatus === "running" ||
          workspace.taskStatus === "canceling"
            ? Boolean(workspace.pending)
            : false,
        taskId:
          typeof workspace.taskId === "string" && workspace.taskId
            ? workspace.taskId
            : undefined,
        taskStatus: workspace.taskStatus,
        taskError:
          typeof workspace.taskError === "string"
            ? workspace.taskError
            : undefined,
        lastPrompt:
          typeof workspace.lastPrompt === "string" ? workspace.lastPrompt : "",
        createdAt:
          typeof workspace.createdAt === "number" &&
          Number.isFinite(workspace.createdAt)
            ? workspace.createdAt
            : Date.now(),
        accent:
          typeof workspace.accent === "number" &&
          Number.isFinite(workspace.accent)
            ? workspace.accent
            : index,
      };
    })
    .filter((workspace): workspace is DrawingWorkspace => Boolean(workspace));

  return workspaces.length > 0 ? workspaces : [createDrawingWorkspace()];
}

export function loadDrawingWorkspaces(): DrawingWorkspace[] {
  if (typeof window === "undefined") {
    return [createDrawingWorkspace()];
  }

  try {
    const raw = window.localStorage.getItem(DRAWING_WORKSPACES_KEY);
    if (!raw) {
      return [createDrawingWorkspace()];
    }

    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [createDrawingWorkspace()];
    }

    return normalizeDrawingWorkspaces(parsed);
  } catch {
    return [createDrawingWorkspace()];
  }
}

export function saveDrawingWorkspaceSnapshot(snapshot: string) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(DRAWING_WORKSPACES_KEY, snapshot);
}

export function loadActiveWorkspaceId() {
  if (typeof window === "undefined") {
    return "";
  }

  return window.localStorage.getItem(DRAWING_ACTIVE_WORKSPACE_KEY) ?? "";
}

export function saveActiveWorkspaceId(workspaceId: string) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(DRAWING_ACTIVE_WORKSPACE_KEY, workspaceId);
}

export function getRequestedDrawingModelId() {
  if (typeof window === "undefined") {
    return "";
  }

  const url = new URL(window.location.href);
  return url.searchParams.get(DRAWING_MODEL_QUERY_PARAM)?.trim() ?? "";
}

export function clearRequestedDrawingModelId() {
  if (typeof window === "undefined") {
    return;
  }

  const url = new URL(window.location.href);
  if (!url.searchParams.has(DRAWING_MODEL_QUERY_PARAM)) {
    return;
  }
  url.searchParams.delete(DRAWING_MODEL_QUERY_PARAM);
  window.history.replaceState(
    window.history.state,
    "",
    `${url.pathname}${url.search}${url.hash}`,
  );
}
