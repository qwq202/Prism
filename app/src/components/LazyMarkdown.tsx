import React, { Suspense } from "react";
import type MarkdownComponent from "@/components/Markdown.tsx";

type MarkdownProps = React.ComponentProps<typeof MarkdownComponent>;

const Markdown = React.lazy(() => import("@/components/Markdown.tsx"));

function LazyMarkdown(props: MarkdownProps) {
  return (
    <Suspense
      fallback={<div className={props.className}>{props.children}</div>}
    >
      <Markdown {...props} />
    </Suspense>
  );
}

export default LazyMarkdown;
