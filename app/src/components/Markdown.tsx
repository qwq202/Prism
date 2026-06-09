import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import remarkBreaks from "remark-breaks";
import rehypeKatex from "rehype-katex";
import rehypeRaw from "rehype-raw";
import "@/assets/markdown/all.less";
import { useEffect, useMemo } from "react";
import { cn } from "@/components/ui/lib/utils.ts";
import { normalizeLatexDelimiters } from "@/utils/markdown-math.ts";
import Label from "@/components/markdown/Label.tsx";
import Link from "@/components/markdown/Link.tsx";
import Code, { CodeProps } from "@/components/markdown/Code.tsx";
import Image from "@/components/markdown/Image.tsx";
import Video from "@/components/markdown/Video.tsx";

type RehypePlugin = NonNullable<
  React.ComponentProps<typeof ReactMarkdown>["rehypePlugins"]
>[number];

const rehypeRawPlugin = rehypeRaw as unknown as RehypePlugin;
const rehypeKatexPlugin = rehypeKatex as unknown as RehypePlugin;

type HastNode = {
  type?: string;
  tagName?: string;
  value?: string;
  properties?: Record<string, unknown>;
  children?: HastNode[];
};

const markdownHtmlBreakPlugin = (() => {
  const skipTags = new Set(["code", "pre"]);

  function splitHtmlBreaks(node: HastNode): HastNode[] {
    const value = node.value ?? "";
    const breakPattern = /<br\s*\/?>/gi;
    const parts: HastNode[] = [];
    let lastIndex = 0;
    let match: RegExpExecArray | null;

    while ((match = breakPattern.exec(value)) !== null) {
      if (match.index > lastIndex) {
        parts.push({
          type: "text",
          value: value.slice(lastIndex, match.index),
        });
      }

      parts.push({
        type: "element",
        tagName: "br",
        properties: {},
        children: [],
      });
      lastIndex = match.index + match[0].length;
    }

    if (lastIndex < value.length) {
      parts.push({ type: "text", value: value.slice(lastIndex) });
    }

    return parts.length > 0 ? parts : [node];
  }

  function transform(node: HastNode) {
    if (node.type === "element" && node.tagName && skipTags.has(node.tagName)) {
      return;
    }
    if (!node.children) return;

    node.children = node.children.flatMap((child) => {
      transform(child);
      if (
        child.type === "text" &&
        child.value &&
        /<br\s*\/?>/i.test(child.value)
      ) {
        return splitHtmlBreaks(child);
      }

      return [child];
    });
  }

  return function rehypeMarkdownHtmlBreaks() {
    return (tree: HastNode) => transform(tree);
  };
})() as RehypePlugin;

type MarkdownProps = {
  children: string;
  className?: string;
  acceptHtml?: boolean;
  codeStyle?: string;
  loading?: boolean;
};

function MarkdownContent({
  children,
  className,
  acceptHtml,
  codeStyle,
  loading,
}: MarkdownProps) {
  const normalizedChildren = useMemo(
    () => normalizeLatexDelimiters(children),
    [children],
  );

  useEffect(() => {
    document.querySelectorAll(".file-instance").forEach((el) => {
      const parent = el.parentElement as HTMLElement;
      if (!parent.classList.contains("file-block"))
        parent.classList.add("file-block");
    });
  }, [normalizedChildren]);

  const rehypePlugins = useMemo(() => {
    const plugins: NonNullable<
      React.ComponentProps<typeof ReactMarkdown>["rehypePlugins"]
    > = [rehypeKatexPlugin, markdownHtmlBreakPlugin];
    return acceptHtml ? [...plugins, rehypeRawPlugin] : plugins;
  }, [acceptHtml]);

  const components = useMemo(() => {
    return {
      p: Label,
      a: Link,
      img: (props: React.ImgHTMLAttributes<HTMLImageElement>) => {
        if (props.alt === "video") {
          return (
            <Video
              src={props.src || ""}
              alt={props.alt}
              className={props.className}
            />
          );
        }
        return <Image {...props} />;
      },
      code: (props: CodeProps) => (
        <Code {...props} loading={loading} codeStyle={codeStyle} />
      ),
    };
  }, [codeStyle, loading]);

  return (
    <ReactMarkdown
      remarkPlugins={[remarkMath, remarkGfm, remarkBreaks]}
      rehypePlugins={rehypePlugins}
      className={cn("markdown-body", className)}
      children={normalizedChildren}
      skipHtml={false}
      components={components}
    />
  );
}

function Markdown({
  children,
  acceptHtml,
  codeStyle,
  className,
  loading,
}: MarkdownProps) {
  // memoize the component
  return useMemo(
    () => (
      <MarkdownContent
        children={children}
        acceptHtml={acceptHtml}
        codeStyle={codeStyle}
        className={className}
        loading={loading}
      />
    ),
    [children, acceptHtml, codeStyle, className, loading],
  );
}

type CodeMarkdownProps = MarkdownProps & {
  filename: string;
  language?: string;
};

export function CodeMarkdown({
  filename,
  language,
  ...props
}: CodeMarkdownProps) {
  const suffix =
    language ?? (filename.includes(".") ? filename.split(".").pop() : "");
  const children = useMemo(() => {
    const content = props.children.toString();

    return `\`\`\`${suffix}\n${content}\n\`\`\``;
  }, [props.children, suffix]);

  return <Markdown {...props}>{children}</Markdown>;
}

export default Markdown;
