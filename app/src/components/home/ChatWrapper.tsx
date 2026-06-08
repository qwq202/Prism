import { useTranslation } from "react-i18next";
import {
  useCallback,
  useEffect,
  lazy,
  useMemo,
  useReducer,
  useState,
  useRef,
  Suspense,
} from "react";
import FileAction, {
  type FileProviderAction,
} from "@/components/FileProvider.tsx";
import { useSelector } from "react-redux";
import { selectAuthenticated, selectInit } from "@/store/auth.ts";
import {
  listenMessageEvent,
  selectConversationLoading,
  selectCurrent,
  selectModel,
  selectSupportModels,
  useMessageActions,
  useMessages,
  useWorking,
} from "@/store/chat.ts";
import { formatMessage } from "@/utils/processor.ts";
import { clearHistoryState, getQueryParam } from "@/utils/path.ts";
import { forgetMemory, popMemory } from "@/utils/memory.ts";
import { alignSelector } from "@/store/settings.ts";
import { FileArray } from "@/api/file.ts";
import {
  DeepSeekThinkingAction,
  FetchAction,
  GeminiThinkingAction,
  LearningModeAction,
  NewConversationAction,
  OpenAIReasoningAction,
  WebAction,
} from "@/components/home/assemblies/ChatAction.tsx";
import ChatSpace from "@/components/home/ChatSpace.tsx";
import ActionButton, {
  ActionCommand,
} from "@/components/home/assemblies/ActionButton.tsx";
import ChatInput from "@/components/home/assemblies/ChatInput.tsx";
import ScrollAction from "@/components/home/assemblies/ScrollAction.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import { goAuth } from "@/utils/app.ts";
import { getModelFromId } from "@/conf/model.ts";
import { ModelArea } from "@/components/home/ModelArea.tsx";
import { toast } from "sonner";
import { VoiceAction } from "@/components/VoiceProvider.tsx";
import { AnimatePresence, motion } from "framer-motion";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { blobEvent, filePanelEvent } from "@/events/blob.ts";

const loadChatInterface = () => import("@/components/home/ChatInterface.tsx");
let chatInterfacePromise: ReturnType<typeof loadChatInterface> | undefined;

function preloadChatInterface() {
  chatInterfacePromise ??= loadChatInterface();
  return chatInterfacePromise;
}

const ChatInterface = lazy(preloadChatInterface);

function hasDraggedFiles(dataTransfer: DataTransfer | null): boolean {
  return Boolean(dataTransfer && Array.from(dataTransfer.types).includes("Files"));
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

type InterfaceProps = {
  scrollable: boolean;
  setTarget: (instance: HTMLElement | null) => void;
};

function ChatContentSkeleton() {
  return (
    <div className="chat-content" aria-busy="true">
      <div className="chat-messages-wrapper gap-6">
        <div className="flex w-full max-w-3xl items-start gap-3">
          <Skeleton className="h-9 w-9 shrink-0 rounded-md" />
          <div className="flex w-full max-w-2xl flex-col gap-3">
            <Skeleton className="h-5 w-3/5" />
            <Skeleton className="h-5 w-11/12" />
            <Skeleton className="h-5 w-4/5" />
          </div>
        </div>
        <div className="ml-auto flex w-full max-w-2xl justify-end">
          <div className="flex w-3/5 flex-col items-end gap-3">
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-2/3" />
          </div>
        </div>
        <div className="flex w-full max-w-3xl items-start gap-3">
          <Skeleton className="h-9 w-9 shrink-0 rounded-md" />
          <div className="flex w-full max-w-2xl flex-col gap-3">
            <Skeleton className="h-5 w-1/2" />
            <Skeleton className="h-5 w-full" />
            <Skeleton className="h-5 w-5/6" />
          </div>
        </div>
      </div>
    </div>
  );
}

function Interface(props: InterfaceProps) {
  const messages = useMessages();
  const current = useSelector(selectCurrent);
  const conversationLoading = useSelector(selectConversationLoading);
  const loadingStage = <ChatContentSkeleton />;
  const emptyConversationStage = (
    <div className="chat-content chat-content-placeholder" aria-busy="false" />
  );

  return conversationLoading && messages.length === 0 ? (
    loadingStage
  ) : messages.length > 0 ? (
    <Suspense fallback={current === -1 ? emptyConversationStage : loadingStage}>
      <ChatInterface {...props} />
    </Suspense>
  ) : current !== -1 ? (
    emptyConversationStage
  ) : (
    <ChatSpace />
  );
}

function fileReducer(state: FileArray, action: FileProviderAction): FileArray {
  switch (action.type) {
    case "add":
      return [...state, action.payload];
    case "remove":
      return state.filter((_, i) => i !== action.payload);
    case "clear":
      return [];
    default:
      return state;
  }
}

function ChatWrapper() {
  const { t } = useTranslation();
  const { send: sendAction } = useMessageActions();
  const process = listenMessageEvent();
  const [files, fileDispatch] = useReducer(fileReducer, []);
  const [input, setInput] = useState("");
  const [visible, setVisibility] = useState(false);
  const init = useSelector(selectInit);
  const current = useSelector(selectCurrent);
  const auth = useSelector(selectAuthenticated);
  const model = useSelector(selectModel);
  const target = useRef<HTMLTextAreaElement>(null);
  const align = useSelector(alignSelector);

  const working = useWorking();
  const supportModels = useSelector(selectSupportModels);

  const requireAuth = useMemo(
    (): boolean => !!getModelFromId(supportModels, model)?.auth,
    [model, supportModels],
  );

  const [instance, setInstance] = useState<HTMLElement | null>(null);

  useEffect(() => {
    void preloadChatInterface();
  }, []);

  useEffect(() => {
    const openFilePanel = (event: DragEvent) => {
      if (!hasDraggedFiles(event.dataTransfer)) return false;
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
      filePanelEvent.emit(undefined);
      return true;
    };

    const handleDragEnter = (event: DragEvent) => {
      openFilePanel(event);
    };

    const handleDragOver = (event: DragEvent) => {
      openFilePanel(event);
    };

    const handleDrop = (event: DragEvent) => {
      if (event.defaultPrevented || !hasDraggedFiles(event.dataTransfer)) {
        return;
      }
      event.preventDefault();

      const files = getDroppedFiles(event.dataTransfer);
      if (files.length > 0) {
        blobEvent.emit(files);
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

  function clearFile() {
    fileDispatch({ type: "clear" });
  }

  const handleLongTextPaste = useCallback(
    (text: string) => {
      const index = files.length + 1;
      const filename = `pasted-text-${index}.txt`;
      fileDispatch({
        type: "add",
        payload: {
          name: filename,
          content: text,
          size: new Blob([text]).size,
        },
      });
      toast.success(t("file.parse-success-prompt", { file: filename }));
    },
    [files.length, t],
  );

  const processSend = useCallback(
    async function processSend(
      data: string,
      passAuth?: boolean,
    ): Promise<boolean> {
      if (requireAuth && !auth && !passAuth) {
        toast(t("login-require"), {
          description: t("login-require-prompt"),
          action: {
            label: t("login"),
            onClick: goAuth,
          },
        });
        return false;
      }

      if (working) return false;

      const message: string = formatMessage(files, data);
      if (message.length > 0 && data.trim().length > 0) {
        if (await sendAction(message)) {
          forgetMemory("history");
          clearFile();
          return true;
        }
      }
      return false;
    },
    [auth, files, requireAuth, sendAction, t, working],
  );

  async function handleSend() {
    // because of the function wrapper, we need to update the selector state using props.
    if (await processSend(input)) {
      setInput("");
    }
  }

  async function handleCancel() {
    process({ id: current, event: "stop" });
  }

  useEffect(() => {
    window.addEventListener("load", () => {
      const el = document.getElementById("input");
      if (el) el.focus();
    });
  }, []);

  useEffect(() => {
    if (!init) return;
    const query = getQueryParam("q").trim();
    if (query.length > 0) processSend(query).then();
    clearHistoryState();
  }, [init, processSend]);

  useEffect(() => {
    const history: string = popMemory("history");
    if (history.length) {
      setInput(history);
      toast(t("chat.recall"), {
        description: t("chat.recall-desc"),
        action: {
          label: t("chat.recall-cancel"),
          onClick: () => {
            setInput("");
          },
        },
      });
    }
  }, [t]);

  return (
    <div className={`chat-container bg-muted/25 dark:bg-muted/10`}>
      <div className={`chat-wrapper`}>
        <Interface setTarget={setInstance} scrollable={!visible} />
        <div className={`chat-input border-t bg-muted/25`}>
          <motion.div
            className={`flex flex-row items-center p-1.5 pb-0.5`}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, ease: "easeOut" }}
          >
            <AnimatePresence key="model">
              <motion.div
                key="model-area"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3 }}
              >
                <ModelArea />
              </motion.div>
              <motion.div
                key="web-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.1 }}
              >
                <WebAction />
              </motion.div>
              <motion.div
                key="fetch-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.15 }}
              >
                <FetchAction />
              </motion.div>
              <motion.div
                key="learning-mode-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.2 }}
              >
                <LearningModeAction />
              </motion.div>
              <motion.div
                key="gemini-thinking-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.25 }}
              >
                <GeminiThinkingAction />
              </motion.div>
              <motion.div
                key="openai-reasoning-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.3 }}
              >
                <OpenAIReasoningAction />
              </motion.div>
              <motion.div
                key="deepseek-thinking-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.35 }}
              >
                <DeepSeekThinkingAction />
              </motion.div>
              <motion.div
                key="file-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.4 }}
              >
                <FileAction files={files} dispatch={fileDispatch} />
              </motion.div>
              <motion.div
                key="voice-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.45 }}
              >
                <VoiceAction
                  value={input}
                  onValueChange={setInput}
                  target={target}
                />
              </motion.div>
              <motion.div
                key="scroll-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.5 }}
              >
                <ScrollAction
                  visible={visible}
                  setVisibility={setVisibility}
                  target={instance}
                />
              </motion.div>
            </AnimatePresence>
            <motion.div
              className={`grow`}
              initial={{ scaleX: 0 }}
              animate={{ scaleX: 1 }}
              transition={{ duration: 0.5, ease: "easeOut" }}
            />
            <AnimatePresence key="new">
              <motion.div
                key="new-conversation-action"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                transition={{ duration: 0.3, delay: 0.6 }}
              >
                <NewConversationAction />
              </motion.div>
            </AnimatePresence>
          </motion.div>
          <div className={`flex flex-col gap-2 px-3 pb-2`}>
            <div className={`relative w-full`}>
              <ChatInput
                className={cn(
                  "rounded-none border-0 bg-transparent w-full",
                  align && "align",
                )}
                target={target}
                value={input}
                onValueChange={setInput}
                onEnterPressed={handleSend}
                onLongTextPaste={handleLongTextPaste}
              />
            </div>
            <div className="flex items-center justify-end gap-2">
              <ActionCommand input={input} />
              <ActionButton
                working={working}
                onClick={() => (working ? handleCancel() : handleSend())}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ChatWrapper;
