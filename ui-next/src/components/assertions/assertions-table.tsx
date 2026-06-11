"use client";

import { useMemo } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import {
  DetailBadges,
  DetailLead,
  DetailMeta,
  DetailUri,
  OutlineBadge,
} from "@/components/detail";
import { listAssertions } from "@/lib/api";
import { fmtDateShort, fmtDateTime, truncate } from "@/lib/format";
import type { AssertionModel } from "@/lib/types";

function detailOf(a: AssertionModel): RowDetail {
  return {
    title: a.kind === "forget" ? "Retired" : "Correction",
    description: a.uri,
    body: (
      <>
        <DetailBadges>
          <OutlineBadge>{a.kind}</OutlineBadge>
        </DetailBadges>
        {a.statement && <DetailLead>{a.statement}</DetailLead>}
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
            Targets
          </span>
          {a.target_uris.length === 0 ? (
            <DetailMeta>None</DetailMeta>
          ) : (
            a.target_uris.map((t) => <DetailUri key={t}>{t}</DetailUri>)
          )}
        </div>
        {a.created_at && (
          <DetailMeta>Created {fmtDateTime(a.created_at)}</DetailMeta>
        )}
      </>
    ),
  };
}

export function AssertionsTable() {
  const columns = useMemo<ColumnDef<AssertionModel>[]>(
    () => [
      {
        id: "statement",
        accessorFn: (a) => a.statement ?? "",
        header: "Statement",
        cell: ({ row }) => (
          <p className="text-foreground line-clamp-2 max-w-md">
            {row.original.statement || "—"}
          </p>
        ),
      },
      {
        id: "kind",
        accessorFn: (a) => a.kind ?? "",
        header: ({ column }) => <SortButton column={column} label="Kind" />,
        cell: ({ row }) => <OutlineBadge>{row.original.kind}</OutlineBadge>,
      },
      {
        id: "targets",
        accessorFn: (a) => a.target_uris?.join(" ") ?? "",
        header: "Targets",
        cell: ({ row }) => {
          const targets = row.original.target_uris ?? [];
          const first = targets[0];
          return (
            <span className="text-muted-foreground font-mono text-xs">
              {first ? truncate(first, 48) : "—"}
              {targets.length > 1 && (
                <span className="text-muted-foreground/70">
                  {" "}
                  +{targets.length - 1}
                </span>
              )}
            </span>
          );
        },
      },
      {
        id: "created",
        accessorFn: (a) => a.created_at ?? "",
        header: ({ column }) => <SortButton column={column} label="Created" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {fmtDateShort(row.original.created_at)}
          </span>
        ),
      },
    ],
    [],
  );

  return (
    <DataTable
      load={() => listAssertions()}
      columns={columns}
      searchText={(a) =>
        [a.statement, a.kind, a.target_uris?.join(" "), a.uri]
          .filter(Boolean)
          .join(" ")
      }
      searchPlaceholder="Search assertions…"
      emptyMessage="No assertions yet."
      initialSorting={[{ id: "created", desc: true }]}
      renderDetail={detailOf}
    />
  );
}
