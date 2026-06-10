"use client";

import { useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";

import { Badge } from "@/components/ui/badge";
import { DataTable, SortButton } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { SessionDetailDialog } from "@/components/sessions/session-detail-dialog";
import { listSessions } from "@/lib/api";
import { fmtDateShort } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { SessionRow } from "@/lib/types";

export function SessionsTable() {
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  const columns = useMemo<ColumnDef<SessionRow>[]>(
    () => [
      {
        accessorKey: "title",
        header: ({ column }) => <SortButton column={column} label="Session" />,
        cell: ({ row }) => {
          const s = row.original;
          const title = s.title?.trim();
          return (
            <div className="flex flex-col">
              <span
                className={cn(
                  "text-foreground font-medium",
                  !title && "font-mono",
                )}
              >
                {title || s.session_key}
              </span>
              <span className="text-muted-foreground text-xs">
                Updated {fmtDateShort(s.updated_at)}
              </span>
            </div>
          );
        },
      },
      {
        accessorKey: "turn_count",
        header: ({ column }) => <SortButton column={column} label="Turns" />,
        cell: ({ row }) => (
          <Badge variant="secondary" className="font-mono">
            {row.original.turn_count}
          </Badge>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => <StatusBadge status={row.original.status} />,
      },
    ],
    [],
  );

  return (
    <>
      <DataTable
        load={listSessions}
        columns={columns}
        searchText={(s) =>
          [s.title, s.session_key, s.abstract, s.status]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search title, key, status…"
        emptyMessage="No sessions yet."
        onRowClick={(s) => setSelectedKey(s.session_key)}
      />
      <SessionDetailDialog
        sessionKey={selectedKey}
        onOpenChange={(open) => {
          if (!open) setSelectedKey(null);
        }}
      />
    </>
  );
}
