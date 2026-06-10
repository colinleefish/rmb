"use client";

import { useMemo } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";
import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import { CategoryBadge } from "@/components/category-badge";
import {
  DetailBadges,
  DetailMeta,
  DetailText,
  DetailUri,
  OutlineBadge,
} from "@/components/detail";
import { listAtoms } from "@/lib/api";
import { fmtDateTime, pick } from "@/lib/format";
import type { AtomModel } from "@/lib/types";

function topicOf(a: AtomModel): string {
  const scene = pick(a, "SceneName", "scene_name");
  const slug = pick(a, "Slug", "slug");
  return [scene, slug].filter(Boolean).join(" · ");
}

function detailOf(a: AtomModel): RowDetail {
  const priority = pick<number>(a, "Priority", "priority");
  const created = pick(a, "CreatedAt", "created_at");
  return {
    title: pick(a, "SceneName", "scene_name") ?? pick(a, "Slug", "slug") ?? "Atom",
    description: pick(a, "URI", "uri"),
    body: (
      <>
        <DetailBadges>
          <CategoryBadge category={pick(a, "Category", "category")} />
          <OutlineBadge>P{priority ?? "—"}</OutlineBadge>
        </DetailBadges>
        <DetailText>{pick(a, "Content", "content")}</DetailText>
        {created && <DetailMeta>{fmtDateTime(created)}</DetailMeta>}
        <DetailUri>{pick(a, "URI", "uri")}</DetailUri>
      </>
    ),
  };
}

export function AtomsTable() {
  const columns = useMemo<ColumnDef<AtomModel>[]>(
    () => [
      {
        id: "content",
        accessorFn: (a) => pick(a, "Content", "content") ?? "",
        header: "Fact",
        cell: ({ row }) => (
          <p className="text-foreground line-clamp-2 max-w-md">
            {pick(row.original, "Content", "content")}
          </p>
        ),
      },
      {
        id: "category",
        accessorFn: (a) => pick(a, "Category", "category") ?? "",
        header: ({ column }) => <SortButton column={column} label="Category" />,
        cell: ({ row }) => (
          <CategoryBadge category={pick(row.original, "Category", "category")} />
        ),
      },
      {
        id: "topic",
        header: "Topic",
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {topicOf(row.original) || "—"}
          </span>
        ),
      },
      {
        id: "priority",
        accessorFn: (a) => pick<number>(a, "Priority", "priority") ?? 0,
        header: ({ column }) => <SortButton column={column} label="Priority" />,
        cell: ({ row }) => {
          const p = pick<number>(row.original, "Priority", "priority") ?? 0;
          return (
            <Badge
              variant="outline"
              className={
                p >= 70
                  ? "border-amber-600/30 bg-amber-500/10 text-amber-700"
                  : "text-muted-foreground"
              }
            >
              {p}
            </Badge>
          );
        },
      },
    ],
    [],
  );

  return (
    <DataTable
      load={listAtoms}
      columns={columns}
      searchText={(a) =>
        [
          pick(a, "Content", "content"),
          pick(a, "Category", "category"),
          pick(a, "SceneName", "scene_name"),
          pick(a, "Slug", "slug"),
          pick(a, "URI", "uri"),
        ]
          .filter(Boolean)
          .join(" ")
      }
      searchPlaceholder="Search facts, category, topic…"
      emptyMessage="No atoms yet."
      renderDetail={detailOf}
    />
  );
}
