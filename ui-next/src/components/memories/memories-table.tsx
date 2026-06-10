"use client";

import { useMemo } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import { CategoryBadge } from "@/components/category-badge";
import {
  DetailBadges,
  DetailLead,
  DetailMeta,
  DetailText,
  DetailUri,
  OutlineBadge,
} from "@/components/detail";
import { listMemories } from "@/lib/api";
import { fmtDateShort, fmtDateTime, pick, truncate } from "@/lib/format";
import type { MemoryModel } from "@/lib/types";

function detailOf(m: MemoryModel): RowDetail {
  const abstract = pick(m, "Abstract", "abstract");
  const body = pick(m, "Body", "body");
  const version = pick<number>(m, "Version", "version");
  const updated = pick(m, "UpdatedAt", "updated_at");
  return {
    title:
      pick(m, "Slug", "slug") ?? pick(m, "Category", "category") ?? "Memory",
    description: pick(m, "URI", "uri"),
    body: (
      <>
        <DetailBadges>
          <CategoryBadge category={pick(m, "Category", "category")} />
          {version != null && <OutlineBadge>v{version}</OutlineBadge>}
        </DetailBadges>
        {abstract && <DetailLead>{abstract}</DetailLead>}
        {body && <DetailText>{body}</DetailText>}
        {updated && <DetailMeta>Updated {fmtDateTime(updated)}</DetailMeta>}
        <DetailUri>{pick(m, "URI", "uri")}</DetailUri>
      </>
    ),
  };
}

export function MemoriesTable() {
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
    <DataTable
      load={listMemories}
      columns={columns}
      searchText={(m) =>
        [
          pick(m, "Abstract", "abstract"),
          pick(m, "Body", "body"),
          pick(m, "Category", "category"),
          pick(m, "Slug", "slug"),
          pick(m, "URI", "uri"),
        ]
          .filter(Boolean)
          .join(" ")
      }
      searchPlaceholder="Search memories…"
      emptyMessage="No memories yet."
      initialSorting={[{ id: "updated", desc: true }]}
      renderDetail={detailOf}
    />
  );
}
