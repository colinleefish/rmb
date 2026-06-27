"use client";

import { useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import {
  BookOpen,
  ChevronRight,
  History,
  Lightbulb,
  Settings2,
  TriangleAlert,
  UserRound,
  type LucideIcon,
} from "lucide-react";

import { ServerDataTable, SortButton } from "@/components/data-table";
import { CategoryBadge } from "@/components/category-badge";
import { MemoryDetailDialog } from "@/components/memories/memory-detail-dialog";
import { pageMemories } from "@/lib/api";
import { fmtRelative, pick, truncate } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { MemoryModel } from "@/lib/types";

/** Icon + tint per memory category, falling back to a neutral knowledge mark. */
const CATEGORY_ICON: Record<string, { icon: LucideIcon; className: string }> = {
  profile: { icon: UserRound, className: "bg-violet-500/10 text-violet-600 dark:text-violet-300" },
  preference: { icon: Settings2, className: "bg-violet-500/10 text-violet-600 dark:text-violet-300" },
  preferences: { icon: Settings2, className: "bg-violet-500/10 text-violet-600 dark:text-violet-300" },
  decision: { icon: Lightbulb, className: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-300" },
  entities: { icon: BookOpen, className: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-300" },
  events: { icon: History, className: "bg-amber-500/10 text-amber-600 dark:text-amber-300" },
  issue: { icon: TriangleAlert, className: "bg-rose-500/10 text-rose-600 dark:text-rose-300" },
  knowledge: { icon: BookOpen, className: "bg-sky-500/10 text-sky-600 dark:text-sky-300" },
  fact: { icon: BookOpen, className: "bg-sky-500/10 text-sky-600 dark:text-sky-300" },
};

function categoryVisual(category: string | null | undefined) {
  const key = (category ?? "").toLowerCase();
  return CATEGORY_ICON[key] ?? { icon: BookOpen, className: "bg-muted text-muted-foreground" };
}

/** Turn a slug like "user-preferences" into "User preferences". */
function humanizeSlug(slug: string | null | undefined): string | null {
  if (!slug) return null;
  const words = slug.replace(/[-_]+/g, " ").trim();
  if (!words) return null;
  return words.charAt(0).toUpperCase() + words.slice(1);
}

export function MemoriesTable() {
  const [selected, setSelected] = useState<MemoryModel | null>(null);

  const columns = useMemo<ColumnDef<MemoryModel>[]>(
    () => [
      {
        id: "memory",
        accessorFn: (m) =>
          pick(m, "Abstract", "abstract") ?? pick(m, "Body", "body") ?? "",
        header: "Memory",
        cell: ({ row }) => {
          const m = row.original;
          const category = pick(m, "Category", "category");
          const { icon: Icon, className } = categoryVisual(category);
          const title =
            humanizeSlug(pick(m, "Slug", "slug")) ??
            (category
              ? category.charAt(0).toUpperCase() + category.slice(1)
              : "Memory");
          const abstract = pick(m, "Abstract", "abstract");
          const body = pick(m, "Body", "body");
          // Show the body as supporting detail only when it adds to the abstract.
          const detail =
            body && body !== abstract ? truncate(body, 140) : null;

          return (
            <div className="flex max-w-xl items-start gap-3">
              <span
                className={cn(
                  "mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md",
                  className,
                )}
                aria-hidden
              >
                <Icon className="size-4" />
              </span>
              <div className="flex min-w-0 flex-col gap-0.5">
                <span className="text-foreground text-sm font-semibold leading-snug">
                  {title}
                </span>
                {abstract && (
                  <span className="text-foreground/80 line-clamp-1 text-sm">
                    {abstract}
                  </span>
                )}
                {detail && (
                  <span className="text-muted-foreground line-clamp-1 text-xs leading-relaxed">
                    {detail}
                  </span>
                )}
              </div>
            </div>
          );
        },
      },
      {
        id: "category",
        accessorFn: (m) => pick(m, "Category", "category") ?? "",
        header: ({ column }) => <SortButton column={column} label="Category" />,
        cell: ({ row }) => (
          <CategoryBadge category={pick(row.original, "Category", "category")} />
        ),
      },
      {
        id: "version",
        accessorFn: (m) => pick<number>(m, "Version", "version") ?? 0,
        header: ({ column }) => <SortButton column={column} label="Revision" />,
        cell: ({ row }) => {
          const v = pick<number>(row.original, "Version", "version");
          return (
            <div className="flex flex-col leading-tight">
              <span className="text-foreground font-mono text-xs">
                v{v ?? "—"}
              </span>
              <span className="text-muted-foreground text-[11px]">
                {v && v > 1 ? `${v} revisions` : "original"}
              </span>
            </div>
          );
        },
      },
      {
        id: "updated",
        accessorFn: (m) => pick(m, "UpdatedAt", "updated_at") ?? "",
        header: ({ column }) => <SortButton column={column} label="Updated" />,
        cell: ({ row }) => (
          <div className="flex items-center justify-between gap-2">
            <span className="text-muted-foreground text-sm whitespace-nowrap">
              {fmtRelative(pick(row.original, "UpdatedAt", "updated_at"))}
            </span>
            <ChevronRight className="text-muted-foreground/40 group-hover/row:text-foreground size-4 shrink-0 transition-colors" />
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <>
      <ServerDataTable
        loadPage={pageMemories}
        columns={columns}
        searchPlaceholder="Search memories…"
        emptyMessage="No memories yet."
        initialSorting={[{ id: "updated", desc: true }]}
        onRowClick={setSelected}
      />
      <MemoryDetailDialog
        memory={selected}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      />
    </>
  );
}
