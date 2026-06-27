"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { type ColumnDef } from "@tanstack/react-table";

import { ServerDataTable, SortButton } from "@/components/data-table";
import { SessionPipelineSummary } from "@/components/sessions/session-pipeline-summary";
import { pageSessions } from "@/lib/api";
import { fmtDateTime, sessionDisplayTitle } from "@/lib/format";
import type { SessionRow } from "@/lib/types";

export function SessionsTable() {
  const router = useRouter();

  const columns = useMemo<ColumnDef<SessionRow>[]>(
    () => [
      {
        id: "title",
        enableSorting: false,
        accessorFn: (s) =>
          s.title?.trim() || s.abstract?.trim() || s.session_key,
        header: "Session",
        cell: ({ row }) => {
          const s = row.original;
          const label = sessionDisplayTitle(s);
          return (
            <div className="flex max-w-xl flex-col gap-0.5">
              <span className="text-foreground line-clamp-2 font-medium leading-snug">
                {label}
              </span>
              <span className="text-muted-foreground break-all font-mono text-xs">
                {s.uri}
              </span>
            </div>
          );
        },
      },
      {
        id: "pipeline",
        header: "Distillation",
        cell: ({ row }) => (
          <SessionPipelineSummary session={row.original} />
        ),
      },
      {
        id: "created",
        accessorFn: (s) => s.created_at,
        header: ({ column }) => <SortButton column={column} label="Created" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm whitespace-nowrap tabular-nums">
            {fmtDateTime(row.original.created_at)}
          </span>
        ),
      },
      {
        id: "updated",
        accessorFn: (s) => s.last_turn_at ?? "",
        header: ({ column }) => <SortButton column={column} label="Updated" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm whitespace-nowrap tabular-nums">
            {fmtDateTime(row.original.last_turn_at)}
          </span>
        ),
      },
    ],
    [],
  );

  return (
    <ServerDataTable
      loadPage={pageSessions}
      columns={columns}
      searchPlaceholder="Search title, summary, key…"
      emptyMessage="No sessions yet."
      initialSorting={[{ id: "updated", desc: true }]}
      onRowClick={(s) =>
        router.push(`/sessions/${encodeURIComponent(s.session_key)}`)
      }
    />
  );
}
