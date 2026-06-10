import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./ui/dialog.tsx";
import {
  Maximize,
  Image,
  MenuSquare,
  PanelRight,
  Eraser,
  RotateCcw,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import "@/assets/common/editor.less";
import { Textarea } from "./ui/textarea.tsx";
import Markdown from "./Markdown.tsx";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { Toggle } from "./ui/toggle.tsx";
import { mobile } from "@/utils/device.ts";
import { Button } from "./ui/button.tsx";
import { ChatAction } from "@/components/home/assemblies/ChatAction.tsx";
import { cn } from "@/components/ui/lib/utils.ts";

type RichEditorProps = {
  value: string;
  onChange: (value: string) => void;
  maxLength?: number;

  formatter?: (value: string) => string;
  isInvalid?: (value: string) => boolean;
  title?: string;
  defaultValue?: string;
  defaultLabel?: string;

  open?: boolean;
  setOpen?: (open: boolean) => void;
  children?: React.ReactNode;

  submittable?: boolean;
  onSubmit?: (value: string) => void;
  closeOnSubmit?: boolean;
};

function RichEditor({
  value,
  onChange,
  maxLength,
  formatter,
  submittable,
  isInvalid,
  onSubmit,
  setOpen,
  closeOnSubmit,
  defaultValue,
  defaultLabel,
}: RichEditorProps) {
  const { t } = useTranslation();
  const input = useRef(null);
  const [openPreview, setOpenPreview] = useState(!mobile);
  const [openInput, setOpenInput] = useState(true);

  const formattedValue = useMemo(() => {
    return formatter ? formatter(value) : value;
  }, [value, formatter]);
  const invalid = useMemo(() => {
    return isInvalid ? isInvalid(value) : false;
  }, [value, isInvalid]);

  useEffect(() => {
    if (!input.current) return;
    const target = input.current as HTMLElement;

    const syncScroll = () => {
      const preview = target.parentElement?.querySelector(
        ".editor-preview",
      ) as HTMLElement | null;
      if (!preview) return;
      preview.scrollTop = target.scrollTop;
    };

    target.addEventListener("scroll", syncScroll);
    if (openInput) target.focus();

    return () => {
      target.removeEventListener("scroll", syncScroll);
    };
  }, [openInput, openPreview]);

  return (
    <div className={`editor-container`}>
      <div className={`editor-toolbar`}>
        <Button
          variant={`outline`}
          className={`h-8 w-8 p-0`}
          onClick={() => value && onChange("")}
        >
          <Eraser className={`h-3.5 w-3.5`} />
        </Button>
        {typeof defaultValue === "string" && (
          <Button
            variant={`outline`}
            className={`h-8 w-8 p-0`}
            title={defaultLabel ?? t("default-config")}
            aria-label={defaultLabel ?? t("default-config")}
            onClick={() => onChange(defaultValue)}
          >
            <RotateCcw className={`h-3.5 w-3.5`} />
          </Button>
        )}
        <div className={`grow`} />
        <Toggle
          variant={`outline`}
          className={`h-8 w-8 p-0`}
          pressed={openInput && !openPreview}
          onClick={() => {
            setOpenPreview(false);
            setOpenInput(true);
          }}
        >
          <MenuSquare className={`h-3.5 w-3.5`} />
        </Toggle>

        <Toggle
          variant={`outline`}
          className={`h-8 w-8 p-0`}
          pressed={openInput && openPreview}
          onClick={() => {
            setOpenPreview(true);
            setOpenInput(true);
          }}
        >
          <PanelRight className={`h-3.5 w-3.5`} />
        </Toggle>

        <Toggle
          variant={`outline`}
          className={`h-8 w-8 p-0`}
          pressed={!openInput && openPreview}
          onClick={() => {
            setOpenPreview(true);
            setOpenInput(false);
          }}
        >
          <Image className={`h-3.5 w-3.5`} />
        </Toggle>
      </div>
      <div className={`editor-wrapper`}>
        <div
          className={cn(
            "editor-object",
            openInput && "show-editor",
            openPreview && "show-preview",
          )}
        >
          {openInput && (
            <Textarea
              placeholder={t("chat.placeholder-raw")}
              value={value}
              className={cn(
                `editor-input thin-scrollbar transition-all`,
                invalid && `error-border`,
              )}
              id={`editor`}
              maxLength={maxLength}
              onChange={(e) => onChange(e.target.value)}
              ref={input}
            />
          )}
          {openPreview &&
            (formattedValue ? (
              <Markdown
                className={`editor-preview thin-scrollbar`}
                children={formattedValue}
              />
            ) : (
              <div
                className={`editor-preview thin-scrollbar inline-flex text-secondary text-xs items-center justify-center whitespace-pre-wrap`}
              >
                <Image
                  className={`h-3.5 w-3.5 mr-1 shrink-0 inline-flex translate-y-[1px]`}
                />
                {t("chat.empty-preview")}
              </div>
            ))}
        </div>
      </div>
      {submittable && (
        <div className={`editor-footer mt-2 flex flex-row`}>
          <Button
            variant={`outline`}
            className={`ml-auto mr-2`}
            onClick={() => setOpen?.(false)}
          >
            {t("cancel")}
          </Button>
          <Button
            variant={`default`}
            onClick={() => {
              onSubmit?.(value);
              (closeOnSubmit ?? true) && setOpen?.(false);
            }}
          >
            {t("submit")}
          </Button>
        </div>
      )}
    </div>
  );
}

function EditorProvider(props: RichEditorProps) {
  const { t } = useTranslation();

  return (
    <>
      <Dialog open={props.open} onOpenChange={props.setOpen}>
        {!props.setOpen && (
          <DialogTrigger asChild>
            {props.children ?? (
              <ChatAction text={t("editor")} className={`hidden md:flex`}>
                <Maximize className={`h-4 w-4`} />
              </ChatAction>
            )}
          </DialogTrigger>
        )}
        <DialogContent className={`editor-dialog flex-dialog`} couldFullScreen>
          <DialogHeader>
            <DialogTitle>{props.title ?? t("edit")}</DialogTitle>
            <DialogDescription asChild>
              <RichEditor {...props} />
            </DialogDescription>
          </DialogHeader>
        </DialogContent>
      </Dialog>
    </>
  );
}

export default EditorProvider;

export function JSONEditorProvider({ ...props }: RichEditorProps) {
  return (
    <EditorProvider
      {...props}
      formatter={(value) => `\`\`\`json\n${value}\n\`\`\``}
      isInvalid={(value) => {
        try {
          JSON.parse(value);
          return false;
        } catch (e) {
          return true;
        }
      }}
    />
  );
}
