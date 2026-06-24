import {
  type ComponentType,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useReducer,
  useState,
} from "react";
import {
  Wand2,
  Plus,
  SlidersHorizontal,
  Palette,
  Ratio,
  Upload,
  ArrowUp,
  Brain,
  FileType2,
  Image as ImageIcon,
  Loader2,
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
import { selectSupportModels, useMessageActions } from "@/store/chat.ts";
import { isDrawingModel } from "@/conf/model.ts";
import ModelAvatar from "@/components/ModelAvatar.tsx";
import FileProvider, {
  type FileProviderAction,
} from "@/components/FileProvider.tsx";
import type { FileArray } from "@/api/file.ts";
import { formatMessage } from "@/utils/processor.ts";
import { toast } from "sonner";
import { blobEvent, filePanelEvent } from "@/events/blob.ts";

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
  thinkingLevels: readonly GeminiImageThinkingLevel[];
  maxReferenceImages: number;
};

type DrawingWorkspace = {
  id: string;
  model: string;
  mode: Mode;
  prompt: string;
  options: DrawingOptions;
  createdAt: number;
  accent: number;
};

const DRAWING_WORKSPACES_KEY = "drawing.workspaces.v1";
const DRAWING_ACTIVE_WORKSPACE_KEY = "drawing.activeWorkspaceId.v1";
const DRAWING_MODEL_QUERY_PARAM = "model";

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
const GEMINI_3_PRO_IMAGE_ASPECT_RATIOS =
  GEMINI_25_FLASH_IMAGE_ASPECT_RATIOS;
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
const GEMINI_3_PRO_IMAGE_SIZES: readonly GeminiImageSize[] = [
  "1K",
  "2K",
  "4K",
];
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
  thinkingLevels: [],
  maxReferenceImages: GEMINI_25_FLASH_IMAGE_REFERENCE_LIMIT,
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

function getDrawingModelCapabilities(modelId: string): DrawingModelCapabilities {
  const normalizedModelId = modelId.trim().toLowerCase();

  if (normalizedModelId.includes("gemini-3.1-flash-image")) {
    return {
      aspectRatios: GEMINI_31_FLASH_IMAGE_ASPECT_RATIOS,
      imageSizes: GEMINI_31_FLASH_IMAGE_SIZES,
      thinkingLevels: GEMINI_31_FLASH_THINKING_LEVELS,
      maxReferenceImages: GEMINI_31_FLASH_IMAGE_REFERENCE_LIMIT,
    };
  }

  if (normalizedModelId.includes("gemini-3-pro-image")) {
    return {
      aspectRatios: GEMINI_3_PRO_IMAGE_ASPECT_RATIOS,
      imageSizes: GEMINI_3_PRO_IMAGE_SIZES,
      thinkingLevels: [],
      maxReferenceImages: GEMINI_3_PRO_IMAGE_REFERENCE_LIMIT,
    };
  }

  if (normalizedModelId.includes("gemini-2.5-flash-image")) {
    return DEFAULT_DRAWING_MODEL_CAPABILITIES;
  }

  return DEFAULT_DRAWING_MODEL_CAPABILITIES;
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
    mimeType: hasStringOption(MIME_TYPE_OPTIONS, mimeType)
      ? mimeType
      : DEFAULT_DRAWING_OPTIONS.mimeType,
    thinkingLevel: hasStringOption(
      capabilities.thinkingLevels,
      thinkingLevel,
    )
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
    mime_type: GeminiImageMimeType;
    aspect_ratio: GeminiImageAspectRatio;
    image_size?: GeminiImageSize;
  } = {
    type: "image",
    mime_type: options.mimeType,
    aspect_ratio: options.aspectRatio,
  };

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

    const workspaces = parsed
      .map((item, index): DrawingWorkspace | null => {
        if (!item || typeof item !== "object") {
          return null;
        }

        const workspace = item as Partial<DrawingWorkspace>;

        return {
          id:
            typeof workspace.id === "string" && workspace.id
              ? workspace.id
              : createWorkspaceId(),
          model: typeof workspace.model === "string" ? workspace.model : "",
          mode: workspace.mode === "edit" ? "edit" : "generate",
          prompt: typeof workspace.prompt === "string" ? workspace.prompt : "",
          options: normalizeDrawingOptions(workspace.options),
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
        <SelectTrigger className="h-9 w-full border-border/60 bg-background/60 px-2.5 text-xs">
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
  const { send } = useMessageActions();
  const supportModels = useSelector(selectSupportModels);
  const [requestedDrawingModelId] = useState(getRequestedDrawingModelId);
  const handledRequestedDrawingModel = useRef(false);
  const [files, dispatchFiles] = useReducer(drawingFileReducer, []);
  const [workspaces, setWorkspaces] = useState<DrawingWorkspace[]>(() =>
    loadDrawingWorkspaces(),
  );
  const [activeWorkspaceId, setActiveWorkspaceId] = useState(() =>
    loadActiveWorkspaceId(),
  );
  const [focused, setFocused] = useState(false);
  const [generating, setGenerating] = useState(false);
  const activeWorkspace =
    workspaces.find((workspace) => workspace.id === activeWorkspaceId) ??
    workspaces[0];
  const activeWorkspaceIdForStorage = activeWorkspace?.id ?? "";
  const drawingModels = useMemo(
    () => supportModels.filter((model) => isDrawingModel(model)),
    [supportModels],
  );
  const selectedDrawingModel =
    drawingModels.find((model) => model.id === activeWorkspace?.model) ??
    drawingModels[0];
  const selectedDrawingModelId = selectedDrawingModel?.id ?? "";
  const drawingModelCapabilities = useMemo(
    () => getDrawingModelCapabilities(selectedDrawingModelId),
    [selectedDrawingModelId],
  );
  const mode = activeWorkspace?.mode ?? "generate";
  const prompt = activeWorkspace?.prompt ?? "";
  const rawOptions = activeWorkspace?.options;
  const options = useMemo(
    () => normalizeDrawingOptions(rawOptions, drawingModelCapabilities),
    [drawingModelCapabilities, rawOptions],
  );
  const referenceImageLimit = drawingModelCapabilities.maxReferenceImages;
  const uploadReferenceTitle = t("drawing.uploadReferenceWithLimit", {
    limit: referenceImageLimit,
  });

  useEffect(() => {
    const openReferencePanel = (event: DragEvent) => {
      if (!hasDraggedFiles(event.dataTransfer)) return false;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
      filePanelEvent.emit(undefined);
      return true;
    };

    const handleDragEnter = (event: DragEvent) => {
      openReferencePanel(event);
    };

    const handleDragOver = (event: DragEvent) => {
      openReferencePanel(event);
    };

    const handleDrop = (event: DragEvent) => {
      if (event.defaultPrevented || !hasDraggedFiles(event.dataTransfer)) {
        return;
      }
      event.preventDefault();

      const droppedFiles = getDroppedFiles(event.dataTransfer);
      if (droppedFiles.length > 0) {
        blobEvent.emit(droppedFiles);
      }
    };

    window.addEventListener("dragenter", handleDragEnter);
    window.addEventListener("dragover", handleDragOver);
    window.addEventListener("drop", handleDrop);

    return () => {
      window.removeEventListener("dragenter", handleDragEnter);
      window.removeEventListener("dragover", handleDragOver);
      window.removeEventListener("drop", handleDrop);
    };
  }, []);

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

    const capabilities = getDrawingModelCapabilities(requestedModel.id);
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

    window.localStorage.setItem(
      DRAWING_WORKSPACES_KEY,
      JSON.stringify(workspaces),
    );
  }, [workspaces]);

  useEffect(() => {
    if (typeof window === "undefined" || !activeWorkspaceIdForStorage) {
      return;
    }

    window.localStorage.setItem(
      DRAWING_ACTIVE_WORKSPACE_KEY,
      activeWorkspaceIdForStorage,
    );
  }, [activeWorkspaceIdForStorage]);

  const updateActiveWorkspace = useCallback(
    (
      updates: Partial<
        Pick<DrawingWorkspace, "model" | "mode" | "prompt" | "options">
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

  const canUseCurrentReferencesWithModel = useCallback(
    (modelId: string) =>
      files.length <= getDrawingModelCapabilities(modelId).maxReferenceImages,
    [files.length],
  );

  const notifyReferenceImageLimit = useCallback(
    (modelId: string) => {
      const capabilities = getDrawingModelCapabilities(modelId);
      const modelName =
        drawingModels.find((model) => model.id === modelId)?.name || modelId;

      toast.error(t("drawing.referenceLimitExceeded"), {
        description: t("drawing.referenceLimitExceededPrompt", {
          count: files.length,
          limit: capabilities.maxReferenceImages,
          model: modelName,
        }),
      });
    },
    [drawingModels, files.length, t],
  );

  const selectDrawingModel = useCallback(
    (modelId: string) => {
      if (!canUseCurrentReferencesWithModel(modelId)) {
        notifyReferenceImageLimit(modelId);
        return;
      }

      updateActiveWorkspace({
        model: modelId,
        options: normalizeDrawingOptions(
          options,
          getDrawingModelCapabilities(modelId),
        ),
      });
    },
    [
      canUseCurrentReferencesWithModel,
      notifyReferenceImageLimit,
      options,
      updateActiveWorkspace,
    ],
  );

  const selectWorkspace = useCallback(
    (workspace: DrawingWorkspace) => {
      const targetModelId = workspace.model || drawingModels[0]?.id || "";
      if (
        targetModelId &&
        !canUseCurrentReferencesWithModel(targetModelId)
      ) {
        notifyReferenceImageLimit(targetModelId);
        return;
      }

      setActiveWorkspaceId(workspace.id);
    },
    [canUseCurrentReferencesWithModel, drawingModels, notifyReferenceImageLimit],
  );

  const addWorkspace = () => {
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
      const nextActiveModelId =
        nextActiveWorkspace?.model || drawingModels[0]?.id || "";

      if (
        nextActiveModelId &&
        !canUseCurrentReferencesWithModel(nextActiveModelId)
      ) {
        notifyReferenceImageLimit(nextActiveModelId);
        return;
      }

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

  const generateImage = async () => {
    const text = prompt.trim();
    if (!text || !selectedDrawingModelId || generating) {
      return;
    }
    if (!canUseCurrentReferencesWithModel(selectedDrawingModelId)) {
      notifyReferenceImageLimit(selectedDrawingModelId);
      return;
    }

    setGenerating(true);
    try {
      if (
        await send(
          formatMessage(files, text),
          selectedDrawingModelId,
          buildDrawingRequestOptions(options, drawingModelCapabilities),
        )
      ) {
        dispatchFiles({ type: "clear" });
      }
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div className="flex h-full min-h-0 w-full bg-background text-foreground overflow-hidden">
      {/* Left Sidebar */}
      <aside className="w-72 min-h-0 flex flex-col z-10 shrink-0 border-r border-border/60 bg-card/50 backdrop-blur-sm">
        <div className="p-5 flex-1 flex flex-col gap-5 overflow-y-auto">
          {/* Provider */}
          <div className="space-y-2.5">
            <div className="flex items-center justify-between">
              <label className="text-xs font-semibold tracking-widest text-muted-foreground uppercase">
                {t("drawing.model")}
              </label>
            </div>
            <Select
              value={selectedDrawingModelId || undefined}
              onValueChange={selectDrawingModel}
              disabled={drawingModels.length === 0}
            >
              <SelectTrigger className="w-full h-10 text-sm border-border/60 bg-background/60">
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
                      <ModelAvatar
                        model={model}
                        size={22}
                        className="shrink-0"
                      />
                      <span className="truncate">{model.name || model.id}</span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {drawingModels.length === 0 && (
              <p className="text-xs leading-relaxed text-muted-foreground/70">
                {t("drawing.noModels")}
              </p>
            )}
            {drawingModels.length > 0 && (
              <div className="space-y-3 border-t border-border/60 pt-4">
                <div className="text-xs font-semibold tracking-widest text-muted-foreground uppercase">
                  {t("drawing.options.title")}
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <DrawingOptionSelect
                    icon={Ratio}
                    label={t("drawing.options.aspectRatio")}
                    value={options.aspectRatio}
                    options={drawingModelCapabilities.aspectRatios}
                    onChange={(aspectRatio) =>
                      updateDrawingOptions({ aspectRatio })
                    }
                  />
                  {drawingModelCapabilities.imageSizes.length > 0 && (
                    <DrawingOptionSelect
                      icon={ImageIcon}
                      label={t("drawing.options.imageSize")}
                      value={options.imageSize}
                      options={drawingModelCapabilities.imageSizes}
                      onChange={(imageSize) =>
                        updateDrawingOptions({ imageSize })
                      }
                    />
                  )}
                  <DrawingOptionSelect
                    icon={FileType2}
                    label={t("drawing.options.mimeType")}
                    value={options.mimeType}
                    options={MIME_TYPE_OPTIONS}
                    getLabel={(value) =>
                      value === "image/jpeg" ? "JPEG" : "PNG"
                    }
                    onChange={(mimeType) => updateDrawingOptions({ mimeType })}
                  />
                  {drawingModelCapabilities.thinkingLevels.length > 0 && (
                    <DrawingOptionSelect
                      icon={Brain}
                      label={t("drawing.options.thinkingLevel")}
                      value={options.thinkingLevel}
                      options={drawingModelCapabilities.thinkingLevels}
                      getLabel={(value) =>
                        t(`drawing.options.thinking.${value}`)
                      }
                      onChange={(thinkingLevel) =>
                        updateDrawingOptions({ thinkingLevel })
                      }
                    />
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </aside>

      {/* Main Area */}
      <main className="relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_80%_50%_at_50%_-20%,hsl(var(--primary)/0.06),transparent)]" />
        <div className="absolute inset-0 bg-[radial-gradient(hsl(var(--muted-foreground))_1px,transparent_1px)] [background-size:28px_28px] opacity-[0.035]" />

        {/* Mode Toggle */}
        <div className="absolute top-6 left-1/2 -translate-x-1/2 z-20">
          <div className="relative grid grid-cols-2 items-center rounded-full border border-border/70 bg-background/80 p-1 shadow-sm backdrop-blur-xl">
            <div
              className={cn(
                "pointer-events-none absolute inset-y-1 left-1 w-[calc(50%-0.25rem)] rounded-full bg-foreground shadow-sm transition-all duration-300 ease-out",
                mode === "edit" && "translate-x-full",
              )}
            />
            {(["generate", "edit"] as const).map((m) => (
              <button
                key={m}
                onClick={() => updateActiveWorkspace({ mode: m })}
                className={cn(
                  "relative z-10 min-w-[76px] px-6 py-2 rounded-full text-sm font-medium transition-colors duration-300",
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

        {/* Empty Canvas */}
        <div className="flex-1 flex flex-col items-center justify-center pb-44 relative z-10">
          <p className="text-base font-semibold text-foreground/70 tracking-wide">
            {t("drawing.emptyTitle")}
          </p>
          <p className="text-sm text-muted-foreground/50 mt-3">
            {t("drawing.emptyPrompt")}
          </p>
        </div>

        {/* Floating Input */}
        <div className="absolute bottom-6 left-0 right-0 px-6 sm:bottom-8 sm:px-10 flex justify-center z-20 pointer-events-none">
          <div
            className={cn(
              "pointer-events-auto w-full max-w-2xl rounded-2xl transition-all duration-200",
              "border bg-background/96 backdrop-blur-2xl",
              focused
                ? "border-border shadow-[0_24px_64px_-12px_rgba(0,0,0,0.16),0_0_0_1px_rgba(0,0,0,0.02)] dark:shadow-[0_24px_64px_-12px_rgba(0,0,0,0.5)]"
                : "border-border/55 shadow-[0_8px_32px_-8px_rgba(0,0,0,0.1)] dark:shadow-[0_8px_32px_-8px_rgba(0,0,0,0.4)]",
            )}
          >
            {/* Meta row */}
            <div className="flex items-center justify-between px-4 pt-3.5 pb-0">
              <div className="flex items-center gap-2">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-foreground text-background">
                  <Wand2 className="h-3 w-3" />
                </div>
                <span className="text-[13px] font-semibold text-foreground">
                  {t("drawing.promptLabel")}
                </span>
              </div>
              <FileProvider
                files={files}
                dispatch={dispatchFiles}
                modelId={selectedDrawingModelId}
                forceImageUpload
                maxFiles={referenceImageLimit}
                trigger={({ disabled, filesCount, open }) => (
                  <button
                    type="button"
                    onClick={open}
                    disabled={disabled}
                    className={cn(
                      "relative rounded-md p-1.5 text-muted-foreground/50 transition-all duration-150 hover:bg-muted/50 hover:text-muted-foreground",
                      disabled && "cursor-not-allowed opacity-50",
                    )}
                    title={uploadReferenceTitle}
                  >
                    <Upload className="h-3.5 w-3.5" />
                    {filesCount > 0 && (
                      <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground">
                        {filesCount}
                      </span>
                    )}
                  </button>
                )}
              />
            </div>

            {/* Textarea */}
            <Textarea
              value={prompt}
              onChange={(e) =>
                updateActiveWorkspace({ prompt: e.target.value })
              }
              onFocus={() => setFocused(true)}
              onBlur={() => setFocused(false)}
              className="min-h-[84px] w-full resize-none border-0 bg-transparent px-4 py-3 text-sm leading-relaxed text-foreground shadow-none placeholder:text-muted-foreground/35 focus-visible:ring-0 focus-visible:ring-offset-0"
              placeholder={t("drawing.promptPlaceholder")}
            />

            {/* Toolbar */}
            <div className="flex items-center justify-between px-2.5 pb-2.5 gap-2">
              <div className="flex items-center">
                {[
                  { icon: Palette, key: "style" },
                  { icon: Ratio, key: "ratio" },
                ].map(({ icon: Icon, key }) => (
                  <button
                    key={key}
                    className="flex items-center gap-1.5 h-8 px-2.5 rounded-lg text-xs text-muted-foreground/70 hover:text-foreground hover:bg-muted/50 transition-all duration-150 font-medium"
                  >
                    <Icon className="h-3.5 w-3.5" />
                    {t(`drawing.tools.${key}`)}
                  </button>
                ))}
                <div className="mx-1 h-4 w-px bg-border/60" />
                <button
                  className="h-8 w-8 rounded-lg flex items-center justify-center text-muted-foreground/50 hover:text-muted-foreground hover:bg-muted/50 transition-all duration-150"
                  title={t("drawing.advanced")}
                >
                  <SlidersHorizontal className="h-3.5 w-3.5" />
                </button>
              </div>

              <button
                onClick={generateImage}
                disabled={!prompt.trim() || !selectedDrawingModelId || generating}
                className={cn(
                  "flex h-9 w-9 items-center justify-center rounded-full transition-all duration-150 shrink-0 select-none",
                  prompt.trim() && selectedDrawingModelId && !generating
                    ? "bg-foreground text-background hover:opacity-85 active:scale-[0.96] shadow-sm"
                    : "bg-muted/60 text-muted-foreground/40 cursor-not-allowed",
                )}
                aria-label={t("drawing.generateImage")}
                title={t("drawing.generateImage")}
              >
                {generating ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <ArrowUp className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>
        </div>
      </main>

      {/* Right Sidebar - History */}
      <aside className="w-[72px] min-h-0 bg-card/50 border-l border-border/60 flex flex-col z-10 shrink-0 backdrop-blur-sm">
        <div className="flex-1 overflow-y-auto p-3 flex flex-col gap-3 items-center no-scrollbar pt-4">
          <button
            type="button"
            onClick={addWorkspace}
            className="w-12 h-12 border-2 border-dashed border-border/60 rounded-2xl flex items-center justify-center text-muted-foreground/60 hover:border-primary/40 hover:text-primary/60 hover:bg-primary/5 transition-all duration-200 group"
            aria-label={t("drawing.addWorkspace")}
            title={t("drawing.addWorkspace")}
          >
            <Plus className="w-4 h-4 group-hover:rotate-90 transition-transform duration-300" />
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

            return (
              <div
                key={workspace.id}
                className="group relative h-12 w-12 shrink-0"
              >
                <button
                  type="button"
                  onClick={() => selectWorkspace(workspace)}
                  className={cn(
                    "relative flex h-full w-full items-center justify-center overflow-hidden rounded-2xl bg-gradient-to-br transition-all duration-200",
                    selected
                      ? "border-2 border-primary/60 shadow-sm ring-2 ring-primary/10"
                      : "border border-border/50 hover:border-border hover:bg-muted/60",
                    selected ? accent.active : accent.idle,
                  )}
                  aria-current={selected ? "true" : undefined}
                  aria-label={label}
                  title={label}
                >
                  <span
                    className={cn(
                      "relative z-10 text-[11px] font-semibold transition-colors",
                      selected
                        ? "text-foreground/70"
                        : "text-muted-foreground/45",
                    )}
                  >
                    {index + 1}
                  </span>
                  {workspace.prompt.trim() && (
                    <span
                      className={cn(
                        "absolute bottom-1.5 h-1 w-4 rounded-full",
                        selected
                          ? "bg-foreground/40"
                          : "bg-muted-foreground/25",
                      )}
                    />
                  )}
                </button>
                {workspaces.length > 1 && (
                  <button
                    type="button"
                    onClick={(event) => {
                      event.stopPropagation();
                      deleteWorkspace(workspace.id);
                    }}
                    className="pointer-events-none absolute -right-1 -top-1 z-20 flex h-5 w-5 items-center justify-center rounded-full border border-border/70 bg-background/95 text-muted-foreground opacity-0 shadow-sm transition-all duration-150 hover:border-destructive/50 hover:bg-destructive hover:text-destructive-foreground group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100"
                    aria-label={deleteLabel}
                    title={deleteLabel}
                  >
                    <X className="h-3 w-3" />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </aside>
    </div>
  );
}

export default Drawing;
