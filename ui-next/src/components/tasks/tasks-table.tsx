"use client";

import { useMemo } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import {
  ServerDataTable,
  SortButton,
  type RowDetail,
} from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import {
  DetailBadges,
  DetailMeta,
  DetailText,
  DetailUri,
  OutlineBadge,
} from "@/components/detail";
import { pageTasks } from "@/lib/api";
import { fmtDateShort, fmtDateTime, pick, shortKey } from "@/lib/format";
import type { TaskModel } from "@/lib/types";

function detailOf(t: TaskModel): RowDetail {
  const error = pick(t, "Error", "error");
  const progress = pick<number>(t, "Progress", "progress") ?? 0;
  const session = pick(t, "SessionID", "session_id");
  const created = pick(t, "CreatedAt", "created_at");
  return {
    title: pick(t, "Kind", "kind") ?? "Task",
    description: pick(t, "ResultURI", "result_uri") ?? undefined,
    body: (
      <>
        <DetailBadges>
          <StatusBadge status={pick(t, "Status", "status")} />
          <OutlineBadge>{progress}%</OutlineBadge>
        </DetailBadges>
        {session && <DetailUri>session {session}</DetailUri>}
        {error ? (
          <DetailText>{error}</DetailText>
        ) : (
          <DetailMeta>No error reported.</DetailMeta>
        )}
        {created && <DetailMeta>{fmtDateTime(created)}</DetailMeta>}
      </>
    ),
  };
}

export function TasksTable() {
  const columns = useMemo<ColumnDef<TaskModel>[]>(
    () => [
      {
        id: "kind",
        accessorFn: (t) => pick(t, "Kind", "kind") ?? "",
        header: ({ column }) => <SortButton column={column} label="Job" />,
        cell: ({ row }) => {
          const t = row.original;
          const error = pick(t, "Error", "error");
          return (
            <div className="flex max-w-md flex-col">
              <span className="text-foreground font-medium">
                {pick(t, "Kind", "kind")}
              </span>
              {error && (
                <span className="text-muted-foreground line-clamp-1 text-xs">
                  {error}
                </span>
              )}
            </div>
          );
        },
      },
      {
        id: "status",
        accessorFn: (t) => pick(t, "Status", "status") ?? "",
        header: ({ column }) => <SortButton column={column} label="Status" />,
        cell: ({ row }) => (
          <StatusBadge status={pick(row.original, "Status", "status")} />
        ),
      },
      {
        id: "session",
        header: "Session",
        cell: ({ row }) => (
          <span className="text-muted-foreground font-mono text-xs">
            {shortKey(pick(row.original, "SessionID", "session_id"))}
          </span>
        ),
      },
      {
        id: "progress",
        accessorFn: (t) => pick<number>(t, "Progress", "progress") ?? 0,
        header: ({ column }) => <SortButton column={column} label="Progress" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {pick<number>(row.original, "Progress", "progress") ?? 0}%
          </span>
        ),
      },
      {
        id: "created",
        accessorFn: (t) => pick(t, "CreatedAt", "created_at") ?? "",
        header: ({ column }) => <SortButton column={column} label="Created" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {fmtDateShort(pick(row.original, "CreatedAt", "created_at"))}
          </span>
        ),
      },
    ],
    [],
  );

  return (
    <ServerDataTable
      loadPage={pageTasks}
      columns={columns}
      searchPlaceholder="Search tasks…"
      emptyMessage="No tasks yet."
      initialSorting={[{ id: "created", desc: true }]}
      renderDetail={detailOf}
    />
  );
}
