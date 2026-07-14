"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { type ColumnDef } from "@tanstack/react-table";
import { ChevronRight, Wand2 } from "lucide-react";

import { ServerDataTable, SortButton } from "@/components/data-table";
import { Badge } from "@/components/ui/badge";
import { pageSkills } from "@/lib/api";
import { fmtRelative } from "@/lib/format";
import { skillDetailHref } from "@/lib/skill-routes";
import { cn } from "@/lib/utils";
import type { SkillRow } from "@/lib/types";

const TAG_CLASS: Record<string, string> = {
  work: "border-emerald-600/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  personal:
    "border-violet-600/30 bg-violet-500/10 text-violet-700 dark:text-violet-300",
};

function SkillTag({ tag }: { tag: string }) {
  const key = tag.toLowerCase();
  return (
    <Badge
      variant="outline"
      className={cn("font-medium", TAG_CLASS[key] ?? "border-border")}
    >
      {tag}
    </Badge>
  );
}

export function SkillsTable() {
  const router = useRouter();

  const columns = useMemo<ColumnDef<SkillRow>[]>(
    () => [
      {
        id: "skill",
        accessorFn: (s) => `${s.name} ${s.description} ${(s.tags ?? []).join(" ")}`,
        header: "Skill",
        cell: ({ row }) => {
          const s = row.original;
          const tags = s.tags ?? [];
          return (
            <div className="flex max-w-2xl items-start gap-3 py-0.5">
              <span
                className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-violet-500/10 text-violet-600 dark:text-violet-300"
                aria-hidden
              >
                <Wand2 className="size-4" />
              </span>
              <div className="flex min-w-0 flex-col gap-1.5">
                <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                  <span className="text-foreground text-sm font-semibold leading-snug">
                    {s.name}
                  </span>
                  {tags.map((tag) => (
                    <SkillTag key={tag} tag={tag} />
                  ))}
                </div>
                <span className="text-muted-foreground font-mono text-xs">
                  {s.uri}
                </span>
                <p className="text-muted-foreground line-clamp-2 text-sm leading-relaxed">
                  {s.description}
                </p>
              </div>
            </div>
          );
        },
      },
      {
        accessorKey: "version",
        header: ({ column }) => <SortButton column={column} label="Revision" />,
        cell: ({ row }) => {
          const v = row.original.version;
          return (
            <div className="flex flex-col leading-tight">
              <span className="text-foreground font-mono text-xs">v{v}</span>
              <span className="text-muted-foreground text-[11px]">
                {v > 1 ? `${v} revisions` : "original"}
              </span>
            </div>
          );
        },
      },
      {
        accessorKey: "updated_at",
        header: ({ column }) => <SortButton column={column} label="Updated" />,
        cell: ({ row }) => (
          <div className="flex items-center justify-end gap-2">
            <span className="text-muted-foreground text-sm whitespace-nowrap">
              {fmtRelative(row.original.updated_at)}
            </span>
            <ChevronRight className="text-muted-foreground/40 group-hover/row:text-foreground size-4 shrink-0 transition-colors" />
          </div>
        ),
      },
    ],
    [],
  );

  return (
    <ServerDataTable
      columns={columns}
      loadPage={pageSkills}
      initialSorting={[{ id: "updated_at", desc: true }]}
      onRowClick={(row) => router.push(skillDetailHref(row.slug))}
      searchPlaceholder="Search skills…"
      emptyMessage="No skills yet."
    />
  );
}
