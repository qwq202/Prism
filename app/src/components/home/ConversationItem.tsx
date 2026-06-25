import { phone } from "@/utils/device.ts";
import { filterMessage } from "@/utils/processor.ts";
import { setMenu } from "@/store/menu.ts";
import {
  Loader2,
  MessageSquare,
  MessagesSquare,
  MoreHorizontal,
  PencilLine,
  Share2,
  Sparkles,
  Star,
  Trash2,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { useDispatch, useSelector } from "react-redux";
import { useTranslation } from "react-i18next";
import { ConversationInstance } from "@/api/types.tsx";
import { useEffect, useRef, useState } from "react";
import { useConversationActions } from "@/store/chat.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import PopupDialog, { popupTypes } from "@/components/PopupDialog.tsx";
import { withNotify } from "@/api/common.ts";
import Clickable from "@/components/ui/clickable.tsx";
import { infoHasTaskModelSelector } from "@/store/info.ts";

type ConversationItemProps = {
  conversation: ConversationInstance;
  current: number;
  operate: (conversation: {
    target: ConversationInstance;
    type: string;
  }) => void;
};

function getRetitleName(data: unknown): string | undefined {
  if (!data || typeof data !== "object" || !("name" in data)) return undefined;

  const { name } = data as { name?: unknown };
  return typeof name === "string" ? name : undefined;
}

function ConversationItem({
  conversation,
  current,
  operate,
}: ConversationItemProps) {
  const conversationTitle = filterMessage(conversation.name);
  const dispatch = useDispatch();
  const hasTaskModel = useSelector(infoHasTaskModelSelector);
  const { t } = useTranslation();
  const { toggle, rename, retitle, favorite } = useConversationActions();
  const [open, setOpen] = useState(false);
  const [offset, setOffset] = useState(0);
  const menuInteractionUntil = useRef(0);

  const [editDialog, setEditDialog] = useState(false);
  const [displayTitle, setDisplayTitle] = useState(conversationTitle);
  const [retitleStage, setRetitleStage] = useState<
    "idle" | "loading" | "typing"
  >("idle");
  const [typingTarget, setTypingTarget] = useState("");

  const pendingLocal = conversation.id === -1;
  const loading = conversation.id === 0;

  const suppressConversationToggle = () => {
    menuInteractionUntil.current = Date.now() + 800;
  };

  const shouldSuppressConversationToggle = () => {
    return open || menuInteractionUntil.current > Date.now();
  };

  const stopMenuEvent = (event: {
    preventDefault: () => void;
    stopPropagation: () => void;
  }) => {
    suppressConversationToggle();
    event.preventDefault();
    event.stopPropagation();
  };

  useEffect(() => {
    if (retitleStage !== "idle") return;
    setDisplayTitle(conversationTitle);
  }, [conversationTitle, retitleStage]);

  useEffect(() => {
    if (retitleStage !== "loading") return;

    const loadingText = t("conversation.retitle-loading");
    let dots = 1;
    setDisplayTitle(`${loadingText}.`);

    const timer = window.setInterval(() => {
      dots = (dots % 3) + 1;
      setDisplayTitle(`${loadingText}${".".repeat(dots)}`);
    }, 350);

    return () => window.clearInterval(timer);
  }, [retitleStage, t]);

  useEffect(() => {
    if (retitleStage !== "typing") return;

    const target = filterMessage(typingTarget || conversation.name);
    const chars = Array.from(target);
    let index = 0;
    setDisplayTitle("");

    if (chars.length === 0) {
      setRetitleStage("idle");
      setTypingTarget("");
      return;
    }

    const timer = window.setInterval(() => {
      index += 1;
      setDisplayTitle(chars.slice(0, index).join(""));
      if (index >= chars.length) {
        window.clearInterval(timer);
        setRetitleStage("idle");
        setTypingTarget("");
      }
    }, 40);

    return () => window.clearInterval(timer);
  }, [conversation.name, retitleStage, typingTarget]);

  return (
    <Clickable
      tapScale={0.975}
      tapDuration={0.01}
      className={cn("conversation", current === conversation.id && "active")}
      onClick={async (e) => {
        if (shouldSuppressConversationToggle()) {
          e.preventDefault();
          e.stopPropagation();
          return;
        }

        const target = e.target as HTMLElement;
        if (
          target.classList.contains("delete") ||
          target.parentElement?.classList.contains("delete")
        )
          return;
        await toggle(conversation.id);
        if (phone) dispatch(setMenu(false));
      }}
      onContextMenu={(e) => {
        e.preventDefault();
        e.stopPropagation();
        setOpen(true);
      }}
    >
      <MessageSquare
        className={`h-6 w-6 p-1 mr-1 text-muted-foreground bg-muted/60 rounded-sm`}
      />
      <div className={`title`}>{displayTitle}</div>
      {conversation.favorite && (
        <Star className={`h-3.5 w-3.5 mx-1 fill-amber-400 text-amber-500`} />
      )}
      {pendingLocal ? (
        <div className={`id`}>
          <MoreHorizontal className={`h-4 w-4 mr-0.5 text-muted-foreground`} />
        </div>
      ) : (
        <DropdownMenu
          open={open}
          onOpenChange={(state: boolean) => {
            suppressConversationToggle();
            setOpen(state);
            if (state) setOffset(new Date().getTime());
          }}
        >
          <DropdownMenuTrigger
            className={`flex flex-row outline-none`}
            onPointerDown={(e) => {
              suppressConversationToggle();
              e.stopPropagation();
            }}
            onClick={(e) => {
              stopMenuEvent(e);
            }}
          >
            <div className={cn("id", loading && "loading")}>
              {loading ? (
                <Loader2 className={`mr-0.5 h-4 w-4 animate-spin`} />
              ) : (
                <MoreHorizontal className={`h-4 w-4 mr-0.5`} />
              )}
            </div>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align={`end`}
            onPointerDown={(e) => {
              suppressConversationToggle();
              e.stopPropagation();
            }}
            onClick={(e) => {
              suppressConversationToggle();
              e.stopPropagation();
            }}
          >
            <DropdownMenuLabel
              className={`inline-flex conversation-id text-left py-0.5 w-full`}
            >
              {conversation.id}

              <MessagesSquare
                className={`inline h-3.5 w-3.5 ml-auto translate-y-0.5 text-muted-foreground`}
              />
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <PopupDialog
              title={t("conversation.edit-title")}
              open={editDialog}
              setOpen={setEditDialog}
              type={popupTypes.Text}
              name={t("title")}
              defaultValue={conversation.name}
              onSubmit={async (name) => {
                const resp = await rename(conversation.id, name);
                withNotify(t, resp, true);
                if (!resp.status) return false;

                setEditDialog(false);
                return true;
              }}
            />
            <DropdownMenuItem
              onClick={(e) => {
                // prevent click event from opening the dropdown menu
                if (offset + 500 > new Date().getTime()) return;

                stopMenuEvent(e);

                setEditDialog(true);
              }}
            >
              <PencilLine className={`h-4 w-4 mx-1`} />
              {t("conversation.edit-title")}
            </DropdownMenuItem>
            {hasTaskModel && (
              <DropdownMenuItem
                disabled={retitleStage !== "idle"}
                onClick={async (e) => {
                  if (offset + 500 > new Date().getTime()) return;

                  stopMenuEvent(e);

                  if (retitleStage !== "idle") return;

                  setOpen(false);
                  setRetitleStage("loading");

                  const resp = await retitle(conversation.id);
                  if (!resp.status) {
                    setRetitleStage("idle");
                    withNotify(t, resp);
                    return;
                  }

                  setTypingTarget(
                    getRetitleName(resp.data) ?? conversation.name,
                  );
                  setRetitleStage("typing");
                }}
              >
                <Sparkles className={`h-4 w-4 mx-1`} />
                {t("conversation.retitle")}
              </DropdownMenuItem>
            )}
            <DropdownMenuItem
              disabled={loading}
              onSelect={async (e) => {
                if (offset + 500 > new Date().getTime()) return;

                stopMenuEvent(e);

                const nextFavorite = !conversation.favorite;
                setOpen(false);

                const resp = await favorite(conversation.id, nextFavorite);
                withNotify(
                  t,
                  resp,
                  true,
                  t(
                    nextFavorite
                      ? "conversation.favorite-success"
                      : "conversation.unfavorite-success",
                  ),
                );
              }}
            >
              <Star
                className={cn(
                  `h-4 w-4 mx-1`,
                  conversation.favorite && `fill-amber-400 text-amber-500`,
                )}
              />
              {conversation.favorite
                ? t("conversation.unfavorite")
                : t("conversation.favorite")}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={(e) => {
                stopMenuEvent(e);
                operate({ target: conversation, type: "share" });

                setOpen(false);
              }}
            >
              <Share2 className={`h-4 w-4 mx-1`} />
              {t("share.share-conversation")}
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={(e) => {
                stopMenuEvent(e);
                operate({ target: conversation, type: "delete" });

                setOpen(false);
              }}
            >
              <Trash2 className={`h-4 w-4 mx-1`} />
              {t("conversation.delete-conversation")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </Clickable>
  );
}

export default ConversationItem;
