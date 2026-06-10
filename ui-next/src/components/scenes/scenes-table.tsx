"use client";

import { useMemo } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import {
  DetailLead,
  DetailMeta,
  DetailText,
  DetailUri,
} from "@/components/detail";
import { listScenes } from "@/lib/api";
import { fmtDateShort, fmtDateTime, pick, truncate } from "@/lib/format";
import type { SceneModel } from "@/lib/types";

function detailOf(s: SceneModel): RowDetail {
  const abstract = pick(s, "Abstract", "abstract");
  const body = pick(s, "Body", "body");
  const updated = pick(s, "UpdatedAt", "updated_at");
  return {
    title:
      pick(s, "DisplayName", "display_name") ??
      truncate(abstract, 60) ??
      "Scene",
    description: pick(s, "URI", "uri"),
    body: (
      <>
        {abstract && <DetailLead>{abstract}</DetailLead>}
        {body && <DetailText>{body}</DetailText>}
        {updated && <DetailMeta>Updated {fmtDateTime(updated)}</DetailMeta>}
        <DetailUri>{pick(s, "URI", "uri")}</DetailUri>
      </>
    ),
  };
}

export function ScenesTable() {
  const columns = useMemo<ColumnDef<SceneModel>[]>(
    () => [
      {
        id: "scene",
        accessorFn: (s) =>
          pick(s, "DisplayName", "display_name") ??
          pick(s, "Abstract", "abstract") ??
          "",
        header: "Scene",
        cell: ({ row }) => {
          const s = row.original;
          const name =
            pick(s, "DisplayName", "display_name") ??
            truncate(pick(s, "Abstract", "abstract"), 60) ??
            "Scene";
          return (
            <div className="flex max-w-md flex-col">
              <span className="text-foreground font-medium">{name}</span>
              <span className="text-muted-foreground font-mono text-xs">
                {truncate(pick(s, "URI", "uri"), 48)}
              </span>
            </div>
          );
        },
      },
      {
        id: "summary",
        header: "Summary",
        cell: ({ row }) => {
          const s = row.original;
          const summary =
            pick(s, "Abstract", "abstract") ??
            truncate(pick(s, "Body", "body"), 160);
          return (
            <p className="text-muted-foreground line-clamp-2 max-w-sm text-sm">
              {summary || "—"}
            </p>
          );
        },
      },
      {
        id: "updated",
        accessorFn: (s) => pick(s, "UpdatedAt", "updated_at") ?? "",
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
      load={listScenes}
      columns={columns}
      searchText={(s) =>
        [
          pick(s, "DisplayName", "display_name"),
          pick(s, "Abstract", "abstract"),
          pick(s, "Body", "body"),
          pick(s, "URI", "uri"),
        ]
          .filter(Boolean)
          .join(" ")
      }
      searchPlaceholder="Search scenes…"
      emptyMessage="No scenes yet."
      initialSorting={[{ id: "updated", desc: true }]}
      renderDetail={detailOf}
    />
  );
}
