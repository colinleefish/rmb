"use client";

import { useMemo } from "react";
import { useRouter } from "next/navigation";
import { type ColumnDef } from "@tanstack/react-table";

import { ServerDataTable } from "@/components/data-table";
import { StatusBadge } from "@/components/status-badge";
import { pagePipelineStates } from "@/lib/api";
import { pick, shortKey } from "@/lib/format";
import type { PipelineRow } from "@/lib/types";

export function PipelineTable() {
  const router = useRouter();

  const columns = useMemo<ColumnDef<PipelineRow>[]>(
    () => [
      {
        id: "session",
        accessorFn: (r) => r.session_key ?? "",
        header: "Session",
        cell: ({ row }) => (
          <span className="text-foreground font-mono text-sm">
            {shortKey(row.original.session_key)}
          </span>
        ),
      },
      {
        id: "t1",
        header: "T1 · atoms",
        cell: ({ row }) => (
          <StatusBadge status={pick(row.original, "T1Status", "t1_status")} />
        ),
      },
      {
        id: "t2",
        header: "T2 · scenes",
        cell: ({ row }) => (
          <StatusBadge status={pick(row.original, "T2Status", "t2_status")} />
        ),
      },
      {
        id: "t3",
        header: "T3 · memories",
        cell: ({ row }) => (
          <StatusBadge status={pick(row.original, "T3Status", "t3_status")} />
        ),
      },
      {
        id: "warmup",
        accessorFn: (r) =>
          pick<number>(r, "WarmupThreshold", "warmup_threshold") ?? 0,
        header: "Warmup",
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {pick<number>(row.original, "WarmupThreshold", "warmup_threshold") ??
              "—"}
          </span>
        ),
      },
    ],
    [],
  );

  return (
    <ServerDataTable
      loadPage={pagePipelineStates}
      columns={columns}
      searchPlaceholder="Search by session key or status…"
      emptyMessage="No pipeline state yet."
      onRowClick={(r) => {
        if (r.session_key) {
          router.push(`/sessions/${encodeURIComponent(r.session_key)}`);
        }
      }}
    />
  );
}
