"use client";

import { cn } from "@/lib/utils";
import type { SkillFileNode } from "@/lib/types";
import { FileCode, FileText, Folder } from "lucide-react";

function TreeNode({
  node,
  selected,
  onSelect,
  depth = 0,
}: {
  node: SkillFileNode;
  selected: string | null;
  onSelect: (path: string) => void;
  depth?: number;
}) {
  const isDir = node.type === "dir";
  const isSelected = !isDir && node.path === selected;
  const Icon = isDir ? Folder : node.path === "SKILL.md" ? FileText : FileCode;

  return (
    <div>
      <button
        type="button"
        onClick={() => !isDir && onSelect(node.path)}
        className={cn(
          "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm",
          isDir
            ? "text-muted-foreground cursor-default"
            : "hover:bg-muted/60",
          isSelected && "bg-muted font-medium",
        )}
        style={{ paddingLeft: `${depth * 12 + 8}px` }}
        disabled={isDir}
      >
        <Icon className="size-3.5 shrink-0 opacity-70" />
        <span className="truncate">{node.name}</span>
      </button>
      {node.children?.map((child) => (
        <TreeNode
          key={child.path}
          node={child}
          selected={selected}
          onSelect={onSelect}
          depth={depth + 1}
        />
      ))}
    </div>
  );
}

export function SkillFileTree({
  tree,
  selected,
  onSelect,
}: {
  tree: SkillFileNode[];
  selected: string | null;
  onSelect: (path: string) => void;
}) {
  return (
    <div className="space-y-0.5">
      {tree.map((node) => (
        <TreeNode
          key={node.path}
          node={node}
          selected={selected}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}
