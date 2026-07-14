import React from "react";
import { Info } from "lucide-react";
import { cn } from "@/components/ui/lib/utils.ts";
import Markdown from "@/components/Markdown.tsx";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion.tsx";
import { Badge } from "@/components/ui/badge.tsx";

// ParagraphProps describes a collapsible section block used throughout the
// admin pages to group related settings under a titled accordion.
export type ParagraphProps = {
  // Renders a gold "Pro" badge next to the title for premium-only sections.
  isPro?: boolean;
  // Heading text shown in the accordion trigger.
  title?: string;
  children: React.ReactNode;
  className?: string;
  // Applies the `config-paragraph` variant used on the system config panels.
  configParagraph?: boolean;
  // When true the accordion can be collapsed back after expanding.
  isCollapsed?: boolean;
};

// Paragraph is a single-open accordion panel that groups a titled block of
// settings. It composes the lower-level ParagraphItem / ParagraphDescription /
// ParagraphSpace / ParagraphFooter helpers exported alongside it.
function Paragraph({
  title,
  children,
  className,
  configParagraph,
  isCollapsed,
  isPro,
}: ParagraphProps) {
  return (
    <Accordion type={`single`} collapsible={isCollapsed} defaultValue={"item"}>
      <AccordionItem
        value={`item`}
        className={cn(
          `paragraph`,
          configParagraph && `config-paragraph`,
          className,
        )}
      >
        <AccordionTrigger className={`paragraph-header`}>
          <div className={`paragraph-title flex flex-row items-center`}>
            {title ?? ""}
            {isPro && (
              <Badge className={`ml-2`} variant={`gold`}>
                Pro
              </Badge>
            )}
          </div>
        </AccordionTrigger>
        <AccordionContent className={`paragraph-content mt-2`}>
          {children}
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}

// ParagraphItem is a single row inside a Paragraph. Set rowLayout to lay its
// children out horizontally instead of the default stacked flow.
function ParagraphItem({
  children,
  className,
  rowLayout,
}: {
  children: React.ReactNode;
  className?: string;
  rowLayout?: boolean;
}) {
  return (
    <div className={cn("paragraph-item", className, rowLayout && "row-layout")}>
      {children}
    </div>
  );
}

type ParagraphDescriptionProps = {
  children: string;
  // Wraps the description in a bordered, padded box to set it off from items.
  border?: boolean;
  // Hides the leading Info icon (useful when the icon is redundant).
  hideIcon?: boolean;
  className?: string;
  // Extra class applied to the inner Markdown renderer rather than the wrapper.
  classNameMarkdown?: string;
};

// ParagraphDescription renders a short Markdown-formatted hint under a setting,
// optionally prefixed with an Info icon and wrapped in a bordered box.
export function ParagraphDescription({
  children,
  border,
  hideIcon,
  className,
  classNameMarkdown,
}: ParagraphDescriptionProps) {
  return (
    <div
      className={cn(
        "paragraph-description",
        border && `px-3 py-2 border rounded-lg`,
        className,
      )}
    >
      {!hideIcon && <Info size={16} />}
      <Markdown
        children={children}
        className={cn("leading-6", classNameMarkdown)}
      />
    </div>
  );
}

// ParagraphSpace inserts a vertical gap between adjacent ParagraphItem blocks.
export function ParagraphSpace() {
  return <div className={`paragraph-space`} />;
}

// ParagraphFooter renders an action area pinned to the bottom of a Paragraph
// (e.g. save/reset buttons for a config section).
function ParagraphFooter({ children }: { children: React.ReactNode }) {
  return <div className={`paragraph-footer`}>{children}</div>;
}

export default Paragraph;
export { ParagraphItem, ParagraphFooter };
