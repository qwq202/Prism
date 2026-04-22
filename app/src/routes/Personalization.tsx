import { useTranslation } from "react-i18next";
import { useDispatch, useSelector } from "react-redux";
import * as settings from "@/store/settings.ts";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Textarea } from "@/components/ui/textarea.tsx";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";
import { Switch } from "@/components/ui/switch.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { cn } from "@/components/ui/lib/utils.ts";
import Icon from "@/components/utils/Icon.tsx";
import {
  deleteMemory,
  listMemories,
  updateMemory,
  type MemoryRecord,
} from "@/api/memory.ts";
import { motion } from "framer-motion";
import {
  Bot,
  Brain,
  Palette,
  UserRound,
  HelpCircle,
  Search,
  MoreHorizontal,
  Pencil,
  Trash2,
} from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

type SelectOption = {
  value: string;
  label: string;
  desc?: string;
};

type MemoryItem = {
  id: number;
  content: string;
  date: string;
  source: string;
  category?: string;
};

function formatMemoryDate(value: string | undefined, locale: string) {
  if (!value) return "";

  const parsed = new Date(value.replace(" ", "T"));
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale || "zh-CN", {
    month: "short",
    day: "numeric",
  }).format(parsed);
}

function MemoryDialog({
  open,
  onOpenChange,
  memories,
  loading,
  onDelete,
  onUpdate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  memories: MemoryRecord[];
  loading: boolean;
  onDelete: (id: number) => Promise<void>;
  onUpdate: (
    id: number,
    content: string,
    category?: string,
  ) => Promise<boolean>;
}) {
  const { t, i18n } = useTranslation();
  const [search, setSearch] = useState("");
  const [editingMemory, setEditingMemory] = useState<MemoryItem | null>(null);
  const [editingContent, setEditingContent] = useState("");
  const [saving, setSaving] = useState(false);
  const filteredMemories: MemoryItem[] = useMemo(
    () =>
      memories
        .filter((item) =>
          item.content.toLowerCase().includes(search.trim().toLowerCase()),
        )
        .map((item) => ({
          id: item.id,
          content: item.content,
          date:
            formatMemoryDate(item.updated_at || item.created_at, i18n.language) ||
            t("settings.personalization.memory.just-now"),
          source: item.source || t("settings.personalization.memory.source.chat"),
          category: item.category,
        })),
    [i18n.language, memories, search, t],
  );

  const closeEditor = () => {
    setEditingMemory(null);
    setEditingContent("");
    setSaving(false);
  };

  const handleStartEdit = (item: MemoryItem) => {
    setEditingMemory(item);
    setEditingContent(item.content);
  };

  const handleSubmitEdit = async () => {
    if (!editingMemory) return;

    const nextContent = editingContent.trim();
    if (!nextContent) return;

    setSaving(true);
    try {
      const ok = await onUpdate(editingMemory.id, nextContent, editingMemory.category);
      if (ok) {
        closeEditor();
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl p-0 overflow-hidden gap-0">
          <DialogHeader className="p-6 pb-2">
            <DialogTitle className="text-xl font-semibold">
              {t("settings.personalization.memory.dialog-title")}
            </DialogTitle>
            <DialogDescription className="text-sm mt-1">
              {t("settings.personalization.memory.dialog-description")}
              <a href="#" className="text-primary hover:underline ml-1">
                {t("learn-more")}
              </a>
            </DialogDescription>
          </DialogHeader>

          <div className="px-6 py-4 flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
              <Input
                placeholder={t("settings.personalization.memory.search-placeholder")}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9 h-10 bg-muted/30 border-none rounded-full"
              />
            </div>
          </div>

          <div className="border-t">
            <ScrollArea className="h-[400px]">
              <div className="flex flex-col">
                {loading && (
                  <div className="px-6 py-8 text-sm text-muted-foreground">
                    {t("settings.personalization.memory.loading")}
                  </div>
                )}
                {!loading && filteredMemories.length === 0 && (
                  <div className="px-6 py-8 text-sm text-muted-foreground">
                    {t("settings.personalization.memory.empty")}
                  </div>
                )}
                {filteredMemories.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-start justify-between p-6 hover:bg-muted/30 transition-colors border-b last:border-0"
                  >
                    <p className="text-sm leading-relaxed flex-1 pr-4 min-w-0">
                      {item.content}
                    </p>
                    <div className="flex shrink-0">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            type="button"
                            className="p-1.5 rounded-md transition-colors hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                            aria-label={t("settings.personalization.memory.more")}
                          >
                            <MoreHorizontal className="w-4 h-4 text-muted-foreground" />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-52 p-1.5">
                          <DropdownMenuItem
                            onClick={() => handleStartEdit(item)}
                            className="cursor-pointer"
                          >
                            <Pencil className="w-4 h-4 mr-2" />
                            {t("edit")}
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => void onDelete(item.id)}
                            className="text-destructive focus:text-destructive focus:bg-destructive/10 cursor-pointer"
                          >
                            <Trash2 className="w-4 h-4 mr-2" />
                            {t("delete")}
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <div className="px-2 py-1.5 text-[10px] text-muted-foreground leading-relaxed">
                            {t("settings.personalization.memory.meta", {
                              date: item.date,
                            })}
                            <span className="underline cursor-pointer">
                              {item.source}
                            </span>
                          </div>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editingMemory} onOpenChange={(nextOpen) => !nextOpen && closeEditor()}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {t("settings.personalization.memory.edit-title")}
            </DialogTitle>
            <DialogDescription>
              {t("settings.personalization.memory.edit-description")}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            value={editingContent}
            onChange={(e) => setEditingContent(e.target.value)}
            placeholder={t("settings.personalization.memory.edit-placeholder")}
            className="min-h-[140px] resize-y"
          />
          <DialogFooter>
            <button
              type="button"
              className="inline-flex items-center justify-center rounded-md border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted"
              onClick={closeEditor}
              disabled={saving}
            >
              {t("close")}
            </button>
            <button
              type="button"
              className="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
              onClick={() => void handleSubmitEdit()}
              disabled={saving || !editingContent.trim()}
            >
              {t("save")}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

type InlineSelectItemProps = {
  label: string;
  description?: string;
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
};

function InlineSelectItem({
  label,
  description,
  value,
  options,
  onChange,
}: InlineSelectItemProps) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center justify-between py-3 gap-2">
      <div className="flex flex-col gap-1 pr-4">
        <span className="text-sm font-medium text-foreground">{label}</span>
        {description && (
          <span className="text-xs text-muted-foreground">{description}</span>
        )}
      </div>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className="w-full sm:w-[180px] shrink-0 h-9 text-sm [&_[data-desc]]:hidden">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((opt) => (
            <SelectItem
              key={opt.value}
              value={opt.value}
              textValue={opt.label}
            >
              {opt.desc ? (
                <div className="flex flex-col gap-0.5 py-0.5">
                  <span className="text-sm">{opt.label}</span>
                  <span data-desc className="text-xs text-muted-foreground hidden sm:inline-block">
                    {opt.desc}
                  </span>
                </div>
              ) : (
                opt.label
              )}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

type InlineSwitchItemProps = {
  label: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
};

function InlineSwitchItem({
  label,
  description,
  checked,
  onCheckedChange,
}: InlineSwitchItemProps) {
  return (
    <div className="flex items-center justify-between py-3 gap-4">
      <div className="flex flex-col gap-1">
        <span className="text-sm font-medium text-foreground">{label}</span>
        {description && (
          <span className="text-xs text-muted-foreground">{description}</span>
        )}
      </div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}

type PersonalizationCardProps = {
  title: string;
  description?: string;
  icon?: React.ReactElement;
  children: React.ReactNode;
  className?: string;
  headerAction?: React.ReactNode;
  showHelp?: boolean;
};

function PersonalizationCard({
  title,
  description,
  icon,
  children,
  className,
  headerAction,
  showHelp,
}: PersonalizationCardProps) {
  return (
    <div
      className={cn(
        "flex flex-col bg-background rounded-lg shadow border overflow-hidden",
        className
      )}
    >
      <div className="select-none inline-flex flex-row items-center justify-between h-fit w-full border-b px-4 py-3 bg-muted/20">
        <div className="flex flex-row items-center">
          <div className="flex items-center mr-3">
            {icon && (
              <Icon
                icon={icon}
                className="w-8 h-8 p-2 rounded-lg bg-muted text-secondary"
              />
            )}
          </div>
          <div className="flex flex-col">
            <div className="flex items-center gap-1.5">
              <p className="text-sm font-medium">{title}</p>
              {showHelp && (
                <HelpCircle className="w-3.5 h-3.5 text-muted-foreground cursor-help" />
              )}
            </div>
            {description && (
              <p className="text-xs text-secondary mt-0.5">{description}</p>
            )}
          </div>
        </div>
        {headerAction}
      </div>
      <div className="p-4 sm:p-5 flex flex-col gap-4">
        {children}
      </div>
    </div>
  );
}

function Personalization() {
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const personaStyle = useSelector(settings.personaStyleSelector);
  const personaWarmth = useSelector(settings.personaWarmthSelector);
  const personaEnthusiasm = useSelector(settings.personaEnthusiasmSelector);
  const personaLists = useSelector(settings.personaListsSelector);
  const personaEmoji = useSelector(settings.personaEmojiSelector);
  const personaCustomInstruction = useSelector(
    settings.personaCustomInstructionSelector,
  );
  const personaNickname = useSelector(settings.personaNicknameSelector);
  const personaOccupation = useSelector(settings.personaOccupationSelector);
  const personaAboutUser = useSelector(settings.personaAboutUserSelector);
  const memoryEnabled = useSelector(settings.memoryEnabledSelector);
  const historyEnabled = useSelector(settings.memoryHistoryEnabledSelector);

  const [memoryDialogOpen, setMemoryDialogOpen] = useState(false);
  const [memories, setMemories] = useState<MemoryRecord[]>([]);
  const [memoryLoading, setMemoryLoading] = useState(false);

  const loadMemories = async (query?: string) => {
    setMemoryLoading(true);
    try {
      setMemories(await listMemories(query));
    } finally {
      setMemoryLoading(false);
    }
  };

  useEffect(() => {
    if (!memoryDialogOpen) return;
    void loadMemories();
  }, [memoryDialogOpen]);

  const handleDeleteMemory = async (id: number) => {
    const resp = await deleteMemory(id);
    if (!resp.status) {
      toast.error(t("settings.personalization.memory.delete-failed"), {
        description:
          resp.message || t("settings.personalization.memory.delete-failed-tip"),
      });
      return;
    }

    setMemories((current) => current.filter((item) => item.id !== id));
  };

  const handleUpdateMemory = async (
    id: number,
    content: string,
    category?: string,
  ) => {
    const resp = await updateMemory(id, content, category);
    if (!resp.status || !resp.data) {
      toast.error(t("settings.personalization.memory.edit-failed"), {
        description:
          resp.message || t("settings.personalization.memory.edit-failed-tip"),
      });
      return false;
    }

    setMemories((current) =>
      current.map((item) => (item.id === id ? resp.data! : item)),
    );
    return true;
  };

  const styleOptions: SelectOption[] = [
    {
      value: "default",
      label: t("settings.personalization.options.style.default"),
      desc: t("settings.personalization.options.style.default-desc"),
    },
    {
      value: "professional",
      label: t("settings.personalization.options.style.professional"),
      desc: t("settings.personalization.options.style.professional-desc"),
    },
    {
      value: "friendly",
      label: t("settings.personalization.options.style.friendly"),
      desc: t("settings.personalization.options.style.friendly-desc"),
    },
    {
      value: "direct",
      label: t("settings.personalization.options.style.direct"),
      desc: t("settings.personalization.options.style.direct-desc"),
    },
    {
      value: "creative",
      label: t("settings.personalization.options.style.creative"),
      desc: t("settings.personalization.options.style.creative-desc"),
    },
    {
      value: "efficient",
      label: t("settings.personalization.options.style.efficient"),
      desc: t("settings.personalization.options.style.efficient-desc"),
    },
    {
      value: "sarcastic",
      label: t("settings.personalization.options.style.sarcastic"),
      desc: t("settings.personalization.options.style.sarcastic-desc"),
    },
  ];

  const warmthOptions: SelectOption[] = [
    {
      value: "high",
      label: t("settings.personalization.options.level.high"),
      desc: t("settings.personalization.options.level.high-desc-warmth"),
    },
    {
      value: "default",
      label: t("settings.personalization.options.level.default"),
    },
    {
      value: "low",
      label: t("settings.personalization.options.level.low"),
      desc: t("settings.personalization.options.level.low-desc-warmth"),
    },
  ];

  const enthusiasmOptions: SelectOption[] = [
    {
      value: "high",
      label: t("settings.personalization.options.level.high"),
      desc: t("settings.personalization.options.level.high-desc-enthusiasm"),
    },
    {
      value: "default",
      label: t("settings.personalization.options.level.default"),
    },
    {
      value: "low",
      label: t("settings.personalization.options.level.low"),
      desc: t("settings.personalization.options.level.low-desc-enthusiasm"),
    },
  ];

  const listOptions: SelectOption[] = [
    {
      value: "structured",
      label: t("settings.personalization.options.list.structured"),
      desc: t("settings.personalization.options.list.structured-desc"),
    },
    {
      value: "balanced",
      label: t("settings.personalization.options.list.balanced"),
      desc: t("settings.personalization.options.list.balanced-desc"),
    },
    {
      value: "default",
      label: t("settings.personalization.options.list.default"),
    },
    {
      value: "minimal",
      label: t("settings.personalization.options.list.minimal"),
      desc: t("settings.personalization.options.list.minimal-desc"),
    },
  ];

  const emojiOptions: SelectOption[] = [
    {
      value: "expressive",
      label: t("settings.personalization.options.emoji.expressive"),
      desc: t("settings.personalization.options.emoji.expressive-desc"),
    },
    {
      value: "light",
      label: t("settings.personalization.options.emoji.light"),
      desc: t("settings.personalization.options.emoji.light-desc"),
    },
    {
      value: "default",
      label: t("settings.personalization.options.emoji.default"),
    },
    {
      value: "none",
      label: t("settings.personalization.options.emoji.none"),
      desc: t("settings.personalization.options.emoji.none-desc"),
    },
  ];

  const pageVariants = {
    hidden: { opacity: 0, y: 18 },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.35,
        ease: "easeOut",
        when: "beforeChildren",
        staggerChildren: 0.08,
      },
    },
  };

  const cardVariants = {
    hidden: { opacity: 0, y: 22 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.4, ease: "easeOut" },
    },
  };

  return (
    <ScrollArea className="relative w-full h-full flex flex-col bg-background">
      <MemoryDialog
        open={memoryDialogOpen}
        onOpenChange={setMemoryDialogOpen}
        memories={memories}
        loading={memoryLoading}
        onDelete={handleDeleteMemory}
        onUpdate={handleUpdateMemory}
      />
      <motion.div
        className="px-4 py-6 md:py-12 lg:py-16 h-full flex flex-col w-full max-w-3xl mx-auto space-y-6"
        variants={pageVariants}
        initial="hidden"
        animate="visible"
      >
        <motion.div variants={cardVariants}>
          <PersonalizationCard
            icon={<Brain />}
            title={t("settings.personalization.memory.title")}
            showHelp
            headerAction={
              <button
                className="text-xs font-medium px-3 py-1 rounded-full border bg-background hover:bg-muted transition-colors"
                onClick={() => setMemoryDialogOpen(true)}
              >
                {t("manage")}
              </button>
            }
          >
            <div className="flex flex-col divide-y divide-border/50">
              <InlineSwitchItem
                label={t("settings.personalization.memory.saved-label")}
                description={t("settings.personalization.memory.saved-desc")}
                checked={memoryEnabled}
                onCheckedChange={(checked) =>
                  dispatch(settings.setMemoryEnabled(checked))
                }
              />
              <InlineSwitchItem
                label={t("settings.personalization.memory.history-label")}
                description={t("settings.personalization.memory.history-desc")}
                checked={historyEnabled}
                onCheckedChange={(checked) =>
                  dispatch(settings.setMemoryHistoryEnabled(checked))
                }
              />
            </div>
            <div className="pt-2">
              <p className="text-xs text-muted-foreground leading-relaxed">
                {t("settings.personalization.memory.footer")}
                <a href="#" className="text-primary hover:underline ml-1">
                  {t("learn-more")}
                </a>
              </p>
            </div>
          </PersonalizationCard>
        </motion.div>

        <motion.div variants={cardVariants}>
          <PersonalizationCard
            icon={<Palette />}
            title={t("settings.personalization.base-style")}
            description={t("settings.personalization.base-style-tip")}
          >
            <div className="flex flex-col divide-y divide-border/50">
              <InlineSelectItem
                label={t("settings.personalization.base-style")}
                value={personaStyle}
                options={styleOptions}
                onChange={(v) => dispatch(settings.setPersonaStyle(v))}
              />
              <InlineSelectItem
                label={t("settings.personalization.warmth")}
                value={personaWarmth}
                options={warmthOptions}
                onChange={(v) => dispatch(settings.setPersonaWarmth(v))}
              />
              <InlineSelectItem
                label={t("settings.personalization.enthusiasm")}
                value={personaEnthusiasm}
                options={enthusiasmOptions}
                onChange={(v) => dispatch(settings.setPersonaEnthusiasm(v))}
              />
              <InlineSelectItem
                label={t("settings.personalization.headings-lists")}
                value={personaLists}
                options={listOptions}
                onChange={(v) => dispatch(settings.setPersonaLists(v))}
              />
              <InlineSelectItem
                label={t("settings.personalization.emoji")}
                value={personaEmoji}
                options={emojiOptions}
                onChange={(v) => dispatch(settings.setPersonaEmoji(v))}
              />
            </div>
          </PersonalizationCard>
        </motion.div>

        <motion.div variants={cardVariants}>
          <PersonalizationCard
            icon={<Bot />}
            title={t("settings.personalization.custom-instruction")}
            description={t("settings.personalization.custom-instruction-placeholder")}
          >
            <div className="flex flex-col gap-2">
              <Textarea
                rows={4}
                value={personaCustomInstruction}
                placeholder={t("settings.personalization.custom-instruction-placeholder")}
                className="resize-y min-h-[100px] text-sm"
                onChange={(e) =>
                  dispatch(settings.setPersonaCustomInstruction(e.target.value))
                }
              />
            </div>
          </PersonalizationCard>
        </motion.div>

        <motion.div variants={cardVariants}>
          <PersonalizationCard
            icon={<UserRound />}
            title={t("settings.personalization.about-you")}
            description={t("settings.personalization.about-user-tip")}
          >
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="flex flex-col gap-2">
                <span className="text-sm font-medium text-foreground">
                  {t("settings.personalization.nickname")}
                </span>
                <Input
                  value={personaNickname}
                  placeholder={t("settings.personalization.nickname-placeholder")}
                  className="h-9 text-sm"
                  onChange={(e) =>
                    dispatch(settings.setPersonaNickname(e.target.value))
                  }
                />
              </div>
              <div className="flex flex-col gap-2">
                <span className="text-sm font-medium text-foreground">
                  {t("settings.personalization.occupation")}
                </span>
                <Input
                  value={personaOccupation}
                  placeholder={t("settings.personalization.occupation-placeholder")}
                  className="h-9 text-sm"
                  onChange={(e) =>
                    dispatch(settings.setPersonaOccupation(e.target.value))
                  }
                />
              </div>
            </div>
            <div className="flex flex-col gap-2 mt-2">
              <span className="text-sm font-medium text-foreground">
                {t("settings.personalization.about-user")}
              </span>
              <Textarea
                rows={4}
                value={personaAboutUser}
                placeholder={t("settings.personalization.about-user-placeholder")}
                className="resize-y min-h-[100px] text-sm"
                onChange={(e) =>
                  dispatch(settings.setPersonaAboutUser(e.target.value))
                }
              />
            </div>
          </PersonalizationCard>
        </motion.div>
      </motion.div>
    </ScrollArea>
  );
}

export default Personalization;
