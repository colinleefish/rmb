"use client";

import { useMemo } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { ServerDataTable, SortButton, type RowDetail } from "@/components/data-table";
import {
  DetailLead,
  DetailMeta,
  DetailUri,
} from "@/components/detail";
import { pageCorrections } from "@/lib/api";
import { fmtDateShort, fmtDateTime, truncate } from "@/lib/format";
import type { CorrectionModel } from "@/lib/types";

function detailOf(a: CorrectionModel): RowDetail {
  return {
    title: "Correction",
    description: a.uri,
    body: (
      <>
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

export function CorrectionsTable() {
  const columns = useMemo<ColumnDef<CorrectionModel>[]>(
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
    <ServerDataTable
      loadPage={(req) => pageCorrections(req)}
      columns={columns}
      searchPlaceholder="Search corrections…"
      emptyMessage="No corrections yet."
      initialSorting={[{ id: "created", desc: true }]}
      renderDetail={detailOf}
    />
  );
}
