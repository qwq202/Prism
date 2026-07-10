import {
  type ComponentType,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  Wand2,
  Plus,
  Ratio,
  Upload,
  ArrowUp,
  Brain,
  Download,
  ImagePlus,
  Sparkles,
  Trash2,
  FileType2,
  Image as ImageIcon,
  Loader2,
  RotateCcw,
  SlidersHorizontal,
  ZoomIn,
  X,
} from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/components/ui/lib/utils";
import { useTranslation } from "react-i18next";
import { useSelector } from "react-redux";
import { selectSupportModels } from "@/store/chat.ts";
import { isDrawingModel } from "@/conf/model.ts";
import ModelAvatar from "@/components/ModelAvatar.tsx";
import FileProvider, {
  type FileProviderAction,
} from "@/components/FileProvider.tsx";
import type { FileArray } from "@/api/file.ts";
import {
  cancelDrawingTask,
  createDrawingTask,
  getDrawingTask,
  listDrawingTasks,
  loadDrawingWorkspaceState,
  saveDrawingWorkspaceState,
  type DrawingTask,
} from "@/api/drawing.ts";
import { formatMessage } from "@/utils/processor.ts";
import { toast } from "sonner";
import { blobEvent } from "@/events/blob.ts";
import { normalizeImageURL } from "@/utils/image-url.ts";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import type { Model } from "@/api/types.ts";

type Mode = "generate" | "edit";
type GeminiImageAspectRatio =
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
type GeminiImageSize = "512px" | "1K" | "2K" | "4K";
type GeminiImageMimeType = "image/png" | "image/jpeg";
type GeminiImageThinkingLevel = "minimal" | "high";

type DrawingOptions = {
  aspectRatio: GeminiImageAspectRatio;
  imageSize: GeminiImageSize;
  mimeType: GeminiImageMimeType;
  thinkingLevel: GeminiImageThinkingLevel;
};

type DrawingModelCapabilities = {
  aspectRatios: readonly GeminiImageAspectRatio[];
  imageSizes: readonly GeminiImageSize[];
  mimeTypes: readonly GeminiImageMimeType[];
  thinkingLevels: readonly GeminiImageThinkingLevel[];
  maxReferenceImages: number;
  supportsEditing: boolean;
};

type DrawingModel = Pick<
  Model,
  "id" | "name" | "channel_type" | "drawing_model"
>;

type DrawingGeneratedImage = {
  id: string;
  src: string;
  prompt: string;
  createdAt: number;
};

type DrawingWorkspace = {
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

const DRAWING_WORKSPACES_KEY = "drawing.workspaces.v1";
const DRAWING_ACTIVE_WORKSPACE_KEY = "drawing.activeWorkspaceId.v1";
const DRAWING_MODEL_QUERY_PARAM = "model";
const DRAWING_TASK_POLL_INTERVAL_MS = 2500;
const MAX_DRAWING_WORKSPACES = 64;

const WORKSPACE_ACCENTS = [
  {
    active: "from-violet-500/14 to-blue-500/12",
    idle: "from-violet-500/8 to-blue-500/6",
  },
  {
    active: "from-emerald-500/14 to-cyan-500/12",
    idle: "from-emerald-500/8 to-cyan-500/6",
  },
  {
    active: "from-rose-500/14 to-amber-500/12",
    idle: "from-rose-500/8 to-amber-500/6",
  },
  {
    active: "from-sky-500/14 to-indigo-500/12",
    idle: "from-sky-500/8 to-indigo-500/6",
  },
];

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

function createDrawingWorkspace(index = 0, model = ""): DrawingWorkspace {
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
    accent: index % WORKSPACE_ACCENTS.length,
  };
}

function drawingFileReducer(
  state: FileArray,
  action: FileProviderAction,
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

function hasDraggedFiles(dataTransfer: DataTransfer | null): boolean {
  return Boolean(
    dataTransfer && Array.from(dataTransfer.types).includes("Files"),
  );
}

function getDroppedFiles(dataTransfer: DataTransfer | null): File[] {
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

function getDrawingModelCapabilities(
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

function normalizeDrawingOptions(
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

function buildDrawingRequestOptions(
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
      };
    })
    .filter((image): image is DrawingGeneratedImage => Boolean(image));
}

function normalizeDrawingReferences(value: unknown): FileArray {
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

function getDrawingImageExtension(source: string): string {
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

  const seen = new Set(current.map((image) => image.src));
  const next = [...current];
  let changed = false;
  incoming.forEach((image) => {
    if (seen.has(image.src)) return;
    seen.add(image.src);
    next.unshift(image);
    changed = true;
  });
  return changed ? next : current;
}

function isActiveDrawingTask(task?: Pick<DrawingTask, "status"> | null) {
  return task?.status === "queued" || task?.status === "running";
}

function applyDrawingTaskToWorkspaces(
  current: DrawingWorkspace[],
  task: DrawingTask<DrawingGeneratedImage>,
): DrawingWorkspace[] {
  let changed = false;
  const next = current.map((workspace) => {
    if (workspace.id !== task.workspace_id) {
      return workspace;
    }

    if (task.status === "succeeded") {
      const images = mergeGeneratedImages(workspace.images, task.images ?? []);
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

function preserveLocalActiveTaskState(
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

function getClipboardImageFiles(clipboardData: DataTransfer | null): File[] {
  if (!clipboardData) return [];

  return Array.from(clipboardData.items)
    .filter((item) => item.kind === "file" && item.type.startsWith("image/"))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file));
}

function normalizeDrawingWorkspaces(value: unknown): DrawingWorkspace[] {
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
          workspace.taskStatus === "running"
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

function loadDrawingWorkspaces(): DrawingWorkspace[] {
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

function loadActiveWorkspaceId() {
  if (typeof window === "undefined") {
    return "";
  }

  return window.localStorage.getItem(DRAWING_ACTIVE_WORKSPACE_KEY) ?? "";
}

function getRequestedDrawingModelId() {
  if (typeof window === "undefined") {
    return "";
  }

  const url = new URL(window.location.href);
  return url.searchParams.get(DRAWING_MODEL_QUERY_PARAM)?.trim() ?? "";
}

function clearRequestedDrawingModelId() {
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

type DrawingOptionSelectProps<T extends string> = {
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: T;
  options: readonly T[];
  getLabel?: (value: T) => string;
  onChange: (value: T) => void;
};

function DrawingOptionSelect<T extends string>({
  icon: Icon,
  label,
  value,
  options,
  getLabel,
  onChange,
}: DrawingOptionSelectProps<T>) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
        <Icon className="h-3 w-3" />
        <span>{label}</span>
      </div>
      <Select value={value} onValueChange={(next) => onChange(next as T)}>
        <SelectTrigger
          aria-label={label}
          className="h-9 w-full border-border/60 bg-background/60 px-2.5 text-xs"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option} value={option}>
              {getLabel ? getLabel(option) : option}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function Drawing() {
  const { t } = useTranslation();
  const supportModels = useSelector(selectSupportModels);
  const [requestedDrawingModelId] = useState(getRequestedDrawingModelId);
  const handledRequestedDrawingModel = useRef(false);
  const dragDepthRef = useRef(0);
  const latestWorkspaceSnapshotRef = useRef("");
  const latestActiveWorkspaceIdRef = useRef("");
  const [workspaces, setWorkspaces] = useState<DrawingWorkspace[]>(() =>
    loadDrawingWorkspaces(),
  );
  const [activeWorkspaceId, setActiveWorkspaceId] = useState(() =>
    loadActiveWorkspaceId(),
  );
  const [cloudSyncEnabled, setCloudSyncEnabled] = useState(false);
  const [cloudSyncReady, setCloudSyncReady] = useState(false);
  const [cloudSyncError, setCloudSyncError] = useState("");
  const [focused, setFocused] = useState(false);
  const [draggingReferences, setDraggingReferences] = useState(false);
  const [referenceUploadPending, setReferenceUploadPending] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [previewImage, setPreviewImage] =
    useState<DrawingGeneratedImage | null>(null);
  const activeWorkspace =
    workspaces.find((workspace) => workspace.id === activeWorkspaceId) ??
    workspaces[0];
  const activeWorkspaceIdForStorage = activeWorkspace?.id ?? "";
  const files = useMemo(
    () => activeWorkspace?.references ?? [],
    [activeWorkspace?.references],
  );
  const drawingModels = useMemo(
    () => supportModels.filter((model) => isDrawingModel(model)),
    [supportModels],
  );
  const selectedDrawingModel =
    drawingModels.find((model) => model.id === activeWorkspace?.model) ??
    drawingModels[0];
  const selectedDrawingModelId = selectedDrawingModel?.id ?? "";
  const drawingModelCapabilities = useMemo(
    () =>
      getDrawingModelCapabilities(
        (selectedDrawingModel as DrawingModel | undefined) ??
          selectedDrawingModelId,
      ),
    [selectedDrawingModel, selectedDrawingModelId],
  );
  const mode = activeWorkspace?.mode ?? "generate";
  const prompt = activeWorkspace?.prompt ?? "";
  const rawOptions = activeWorkspace?.options;
  const options = useMemo(
    () => normalizeDrawingOptions(rawOptions, drawingModelCapabilities),
    [drawingModelCapabilities, rawOptions],
  );
  const referenceImageLimit = drawingModelCapabilities.maxReferenceImages;
  const canAcceptReferences = referenceImageLimit > 0;
  const uploadReferenceTitle = t("drawing.uploadReferenceWithLimit", {
    limit: referenceImageLimit,
  });
  const generatedImages = activeWorkspace?.images ?? [];
  const activeWorkspacePending =
    Boolean(activeWorkspace?.pending) ||
    isActiveDrawingTask(
      activeWorkspace?.taskStatus
        ? { status: activeWorkspace.taskStatus }
        : undefined,
    );
  const requestInFlight = activeWorkspacePending;
  const referencesRemaining = Math.max(0, referenceImageLimit - files.length);
  const canGenerate = Boolean(
    prompt.trim() &&
    selectedDrawingModelId &&
    cloudSyncReady &&
    !requestInFlight &&
    !referenceUploadPending &&
    (mode !== "edit" || files.length > 0),
  );
  const generateDisabledReason = !selectedDrawingModelId
    ? t("drawing.needModel")
    : !cloudSyncReady
      ? t("drawing.loadingWorkspace")
      : !prompt.trim()
        ? t("drawing.needPrompt")
        : referenceUploadPending
          ? t("drawing.uploadProcessing")
          : mode === "edit" && files.length === 0
            ? t("drawing.editRequiresReference")
            : requestInFlight
              ? t("drawing.generating")
              : "";
  const activeTaskIds = useMemo(
    () =>
      workspaces
        .filter(
          (workspace) =>
            workspace.taskId &&
            (isActiveDrawingTask(
              workspace.taskStatus
                ? { status: workspace.taskStatus }
                : undefined,
            ) ||
              workspace.pending),
        )
        .map((workspace) => workspace.taskId as string),
    [workspaces],
  );
  const activeTaskKey = useMemo(
    () => [...activeTaskIds].sort().join("|"),
    [activeTaskIds],
  );

  useEffect(() => {
    const resetDragState = () => {
      dragDepthRef.current = 0;
      setDraggingReferences(false);
    };

    const handleDragEnter = (event: DragEvent) => {
      if (!canAcceptReferences) return;
      if (!hasDraggedFiles(event.dataTransfer)) return;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
      dragDepthRef.current += 1;
      setDraggingReferences(true);
    };

    const handleDragOver = (event: DragEvent) => {
      if (!canAcceptReferences) return;
      if (!hasDraggedFiles(event.dataTransfer)) return;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
      setDraggingReferences(true);
    };

    const handleDragLeave = (event: DragEvent) => {
      if (!canAcceptReferences) return;
      if (!hasDraggedFiles(event.dataTransfer)) return;
      dragDepthRef.current = Math.max(0, dragDepthRef.current - 1);
      if (dragDepthRef.current === 0) {
        setDraggingReferences(false);
      }
    };

    const handleDrop = (event: DragEvent) => {
      if (!canAcceptReferences) return;
      if (!hasDraggedFiles(event.dataTransfer)) return;
      event.preventDefault();
      resetDragState();

      const droppedFiles = getDroppedFiles(event.dataTransfer);
      if (droppedFiles.length > 0) {
        blobEvent.emit(droppedFiles);
      }
    };

    window.addEventListener("dragenter", handleDragEnter);
    window.addEventListener("dragover", handleDragOver);
    window.addEventListener("dragleave", handleDragLeave);
    window.addEventListener("drop", handleDrop);

    return () => {
      window.removeEventListener("dragenter", handleDragEnter);
      window.removeEventListener("dragover", handleDragOver);
      window.removeEventListener("dragleave", handleDragLeave);
      window.removeEventListener("drop", handleDrop);
    };
  }, [canAcceptReferences]);

  useEffect(() => {
    let cancelled = false;

    void loadDrawingWorkspaceState<DrawingWorkspace>().then(
      async (response) => {
        if (cancelled) return;

        if (!response.status || !response.data) {
          setCloudSyncEnabled(false);
          setCloudSyncError(
            response.message || response.error || t("drawing.syncUnavailable"),
          );
          setCloudSyncReady(true);
          return;
        }

        setCloudSyncEnabled(true);
        setCloudSyncError("");
        let normalized: DrawingWorkspace[] | undefined;
        const serverWorkspaces = response.data.workspaces;
        if (Array.isArray(serverWorkspaces) && serverWorkspaces.length > 0) {
          normalized = normalizeDrawingWorkspaces(serverWorkspaces);

          const serverActiveWorkspaceId =
            response.data.active_workspace_id &&
            normalized.some(
              (workspace) =>
                workspace.id === response.data?.active_workspace_id,
            )
              ? response.data.active_workspace_id
              : normalized[0]?.id;
          if (serverActiveWorkspaceId) {
            setActiveWorkspaceId(serverActiveWorkspaceId);
          }
        }

        const tasksResponse = await listDrawingTasks<DrawingGeneratedImage>();
        if (cancelled) return;
        if (!tasksResponse.status) {
          setCloudSyncError(
            tasksResponse.message ||
              tasksResponse.error ||
              t("drawing.syncUnavailable"),
          );
        }
        if (normalized) {
          const tasks = tasksResponse.status ? tasksResponse.data || [] : [];
          setWorkspaces(tasks.reduce(applyDrawingTaskToWorkspaces, normalized));
        }

        setCloudSyncReady(true);
      },
    );

    return () => {
      cancelled = true;
    };
  }, [t]);

  useEffect(() => {
    const firstWorkspaceId = workspaces[0]?.id;
    if (
      firstWorkspaceId &&
      !workspaces.some((workspace) => workspace.id === activeWorkspaceId)
    ) {
      setActiveWorkspaceId(firstWorkspaceId);
    }
  }, [activeWorkspaceId, workspaces]);

  useEffect(() => {
    const defaultModelId = drawingModels[0]?.id;
    if (!defaultModelId) return;

    setWorkspaces((current) => {
      let changed = false;
      const next = current.map((workspace) => {
        if (workspace.model) return workspace;
        changed = true;
        return { ...workspace, model: defaultModelId };
      });
      return changed ? next : current;
    });
  }, [drawingModels]);

  useEffect(() => {
    if (!requestedDrawingModelId || handledRequestedDrawingModel.current) {
      return;
    }
    if (drawingModels.length === 0) {
      return;
    }

    handledRequestedDrawingModel.current = true;
    const requestedModel = drawingModels.find(
      (model) => model.id === requestedDrawingModelId,
    );
    clearRequestedDrawingModelId();
    if (!requestedModel) {
      return;
    }

    const capabilities = getDrawingModelCapabilities(requestedModel);
    if (files.length > capabilities.maxReferenceImages) {
      toast.error(t("drawing.referenceLimitExceeded"), {
        description: t("drawing.referenceLimitExceededPrompt", {
          count: files.length,
          limit: capabilities.maxReferenceImages,
          model: requestedModel.name || requestedModel.id,
        }),
      });
      return;
    }

    setWorkspaces((current) => {
      const targetWorkspaceId = activeWorkspaceIdForStorage || current[0]?.id;
      if (!targetWorkspaceId) {
        return current;
      }

      return current.map((workspace) =>
        workspace.id === targetWorkspaceId
          ? {
              ...workspace,
              model: requestedModel.id,
              mode: capabilities.supportsEditing ? workspace.mode : "generate",
              options: normalizeDrawingOptions(workspace.options, capabilities),
            }
          : workspace,
      );
    });
    if (activeWorkspaceIdForStorage) {
      setActiveWorkspaceId(activeWorkspaceIdForStorage);
    }
  }, [
    activeWorkspaceIdForStorage,
    drawingModels,
    files.length,
    requestedDrawingModelId,
    t,
  ]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    latestWorkspaceSnapshotRef.current = JSON.stringify(workspaces);
    latestActiveWorkspaceIdRef.current = activeWorkspaceIdForStorage;

    try {
      window.localStorage.setItem(
        DRAWING_WORKSPACES_KEY,
        latestWorkspaceSnapshotRef.current,
      );
    } catch (error) {
      console.debug("[drawing] local workspace cache is full", error);
    }
  }, [activeWorkspaceIdForStorage, workspaces]);

  useEffect(() => {
    if (typeof window === "undefined" || !activeWorkspaceIdForStorage) {
      return;
    }

    try {
      window.localStorage.setItem(
        DRAWING_ACTIVE_WORKSPACE_KEY,
        activeWorkspaceIdForStorage,
      );
    } catch (error) {
      console.debug("[drawing] failed to cache active workspace", error);
    }
  }, [activeWorkspaceIdForStorage]);

  useEffect(() => {
    if (!cloudSyncReady || !cloudSyncEnabled) {
      return;
    }

    const workspacesSnapshot = JSON.stringify(workspaces);
    const activeWorkspaceIdSnapshot = activeWorkspaceIdForStorage;

    const timer = setTimeout(() => {
      void saveDrawingWorkspaceState<DrawingWorkspace>({
        active_workspace_id: activeWorkspaceIdSnapshot,
        workspaces,
      }).then((response) => {
        if (!response.status || !response.data) {
          setCloudSyncError(
            response.message || response.error || t("drawing.syncUnavailable"),
          );
          return;
        }
        setCloudSyncError("");

        if (
          latestWorkspaceSnapshotRef.current !== workspacesSnapshot ||
          latestActiveWorkspaceIdRef.current !== activeWorkspaceIdSnapshot
        ) {
          return;
        }

        const normalized = preserveLocalActiveTaskState(
          normalizeDrawingWorkspaces(response.data.workspaces),
          workspaces,
        );
        const normalizedSnapshot = JSON.stringify(normalized);
        if (normalizedSnapshot !== workspacesSnapshot) {
          setWorkspaces(normalized);
        }

        const nextActiveWorkspaceId =
          response.data.active_workspace_id &&
          normalized.some(
            (workspace) => workspace.id === response.data?.active_workspace_id,
          )
            ? response.data.active_workspace_id
            : normalized[0]?.id;
        if (
          nextActiveWorkspaceId &&
          nextActiveWorkspaceId !== activeWorkspaceIdSnapshot
        ) {
          setActiveWorkspaceId(nextActiveWorkspaceId);
        }
      });
    }, 700);

    return () => {
      clearTimeout(timer);
    };
  }, [
    activeWorkspaceIdForStorage,
    cloudSyncEnabled,
    cloudSyncReady,
    t,
    workspaces,
  ]);

  const updateActiveWorkspace = useCallback(
    (
      updates: Partial<
        Pick<
          DrawingWorkspace,
          | "model"
          | "mode"
          | "prompt"
          | "options"
          | "references"
          | "images"
          | "pending"
          | "taskId"
          | "taskStatus"
          | "taskError"
          | "lastPrompt"
        >
      >,
    ) => {
      if (!activeWorkspace) {
        return;
      }

      setWorkspaces((current) =>
        current.map((workspace) =>
          workspace.id === activeWorkspace.id
            ? { ...workspace, ...updates }
            : workspace,
        ),
      );
    },
    [activeWorkspace],
  );

  const dispatchFiles = useCallback(
    (action: FileProviderAction) => {
      const workspaceId = activeWorkspaceIdForStorage;
      if (!workspaceId) {
        return;
      }

      setWorkspaces((current) =>
        current.map((workspace) =>
          workspace.id === workspaceId
            ? {
                ...workspace,
                references: drawingFileReducer(
                  normalizeDrawingReferences(workspace.references),
                  action,
                ),
              }
            : workspace,
        ),
      );
    },
    [activeWorkspaceIdForStorage],
  );

  const updateWorkspaceById = useCallback(
    (workspaceId: string, updates: Partial<DrawingWorkspace>) => {
      setWorkspaces((current) =>
        current.map((workspace) =>
          workspace.id === workspaceId
            ? { ...workspace, ...updates }
            : workspace,
        ),
      );
    },
    [],
  );

  useEffect(() => {
    if (!activeTaskKey) {
      return;
    }

    let cancelled = false;
    let timer: number | undefined;
    const controller = new AbortController();
    const taskIds = activeTaskKey.split("|").filter(Boolean);
    const poll = async () => {
      for (const taskId of taskIds) {
        const response = await getDrawingTask<DrawingGeneratedImage>(
          taskId,
          controller.signal,
        );
        if (cancelled) return;
        if (!response.status || !response.data) {
          setCloudSyncError(
            response.message || response.error || t("drawing.syncUnavailable"),
          );
          continue;
        }

        setCloudSyncError("");
        setWorkspaces((current) =>
          applyDrawingTaskToWorkspaces(current, response.data!),
        );
      }

      if (!cancelled) {
        timer = window.setTimeout(poll, DRAWING_TASK_POLL_INTERVAL_MS);
      }
    };

    void poll();

    return () => {
      cancelled = true;
      controller.abort();
      if (timer !== undefined) window.clearTimeout(timer);
    };
  }, [activeTaskKey, t]);

  const getCapabilitiesForModelId = useCallback(
    (modelId: string) =>
      getDrawingModelCapabilities(
        (drawingModels.find((model) => model.id === modelId) as
          | DrawingModel
          | undefined) ?? modelId,
      ),
    [drawingModels],
  );

  const canUseReferencesWithModel = useCallback(
    (modelId: string, references: FileArray = files) =>
      references.length <=
      getCapabilitiesForModelId(modelId).maxReferenceImages,
    [files, getCapabilitiesForModelId],
  );

  const notifyReferenceImageLimit = useCallback(
    (
      modelId: string,
      count = files.length,
      titleKey:
        | "referenceLimitExceeded"
        | "referenceLimitReached" = "referenceLimitExceeded",
    ) => {
      const capabilities = getCapabilitiesForModelId(modelId);
      const modelName =
        drawingModels.find((model) => model.id === modelId)?.name || modelId;

      toast.error(t(`drawing.${titleKey}`), {
        description: t("drawing.referenceLimitExceededPrompt", {
          count,
          limit: capabilities.maxReferenceImages,
          model: modelName,
        }),
      });
    },
    [drawingModels, files.length, getCapabilitiesForModelId, t],
  );

  const selectDrawingModel = useCallback(
    (modelId: string) => {
      if (!canUseReferencesWithModel(modelId)) {
        notifyReferenceImageLimit(modelId);
        return;
      }

      const capabilities = getCapabilitiesForModelId(modelId);
      updateActiveWorkspace({
        model: modelId,
        mode: capabilities.supportsEditing ? mode : "generate",
        options: normalizeDrawingOptions(options, capabilities),
      });
      setSettingsOpen(false);
    },
    [
      canUseReferencesWithModel,
      getCapabilitiesForModelId,
      mode,
      notifyReferenceImageLimit,
      options,
      updateActiveWorkspace,
    ],
  );

  const selectWorkspace = useCallback((workspace: DrawingWorkspace) => {
    setActiveWorkspaceId(workspace.id);
  }, []);

  const addWorkspace = () => {
    if (workspaces.length >= MAX_DRAWING_WORKSPACES) {
      toast.error(
        t("drawing.workspaceLimitReached", {
          count: MAX_DRAWING_WORKSPACES,
        }),
      );
      return;
    }

    const workspace = createDrawingWorkspace(
      workspaces.length,
      selectedDrawingModelId,
    );
    setWorkspaces((current) => [...current, workspace]);
    setActiveWorkspaceId(workspace.id);
  };

  const deleteWorkspace = (workspaceId: string) => {
    const workspaceIndex = workspaces.findIndex(
      (workspace) => workspace.id === workspaceId,
    );

    if (workspaceIndex === -1) {
      return;
    }
    if (
      workspaces[workspaceIndex]?.pending ||
      isActiveDrawingTask(
        workspaces[workspaceIndex]?.taskStatus
          ? { status: workspaces[workspaceIndex]?.taskStatus }
          : undefined,
      )
    ) {
      toast.info(t("drawing.generating"));
      return;
    }

    if (workspaces.length === 1) {
      const workspace = createDrawingWorkspace(0, selectedDrawingModelId);
      setWorkspaces([workspace]);
      setActiveWorkspaceId(workspace.id);
      return;
    }

    const nextWorkspaces = workspaces.filter(
      (workspace) => workspace.id !== workspaceId,
    );

    if (workspaceId === activeWorkspaceIdForStorage) {
      const nextActiveWorkspace =
        nextWorkspaces[Math.min(workspaceIndex, nextWorkspaces.length - 1)] ??
        nextWorkspaces[0];

      setWorkspaces(nextWorkspaces);

      if (nextActiveWorkspace) {
        setActiveWorkspaceId(nextActiveWorkspace.id);
      }

      return;
    }

    setWorkspaces(nextWorkspaces);
  };

  const updateDrawingOptions = (updates: Partial<DrawingOptions>) => {
    updateActiveWorkspace({
      options: normalizeDrawingOptions(
        {
          ...options,
          ...updates,
        },
        drawingModelCapabilities,
      ),
    });
  };

  const removeReferenceFile = (index: number) => {
    dispatchFiles({ type: "remove", payload: index });
  };

  const clearGeneratedImages = () => {
    updateActiveWorkspace({ images: [] });
  };

  const removeGeneratedImage = (imageId: string) => {
    updateActiveWorkspace({
      images: generatedImages.filter((image) => image.id !== imageId),
    });
    if (previewImage?.id === imageId) setPreviewImage(null);
  };

  const addGeneratedImageAsReference = (image: DrawingGeneratedImage) => {
    if (files.length >= referenceImageLimit) {
      notifyReferenceImageLimit(
        selectedDrawingModelId,
        files.length,
        "referenceLimitReached",
      );
      return;
    }

    const extension = getDrawingImageExtension(image.src);
    dispatchFiles({
      type: "add",
      payload: {
        name: `drawing-${new Date(image.createdAt).toISOString()}.${extension}`,
        content: image.src,
      },
    });
    toast.success(t("drawing.referenceAdded"));
  };

  const generateImage = async () => {
    const text = prompt.trim();
    const workspaceId = activeWorkspaceIdForStorage;
    if (
      !text ||
      !selectedDrawingModelId ||
      requestInFlight ||
      referenceUploadPending ||
      !workspaceId
    ) {
      return;
    }
    if (mode === "edit" && files.length === 0) {
      toast.info(t("drawing.editRequiresReference"));
      return;
    }
    if (!canUseReferencesWithModel(selectedDrawingModelId)) {
      notifyReferenceImageLimit(
        selectedDrawingModelId,
        files.length,
        "referenceLimitReached",
      );
      return;
    }

    updateWorkspaceById(workspaceId, {
      model: selectedDrawingModelId,
      pending: true,
      taskId: undefined,
      taskStatus: "queued",
      taskError: undefined,
      lastPrompt: text,
    });
    try {
      const requestOptions = buildDrawingRequestOptions(
        options,
        drawingModelCapabilities,
      );
      const response = await createDrawingTask<DrawingGeneratedImage>({
        workspace_id: workspaceId,
        message: formatMessage(files, text),
        prompt: text,
        model: selectedDrawingModelId,
        response_format: requestOptions.response_format,
        thinking: requestOptions.thinking,
      });
      if (!response.status || !response.data) {
        throw new Error(response.message || response.error || "");
      }
      setWorkspaces((current) =>
        applyDrawingTaskToWorkspaces(current, response.data!),
      );
    } catch (error) {
      const message =
        error instanceof Error && error.message
          ? error.message
          : t("drawing.generateFailed");
      updateWorkspaceById(workspaceId, {
        pending: false,
        taskStatus: "failed",
        taskError: message,
      });
      toast.error(t("drawing.generateFailed"), {
        description: message,
      });
    }
  };

  const cancelActiveTask = async () => {
    const taskId = activeWorkspace?.taskId;
    if (!taskId || !requestInFlight) {
      return;
    }

    const response = await cancelDrawingTask<DrawingGeneratedImage>(taskId);
    if (!response.status || !response.data) {
      toast.error(t("drawing.cancelFailed"), {
        description: response.message || response.error,
      });
      return;
    }

    setWorkspaces((current) =>
      applyDrawingTaskToWorkspaces(current, response.data!),
    );
  };

  const hasGenerationOptions =
    drawingModelCapabilities.aspectRatios.length > 0 ||
    drawingModelCapabilities.imageSizes.length > 0 ||
    drawingModelCapabilities.mimeTypes.length > 0 ||
    drawingModelCapabilities.thinkingLevels.length > 0;
  const terminalTaskStatus =
    activeWorkspace?.taskStatus === "failed" ||
    activeWorkspace?.taskStatus === "canceled";

  const settingsPanel = (
    <div className="space-y-5">
      <div className="space-y-2.5">
        <label className="text-sm font-medium text-foreground">
          {t("drawing.model")}
        </label>
        <Select
          value={selectedDrawingModelId || undefined}
          onValueChange={selectDrawingModel}
          disabled={drawingModels.length === 0}
        >
          <SelectTrigger
            aria-label={t("drawing.model")}
            className="h-11 w-full border-border/60 bg-background/60 text-sm"
          >
            {selectedDrawingModel ? (
              <div className="flex min-w-0 items-center gap-2">
                <ModelAvatar
                  model={selectedDrawingModel}
                  size={22}
                  className="shrink-0"
                />
                <span className="truncate">
                  {selectedDrawingModel.name || selectedDrawingModel.id}
                </span>
              </div>
            ) : (
              <SelectValue placeholder={t("drawing.selectModel")} />
            )}
          </SelectTrigger>
          <SelectContent>
            {drawingModels.map((model) => (
              <SelectItem key={model.id} value={model.id}>
                <div className="flex min-w-0 items-center gap-2">
                  <ModelAvatar model={model} size={22} className="shrink-0" />
                  <span className="truncate">{model.name || model.id}</span>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {drawingModels.length === 0 && (
          <p className="text-xs leading-relaxed text-muted-foreground">
            {t("drawing.noModels")}
          </p>
        )}
      </div>

      {drawingModels.length > 0 && hasGenerationOptions && (
        <div className="space-y-3 border-t border-border/60 pt-4">
          <div className="text-sm font-medium text-foreground">
            {t("drawing.options.title")}
          </div>
          <div className="grid grid-cols-2 gap-3">
            {drawingModelCapabilities.aspectRatios.length > 0 && (
              <DrawingOptionSelect
                icon={Ratio}
                label={t("drawing.options.aspectRatio")}
                value={options.aspectRatio}
                options={drawingModelCapabilities.aspectRatios}
                onChange={(aspectRatio) =>
                  updateDrawingOptions({ aspectRatio })
                }
              />
            )}
            {drawingModelCapabilities.imageSizes.length > 0 && (
              <DrawingOptionSelect
                icon={ImageIcon}
                label={t("drawing.options.imageSize")}
                value={options.imageSize}
                options={drawingModelCapabilities.imageSizes}
                onChange={(imageSize) => updateDrawingOptions({ imageSize })}
              />
            )}
            {drawingModelCapabilities.mimeTypes.length > 0 && (
              <DrawingOptionSelect
                icon={FileType2}
                label={t("drawing.options.mimeType")}
                value={options.mimeType}
                options={drawingModelCapabilities.mimeTypes}
                getLabel={(value) => (value === "image/jpeg" ? "JPEG" : "PNG")}
                onChange={(mimeType) => updateDrawingOptions({ mimeType })}
              />
            )}
            {drawingModelCapabilities.thinkingLevels.length > 0 && (
              <DrawingOptionSelect
                icon={Brain}
                label={t("drawing.options.thinkingLevel")}
                value={options.thinkingLevel}
                options={drawingModelCapabilities.thinkingLevels}
                getLabel={(value) => t(`drawing.options.thinking.${value}`)}
                onChange={(thinkingLevel) =>
                  updateDrawingOptions({ thinkingLevel })
                }
              />
            )}
          </div>
        </div>
      )}
    </div>
  );

  const renderWorkspaceRail = (vertical: boolean) => (
    <aside
      className={cn(
        "z-20 shrink-0 border-border/60 bg-card/70",
        vertical
          ? "hidden min-h-0 w-[72px] flex-col border-l lg:flex"
          : "flex h-[68px] w-full items-center border-t px-2 pb-[max(0px,env(safe-area-inset-bottom))] lg:hidden",
      )}
    >
      <div
        className={cn(
          "no-scrollbar flex min-h-0 min-w-0 flex-1 gap-2 overflow-auto p-2",
          vertical ? "flex-col items-center pt-3" : "flex-row items-center",
        )}
      >
        <button
          type="button"
          onClick={addWorkspace}
          disabled={workspaces.length >= MAX_DRAWING_WORKSPACES}
          className="group flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-dashed border-border text-muted-foreground transition-colors hover:border-primary/50 hover:bg-primary/5 hover:text-primary disabled:cursor-not-allowed disabled:opacity-40"
          aria-label={t("drawing.addWorkspace")}
          title={t("drawing.addWorkspace")}
        >
          <Plus className="h-4 w-4 motion-safe:transition-transform motion-safe:duration-200 group-hover:rotate-90" />
        </button>
        {workspaces.map((workspace, index) => {
          const selected = workspace.id === activeWorkspaceIdForStorage;
          const accent =
            WORKSPACE_ACCENTS[workspace.accent % WORKSPACE_ACCENTS.length] ??
            WORKSPACE_ACCENTS[0];
          const label = selected
            ? t("drawing.activeWorkspaceTitle", { index: index + 1 })
            : t("drawing.workspaceTitle", { index: index + 1 });
          const deleteLabel = t("drawing.deleteWorkspace", {
            index: index + 1,
          });
          const workspacePending =
            workspace.pending ||
            isActiveDrawingTask(
              workspace.taskStatus
                ? { status: workspace.taskStatus }
                : undefined,
            );
          const thumbnail = workspace.images.reduce<
            DrawingGeneratedImage | undefined
          >(
            (latest, image) =>
              !latest || image.createdAt > latest.createdAt ? image : latest,
            undefined,
          );

          return (
            <div
              key={workspace.id}
              className="group relative h-11 w-11 shrink-0"
            >
              <button
                type="button"
                onClick={() => selectWorkspace(workspace)}
                className={cn(
                  "relative flex h-full w-full items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br transition-colors",
                  selected
                    ? "border-2 border-primary/70 ring-2 ring-primary/10"
                    : "border border-border/60 hover:border-foreground/30",
                  selected ? accent.active : accent.idle,
                )}
                aria-current={selected ? "true" : undefined}
                aria-label={label}
                title={label}
              >
                {thumbnail && (
                  <img
                    src={thumbnail.src}
                    alt=""
                    className="absolute inset-0 h-full w-full object-cover"
                  />
                )}
                <span className="relative z-10 flex h-5 min-w-5 items-center justify-center rounded-md bg-background/85 px-1 text-[10px] font-semibold text-foreground shadow-sm">
                  {index + 1}
                </span>
                {workspacePending ? (
                  <span className="absolute right-1 top-1 z-10 flex h-4 w-4 items-center justify-center rounded-full bg-background/90 text-primary">
                    <Loader2 className="h-2.5 w-2.5 animate-spin motion-reduce:animate-none" />
                  </span>
                ) : workspace.taskStatus === "failed" ? (
                  <span className="absolute right-1 top-1 z-10 h-2 w-2 rounded-full bg-destructive ring-2 ring-background" />
                ) : null}
              </button>
              {vertical && workspaces.length > 1 && (
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    deleteWorkspace(workspace.id);
                  }}
                  disabled={workspacePending}
                  className="pointer-events-none absolute -right-2 -top-2 z-20 flex h-7 w-7 items-center justify-center rounded-full border border-border bg-background text-muted-foreground opacity-0 transition-colors hover:border-destructive/50 hover:bg-destructive hover:text-destructive-foreground focus-visible:pointer-events-auto focus-visible:opacity-100 disabled:cursor-not-allowed disabled:opacity-40 group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100"
                  aria-label={deleteLabel}
                  title={
                    workspacePending ? t("drawing.generating") : deleteLabel
                  }
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          );
        })}
      </div>
      {!vertical && workspaces.length > 1 && (
        <button
          type="button"
          onClick={() => deleteWorkspace(activeWorkspaceIdForStorage)}
          disabled={activeWorkspacePending}
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:cursor-not-allowed disabled:opacity-35"
          aria-label={t("drawing.deleteWorkspace", {
            index:
              workspaces.findIndex(
                (workspace) => workspace.id === activeWorkspaceIdForStorage,
              ) + 1,
          })}
          title={
            activeWorkspacePending
              ? t("drawing.generating")
              : t("drawing.deleteWorkspace", {
                  index:
                    workspaces.findIndex(
                      (workspace) =>
                        workspace.id === activeWorkspaceIdForStorage,
                    ) + 1,
                })
          }
        >
          <Trash2 className="h-4 w-4" />
        </button>
      )}
    </aside>
  );

  return (
    <div
      className="flex h-full min-h-0 w-full flex-col overflow-hidden bg-background text-foreground lg:flex-row"
      onPaste={(event) => {
        if (!canAcceptReferences) return;
        const pastedImages = getClipboardImageFiles(event.clipboardData);
        if (pastedImages.length === 0) return;

        event.preventDefault();
        blobEvent.emit(pastedImages);
      }}
    >
      <aside className="z-10 hidden min-h-0 w-72 shrink-0 flex-col border-r border-border/60 bg-card/50 lg:flex">
        <div className="flex-1 overflow-y-auto p-5">{settingsPanel}</div>
      </aside>

      {/* Main Area */}
      <main className="relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <div className="relative z-30 flex h-14 shrink-0 items-center justify-between border-b border-border/60 bg-background px-3 lg:hidden">
          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className="flex h-11 min-w-0 flex-1 items-center gap-2 rounded-xl px-2 text-left transition-colors hover:bg-muted/60"
            aria-label={t("drawing.settings")}
          >
            {selectedDrawingModel ? (
              <>
                <ModelAvatar
                  model={selectedDrawingModel}
                  size={24}
                  className="shrink-0"
                />
                <span className="truncate text-sm font-medium">
                  {selectedDrawingModel.name || selectedDrawingModel.id}
                </span>
              </>
            ) : (
              <span className="truncate text-sm text-muted-foreground">
                {t("drawing.selectModel")}
              </span>
            )}
          </button>
          <button
            type="button"
            onClick={() => setSettingsOpen(true)}
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label={t("drawing.settings")}
            title={t("drawing.settings")}
          >
            <SlidersHorizontal className="h-4 w-4" />
          </button>
        </div>
        <Drawer open={settingsOpen} onOpenChange={setSettingsOpen}>
          <DrawerContent className="max-h-[88dvh]">
            <DrawerHeader className="text-left">
              <DrawerTitle>{t("drawing.settings")}</DrawerTitle>
              <DrawerDescription>
                {t("drawing.options.title")}
              </DrawerDescription>
            </DrawerHeader>
            <div className="overflow-y-auto px-4 pb-[max(1.5rem,env(safe-area-inset-bottom))]">
              {settingsPanel}
            </div>
          </DrawerContent>
        </Drawer>

        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_50%_at_50%_-20%,hsl(var(--primary)/0.06),transparent)]" />
        <div className="absolute inset-0 bg-[radial-gradient(hsl(var(--muted-foreground))_1px,transparent_1px)] [background-size:28px_28px] opacity-[0.035]" />
        {draggingReferences && (
          <div className="pointer-events-none absolute inset-4 z-30 flex items-center justify-center rounded-3xl border-2 border-dashed border-primary/45 bg-background/80 shadow-2xl backdrop-blur-xl">
            <div className="flex flex-col items-center text-center">
              <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <Upload className="h-6 w-6" />
              </div>
              <div className="text-base font-semibold text-foreground">
                {t("drawing.dropReferenceTitle")}
              </div>
              <div className="mt-1 text-sm text-muted-foreground">
                {t("drawing.dropReferencePrompt", {
                  remaining: referencesRemaining,
                })}
              </div>
            </div>
          </div>
        )}

        {/* Mode Toggle */}
        {drawingModelCapabilities.supportsEditing && (
          <div className="absolute left-1/2 top-[4.5rem] z-20 -translate-x-1/2 lg:top-6">
            <div className="relative grid grid-cols-2 items-center rounded-full border border-border/70 bg-background/80 p-1 shadow-sm backdrop-blur-xl">
              <div
                className={cn(
                  "pointer-events-none absolute inset-y-1 left-1 w-[calc(50%-0.25rem)] rounded-full bg-foreground shadow-sm motion-safe:transition-transform motion-safe:duration-300 motion-safe:ease-out",
                  mode === "edit" && "translate-x-full",
                )}
              />
              {(["generate", "edit"] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => updateActiveWorkspace({ mode: m })}
                  className={cn(
                    "relative z-10 min-h-10 min-w-[76px] rounded-full px-5 py-2 text-sm font-medium transition-colors duration-200",
                    mode === m
                      ? "text-background"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                  aria-pressed={mode === m}
                >
                  {t(`drawing.mode.${m}`)}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Canvas / Results */}
        <div className="relative z-10 flex-1 min-h-0 overflow-y-auto">
          <div
            className={cn(
              "mx-auto min-h-full w-full max-w-6xl px-4 pb-6 sm:px-6",
              drawingModelCapabilities.supportsEditing
                ? "pt-20 lg:pt-24"
                : "pt-6",
            )}
          >
            {!cloudSyncReady ? (
              <div
                className="flex min-h-[24rem] flex-col items-center justify-center gap-3 text-sm text-muted-foreground"
                role="status"
                aria-live="polite"
              >
                <Loader2 className="h-5 w-5 animate-spin motion-reduce:animate-none" />
                {t("drawing.loadingWorkspace")}
              </div>
            ) : (
              <>
                {cloudSyncError && (
                  <div
                    className="mb-4 rounded-xl border border-amber-500/30 bg-amber-500/8 px-3 py-2.5 text-sm text-foreground"
                    role="status"
                  >
                    <span className="font-medium">
                      {t("drawing.syncUnavailable")}
                    </span>
                    <span className="ml-2 text-muted-foreground">
                      {cloudSyncError}
                    </span>
                  </div>
                )}

                {activeWorkspacePending && (
                  <div
                    className="mb-4 rounded-xl border border-border/60 bg-card p-4"
                    role="status"
                    aria-live="polite"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                        <Loader2 className="h-5 w-5 animate-spin motion-reduce:animate-none" />
                      </div>
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-foreground">
                          {activeWorkspace?.taskStatus === "queued"
                            ? t("drawing.queuedTitle")
                            : t("drawing.generatingTitle")}
                        </div>
                        <div className="mt-1 truncate text-xs text-muted-foreground">
                          {activeWorkspace?.lastPrompt || prompt}
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {terminalTaskStatus && (
                  <div
                    className={cn(
                      "mb-4 flex flex-col gap-3 rounded-xl border p-4 sm:flex-row sm:items-center sm:justify-between",
                      activeWorkspace?.taskStatus === "failed"
                        ? "border-destructive/30 bg-destructive/5"
                        : "border-border bg-muted/35",
                    )}
                    role={
                      activeWorkspace?.taskStatus === "failed"
                        ? "alert"
                        : "status"
                    }
                  >
                    <div className="min-w-0">
                      <div
                        className={cn(
                          "text-sm font-medium",
                          activeWorkspace?.taskStatus === "failed"
                            ? "text-destructive"
                            : "text-foreground",
                        )}
                      >
                        {activeWorkspace?.taskStatus === "failed"
                          ? t("drawing.generateFailed")
                          : t("drawing.canceledTitle")}
                      </div>
                      {activeWorkspace?.taskError && (
                        <div className="mt-1 break-words text-xs text-muted-foreground">
                          {activeWorkspace.taskError}
                        </div>
                      )}
                    </div>
                    <button
                      type="button"
                      onClick={() => void generateImage()}
                      disabled={!canGenerate}
                      className="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition-opacity hover:opacity-85 disabled:cursor-not-allowed disabled:opacity-40"
                    >
                      <RotateCcw className="h-4 w-4" />
                      {t("drawing.retry")}
                    </button>
                  </div>
                )}

                {drawingModels.length === 0 ? (
                  <div className="flex min-h-[28rem] flex-col items-center justify-center text-center">
                    <div className="mb-5 flex h-16 w-16 items-center justify-center rounded-2xl border border-border/70 bg-background">
                      <ImageIcon className="h-7 w-7 text-muted-foreground" />
                    </div>
                    <p className="text-base font-semibold text-foreground">
                      {t("drawing.noModels")}
                    </p>
                    <p className="mt-3 max-w-sm text-sm leading-relaxed text-muted-foreground">
                      {t("drawing.noModelsPrompt")}
                    </p>
                    <a
                      href="/model"
                      className="mt-5 inline-flex h-10 items-center rounded-xl bg-foreground px-4 text-sm font-medium text-background transition-opacity hover:opacity-85"
                    >
                      {t("drawing.goSettings")}
                    </a>
                  </div>
                ) : generatedImages.length === 0 &&
                  !activeWorkspacePending &&
                  !terminalTaskStatus ? (
                  <div className="flex min-h-[28rem] flex-col items-center justify-center px-4 text-center">
                    <div className="mb-5 flex h-16 w-16 items-center justify-center rounded-2xl border border-border/70 bg-background">
                      <Sparkles className="h-7 w-7 text-muted-foreground" />
                    </div>
                    <p className="text-base font-semibold text-foreground">
                      {t("drawing.emptyTitle")}
                    </p>
                    <p className="mt-3 max-w-md text-sm text-muted-foreground">
                      {t("drawing.emptyPrompt")}
                    </p>
                  </div>
                ) : generatedImages.length > 0 ? (
                  <div>
                    <div className="mb-4 flex items-center justify-between gap-3">
                      <div>
                        <div className="text-sm font-semibold text-foreground">
                          {t("drawing.resultsTitle")}
                        </div>
                        <div className="mt-0.5 text-xs text-muted-foreground">
                          {t("drawing.resultsCount", {
                            count: generatedImages.length,
                          })}
                        </div>
                      </div>
                      <button
                        type="button"
                        onClick={clearGeneratedImages}
                        className="inline-flex h-10 items-center gap-1.5 rounded-xl px-3 text-xs font-medium text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                        {t("drawing.clearResults")}
                      </button>
                    </div>
                    <div className="grid grid-cols-1 items-start gap-4 min-[480px]:grid-cols-2 xl:grid-cols-3">
                      {generatedImages.map((image, index) => {
                        const extension = getDrawingImageExtension(image.src);
                        return (
                          <article
                            key={image.id}
                            className="group relative overflow-hidden rounded-xl border border-border/60 bg-card"
                          >
                            <button
                              type="button"
                              onClick={() => setPreviewImage(image)}
                              className="relative block w-full overflow-hidden bg-muted/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
                              aria-label={t("drawing.generatedImage")}
                            >
                              <img
                                src={image.src}
                                alt={t("drawing.generatedImage")}
                                loading="lazy"
                                className="block h-auto max-h-[34rem] w-full object-contain motion-safe:transition-transform motion-safe:duration-300 group-hover:scale-[1.01]"
                              />
                              <span className="absolute bottom-2 left-2 flex h-10 w-10 items-center justify-center rounded-xl bg-background/90 text-muted-foreground opacity-100 sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100">
                                <ZoomIn className="h-4 w-4" />
                              </span>
                            </button>
                            <div className="absolute right-2 top-2 flex gap-1.5 opacity-100 sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100 sm:group-focus-within:opacity-100">
                              {canAcceptReferences && (
                                <button
                                  type="button"
                                  onClick={() =>
                                    addGeneratedImageAsReference(image)
                                  }
                                  aria-label={t("drawing.useAsReference")}
                                  title={t("drawing.useAsReference")}
                                  className="flex h-10 w-10 items-center justify-center rounded-xl bg-background/95 text-muted-foreground shadow-sm transition-colors hover:text-foreground"
                                >
                                  <ImagePlus className="h-4 w-4" />
                                </button>
                              )}
                              <a
                                href={image.src}
                                download={`drawing-${index + 1}.${extension}`}
                                target="_blank"
                                rel="noreferrer"
                                aria-label={t("drawing.downloadImage")}
                                title={t("drawing.downloadImage")}
                                className="flex h-10 w-10 items-center justify-center rounded-xl bg-background/95 text-muted-foreground shadow-sm transition-colors hover:text-foreground"
                              >
                                <Download className="h-4 w-4" />
                              </a>
                              <button
                                type="button"
                                onClick={() => removeGeneratedImage(image.id)}
                                aria-label={t("remove")}
                                title={t("remove")}
                                className="flex h-10 w-10 items-center justify-center rounded-xl bg-background/95 text-muted-foreground shadow-sm transition-colors hover:bg-destructive hover:text-destructive-foreground"
                              >
                                <Trash2 className="h-4 w-4" />
                              </button>
                            </div>
                            {image.prompt && (
                              <p className="line-clamp-2 px-3 py-2.5 text-xs leading-relaxed text-muted-foreground">
                                {image.prompt}
                              </p>
                            )}
                          </article>
                        );
                      })}
                    </div>
                  </div>
                ) : null}
              </>
            )}
          </div>
        </div>

        {/* Composer */}
        <div className="relative z-20 flex shrink-0 justify-center px-3 pb-3 sm:px-6 sm:pb-6">
          <div
            className={cn(
              "w-full max-w-2xl rounded-2xl border bg-background transition-colors duration-200",
              focused
                ? "border-primary/55 ring-2 ring-primary/10"
                : "border-border/70 shadow-sm",
            )}
          >
            {/* Meta row */}
            <div className="flex items-center justify-between px-4 pt-3.5 pb-0">
              <div className="flex items-center gap-2">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-foreground text-background">
                  <Wand2 className="h-3 w-3" />
                </div>
                <span
                  id="drawing-prompt-label"
                  className="text-[13px] font-semibold text-foreground"
                >
                  {t("drawing.promptLabel")}
                </span>
              </div>
              <FileProvider
                files={files}
                dispatch={dispatchFiles}
                modelId={selectedDrawingModelId}
                forceImageUpload
                maxFiles={referenceImageLimit}
                onUploadingChange={setReferenceUploadPending}
                trigger={({ disabled, filesCount, open }) =>
                  canAcceptReferences ? (
                    <button
                      type="button"
                      onClick={open}
                      disabled={disabled}
                      className={cn(
                        "relative flex h-10 w-10 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                        disabled && "cursor-not-allowed opacity-50",
                      )}
                      aria-label={uploadReferenceTitle}
                      title={uploadReferenceTitle}
                    >
                      {referenceUploadPending ? (
                        <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" />
                      ) : (
                        <Upload className="h-4 w-4" />
                      )}
                      {filesCount > 0 && (
                        <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground">
                          {filesCount}
                        </span>
                      )}
                    </button>
                  ) : null
                }
              />
            </div>

            {files.length > 0 && (
              <div className="mx-4 mt-3 flex gap-2 overflow-x-auto pb-1">
                {files.map((file, index) => {
                  const imageUrl = normalizeImageURL(file.content);
                  return (
                    <div
                      key={`${file.name}-${index}`}
                      className="group relative h-16 w-16 shrink-0 overflow-hidden rounded-xl border border-border/70 bg-muted/30"
                      title={file.name}
                    >
                      <img
                        src={imageUrl}
                        alt={file.name}
                        className="h-full w-full object-cover"
                        loading="lazy"
                      />
                      <button
                        type="button"
                        onClick={() => removeReferenceFile(index)}
                        aria-label={t("remove")}
                        className="absolute right-1 top-1 flex h-8 w-8 items-center justify-center rounded-xl bg-background/95 text-muted-foreground opacity-100 shadow-sm transition-colors hover:bg-destructive hover:text-destructive-foreground sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100 sm:focus-visible:opacity-100"
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Textarea */}
            <Textarea
              id="drawing-prompt"
              aria-labelledby="drawing-prompt-label"
              value={prompt}
              onChange={(e) =>
                updateActiveWorkspace({ prompt: e.target.value })
              }
              onKeyDown={(event) => {
                if (
                  event.key !== "Enter" ||
                  (!event.metaKey && !event.ctrlKey)
                ) {
                  return;
                }
                event.preventDefault();
                void generateImage();
              }}
              onFocus={() => setFocused(true)}
              onBlur={() => setFocused(false)}
              className="min-h-[76px] w-full resize-none border-0 bg-transparent px-4 py-3 text-sm leading-relaxed text-foreground shadow-none placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0"
              placeholder={t("drawing.promptPlaceholder")}
            />

            {(referenceUploadPending ||
              (mode === "edit" && files.length === 0)) && (
              <div
                className="px-4 pb-2 text-xs text-muted-foreground"
                role="status"
              >
                {referenceUploadPending
                  ? t("drawing.uploadProcessing")
                  : t("drawing.editRequiresReference")}
              </div>
            )}

            {/* Toolbar */}
            <div className="flex items-center justify-between px-2.5 pb-2.5 gap-2">
              <div className="flex min-w-0 items-center gap-1.5 overflow-hidden">
                {drawingModelCapabilities.aspectRatios.length > 0 && (
                  <span className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-muted/45 px-2.5 text-xs font-medium text-muted-foreground">
                    <Ratio className="h-3.5 w-3.5" />
                    {options.aspectRatio}
                  </span>
                )}
                {drawingModelCapabilities.imageSizes.length > 0 && (
                  <span className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-muted/45 px-2.5 text-xs font-medium text-muted-foreground">
                    <ImageIcon className="h-3.5 w-3.5" />
                    {options.imageSize}
                  </span>
                )}
                {canAcceptReferences && (
                  <span className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-muted/45 px-2.5 text-xs font-medium text-muted-foreground">
                    <Upload className="h-3.5 w-3.5" />
                    {files.length}/{referenceImageLimit}
                  </span>
                )}
              </div>

              <button
                onClick={() =>
                  requestInFlight
                    ? void cancelActiveTask()
                    : void generateImage()
                }
                disabled={!requestInFlight && !canGenerate}
                className={cn(
                  "flex h-11 w-11 shrink-0 select-none items-center justify-center rounded-full transition-all duration-150",
                  requestInFlight
                    ? "bg-destructive text-destructive-foreground hover:opacity-85 active:scale-[0.96] shadow-sm"
                    : canGenerate
                      ? "bg-foreground text-background hover:opacity-85 active:scale-[0.96] shadow-sm"
                      : "bg-muted/60 text-muted-foreground/40 cursor-not-allowed",
                )}
                aria-label={
                  requestInFlight ? t("cancel") : t("drawing.generateImage")
                }
                title={
                  requestInFlight
                    ? t("cancel")
                    : canGenerate
                      ? t("drawing.generateImage")
                      : generateDisabledReason
                }
              >
                {requestInFlight ? (
                  <X className="h-4 w-4" />
                ) : (
                  <ArrowUp className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>
        </div>
      </main>

      {renderWorkspaceRail(false)}
      {renderWorkspaceRail(true)}

      <Dialog
        open={Boolean(previewImage)}
        onOpenChange={(open) => {
          if (!open) setPreviewImage(null);
        }}
      >
        <DialogContent className="max-h-[94dvh] max-w-[96vw] overflow-hidden p-3 sm:p-4 md:max-w-5xl">
          <DialogHeader className="sr-only">
            <DialogTitle>{t("drawing.generatedImage")}</DialogTitle>
            <DialogDescription>
              {previewImage?.prompt || t("drawing.generatedImage")}
            </DialogDescription>
          </DialogHeader>
          {previewImage && (
            <div className="flex min-h-0 flex-col gap-3">
              <div className="flex min-h-0 flex-1 items-center justify-center overflow-auto rounded-xl bg-muted/25">
                <img
                  src={previewImage.src}
                  alt={t("drawing.generatedImage")}
                  className="max-h-[78dvh] max-w-full object-contain"
                />
              </div>
              <div className="flex items-center justify-between gap-3 px-1">
                <p className="line-clamp-2 min-w-0 text-xs text-muted-foreground">
                  {previewImage.prompt}
                </p>
                <a
                  href={previewImage.src}
                  download={`drawing.${getDrawingImageExtension(previewImage.src)}`}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex h-10 shrink-0 items-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition-opacity hover:opacity-85"
                >
                  <Download className="h-4 w-4" />
                  {t("drawing.downloadImage")}
                </a>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default Drawing;
