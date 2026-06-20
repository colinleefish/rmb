"use client";

import { useCallback, useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { ArrowRight, Check, X } from "lucide-react";

import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import { DetailLead, DetailMeta, DetailUri } from "@/components/detail";
import { Button } from "@/components/ui/button";
import {
  confirmAliasCandidate,
  listAliasCandidates,
  rejectAliasCandidate,
} from "@/lib/api";
import { fmtDateShort, fmtDateTime, truncate } from "@/lib/format";
import type { AliasCandidateModel } from "@/lib/types";

function detailOf(
  c: AliasCandidateModel,
  onConfirm: (id: string) => void,
  onReject: (id: string) => void,
  busyID: string | null,
): RowDetail {
  const busy = busyID === c.id;
  return {
    title: "Alias suggestion",
    description: `${(c.similarity * 100).toFixed(1)}% similar`,
    body: (
      <>
        <div className="flex flex-wrap items-center gap-2 font-mono text-xs">
          <DetailUri>{c.alias_uri}</DetailUri>
          <ArrowRight className="text-muted-foreground size-3.5 shrink-0" />
          <DetailUri>{c.canonical_uri}</DetailUri>
        </div>
        {c.rationale && <DetailLead>{c.rationale}</DetailLead>}
        {c.created_at && (
          <DetailMeta>Proposed {fmtDateTime(c.created_at)}</DetailMeta>
        )}
        <div className="flex gap-2 pt-2">
          <Button size="sm" disabled={busy} onClick={() => onConfirm(c.id)}>
            <Check />
            {busy ? "Working…" : "Confirm alias"}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={busy}
            onClick={() => onReject(c.id)}
          >
            <X />
            Reject
          </Button>
        </div>
      </>
    ),
  };
}

export function AliasCandidatesView() {
  const [reloadKey, setReloadKey] = useState(0);
  const [busyID, setBusyID] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  const run = useCallback(
    (id: string, fn: (id: string) => Promise<unknown>) => {
      setBusyID(id);
      setActionError(null);
      fn(id)
        .then(() => reload())
        .catch((err: Error) => setActionError(err.message))
        .finally(() => setBusyID(null));
    },
    [reload],
  );

  const handleConfirm = useCallback(
    (id: string) => run(id, confirmAliasCandidate),
    [run],
  );
  const handleReject = useCallback(
    (id: string) => run(id, rejectAliasCandidate),
    [run],
  );

  const columns = useMemo<ColumnDef<AliasCandidateModel>[]>(
    () => [
      {
        id: "alias",
        accessorFn: (c) => c.alias_uri,
        header: "Alias",
        cell: ({ row }) => (
          <span className="text-muted-foreground font-mono text-xs">
            {truncate(row.original.alias_uri, 48)}
          </span>
        ),
      },
      {
        id: "arrow",
        header: "",
        cell: () => (
          <ArrowRight className="text-muted-foreground mx-auto size-3.5" />
        ),
        enableSorting: false,
      },
      {
        id: "canonical",
        accessorFn: (c) => c.canonical_uri,
        header: "Canonical",
        cell: ({ row }) => (
          <span className="text-foreground font-mono text-xs">
            {truncate(row.original.canonical_uri, 48)}
          </span>
        ),
      },
      {
        id: "similarity",
        accessorFn: (c) => c.similarity,
        header: ({ column }) => <SortButton column={column} label="Similarity" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm tabular-nums">
            {(row.original.similarity * 100).toFixed(1)}%
          </span>
        ),
      },
      {
        id: "created",
        accessorFn: (c) => c.created_at ?? "",
        header: ({ column }) => <SortButton column={column} label="Proposed" />,
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
    <div className="flex flex-col gap-6">
      {actionError && (
        <p className="text-destructive text-sm">Action failed: {actionError}</p>
      )}
      <DataTable
        key={reloadKey}
        load={() => listAliasCandidates("pending")}
        columns={columns}
        searchText={(c) =>
          [c.alias_uri, c.canonical_uri, c.rationale].filter(Boolean).join(" ")
        }
        searchPlaceholder="Search suggestions…"
        emptyMessage="No pending suggestions. Run the alias-suggest worker to generate some."
        initialSorting={[{ id: "similarity", desc: true }]}
        renderDetail={(c) => detailOf(c, handleConfirm, handleReject, busyID)}
      />
    </div>
  );
}
