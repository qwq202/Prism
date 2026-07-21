import { Message, UserRole } from "@/api/types.tsx";
import Markdown from "@/components/Markdown.tsx";
import {
  CalendarCheck2,
  CircleSlash,
  Cloud,
  CloudCog,
  Copy,
  File,
  Loader2,
  SquareMousePointer,
  PencilLine,
  Power,
  RotateCcw,
  Trash,
} from "lucide-react";
import {
  filterMessage,
  getFileMarkdown,
  splitFileMessage,
} from "@/utils/processor.ts";
import { copyClipboard, isContainDom, saveAsFile } from "@/utils/dom.ts";
import { useTranslation } from "react-i18next";
import React, { Ref, useMemo, useRef, useState } from "react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import EditorProvider from "@/components/EditorProvider.tsx";
import Avatar from "@/components/Avatar.tsx";
import { useSelector } from "react-redux";
import { selectUsername } from "@/store/auth.ts";
import { motion } from "framer-motion";
import { ThinkContent } from "@/components/ThinkContent";
import ModelAvatar from "@/components/ModelAvatar.tsx";
import { selectSupportModels } from "@/store/chat.ts";
import { parseThinkContent } from "@/utils/thinking";
import { showQuotaSelector } from "@/store/settings.ts";
import { MarkdownFile } from "@/components/plugins/file.tsx";
import {
  type AskUserResult,
  getAskUserToolCalls,
  isPendingAskUserToolCall,
} from "@/api/ask-user.ts";
import { AskUserCard } from "@/components/AskUserCard.tsx";

type MessageProps = {
  index: number;
  message: Message;
  end?: boolean;
  username?: string;
  model?: string;
  onEvent?: (event: string, index?: number, message?: string) => void;
  onAskUserAnswer?: (
    toolCallId: string,
    result: AskUserResult,
  ) => boolean | Promise<boolean>;
  ref?: Ref<HTMLElement>;
  sharing?: boolean;

  selected?: boolean;
  onFocus?: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
  onFocusLeave?: (event: React.MouseEvent<HTMLDivElement, MouseEvent>) => void;
  onMenuOpenChange?: (open: boolean) => void;
};

function MessageSegment(props: MessageProps) {
  const ref = useRef(null);
  const menuOpenRef = useRef(false);
  const { message } = props;

  return (
    <div
      className={`message ${message.role}`}
      ref={ref}
      onClick={props.onFocus}
      onMouseEnter={props.onFocus}
      onMouseLeave={(event) => {
        if (menuOpenRef.current) return;
        try {
          if (isContainDom(ref.current, event.relatedTarget as HTMLElement))
            return;
          props.onFocusLeave && props.onFocusLeave(event);
        } catch {
          props.onFocusLeave && props.onFocusLeave(event);
        }
      }}
    >
      <MessageContent
        {...props}
        onMenuOpenChange={(open) => {
          menuOpenRef.current = open;
          props.onMenuOpenChange?.(open);
        }}
      />
      <MessageQuota message={message} />
    </div>
  );
}

type MessageQuotaProps = {
  message: Message;
};

function MessageQuota({ message }: MessageQuotaProps) {
  const [detail, setDetail] = useState(false);
  const showQuota = useSelector(showQuotaSelector);
  const quota = Number(message.quota);

  if (
    !showQuota ||
    message.role === UserRole ||
    !Number.isFinite(quota) ||
    quota <= 0
  ) {
    return null;
  }

  return (
    <motion.div
      className={cn("message-quota", message.plan && "subscription")}
      onClick={() => setDetail(!detail)}
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.3, ease: "easeOut" }}
      whileHover={{ scale: 1.05 }}
      whileTap={{ scale: 0.95 }}
    >
      <motion.div
        initial={{ rotate: 0 }}
        animate={{ rotate: detail ? 360 : 0 }}
        transition={{ duration: 0.5, ease: "easeInOut" }}
      >
        {message.plan ? (
          <CalendarCheck2 className={`h-4 w-4 icon`} />
        ) : detail ? (
          <CloudCog className={`h-4 w-4 icon`} />
        ) : (
          <Cloud className={`h-4 w-4 icon`} />
        )}
      </motion.div>
      <motion.span
        className={`quota`}
        initial={{ y: 10, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ delay: 0.2, duration: 0.3 }}
      >
        {quota.toFixed(detail ? 6 : 2)}
      </motion.span>
    </motion.div>
  );
}

type MessageMenuProps = {
  children?: React.ReactNode;
  message: Message;
  end?: boolean;
  index: number;
  onEvent?: (event: string, index?: number, message?: string) => void;
  editedMessage?: string;
  setEditedMessage: (message: string) => void;
  setOpen: (open: boolean) => void;
  align?: "start" | "end";
  onOpenChange?: (open: boolean) => void;
};

function MessageMenu({
  children,
  align,
  message,
  end,
  index,
  onEvent,
  editedMessage,
  setEditedMessage,
  setOpen,
  onOpenChange,
}: MessageMenuProps) {
  const { t } = useTranslation();
  const isAssistant = message.role === "assistant";
  const notInOutput = message.end !== false;
  const disableDelete = isAssistant && end && !notInOutput;
  const [dropdown, setDropdown] = useState(false);

  const handleDropdownChange = (open: boolean) => {
    setDropdown(open);
    onOpenChange?.(open);
  };

  return (
    <DropdownMenu open={dropdown} onOpenChange={handleDropdownChange}>
      <DropdownMenuTrigger asChild>{children}</DropdownMenuTrigger>
      <DropdownMenuContent align={align}>
        {isAssistant && end ? (
          <DropdownMenuItem
            onClick={() => {
              onEvent && onEvent(message.end !== false ? "restart" : "stop");
              setDropdown(false);
            }}
          >
            {notInOutput ? (
              <>
                <RotateCcw className={`h-4 w-4 mr-1.5`} />
                {t("message.restart")}
              </>
            ) : (
              <>
                <Power className={`h-4 w-4 mr-1.5`} />
                {t("message.stop")}
              </>
            )}
          </DropdownMenuItem>
        ) : (
          notInOutput && (
            <DropdownMenuItem
              onClick={() => {
                onEvent && onEvent("restart");
                setDropdown(false);
              }}
            >
              <RotateCcw className={`h-4 w-4 mr-1.5`} />
              {t("message.restart")}
            </DropdownMenuItem>
          )
        )}
        <DropdownMenuItem
          onClick={() => copyClipboard(filterMessage(message.content))}
        >
          <Copy className={`h-4 w-4 mr-1.5`} />
          {t("message.copy")}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => {
            const input = document.getElementById("input") as HTMLInputElement;
            if (input) {
              input.value = filterMessage(message.content);
              input.focus();
            }
          }}
        >
          <SquareMousePointer className={`h-4 w-4 mr-1.5`} />
          {t("message.use")}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={disableDelete}
          onClick={() => {
            editedMessage?.length === 0 && setEditedMessage(message.content);
            setOpen(true);
          }}
        >
          <PencilLine className={`h-4 w-4 mr-1.5`} />
          {t("message.edit")}
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={disableDelete}
          onClick={() => onEvent && onEvent("remove", index)}
        >
          <Trash className={`h-4 w-4 mr-1.5`} />
          {t("message.remove")}
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() =>
            saveAsFile(
              `message-${message.role}.txt`,
              filterMessage(message.content),
            )
          }
        >
          <File className={`h-4 w-4 mr-1.5`} />
          {t("message.save")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

type MessageMarkdownProps = {
  content: string;
  loading?: boolean;
};

function MessageMarkdown({ content, loading }: MessageMarkdownProps) {
  const parts = useMemo(() => splitFileMessage(content), [content]);
  const hasFiles = parts.some((part) => part.type === "file");

  if (!hasFiles) {
    return <Markdown loading={loading} children={content} acceptHtml={false} />;
  }

  return (
    <>
      {parts.map((part, index) => {
        if (part.type === "text") {
          if (part.content.trim().length === 0) return null;

          return (
            <Markdown
              key={`text-${index}`}
              loading={loading}
              children={part.content}
              acceptHtml={false}
            />
          );
        }

        return (
          <MarkdownFile
            key={`file-${index}`}
            children={getFileMarkdown(part.file)}
          />
        );
      })}
    </>
  );
}

function MessageContent({
  message,
  end,
  index,
  onEvent,
  onAskUserAnswer,
  selected,
  username,
  model,
  onMenuOpenChange,
}: MessageProps) {
  const isUser = message.role === "user";
  const hasContent = message.content.length > 0;
  const isAssistant = message.role === "assistant";
  const isOutput = message.end === false || message.status === "streaming";
  const askUserToolCalls = isAssistant
    ? getAskUserToolCalls(message.tool_calls)
    : [];
  const hasAskUserToolCalls = askUserToolCalls.length > 0;
  const isPendingAskUserOnlyMessage =
    !hasContent &&
    hasAskUserToolCalls &&
    askUserToolCalls.every(isPendingAskUserToolCall);
  const user = useSelector(selectUsername);
  const supportModels = useSelector(selectSupportModels);

  const [open, setOpen] = useState(false);
  const [editedMessage, setEditedMessage] = useState<string | undefined>("");

  const modelId = model;
  const messageModel = supportModels.find((item) => item.id === modelId) || {
    id: modelId || "assistant",
    name: modelId || "assistant",
    avatar: "",
  };
  const useModelAvatar = !isUser && !selected;

  const parsedContent =
    isAssistant && message.content.length
      ? parseThinkContent(message.content)
      : null;

  return (
    <div className={"content-wrapper"}>
      <EditorProvider
        submittable={true}
        onSubmit={(value) => onEvent && onEvent("edit", index, value)}
        open={open}
        setOpen={setOpen}
        value={editedMessage ?? ""}
        onChange={setEditedMessage}
      />
      <div
        className={cn(
          "message-avatar-wrapper",
          useModelAvatar && "message-avatar-wrapper-round",
        )}
      >
        {!selected ? (
          isUser ? (
            <Avatar
              className={`message-avatar animate-fade-in`}
              username={username ?? user}
            />
          ) : (
            <ModelAvatar
              model={messageModel}
              size={36}
              className={`message-avatar animate-fade-in`}
            />
          )
        ) : (
          <MessageMenu
            message={message}
            end={end}
            index={index}
            onEvent={onEvent}
            editedMessage={editedMessage}
            setEditedMessage={setEditedMessage}
            setOpen={setOpen}
            align={isUser ? "end" : "start"}
            onOpenChange={onMenuOpenChange}
          >
            <button
              type="button"
              className={`message-avatar flex flex-row items-center justify-center cursor-pointer select-none opacity-0 animate-fade-in border-0 bg-transparent p-0 text-foreground`}
              onClick={(event) => event.stopPropagation()}
            >
              <PencilLine className={`h-4 w-4`} />
            </button>
          </MessageMenu>
        )}
      </div>
      <div
        className={cn(
          "relative message-content",
          isPendingAskUserOnlyMessage
            ? "ask-user-only-message"
            : "dark:bg-muted/40 border dark:border-transparent hover:border-border",
        )}
      >
        {hasContent ? (
          <>
            {parsedContent ? (
              <>
                <ThinkContent
                  content={parsedContent.thinkContent}
                  isComplete={parsedContent.isComplete}
                />
                {parsedContent.restContent && (
                  <MessageMarkdown
                    loading={isOutput}
                    content={parsedContent.restContent}
                  />
                )}
              </>
            ) : (
              <MessageMarkdown loading={isOutput} content={message.content} />
            )}
          </>
        ) : message.end === true && !hasAskUserToolCalls ? (
          <CircleSlash className={`h-5 w-5 m-1`} />
        ) : !hasAskUserToolCalls ? (
          <Loader2 className={`h-5 w-5 m-1 animate-spin`} />
        ) : null}

        {askUserToolCalls.map((toolCall) => (
          <AskUserCard
            key={toolCall.id}
            toolCall={toolCall}
            onSubmit={async (toolCallId, result) =>
              (await onAskUserAnswer?.(toolCallId, result)) ?? false
            }
          />
        ))}

        {isAssistant && hasContent && isOutput && (
          <Loader2
            className={`absolute right-0 bottom-0 h-3.5 w-3.5 m-1 animate-spin`}
          />
        )}
      </div>
    </div>
  );
}

export default MessageSegment;
