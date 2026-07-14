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
  Keyboard,
  RotateCcw,
  SlidersHorizontal,
  X,
  Coins,
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
  acknowledgeDrawingTask,
  cancelDrawingTask,
  createDrawingTask,
  getDrawingTask,
  listDrawingTasks,
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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog.tsx";
import {
  DRAWING_TASK_POLL_INTERVAL_MS,
  MAX_DRAWING_WORKSPACES,
  applyDrawingTaskToWorkspaces,
  buildDrawingRequestOptions,
  clearRequestedDrawingModelId,
  createDrawingWorkspace,
  drawingFileReducer,
  getClipboardImageFiles,
  getDrawingImageExtension,
  getDrawingModelCapabilities,
  getDroppedFiles,
  getRequestedDrawingModelId,
  hasDraggedFiles,
  isActiveDrawingTask,
  loadActiveWorkspaceId,
  loadDrawingWorkspaces,
  normalizeDrawingOptions,
  normalizeDrawingReferences,
  normalizeDrawingWorkspaces,
  type DrawingGeneratedImage,
  type DrawingModel,
  type DrawingOptions,
  type DrawingWorkspace,
} from "@/routes/drawing/domain.ts";
import {
  loadDrawingLocalBootstrap,
  loadDrawingLocalState,
  saveDrawingLocalBootstrap,
  saveDrawingLocalState,
} from "@/routes/drawing/storage.ts";
import {
  imageBilling,
  imageBillingModeOfficialUsage,
  nonBilling,
} from "@/admin/charge.ts";
import { estimateImageQuota } from "@/admin/image-charge.ts";
import { formatDecimal } from "@/utils/base.ts";

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

type DrawingOptionSelectProps<T extends string> = {
  icon: ComponentType<{ className?: string }>;
  label: string;
  value: T;
  options: readonly T[];
  getLabel?: (value: T) => string;
  onChange: (value: T) => void;
};

type GeneratedImageMetadata = {
  model: string;
  aspectRatio: string;
  imageSize: string;
  mimeType: string;
  thinkingLevel: string;
};

function getGeneratedImageMetadata(
  image: DrawingGeneratedImage,
  fallbackModel: string,
  fallbackOptions: DrawingOptions,
): GeneratedImageMetadata {
  const responseFormat = image.options?.response_format;
  return {
    model: image.model || fallbackModel,
    aspectRatio: responseFormat?.aspect_ratio || fallbackOptions.aspectRatio,
    imageSize: responseFormat?.image_size || fallbackOptions.imageSize,
    mimeType: responseFormat?.mime_type || fallbackOptions.mimeType,
    thinkingLevel:
      image.options?.thinking?.thinking_level || fallbackOptions.thinkingLevel,
  };
}

function formatGeneratedImageDate(timestamp: number, locale: string) {
  try {
    return new Intl.DateTimeFormat(locale, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(timestamp));
  } catch {
    return new Date(timestamp).toLocaleString();
  }
}

function readBlobAsDataURL(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ""));
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(blob);
  });
}

async function localizeDrawingImageSource(source: string): Promise<string> {
  if (source.startsWith("data:image/")) return source;

  const response = await fetch(normalizeImageURL(source), {
    credentials: "same-origin",
  });
  if (!response.ok) {
    throw new Error(`Failed to cache drawing image: ${response.status}`);
  }
  return readBlobAsDataURL(await response.blob());
}

async function localizeDrawingWorkspaceAssets(
  workspaces: DrawingWorkspace[],
): Promise<DrawingWorkspace[]> {
  return Promise.all(
    workspaces.map(async (workspace) => ({
      ...workspace,
      references: await Promise.all(
        workspace.references.map(async (reference) => {
          try {
            return {
              ...reference,
              content: await localizeDrawingImageSource(reference.content),
            };
          } catch (error) {
            console.debug("[drawing] failed to migrate local reference", error);
            return reference;
          }
        }),
      ),
      images: await Promise.all(
        workspace.images.map(async (image) => {
          try {
            return {
              ...image,
              src: await localizeDrawingImageSource(image.src),
            };
          } catch (error) {
            console.debug("[drawing] failed to migrate local result", error);
            return image;
          }
        }),
      ),
    })),
  );
}

async function localizeDrawingTaskImages(
  task: DrawingTask<DrawingGeneratedImage>,
): Promise<DrawingTask<DrawingGeneratedImage>> {
  if (task.status !== "succeeded" || !task.images?.length) return task;

  const images = await Promise.all(
    task.images.map(async (image) => {
      return {
        ...image,
        src: await localizeDrawingImageSource(image.src),
      };
    }),
  );

  return { ...task, images };
}

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
  const { t, i18n } = useTranslation();
  const supportModels = useSelector(selectSupportModels);
  const [requestedDrawingModelId] = useState(getRequestedDrawingModelId);
  const handledRequestedDrawingModel = useRef(false);
  const dragDepthRef = useRef(0);
  const pendingTaskAcksRef = useRef(new Set<string>());
  const [initialLocalState] = useState(() => {
    const bootstrap = loadDrawingLocalBootstrap();
    return {
      state: bootstrap ?? {
        workspaces: loadDrawingWorkspaces(),
        activeWorkspaceId: loadActiveWorkspaceId(),
      },
      hasBootstrap: bootstrap !== null,
    };
  });
  const [workspaces, setWorkspaces] = useState<DrawingWorkspace[]>(
    initialLocalState.state.workspaces,
  );
  const [activeWorkspaceId, setActiveWorkspaceId] = useState(
    initialLocalState.state.activeWorkspaceId,
  );
  const legacyLocalStateRef = useRef(initialLocalState.state);
  const [localStateReady, setLocalStateReady] = useState(false);
  const settingsStateReady = initialLocalState.hasBootstrap || localStateReady;
  const [focused, setFocused] = useState(false);
  const [draggingReferences, setDraggingReferences] = useState(false);
  const [referenceUploadPending, setReferenceUploadPending] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const [previewImage, setPreviewImage] =
    useState<DrawingGeneratedImage | null>(null);
  const [workspaceToDelete, setWorkspaceToDelete] = useState("");
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
  const drawingQuotaEstimate = useMemo(() => {
    const price = selectedDrawingModel?.price;
    if (!price || price.type === nonBilling) {
      return price?.type === nonBilling ? 0 : null;
    }

    return estimateImageQuota(
      price,
      {
        size:
          drawingModelCapabilities.imageSizes.length > 0
            ? options.imageSize
            : "",
        mimeType:
          drawingModelCapabilities.mimeTypes.length > 0
            ? options.mimeType
            : "",
        aspectRatio:
          drawingModelCapabilities.aspectRatios.length > 0
            ? options.aspectRatio
            : "",
      },
      files.length,
    );
  }, [
    drawingModelCapabilities.aspectRatios.length,
    drawingModelCapabilities.imageSizes.length,
    drawingModelCapabilities.mimeTypes.length,
    files.length,
    options.aspectRatio,
    options.imageSize,
    options.mimeType,
    selectedDrawingModel,
  ]);
  const drawingQuotaEstimateState = useMemo(() => {
    const price = selectedDrawingModel?.price;
    if (!price || price.type === nonBilling) {
      return price?.type === nonBilling ? "free" : null;
    }
    if (drawingQuotaEstimate !== null) {
      return "value";
    }
    if (
      price.type === imageBilling &&
      price.image?.mode === imageBillingModeOfficialUsage
    ) {
      return "usage";
    }
    return "unavailable";
  }, [drawingQuotaEstimate, selectedDrawingModel]);
  const uploadReferenceTitle = t("drawing.uploadReferenceWithLimit", {
    limit: referenceImageLimit,
  });
  const generatedImages = activeWorkspace?.images ?? [];
  const previewMetadata = previewImage
    ? getGeneratedImageMetadata(
        previewImage,
        activeWorkspace?.model || selectedDrawingModelId,
        options,
      )
    : null;
  const previewModelLabel = previewMetadata
    ? drawingModels.find((model) => model.id === previewMetadata.model)?.name ||
      previewMetadata.model ||
      t("drawing.selectModel")
    : "";
  const activeWorkspacePending =
    Boolean(activeWorkspace?.pending) ||
    isActiveDrawingTask(
      activeWorkspace?.taskStatus
        ? { status: activeWorkspace.taskStatus }
        : undefined,
    );
  const requestInFlight = activeWorkspacePending;
  const canCancelActiveTask = Boolean(
    requestInFlight &&
    activeWorkspace?.taskId &&
    activeWorkspace.taskStatus !== "canceling",
  );
  const referencesRemaining = Math.max(0, referenceImageLimit - files.length);
  const canGenerate = Boolean(
    prompt.trim() &&
    selectedDrawingModelId &&
    localStateReady &&
    !requestInFlight &&
    !referenceUploadPending,
  );
  const generateDisabledReason = !selectedDrawingModelId
    ? t("drawing.needModel")
    : !localStateReady
      ? t("drawing.loadingWorkspace")
      : !prompt.trim()
        ? t("drawing.needPrompt")
        : referenceUploadPending
          ? t("drawing.uploadProcessing")
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

    const initializeLocalState = async () => {
      const legacyState = legacyLocalStateRef.current;
      let nextWorkspaces = normalizeDrawingWorkspaces(legacyState.workspaces);
      let nextActiveWorkspaceId = legacyState.activeWorkspaceId;

      try {
        const stored = await loadDrawingLocalState();
        if (stored?.workspaces?.length) {
          nextWorkspaces = normalizeDrawingWorkspaces(stored.workspaces);
          nextActiveWorkspaceId = stored.activeWorkspaceId;
        }
        nextWorkspaces = await localizeDrawingWorkspaceAssets(nextWorkspaces);

        const tasksResponse = await listDrawingTasks<DrawingGeneratedImage>();
        if (cancelled) return;
        if (tasksResponse.status && tasksResponse.data?.length) {
          const localizedTasks = await Promise.all(
            tasksResponse.data.map(async (task) => {
              try {
                return await localizeDrawingTaskImages(task);
              } catch (error) {
                console.debug(
                  "[drawing] failed to cache recovered task images",
                  error,
                );
                return null;
              }
            }),
          );
          nextWorkspaces = localizedTasks
            .filter(
              (task): task is DrawingTask<DrawingGeneratedImage> =>
                task !== null,
            )
            .reduce(applyDrawingTaskToWorkspaces, nextWorkspaces);
          localizedTasks.forEach((task) => {
            if (
              task &&
              (task.status === "succeeded" ||
                task.status === "failed" ||
                task.status === "canceled")
            ) {
              pendingTaskAcksRef.current.add(task.task_id);
            }
          });
        }
      } catch (error) {
        console.debug("[drawing] failed to load local drawing state", error);
      }

      if (cancelled) return;
      setWorkspaces(nextWorkspaces);
      setActiveWorkspaceId(
        nextWorkspaces.some(
          (workspace) => workspace.id === nextActiveWorkspaceId,
        )
          ? nextActiveWorkspaceId
          : nextWorkspaces[0]?.id || "",
      );
      setLocalStateReady(true);
    };

    void initializeLocalState();

    return () => {
      cancelled = true;
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
    if (!localStateReady) {
      return;
    }

    saveDrawingLocalBootstrap({
      activeWorkspaceId: activeWorkspaceIdForStorage,
      workspaces,
    });

    const timer = setTimeout(() => {
      void saveDrawingLocalState({
        activeWorkspaceId: activeWorkspaceIdForStorage,
        workspaces,
      })
        .then(async () => {
          const taskIds = [...pendingTaskAcksRef.current];
          await Promise.all(
            taskIds.map(async (taskId) => {
              if (await acknowledgeDrawingTask(taskId)) {
                pendingTaskAcksRef.current.delete(taskId);
              }
            }),
          );
        })
        .catch((error) => {
          console.debug("[drawing] failed to save local drawing state", error);
        });
    }, 350);

    return () => {
      clearTimeout(timer);
    };
  }, [activeWorkspaceIdForStorage, localStateReady, workspaces]);

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
          console.debug(
            "[drawing] failed to poll generation task",
            response.message || response.error,
          );
          continue;
        }

        try {
          const localizedTask = await localizeDrawingTaskImages(response.data);
          if (cancelled) return;
          if (
            localizedTask.status === "succeeded" ||
            localizedTask.status === "failed" ||
            localizedTask.status === "canceled"
          ) {
            pendingTaskAcksRef.current.add(localizedTask.task_id);
          }
          setWorkspaces((current) =>
            applyDrawingTaskToWorkspaces(current, localizedTask),
          );
        } catch (error) {
          console.debug("[drawing] failed to cache generated images", error);
        }
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
  }, [activeTaskKey]);

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

  const switchWorkspaceByOffset = useCallback(
    (offset: number) => {
      if (workspaces.length < 2) return;

      const currentIndex = workspaces.findIndex(
        (workspace) => workspace.id === activeWorkspaceIdForStorage,
      );
      const nextIndex =
        (Math.max(0, currentIndex) + offset + workspaces.length) %
        workspaces.length;
      const nextWorkspace = workspaces[nextIndex];
      if (nextWorkspace) setActiveWorkspaceId(nextWorkspace.id);
    },
    [activeWorkspaceIdForStorage, workspaces],
  );

  const addWorkspace = useCallback(() => {
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
  }, [selectedDrawingModelId, t, workspaces]);

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

  const requestDeleteWorkspace = (workspaceId: string) => {
    const workspace = workspaces.find((item) => item.id === workspaceId);
    if (!workspace) return;
    if (
      workspace.pending ||
      isActiveDrawingTask(
        workspace.taskStatus ? { status: workspace.taskStatus } : undefined,
      )
    ) {
      toast.info(t("drawing.generating"));
      return;
    }
    setWorkspaceToDelete(workspaceId);
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

  const generateImage = useCallback(async () => {
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
      toast.warning(t("drawing.editRequiresReference"));
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
  }, [
    activeWorkspaceIdForStorage,
    canUseReferencesWithModel,
    drawingModelCapabilities,
    files,
    mode,
    notifyReferenceImageLimit,
    options,
    prompt,
    referenceUploadPending,
    requestInFlight,
    selectedDrawingModelId,
    t,
    updateWorkspaceById,
  ]);

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

  const toggleDrawingMode = useCallback(() => {
    if (!drawingModelCapabilities.supportsEditing) return;
    updateActiveWorkspace({ mode: mode === "edit" ? "generate" : "edit" });
  }, [drawingModelCapabilities.supportsEditing, mode, updateActiveWorkspace]);

  useEffect(() => {
    const handleDrawingShortcut = (event: KeyboardEvent) => {
      if (event.defaultPrevented || event.repeat || event.isComposing) return;
      if (
        shortcutsOpen ||
        settingsOpen ||
        Boolean(previewImage) ||
        Boolean(workspaceToDelete)
      ) {
        return;
      }

      const primaryModifier = event.ctrlKey || event.metaKey;
      if (primaryModifier && !event.altKey && event.key === "Enter") {
        event.preventDefault();
        void generateImage();
        return;
      }

      const target = event.target instanceof HTMLElement ? event.target : null;
      if (
        target?.closest(
          'input, textarea, select, button, a, [contenteditable="true"], [role="button"], [role="dialog"], [role="menu"], [role="listbox"]',
        )
      ) {
        return;
      }
      if (event.altKey || event.ctrlKey || event.metaKey) return;

      if (event.key === "?") {
        event.preventDefault();
        setShortcutsOpen(true);
        return;
      }
      if (event.shiftKey) return;

      switch (event.key.toLowerCase()) {
        case "n":
          event.preventDefault();
          addWorkspace();
          break;
        case "e":
          if (!drawingModelCapabilities.supportsEditing) return;
          event.preventDefault();
          toggleDrawingMode();
          break;
        case "[":
          event.preventDefault();
          switchWorkspaceByOffset(-1);
          break;
        case "]":
          event.preventDefault();
          switchWorkspaceByOffset(1);
          break;
      }
    };

    window.addEventListener("keydown", handleDrawingShortcut);
    return () => window.removeEventListener("keydown", handleDrawingShortcut);
  }, [
    addWorkspace,
    drawingModelCapabilities.supportsEditing,
    generateImage,
    previewImage,
    settingsOpen,
    shortcutsOpen,
    switchWorkspaceByOffset,
    toggleDrawingMode,
    workspaceToDelete,
  ]);

  const shortcutItems = [
    {
      keys: ["Ctrl / ⌘", "Enter"],
      label: t("drawing.shortcuts.generate"),
    },
    { keys: ["N"], label: t("drawing.shortcuts.newWorkspace") },
    ...(drawingModelCapabilities.supportsEditing
      ? [{ keys: ["E"], label: t("drawing.shortcuts.toggleMode") }]
      : []),
    { keys: ["["], label: t("drawing.shortcuts.previousWorkspace") },
    { keys: ["]"], label: t("drawing.shortcuts.nextWorkspace") },
    { keys: ["?"], label: t("drawing.shortcuts.showHelp") },
  ];

  const hasGenerationOptions =
    drawingModelCapabilities.aspectRatios.length > 0 ||
    drawingModelCapabilities.imageSizes.length > 0 ||
    drawingModelCapabilities.mimeTypes.length > 0 ||
    drawingModelCapabilities.thinkingLevels.length > 0;
  const terminalTaskStatus =
    activeWorkspace?.taskStatus === "failed" ||
    activeWorkspace?.taskStatus === "canceled";

  const settingsPanel = settingsStateReady ? (
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

      {drawingModels.length > 0 && drawingQuotaEstimateState && (
        <div className="rounded-xl border border-border/60 bg-muted/20 px-3.5 py-3">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Coins className="h-3.5 w-3.5" />
            <span>{t("drawing.estimatedQuota")}</span>
          </div>
          <p className="mt-1.5 text-xl font-semibold tabular-nums text-foreground">
            {drawingQuotaEstimateState === "free" &&
              t("drawing.estimatedQuotaFree")}
            {drawingQuotaEstimateState === "usage" &&
              t("drawing.estimatedQuotaUsage")}
            {drawingQuotaEstimateState === "unavailable" &&
              t("drawing.estimatedQuotaUnavailable")}
            {drawingQuotaEstimateState === "value" && (
              <>
                {formatDecimal(drawingQuotaEstimate ?? 0)}
                <span className="ml-1.5 text-sm font-normal text-muted-foreground">
                  {t("quota")}
                </span>
              </>
            )}
          </p>
        </div>
      )}
    </div>
  ) : null;

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
                  <span className="absolute right-1 top-1 z-10 h-2 w-2 rounded-full bg-primary ring-2 ring-background" />
                ) : workspace.taskStatus === "failed" ? (
                  <span className="absolute right-1 top-1 z-10 h-2 w-2 rounded-full bg-destructive ring-2 ring-background" />
                ) : null}
              </button>
              {vertical && workspaces.length > 1 && (
                <button
                  type="button"
                  onClick={(event) => {
                    event.stopPropagation();
                    requestDeleteWorkspace(workspace.id);
                  }}
                  disabled={workspacePending}
                  className="group/workspace-delete pointer-events-none absolute -right-2 -top-2 z-20 flex h-6 w-6 items-center justify-center text-muted-foreground opacity-0 focus-visible:pointer-events-auto focus-visible:opacity-100 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-40 group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100"
                  aria-label={deleteLabel}
                  title={
                    workspacePending ? t("drawing.generating") : deleteLabel
                  }
                >
                  <span className="flex h-5 w-5 items-center justify-center rounded-full border border-border bg-background transition-colors group-hover/workspace-delete:border-destructive/50 group-hover/workspace-delete:bg-destructive group-hover/workspace-delete:text-destructive-foreground group-focus-visible/workspace-delete:ring-2 group-focus-visible/workspace-delete:ring-ring group-focus-visible/workspace-delete:ring-offset-1">
                    <X className="h-2.5 w-2.5" />
                  </span>
                </button>
              )}
            </div>
          );
        })}
      </div>
      {!vertical && workspaces.length > 1 && (
        <button
          type="button"
          onClick={() => requestDeleteWorkspace(activeWorkspaceIdForStorage)}
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
          <div className="flex shrink-0 items-center">
            <button
              type="button"
              onClick={() => setShortcutsOpen(true)}
              className="flex h-11 w-11 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              aria-label={t("drawing.shortcuts.open")}
              title={`${t("drawing.shortcuts.open")} (?)`}
            >
              <Keyboard className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={() => setSettingsOpen(true)}
              className="flex h-11 w-11 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              aria-label={t("drawing.settings")}
              title={t("drawing.settings")}
            >
              <SlidersHorizontal className="h-4 w-4" />
            </button>
          </div>
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

        <button
          type="button"
          onClick={() => setShortcutsOpen(true)}
          className="absolute right-6 top-6 z-20 hidden h-10 w-10 items-center justify-center rounded-xl text-muted-foreground transition-colors hover:bg-muted hover:text-foreground lg:flex"
          aria-label={t("drawing.shortcuts.open")}
          title={`${t("drawing.shortcuts.open")} (?)`}
        >
          <Keyboard className="h-4 w-4" />
        </button>

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
        <div className="thin-scrollbar relative z-10 flex-1 min-h-0 overflow-y-auto">
          <div
            className={cn(
              "mx-auto min-h-full w-full max-w-6xl px-4 sm:px-6",
              files.length > 0 ? "pb-80" : "pb-56",
              drawingModelCapabilities.supportsEditing
                ? "pt-20 lg:pt-24"
                : "pt-6",
            )}
          >
            {localStateReady && (
              <>
                {activeWorkspacePending && (
                  <div
                    className="mb-4 rounded-xl border border-border/60 bg-card p-4"
                    role="status"
                    aria-live="polite"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
                        <Sparkles className="h-5 w-5" />
                      </div>
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-foreground">
                          {activeWorkspace?.taskStatus === "queued"
                            ? t("drawing.queuedTitle")
                            : activeWorkspace?.taskStatus === "canceling"
                              ? t("drawing.cancelingTitle")
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
                    <div className="grid grid-cols-1 items-start gap-4 md:grid-cols-2">
                      {generatedImages.map((image, index) => {
                        const extension = getDrawingImageExtension(image.src);
                        const metadata = getGeneratedImageMetadata(
                          image,
                          activeWorkspace?.model || selectedDrawingModelId,
                          options,
                        );
                        const modelLabel =
                          drawingModels.find(
                            (model) => model.id === metadata.model,
                          )?.name || metadata.model;
                        return (
                          <article
                            key={image.id}
                            className="group relative grid min-h-40 grid-cols-[42%_minmax(0,1fr)] overflow-hidden rounded-xl border border-border/60 bg-card transition-colors duration-200 hover:border-border-hover sm:grid-cols-[11rem_minmax(0,1fr)]"
                          >
                            <button
                              type="button"
                              onClick={() => setPreviewImage(image)}
                              className="absolute inset-0 z-10 rounded-[inherit] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:ring-inset"
                              aria-label={t("drawing.openDetails")}
                            />
                            <div className="relative min-h-40 overflow-hidden bg-muted/25">
                              <img
                                src={image.src}
                                alt={t("drawing.generatedImage")}
                                loading="lazy"
                                className="absolute inset-0 h-full w-full object-cover motion-safe:transition-transform motion-safe:duration-300 group-hover:scale-[1.02]"
                              />
                              <div className="absolute left-2 top-2 flex max-w-[calc(100%-1rem)] flex-wrap gap-1">
                                <span className="rounded-md bg-black/70 px-1.5 py-0.5 text-[10px] font-medium text-white">
                                  {metadata.aspectRatio}
                                </span>
                                <span className="rounded-md bg-black/70 px-1.5 py-0.5 text-[10px] font-medium text-white">
                                  {metadata.imageSize}
                                </span>
                              </div>
                            </div>
                            <div className="flex min-w-0 flex-col p-3">
                              <p className="line-clamp-3 text-sm font-medium leading-relaxed text-foreground">
                                {image.prompt || t("drawing.generatedImage")}
                              </p>
                              <div className="mt-2 flex min-w-0 flex-wrap gap-1.5">
                                {modelLabel && (
                                  <span className="max-w-full truncate rounded-md bg-muted/55 px-2 py-1 text-[10px] text-muted-foreground">
                                    {modelLabel}
                                  </span>
                                )}
                                <span className="rounded-md bg-muted/55 px-2 py-1 text-[10px] uppercase text-muted-foreground">
                                  {metadata.mimeType.replace("image/", "")}
                                </span>
                              </div>
                              <div className="relative z-20 mt-auto flex items-center justify-end gap-1 pt-3">
                                {canAcceptReferences && (
                                  <button
                                    type="button"
                                    onClick={() =>
                                      addGeneratedImageAsReference(image)
                                    }
                                    aria-label={t("drawing.useAsReference")}
                                    title={t("drawing.useAsReference")}
                                    className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                                  >
                                    <ImagePlus className="h-3.5 w-3.5" />
                                  </button>
                                )}
                                <a
                                  href={image.src}
                                  download={`drawing-${index + 1}.${extension}`}
                                  target="_blank"
                                  rel="noreferrer"
                                  aria-label={t("drawing.downloadImage")}
                                  title={t("drawing.downloadImage")}
                                  className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                                >
                                  <Download className="h-3.5 w-3.5" />
                                </a>
                                <button
                                  type="button"
                                  onClick={() => removeGeneratedImage(image.id)}
                                  aria-label={t("remove")}
                                  title={t("remove")}
                                  className="flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </button>
                              </div>
                            </div>
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
        <div className="pointer-events-none absolute inset-x-0 bottom-0 z-20 flex justify-center px-3 pb-3 sm:px-6 sm:pb-6">
          <div
            className={cn(
              "pointer-events-auto w-full max-w-2xl rounded-2xl border bg-background shadow-md transition-colors duration-200",
              focused
                ? "border-border/70 ring-2 ring-primary/10"
                : "border-border/70",
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
                localImageOnly
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
              onFocus={() => setFocused(true)}
              onBlur={() => setFocused(false)}
              className="min-h-[76px] w-full resize-none border-0 bg-transparent px-4 py-3 text-sm leading-relaxed text-foreground shadow-none placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0"
              placeholder={t("drawing.promptPlaceholder")}
            />

            {referenceUploadPending && (
              <div
                className="px-4 pb-2 text-xs text-muted-foreground"
                role="status"
              >
                {t("drawing.uploadProcessing")}
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
                    ? canCancelActiveTask
                      ? void cancelActiveTask()
                      : undefined
                    : void generateImage()
                }
                disabled={requestInFlight ? !canCancelActiveTask : !canGenerate}
                className={cn(
                  "flex h-11 w-11 shrink-0 select-none items-center justify-center rounded-full transition-all duration-150",
                  requestInFlight
                    ? canCancelActiveTask
                      ? "bg-destructive text-destructive-foreground shadow-sm hover:opacity-85 active:scale-[0.96]"
                      : "cursor-wait bg-muted text-muted-foreground"
                    : canGenerate
                      ? "bg-foreground text-background hover:opacity-85 active:scale-[0.96] shadow-sm"
                      : "bg-muted/60 text-muted-foreground/40 cursor-not-allowed",
                )}
                aria-label={
                  requestInFlight
                    ? canCancelActiveTask
                      ? t("cancel")
                      : activeWorkspace?.taskStatus === "canceling"
                        ? t("drawing.cancelingTitle")
                        : t("drawing.generating")
                    : t("drawing.generateImage")
                }
                title={
                  requestInFlight
                    ? canCancelActiveTask
                      ? t("cancel")
                      : activeWorkspace?.taskStatus === "canceling"
                        ? t("drawing.cancelingTitle")
                        : t("drawing.generating")
                    : canGenerate
                      ? t("drawing.generateImage")
                      : generateDisabledReason
                }
              >
                {requestInFlight && !canCancelActiveTask ? (
                  <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" />
                ) : requestInFlight ? (
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

      <Dialog open={shortcutsOpen} onOpenChange={setShortcutsOpen}>
        <DialogContent className="max-w-[calc(100vw-2rem)] rounded-xl sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("drawing.shortcuts.title")}</DialogTitle>
            <DialogDescription>
              {t("drawing.shortcuts.description")}
            </DialogDescription>
          </DialogHeader>
          <div className="divide-y divide-border/60" role="list">
            {shortcutItems.map((shortcut) => (
              <div
                key={shortcut.label}
                className="flex min-h-12 items-center justify-between gap-4 py-2.5"
                role="listitem"
              >
                <span className="text-sm text-foreground">
                  {shortcut.label}
                </span>
                <span className="flex shrink-0 items-center gap-1.5">
                  {shortcut.keys.map((key, index) => (
                    <span key={key} className="flex items-center gap-1.5">
                      {index > 0 && (
                        <span className="text-xs text-muted-foreground">+</span>
                      )}
                      <kbd className="min-w-7 rounded-md border border-border bg-muted/60 px-2 py-1 text-center font-mono text-[11px] font-medium text-foreground shadow-sm">
                        {key}
                      </kbd>
                    </span>
                  ))}
                </span>
              </div>
            ))}
          </div>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(workspaceToDelete)}
        onOpenChange={(open) => {
          if (!open) setWorkspaceToDelete("");
        }}
      >
        <AlertDialogContent className="max-w-[calc(100vw-2rem)] rounded-xl sm:max-w-md">
          <AlertDialogHeader notTextCentered>
            <AlertDialogTitle>
              {t("drawing.deleteWorkspace", {
                index:
                  workspaces.findIndex(
                    (workspace) => workspace.id === workspaceToDelete,
                  ) + 1,
              })}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("drawing.deleteWorkspacePrompt")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => {
                const workspaceId = workspaceToDelete;
                setWorkspaceToDelete("");
                deleteWorkspace(workspaceId);
              }}
            >
              {t("delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog
        open={Boolean(previewImage)}
        onOpenChange={(open) => {
          if (!open) setPreviewImage(null);
        }}
      >
        <DialogContent className="flex h-[min(600px,88dvh)] w-[calc(100vw-2rem)] max-w-[960px] flex-col gap-0 overflow-hidden rounded-2xl p-0 md:max-w-[960px]">
          <DialogHeader className="sr-only">
            <DialogTitle>{t("drawing.detailsTitle")}</DialogTitle>
            <DialogDescription>
              {previewImage?.prompt || t("drawing.generatedImage")}
            </DialogDescription>
          </DialogHeader>
          {previewImage && previewMetadata && (
            <div className="grid min-h-0 flex-1 grid-rows-[minmax(240px,1fr)_auto] md:grid-cols-2 md:grid-rows-1">
              <div className="relative min-h-0 overflow-hidden bg-muted/40">
                <img
                  src={previewImage.src}
                  alt={t("drawing.generatedImage")}
                  className="absolute inset-0 h-full w-full object-cover"
                />
              </div>

              <div className="flex min-h-0 flex-col border-t border-border/50 p-5 sm:p-6 md:border-l md:border-t-0">
                <DialogHeader notTextCentered className="pr-8">
                  <DialogTitle>{t("drawing.detailsTitle")}</DialogTitle>
                  <DialogDescription>
                    {formatGeneratedImageDate(
                      previewImage.createdAt,
                      i18n.language,
                    )}
                  </DialogDescription>
                </DialogHeader>

                <section className="mt-5">
                  <div className="text-xs font-medium text-muted-foreground">
                    {t("drawing.promptLabel")}
                  </div>
                  <p className="mt-2 whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground">
                    {previewImage.prompt || t("drawing.generatedImage")}
                  </p>
                </section>

                <section className="mt-5">
                  <div className="text-xs font-medium text-muted-foreground">
                    {t("drawing.options.title")}
                  </div>
                  <div className="mt-3 grid grid-cols-2 gap-2">
                    {[
                      [t("drawing.model"), previewModelLabel],
                      [
                        t("drawing.options.aspectRatio"),
                        previewMetadata.aspectRatio,
                      ],
                      [
                        t("drawing.options.imageSize"),
                        previewMetadata.imageSize,
                      ],
                      [
                        t("drawing.options.mimeType"),
                        previewMetadata.mimeType.replace("image/", ""),
                      ],
                      [
                        t("drawing.options.thinkingLevel"),
                        ["minimal", "high"].includes(
                          previewMetadata.thinkingLevel,
                        )
                          ? t(
                              `drawing.options.thinking.${previewMetadata.thinkingLevel}`,
                            )
                          : previewMetadata.thinkingLevel,
                      ],
                    ].map(([label, value]) => (
                      <div
                        key={label}
                        className="min-w-0 rounded-xl bg-muted/40 px-3 py-2.5"
                      >
                        <div className="text-[11px] text-muted-foreground">
                          {label}
                        </div>
                        <div className="mt-1 truncate text-xs font-medium text-foreground">
                          {value}
                        </div>
                      </div>
                    ))}
                  </div>
                </section>

                <div className="mt-6 flex items-center justify-end gap-2 border-t border-border/60 pt-4 md:mt-auto">
                  {canAcceptReferences && (
                    <button
                      type="button"
                      onClick={() =>
                        addGeneratedImageAsReference(previewImage)
                      }
                      aria-label={t("drawing.useAsReference")}
                      title={t("drawing.useAsReference")}
                      className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-border bg-background text-foreground transition-colors hover:bg-muted"
                    >
                      <ImagePlus className="h-4 w-4" />
                    </button>
                  )}
                  <a
                    href={previewImage.src}
                    download={`drawing.${getDrawingImageExtension(previewImage.src)}`}
                    target="_blank"
                    rel="noreferrer"
                    aria-label={t("drawing.downloadImage")}
                    title={t("drawing.downloadImage")}
                    className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-foreground text-background transition-opacity hover:opacity-85"
                  >
                    <Download className="h-4 w-4" />
                  </a>
                  <button
                    type="button"
                    onClick={() => removeGeneratedImage(previewImage.id)}
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-destructive/10 px-3.5 text-sm font-medium text-destructive transition-colors hover:bg-destructive hover:text-destructive-foreground"
                  >
                    <Trash2 className="h-4 w-4 shrink-0" />
                    {t("delete")}
                  </button>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default Drawing;
