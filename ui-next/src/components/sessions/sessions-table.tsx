"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { type ColumnDef } from "@tanstack/react-table";
import { ChevronRight } from "lucide-react";

import { ServerDataTable, SortButton } from "@/components/data-table";
import { SessionPipelineSummary } from "@/components/sessions/session-pipeline-summary";
import { pageSessions } from "@/lib/api";
import { fmtDateTime, sessionDisplayTitle, statusTone } from "@/lib/format";
import { sessionDetailHref } from "@/lib/session-routes";
import { cn } from "@/lib/utils";
import type { SessionRow } from "@/lib/types";

const STATUS_DOT: Record<ReturnType<typeof statusTone>, string> = {
  neutral: "bg-muted-foreground/40",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  destructive: "bg-destructive",
  info: "bg-sky-500",
};

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
            <div className="flex max-w-xl items-start gap-2.5">
              <span
                className={cn(
                  "mt-1.5 size-2 shrink-0 rounded-full",
                  STATUS_DOT[statusTone(s.status)],
                )}
                title={s.status || "unknown"}
                aria-hidden
              />
              <div className="flex flex-col gap-0.5">
                <span className="text-foreground line-clamp-2 font-medium leading-snug">
                  {label}
                </span>
                <span className="text-muted-foreground break-all font-mono text-xs">
                  {s.uri}
                </span>
              </div>
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
      {
        id: "chevron",
        enableSorting: false,
        header: () => <span className="sr-only">Open</span>,
        cell: () => (
          <ChevronRight className="text-muted-foreground/40 group-hover/row:text-foreground size-4 transition-colors" />
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
      onRowClick={(s) => router.push(sessionDetailHref(s.session_key))}
    />
  );
}
