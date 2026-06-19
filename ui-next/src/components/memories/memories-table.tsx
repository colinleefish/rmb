"use client";

import { useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { ServerDataTable, SortButton } from "@/components/data-table";
import { CategoryBadge } from "@/components/category-badge";
import { MemoryDetailDialog } from "@/components/memories/memory-detail-dialog";
import { pageMemories } from "@/lib/api";
import { fmtDateShort, pick, truncate } from "@/lib/format";
import type { MemoryModel } from "@/lib/types";

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
          const primary =
            pick(m, "Abstract", "abstract") ??
            truncate(pick(m, "Body", "body"), 120);
          return (
            <p className="text-foreground line-clamp-2 max-w-md">
              {primary || "—"}
            </p>
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
        id: "slug",
        accessorFn: (m) => pick(m, "Slug", "slug") ?? "",
        header: "Slug",
        cell: ({ row }) => (
          <span className="text-muted-foreground font-mono text-xs">
            {pick(row.original, "Slug", "slug") ?? "—"}
          </span>
        ),
      },
      {
        id: "version",
        accessorFn: (m) => pick<number>(m, "Version", "version") ?? 0,
        header: ({ column }) => <SortButton column={column} label="Ver" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            v{pick<number>(row.original, "Version", "version") ?? "—"}
          </span>
        ),
      },
      {
        id: "updated",
        accessorFn: (m) => pick(m, "UpdatedAt", "updated_at") ?? "",
        header: ({ column }) => <SortButton column={column} label="Updated" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {fmtDateShort(pick(row.original, "UpdatedAt", "updated_at"))}
          </span>
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
