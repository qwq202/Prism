import { useTranslation } from "react-i18next";
import { useDispatch, useSelector } from "react-redux";
import { selectAuthenticated, selectInit } from "@/store/auth.ts";
import {
  selectCurrent,
  selectHistory,
  selectMaskItem,
  useConversationActions,
} from "@/store/chat.ts";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { ConversationInstance } from "@/api/types.tsx";
import { extractMessage, filterMessage } from "@/utils/processor.ts";
import { copyClipboard } from "@/utils/dom.ts";
import {
  useEffectAsync,
  useAnimation as animateElement,
} from "@/utils/hook.ts";
import { openWindow, phone } from "@/utils/device.ts";
import { Button } from "@/components/ui/button.tsx";
import { selectMenu, setMenu } from "@/store/menu.ts";
import {
  Copy,
  Eraser,
  Paintbrush,
  Plus,
  RotateCw,
  Search,
  Star,
  User,
} from "lucide-react";
import ConversationItem from "./ConversationItem.tsx";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog.tsx";
import { getSharedLink, shareConversation } from "@/api/sharing.ts";
import { Input } from "@/components/ui/input.tsx";
import { goAuth } from "@/utils/app.ts";
import { cn } from "@/components/ui/lib/utils.ts";
import { toast } from "sonner";
import { AnimatePresence, motion } from "framer-motion";
import { isConversationListCacheStorageKey } from "@/utils/conversation-cache.ts";

type Operation = {
  target: ConversationInstance | null;
  type: string;
};

type SidebarActionProps = {
  search: string;
  favoriteOnly: boolean;
  setSearch: (search: string) => void;
  setFavoriteOnly: (favoriteOnly: boolean) => void;
  setOperateConversation: (operation: Operation) => void;
};

type ConversationListProps = {
  search: string;
  favoriteOnly: boolean;
  operateConversation: Operation;
  setOperateConversation: (operation: Operation) => void;
};

const conversationAutoRefreshIntervalMs = 10_000;
const conversationFocusRefreshThrottleMs = 2_000;

function SidebarAction({
  search,
  favoriteOnly,
  setSearch,
  setFavoriteOnly,
  setOperateConversation,
}: SidebarActionProps) {
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const {
    toggle,
    refresh: refreshAction,
    removeAll: removeAllAction,
  } = useConversationActions();
  const refreshRef = useRef(null);
  const [removeAll, setRemoveAll] = useState<boolean>(false);

  const current = useSelector(selectCurrent);
  const mask = useSelector(selectMaskItem);

  async function handleDeleteAll(e: React.MouseEvent<HTMLButtonElement>) {
    e.preventDefault();
    e.stopPropagation();

    (await removeAllAction())
      ? toast.success(t("conversation.delete-success"), {
          description: t("conversation.delete-success-prompt"),
        })
      : toast.error(t("conversation.delete-failed"), {
          description: t("conversation.delete-failed-prompt"),
        });

    await refreshAction({ useCache: false });
    setOperateConversation({ target: null, type: "" });
    setRemoveAll(false);
  }

  return (
    <motion.div
      className={`sidebar-action-wrapper flex flex-col w-full h-fit px-1.5`}
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <motion.div
        className={`sidebar-action`}
        initial={{ scale: 0.9 }}
        animate={{ scale: 1 }}
        transition={{ duration: 0.3 }}
      >
        <motion.div whileTap={{ scale: 0.9 }}>
          <Button
            variant={`ghost`}
            size={`icon`}
            onClick={async () => {
              await toggle(-1);
              if (phone) dispatch(setMenu(false));
            }}
          >
            {current === -1 && mask ? (
              <Paintbrush className={`h-4 w-4`} />
            ) : (
              <Plus className={`h-4 w-4`} />
            )}
          </Button>
        </motion.div>
        <div className={`grow`} />
        <AlertDialog open={removeAll} onOpenChange={setRemoveAll}>
          <AlertDialogTrigger asChild>
            <motion.div whileTap={{ scale: 0.9 }}>
              <Button variant={`ghost`} size={`icon`}>
                <Eraser className={`h-4 w-4`} />
              </Button>
            </motion.div>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t("conversation.remove-all-title")}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t("conversation.remove-all-description")}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t("conversation.cancel")}</AlertDialogCancel>
              <Button
                variant={`destructive`}
                loading={true}
                onClick={handleDeleteAll}
                unClickable
              >
                {t("conversation.delete")}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <motion.div whileTap={{ scale: 0.9 }}>
          <Button
            variant={`ghost`}
            size={`icon`}
            aria-pressed={favoriteOnly}
            aria-label={t("conversation.favorites")}
            title={t("conversation.favorites")}
            className={cn(favoriteOnly && `bg-muted text-foreground`)}
            onClick={() => setFavoriteOnly(!favoriteOnly)}
          >
            <Star
              className={cn(
                `h-4 w-4`,
                favoriteOnly && `fill-amber-400 text-amber-500`,
              )}
            />
          </Button>
        </motion.div>
        <motion.div whileTap={{ scale: 0.9 }} className={`refresh-action`}>
          <Button
            variant={`ghost`}
            size={`icon`}
            id={`refresh`}
            ref={refreshRef}
            onClick={() => {
              const hook = animateElement(refreshRef, "active", 500);
              refreshAction({ useCache: false }).finally(hook);
            }}
          >
            <RotateCw className={`h-4 w-4`} />
          </Button>
        </motion.div>
      </motion.div>
      <motion.div
        className={`relative w-full h-fit`}
        initial={{ opacity: 0, x: -20 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.5, delay: 0.2 }}
      >
        <Search
          className={`absolute h-3.5 w-3.5 top-1/2 left-3.5 transform -translate-y-1/2`}
        />
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("conversation.search")}
          className={`w-full pl-9`}
        />
      </motion.div>
    </motion.div>
  );
}

function SidebarConversationList({
  search,
  favoriteOnly,
  operateConversation,
  setOperateConversation,
}: ConversationListProps) {
  const { t } = useTranslation();
  const { remove } = useConversationActions();
  const auth = useSelector(selectAuthenticated);
  const history: ConversationInstance[] = useSelector(selectHistory);
  const [shared, setShared] = useState<string>("");
  const current = useSelector(selectCurrent);

  const filteredHistory = useMemo(() => {
    const visibleHistory = favoriteOnly
      ? history.filter((conversation) => conversation.favorite)
      : history;

    if (search.trim().length === 0) return visibleHistory;

    const searchItems = search
      .trim()
      .toLowerCase()
      .split(" ")
      .filter((item) => item.length > 0);

    return visibleHistory.filter((conversation) => {
      const name = conversation.name.toLowerCase();
      const id = conversation.id.toString();
      return searchItems.every(
        (item) => name.includes(item) || id.includes(item),
      );
    });
  }, [favoriteOnly, history, search]);

  async function handleDelete(e: React.MouseEvent<HTMLButtonElement>) {
    e.preventDefault();
    e.stopPropagation();

    if (await remove(operateConversation?.target?.id || -1))
      toast.success(t("conversation.delete-success"), {
        description: t("conversation.delete-success-prompt"),
      });
    else
      toast.error(t("conversation.delete-failed"), {
        description: t("conversation.delete-failed-prompt"),
      });
    setOperateConversation({ target: null, type: "" });
  }

  async function handleShare(e: React.MouseEvent<HTMLButtonElement>) {
    e.preventDefault();
    e.stopPropagation();

    const resp = await shareConversation(operateConversation?.target?.id || -1);
    if (resp.status) setShared(getSharedLink(resp.data));
    else
      toast.error(t("share.failed"), {
        description: resp.message,
      });

    setOperateConversation({ target: null, type: "" });
  }

  return (
    <>
      <div className={`conversation-list`}>
        <AnimatePresence>
          {filteredHistory.length ? (
            filteredHistory.map((conversation, i) => (
              <motion.div
                key={conversation.local_key ?? conversation.id}
                initial={
                  conversation.id === -1 ? false : { opacity: 0, y: 20 }
                }
                animate={{ opacity: 1, y: 0 }}
                exit={
                  conversation.id === -1 ? undefined : { opacity: 0, y: -20 }
                }
                transition={
                  conversation.id === -1
                    ? { duration: 0 }
                    : { duration: 0.3, delay: i * 0.05 }
                }
              >
                <ConversationItem
                  operate={setOperateConversation}
                  conversation={conversation}
                  current={current}
                />
              </motion.div>
            ))
          ) : (
            <motion.div
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.5 }}
              className={`empty text-center px-6`}
            >
              {auth
                ? t(
                    favoriteOnly
                      ? "conversation.empty-favorites"
                      : "conversation.empty",
                  )
                : t("conversation.empty-anonymous")}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
      <AlertDialog
        open={
          operateConversation.type === "delete" && !!operateConversation.target
        }
        onOpenChange={(open) => {
          if (!open) setOperateConversation({ target: null, type: "" });
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("conversation.remove-title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("conversation.remove-description")}
              <strong className={`conversation-name`}>
                {extractMessage(
                  filterMessage(operateConversation?.target?.name || ""),
                )}
              </strong>
              {t("end")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("conversation.cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>
              {t("conversation.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={
          operateConversation.type === "share" && !!operateConversation.target
        }
        onOpenChange={(open) => {
          if (!open) setOperateConversation({ target: null, type: "" });
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("share.title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("share.description")}
              <strong className={`conversation-name`}>
                {extractMessage(
                  filterMessage(operateConversation?.target?.name || ""),
                )}
              </strong>
              {t("end")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("conversation.cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={handleShare}>
              {t("share.title")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog
        open={shared.length > 0}
        onOpenChange={(open) => {
          if (!open) {
            setShared("");
            setOperateConversation({ target: null, type: "" });
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("share.success")}</AlertDialogTitle>
            <AlertDialogDescription>
              <div className={`share-wrapper mt-4 mb-2`}>
                <Input value={shared} />
                <Button
                  variant={`default`}
                  size={`icon`}
                  onClick={async () => {
                    const copied = await copyClipboard(shared);
                    if (copied) {
                      toast.success(t("share.copied"), {
                        description: t("share.copied-description"),
                      });
                    } else {
                      toast.error(t("copied.failed"));
                    }
                  }}
                >
                  <Copy className={`h-4 w-4`} />
                </Button>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("close")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={async (e) => {
                e.preventDefault();
                e.stopPropagation();
                openWindow(shared, "_blank");
              }}
            >
              {t("share.view")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function SideBar() {
  const { t } = useTranslation();
  const { restore, refresh } = useConversationActions();
  const open = useSelector(selectMenu);
  const auth = useSelector(selectAuthenticated);
  const init = useSelector(selectInit);
  const refreshRef = useRef(refresh);
  const refreshingRef = useRef(false);
  const lastAutoRefreshRef = useRef(0);
  const [search, setSearch] = useState<string>("");
  const [favoriteOnly, setFavoriteOnly] = useState(false);
  const [operateConversation, setOperateConversation] = useState<Operation>({
    target: null,
    type: "",
  });

  useEffect(() => {
    refreshRef.current = refresh;
  }, [refresh]);

  useEffectAsync(async () => {
    if (!init || !auth) return;
    await restore({ useCache: false });
  }, [auth, init]);

  useEffect(() => {
    if (!init || !auth) return;

    const syncConversations = async (force = false) => {
      if (refreshingRef.current || document.visibilityState === "hidden") {
        return;
      }

      const now = Date.now();
      if (
        !force &&
        now - lastAutoRefreshRef.current < conversationFocusRefreshThrottleMs
      ) {
        return;
      }

      refreshingRef.current = true;
      try {
        await refreshRef.current({ useCache: false });
        lastAutoRefreshRef.current = Date.now();
      } finally {
        refreshingRef.current = false;
      }
    };

    const interval = window.setInterval(() => {
      void syncConversations();
    }, conversationAutoRefreshIntervalMs);
    const handleFocus = () => void syncConversations(true);
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        void syncConversations(true);
      }
    };

    window.addEventListener("focus", handleFocus);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    void syncConversations(true);

    return () => {
      window.clearInterval(interval);
      window.removeEventListener("focus", handleFocus);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [auth, init]);

  useEffect(() => {
    if (!init || !auth) return;

    const handleStorage = (event: StorageEvent) => {
      if (!isConversationListCacheStorageKey(event.key)) return;
      void refreshRef.current({ useCache: false });
    };

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [auth, init]);

  return (
    <div className={cn("sidebar", open && "open")}>
      <div className={`sidebar-content`}>
        <SidebarAction
          search={search}
          favoriteOnly={favoriteOnly}
          setSearch={setSearch}
          setFavoriteOnly={setFavoriteOnly}
          setOperateConversation={setOperateConversation}
        />
        <SidebarConversationList
          search={search}
          favoriteOnly={favoriteOnly}
          operateConversation={operateConversation}
          setOperateConversation={setOperateConversation}
        />
        {init && !auth && (
          <Button
            className={`login-action min-h-10 h-max`}
            variant={`default`}
            onClick={goAuth}
          >
            <User className={`h-4 w-4 mr-1.5 shrink-0`} /> {t("login-action")}
          </Button>
        )}
      </div>
    </div>
  );
}

export default SideBar;
