import {
  type Dispatch,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
} from "react";
import {
  Loader2,
  Paperclip,
  X,
  FileIcon,
  FileTextIcon,
  FileImageIcon,
  FileVideoIcon,
  FileAudioIcon,
  FileSpreadsheetIcon,
  FileArchiveIcon,
  FileCodeIcon,
  FileJsonIcon,
  FileVideo2Icon,
  FileDigitIcon,
  AlarmClock,
} from "lucide-react";

import "@/assets/common/file.less";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { useTranslation } from "react-i18next";
import { useDraggableInput as bindDraggableInput } from "@/utils/dom.ts";
import { FileObject, FileArray, quickBlobParser } from "@/api/file.ts";
import { useSelector } from "react-redux";
import {
  getModelFromId,
  isHighContextModel,
  supportsImageUpload,
} from "@/conf/model.ts";
import { selectModel, selectSupportModels } from "@/store/chat.ts";
import { ChatAction } from "@/components/home/assemblies/ChatAction.tsx";
import { blobEvent, filePanelEvent } from "@/events/blob.ts";
import { isB64Image } from "@/utils/base.ts";
import { toast } from "sonner";
import { Badge } from "./ui/badge.tsx";
import { AnimatePresence, motion } from "framer-motion";
import { cn } from "./ui/lib/utils.ts";
import { Progress } from "./ui/progress.tsx";
import { normalizeImageURL } from "@/utils/image-url.ts";

const MaxFileSize = 1024 * 1024 * 100; // 100MB File Size Limit
const MaxPromptSize = 10 * 1024; // 10KB Prompt Size Limit (to avoid token overflow)

function getClipboardImageFiles(clipboardData: DataTransfer | null): File[] {
  if (!clipboardData) return [];

  return Array.from(clipboardData.items)
    .filter((item) => item.kind === "file" && item.type.startsWith("image/"))
    .map((item) => item.getAsFile())
    .filter((file): file is File => Boolean(file));
}

type FileTask = {
  id: number;
  file: File;
  progress: number;
};

type FileTaskState = {
  tasks: FileTask[];
};

type FileTaskAction =
  | { type: "add"; payload: FileTask }
  | { type: "remove"; payload: number }
  | { type: "update-progress"; payload: { id: number; progress: number } };

export type FileProviderAction =
  | { type: "add"; payload: FileObject }
  | { type: "remove"; payload: number }
  | { type: "clear" };

function fileTaskReducer(
  state: FileTaskState,
  action: FileTaskAction,
): FileTaskState {
  switch (action.type) {
    case "add":
      return { ...state, tasks: [...state.tasks, action.payload] };
    case "remove":
      return {
        ...state,
        tasks: state.tasks.filter((task) => task.id !== action.payload),
      };
    case "update-progress":
      return {
        ...state,
        tasks: state.tasks.map((task) =>
          task.id === action.payload.id
            ? { ...task, progress: action.payload.progress }
            : task,
        ),
      };
    default:
      return state;
  }
}

type FileProviderProps = {
  files: FileArray;
  dispatch: Dispatch<FileProviderAction>;
  modelId?: string;
  forceImageUpload?: boolean;
  localImageOnly?: boolean;
  maxFiles?: number;
  onUploadingChange?: (uploading: boolean) => void;
  trigger?: (props: {
    disabled: boolean;
    filesCount: number;
    open: () => void;
  }) => ReactNode;
};

function FileProvider({
  files,
  dispatch,
  modelId,
  forceImageUpload = false,
  localImageOnly = false,
  maxFiles,
  onUploadingChange,
  trigger,
}: FileProviderProps) {
  const { t } = useTranslation();
  const selectedModel = useSelector(selectModel);
  const model = modelId ?? selectedModel;
  const [open, setOpen] = useState(false);
  const [previewFile, setPreviewFile] = useState<FileObject | null>(null);
  const nextTaskId = useRef(0);

  const [tasks, taskDispatch] = useReducer(fileTaskReducer, {
    tasks: [],
  } as FileTaskState);
  const uploading = tasks.tasks.length > 0;

  useEffect(() => {
    onUploadingChange?.(uploading);
  }, [onUploadingChange, uploading]);

  useEffect(
    () => () => {
      onUploadingChange?.(false);
    },
    [onUploadingChange],
  );

  const supportModels = useSelector(selectSupportModels);
  const currentModelInfo = useMemo(
    () =>
      getModelFromId(supportModels, model) ?? {
        id: model,
        ocr_model: false,
        vision_model: false,
        reverse_model: false,
      },
    [supportModels, model],
  );
  const uploadModelInfo = useMemo(
    () =>
      forceImageUpload
        ? { ...currentModelInfo, vision_model: true }
        : currentModelInfo,
    [currentModelInfo, forceImageUpload],
  );
  const canUploadImage =
    model.trim().length > 0 &&
    (forceImageUpload || supportsImageUpload(currentModelInfo));
  const canOpenFilePanel = canUploadImage || files.length > 0;
  const fileLimit =
    typeof maxFiles === "number" ? Math.max(0, Math.floor(maxFiles)) : null;
  const hasFileLimit = fileLimit !== null;
  const fileLimitReached =
    hasFileLimit && files.length + tasks.tasks.length >= (fileLimit ?? 0);

  const showFileLimitToast = useCallback(() => {
    if (!hasFileLimit) return;
    toast.error(t("file.max-count"), {
      description: t("file.max-count-prompt", { count: fileLimit ?? 0 }),
    });
  }, [fileLimit, hasFileLimit, t]);

  const addFile = useCallback(
    (file: FileObject) => {
      console.debug(
        `[file] new file was added (filename: ${file.name}, size: ${file.size}, prompt: ${file.content.length})`,
      );
      if (
        file.content.length > MaxPromptSize &&
        !isHighContextModel(supportModels, model) &&
        !isB64Image(file.content)
      ) {
        file.content = file.content.slice(0, MaxPromptSize);
        toast(t("file.max-length"), {
          description: t("file.max-length-prompt"),
        });
      }

      dispatch({ type: "add", payload: file });
    },
    [dispatch, model, supportModels, t],
  );

  const triggerFile = useCallback(
    async (incomingFiles: (File | null)[]) => {
      if (!canUploadImage) {
        toast.info(t("file.vision-model-required"));
        return;
      }

      const availableFiles = incomingFiles.filter(
        (file): file is File => Boolean(file),
      );

      if (availableFiles.length === 0) {
        return;
      }

      if (
        hasFileLimit &&
        files.length + tasks.tasks.length + availableFiles.length >
          (fileLimit ?? 0)
      ) {
        showFileLimitToast();
        return;
      }

      for (const file of availableFiles) {
        if (file.size > MaxFileSize) {
          toast.error(t("file.over-size"), {
            description: t("file.over-size-prompt", {
              size: (MaxFileSize / 1024 / 1024).toFixed(),
            }),
          });
        } else {
          nextTaskId.current += 1;
          const id = nextTaskId.current;
          taskDispatch({
            type: "add",
            payload: { id, file, progress: 0 },
          });

          const task = quickBlobParser(
            file,
            uploadModelInfo,
            (progress) => {
              console.debug(
                `[parser] task ${id} progress: ${progress.toFixed(2)}%`,
              );
              taskDispatch({
                type: "update-progress",
                payload: { id, progress },
              });
            },
            localImageOnly,
          );

          toast.promise(task, {
            loading: t("file.uploading-prompt"),
            success: (content: string) => {
              addFile({ name: file.name, content, size: file.size });
              taskDispatch({
                type: "remove",
                payload: id,
              });
              return t("file.parse-success-prompt", { file: file.name });
            },
            error: (error: Error) => {
              taskDispatch({
                type: "remove",
                payload: id,
              });
              const reason =
                error.message ===
                "The current model does not support image recognition"
                  ? t("file.vision-model-required")
                  : error.message === "Only image uploads are supported"
                  ? t("file.image-only")
                  : error.message;
              return t("file.parse-error-prompt", { reason });
            },
          });
        }
      }
    },
    [
      addFile,
      canUploadImage,
      fileLimit,
      files.length,
      hasFileLimit,
      localImageOnly,
      showFileLimitToast,
      t,
      tasks.tasks.length,
      uploadModelInfo,
    ],
  );

  const handlePasteImages = useCallback(
    (clipboardData: DataTransfer | null) => {
      const pastedImages = getClipboardImageFiles(clipboardData);
      if (pastedImages.length === 0) return false;

      void triggerFile(pastedImages);
      return true;
    },
    [triggerFile],
  );

  useEffect(() => {
    blobEvent.bind(async (file: File | File[]) => {
      if (!canUploadImage) {
        toast.info(t("file.vision-model-required"));
        return;
      }
      setOpen?.(true);
      await triggerFile(Array.isArray(file) ? file : [file]);
    });
  }, [canUploadImage, t, triggerFile]);

  useEffect(() => {
    filePanelEvent.bind(() => {
      if (!canOpenFilePanel) return;
      setOpen(true);
    });
  }, [canOpenFilePanel]);

  useEffect(() => {
    if (canUploadImage || canOpenFilePanel) return;

    if (open) {
      setOpen(false);
    }
  }, [canOpenFilePanel, canUploadImage, open]);

  function removeFile(index: number) {
    dispatch({ type: "remove", payload: index });
  }

  const openFilePanel = () => {
    if (!canOpenFilePanel) {
      toast.info(t("file.vision-model-required"));
      return;
    }
    setOpen(true);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!canOpenFilePanel && next) {
          toast.info(t("file.vision-model-required"));
          return;
        }
        setOpen(next);
      }}
    >
      {trigger ? (
        trigger({
          disabled: !canOpenFilePanel,
          filesCount: files.length,
          open: openFilePanel,
        })
      ) : (
        <ChatAction
          text={
            canOpenFilePanel ? t("file.file") : t("file.vision-model-required")
          }
          active={files.length}
          badge={files.length}
          className={!canOpenFilePanel ? "opacity-50" : undefined}
          disabled={!canOpenFilePanel}
          onClick={openFilePanel}
        >
          <Paperclip className={`h-4 w-4`} />
        </ChatAction>
      )}
      <DialogContent
        className={`file-dialog flex-dialog`}
        onPaste={(event) => {
          if (handlePasteImages(event.clipboardData)) {
            event.preventDefault();
          }
        }}
      >
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center">
            {t("file.file")}
            <Badge variant="secondary" className="ml-2">
              {files.length}
            </Badge>
          </DialogTitle>
          <DialogDescription asChild>
            <motion.div
              className={`file-wrapper`}
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              <AnimatePresence key="files">
                <FileList
                  value={files}
                  removeFile={removeFile}
                  previewFile={setPreviewFile}
                />
              </AnimatePresence>
              <AnimatePresence key="tasks">
                {tasks.tasks.map((task, index) => (
                  <motion.div
                    key={task.id}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -20 }}
                    transition={{ duration: 0.1, delay: index * 0.1 }}
                  >
                    <FileTaskItem task={task} />
                  </motion.div>
                ))}
              </AnimatePresence>
              <motion.div
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: 0.2, duration: 0.3 }}
              >
                {canUploadImage && !fileLimitReached && (
                  <FileInput
                    loading={tasks.tasks.length > 0}
                    id={"file"}
                    className={"file"}
                    handleEvent={triggerFile}
                  />
                )}
                {canUploadImage && fileLimitReached && (
                  <div className="drop-window cursor-default">
                    <p>
                      {t("file.max-count-reached", { count: fileLimit ?? 0 })}
                    </p>
                  </div>
                )}
              </motion.div>
            </motion.div>
          </DialogDescription>
        </DialogHeader>
      </DialogContent>
      <FilePreviewDialog
        file={previewFile}
        onOpenChange={(open) => {
          if (!open) {
            setPreviewFile(null);
          }
        }}
      />
    </Dialog>
  );
}

type FileTaskItemProps = {
  task: FileTask;
};

function FileTaskItem({ task }: FileTaskItemProps) {
  return (
    <div className="w-full h-fit flex flex-row items-center py-0.5 select-none">
      <AlarmClock className="w-3.5 h-3.5 mr-1" />
      <div className="truncate">{task.file.name}</div>
      <div className="mr-1 ml-auto text-xs">{task.progress.toFixed()}%</div>
      <Progress value={task.progress} className="w-16 md:w-24 h-2" />
    </div>
  );
}

type FileBadgeProps = {
  name: string;
};

function getFileExtension(name: string) {
  return name.split(".").pop()?.toLowerCase() || "";
}

const previewImageExtensions = new Set([
  "jpg",
  "jpeg",
  "png",
  "gif",
  "webp",
  "heif",
  "heic",
  "bmp",
  "svg",
]);

function isPreviewableImageFile(name: string): boolean {
  return previewImageExtensions.has(getFileExtension(name));
}

function getFilePreviewImageURL(file: FileObject): string {
  if (!isPreviewableImageFile(file.name) && !isB64Image(file.content)) {
    return "";
  }

  return normalizeImageURL(file.content);
}

function getFileIcon(name: string) {
  const extension = getFileExtension(name);
  switch (extension) {
    case "pdf":
      return FileTextIcon;
    case "doc":
    case "docx":
    case "txt":
      return FileDigitIcon;
    case "xls":
    case "xlsx":
    case "csv":
      return FileSpreadsheetIcon;
    case "ppt":
    case "pptx":
      return FileVideo2Icon;
    case "jpg":
    case "jpeg":
    case "png":
    case "gif":
    case "svg":
      return FileImageIcon;
    case "mp4":
    case "avi":
    case "mov":
      return FileVideoIcon;
    case "mp3":
    case "wav":
      return FileAudioIcon;
    case "zip":
    case "rar":
    case "7z":
      return FileArchiveIcon;
    case "js":
    case "ts":
    case "py":
    case "java":
    case "cpp":
    case "c":
    case "h":
    case "rs":
    case "swift":
    case "kt":
    case "ktm":
    case "php":
    case "rb":
    case "sh":
    case "html":
    case "css":
    case "scss":
    case "less":
    case "sass":
    case "styl":
    case "vue":
    case "svelte":
    case "astro":
    case "tsx":
    case "jsx":
      return FileCodeIcon;
    case "json":
    case "xml":
    case "jsonl":
    case "yaml":
    case "yml":
    case "toml":
    case "ini":
    case "cfg":
    case "conf":
      return FileJsonIcon;
    default:
      return FileIcon;
  }
}

function FileIconObject({ name }: FileBadgeProps) {
  const IconComponent = useMemo(() => getFileIcon(name), [name]);

  return (
    <div className="w-fit h-fit relative">
      <IconComponent className="stroke-[1.25] h-8 w-8 text-primary/70 group-hover:text-primary transition-colors duration-200" />
    </div>
  );
}

function FileBadge({ name }: FileBadgeProps) {
  const extension = getFileExtension(name);
  return (
    <span
      className={cn(
        "px-1 inline-block mr-1 rounded-sm bg-muted/50 text-2xs text-primary",
        {
          // pdf&ppt: red-500
          "bg-red-500/10 text-red-500":
            extension === "pdf" || extension === "ppt" || extension === "pptx",
          // doc: blue-500
          "bg-blue-500/10 text-blue-500":
            extension === "doc" || extension === "docx",
          // xls: green-500
          "bg-green-500/10 text-green-500":
            extension === "xls" || extension === "xlsx" || extension === "csv",
          // json/xml/etc: orange-500
          "bg-orange-500/10 text-orange-500":
            extension === "json" ||
            extension === "xml" ||
            extension === "jsonl" ||
            extension === "yaml" ||
            extension === "yml" ||
            extension === "toml" ||
            extension === "ini" ||
            extension === "cfg" ||
            extension === "conf",
          // code: violet-500
          "bg-violet-500/10 text-violet-500":
            extension === "js" ||
            extension === "ts" ||
            extension === "py" ||
            extension === "java" ||
            extension === "cpp" ||
            extension === "go" ||
            extension === "c" ||
            extension === "h" ||
            extension === "rs" ||
            extension === "swift" ||
            extension === "kt" ||
            extension === "ktm" ||
            extension === "php" ||
            extension === "rb" ||
            extension === "sh" ||
            extension === "html" ||
            extension === "css" ||
            extension === "scss" ||
            extension === "less" ||
            extension === "sass" ||
            extension === "styl" ||
            extension === "vue" ||
            extension === "svelte" ||
            extension === "astro" ||
            extension === "tsx" ||
            extension === "jsx" ||
            extension === "ts" ||
            extension === "jsx",
        },
      )}
    >
      {extension.toUpperCase()}
    </span>
  );
}

type FileListProps = {
  value: FileArray;
  removeFile: (index: number) => void;
  previewFile: (file: FileObject) => void;
};

function FileList({ value, removeFile, previewFile }: FileListProps) {
  const { t } = useTranslation();

  if (value.length === 0) return null;

  const listVariants = {
    hidden: { opacity: 0, height: 0 },
    visible: { opacity: 1, height: "auto" },
  };

  const itemVariants = {
    hidden: { opacity: 0, y: 10 },
    visible: { opacity: 1, y: 0 },
  };

  return (
    <motion.div
      className={`file-list compact-file-list`}
      initial="hidden"
      animate="visible"
      variants={listVariants}
    >
      <AnimatePresence>
        {value.map((file, index) => (
          <motion.div
            className={`file-card-compact group relative flex cursor-pointer flex-row items-center gap-2 rounded-lg border bg-gradient-to-tr from-background to-muted/25 p-2 pr-8 shadow-sm transition-all duration-200 ease-in-out hover:border-primary/40`}
            key={index}
            initial="hidden"
            animate="visible"
            exit="hidden"
            variants={itemVariants}
            transition={{ delay: index * 0.1 }}
            role="button"
            tabIndex={0}
            onClick={() => previewFile(file)}
            onKeyDown={(event) => {
              if (event.target !== event.currentTarget) return;
              if (event.key !== "Enter" && event.key !== " ") return;

              event.preventDefault();
              previewFile(file);
            }}
          >
            <div className="flex h-fit shrink-0 items-center">
              <FileIconObject name={file.name} />
            </div>
            <div className="flex min-w-0 flex-1 flex-col">
              <span
                className={`truncate text-sm font-medium text-foreground`}
              >
                {file.name}
              </span>
              <span
                className={`mt-1 flex flex-row items-center text-xs text-muted-foreground`}
              >
                <FileBadge name={file.name} />
                {((file.size || file.content.length) / 1024).toFixed(2)}KB
              </span>
            </div>
            <button
              type="button"
              aria-label={t("remove")}
              className="absolute right-2 top-2 h-fit w-fit rounded-full p-1 transition-colors duration-200 hover:bg-secondary/10"
              onClick={(e) => {
                e.stopPropagation();
                removeFile(index);
              }}
            >
              <X
                className={`h-4 w-4 text-secondary transition-colors duration-200 hover:text-destructive`}
              />
            </button>
          </motion.div>
        ))}
      </AnimatePresence>
    </motion.div>
  );
}

type FilePreviewDialogProps = {
  file: FileObject | null;
  onOpenChange: (open: boolean) => void;
};

function FilePreviewDialog({ file, onOpenChange }: FilePreviewDialogProps) {
  const { t } = useTranslation();
  const imageURL = useMemo(
    () => (file ? getFilePreviewImageURL(file) : ""),
    [file],
  );

  return (
    <Dialog open={Boolean(file)} onOpenChange={onOpenChange}>
      <DialogContent className="flex-dialog md:max-w-[min(90vw,900px)]">
        <DialogHeader>
          <DialogTitle className="flex flex-row items-center select-none">
            <Paperclip className="h-4 w-4 mr-2" />
            {file?.name ?? t("file.file")}
          </DialogTitle>
        </DialogHeader>
        {file && (
          <div className="flex max-h-[70vh] min-h-48 items-center justify-center overflow-auto rounded-md bg-muted/30 p-2">
            {imageURL ? (
              <img
                src={imageURL}
                alt={file.name}
                className="max-h-[68vh] max-w-full rounded-sm object-contain"
              />
            ) : (
              <pre className="max-h-[68vh] w-full whitespace-pre-wrap break-words rounded-sm p-2 text-sm">
                {file.content}
              </pre>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

type FileInputProps = {
  id: string;
  loading: boolean;
  className?: string;
  handleEvent: (files: (File | null)[]) => void;
};

function FileInput({
  id,
  loading,
  className,
  handleEvent,
}: FileInputProps) {
  const { t } = useTranslation();
  const ref = useRef(null);

  useEffect(() => {
    return bindDraggableInput(window.document.body, handleEvent);
  }, [handleEvent]);

  return (
    <>
      <label className={`drop-window`} htmlFor={id} ref={ref}>
        {loading && <Loader2 className={`h-4 w-4 animate-spin mr-2`} />}
        <p>{t("file.drop")}</p>

        <div className="mt-6 flex flex-wrap justify-center gap-2">
          {[{ icon: FileImageIcon, text: "Image" }].map((item, index) => (
            <motion.div
              key={index}
              className="flex flex-col items-center"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.1, duration: 0.3 }}
            >
              <div className="w-10 h-10 rounded-full bg-muted/50 p-2 hover:bg-muted/70 transition-colors duration-200">
                <item.icon className="w-full h-full text-secondary stroke-[1.5]" />
              </div>
              <span className="mt-0.5 text-xs font-medium text-muted-foreground">
                {item.text}
              </span>
            </motion.div>
          ))}
        </div>
      </label>
      <input
        id={id}
        type="file"
        className={className}
        onChange={(e) => {
          handleEvent(Array.from(e.target?.files || []));
          e.currentTarget.value = "";
        }}
        accept="image/*"
        style={{ display: "none" }}
        multiple={true}
        // on transfer file
        onPaste={(e) => {
          const items = e.clipboardData.items;
          const files = Array.from(items).filter(
            (item) => item.kind === "file",
          );
          handleEvent(files.map((file) => file.getAsFile()));
        }}
      />
    </>
  );
}

export default FileProvider;
