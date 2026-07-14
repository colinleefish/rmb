"use client";

import { ScrollArea } from "@/components/ui/scroll-area";

export function SkillFileViewer({
  path,
  content,
}: {
  path: string | null;
  content: string;
}) {
  if (!path) {
    return (
      <p className="text-muted-foreground text-sm">Select a file to preview.</p>
    );
  }

  const isMarkdown = path.endsWith(".md");

  return (
    <ScrollArea className="h-[min(70vh,640px)] rounded-md border bg-muted/20 p-4">
      <div className="mb-3 text-muted-foreground font-mono text-xs">{path}</div>
      {isMarkdown ? (
        <pre className="whitespace-pre-wrap text-sm leading-relaxed">{content}</pre>
      ) : (
        <pre className="font-mono text-xs leading-relaxed whitespace-pre-wrap">
          {content}
        </pre>
      )}
    </ScrollArea>
  );
}
