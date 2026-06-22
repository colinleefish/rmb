"use client";

import { useCallback, useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { ArrowLeftRight, ArrowRight, Check, X } from "lucide-react";

import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import { DetailLead, DetailMeta } from "@/components/detail";
import { Button } from "@/components/ui/button";
import {
  confirmAliasCandidate,
  createAlias,
  listAliasCandidates,
  rejectAliasCandidate,
} from "@/lib/api";
import { fmtDateShort, fmtDateTime, truncate } from "@/lib/format";
import type { AliasCandidateModel } from "@/lib/types";

function CandidateDetail({
  c,
  onConfirm,
  onReject,
  busy,
}: {
  c: AliasCandidateModel;
  onConfirm: (c: AliasCandidateModel, swapped: boolean) => void;
  onReject: (id: string) => void;
  busy: boolean;
}) {
  const [swapped, setSwapped] = useState(false);
  const canonicalUri = swapped ? c.alias_uri : c.canonical_uri;
  const aliasUri = swapped ? c.canonical_uri : c.alias_uri;

  return (
    <>
      <div className="relative flex flex-col gap-3 pr-10">
        <div>
          <p className="text-muted-foreground mb-1 text-xs">canonical</p>
          <div className="bg-muted rounded-lg px-3 py-2 font-mono text-xs break-all">
            {canonicalUri}
          </div>
        </div>
        <div>
          <p className="text-muted-foreground mb-1 text-xs">alias</p>
          <div className="bg-muted rounded-lg px-3 py-2 font-mono text-xs break-all">
            {aliasUri}
          </div>
        </div>
        <button
          type="button"
          title="Swap alias and canonical"
          onClick={() => setSwapped((s) => !s)}
          className={`absolute right-0 top-1/2 -translate-y-1/2 rounded-full p-1.5 transition-colors ${
            swapped
              ? "bg-primary text-primary-foreground"
              : "text-muted-foreground hover:bg-muted hover:text-foreground"
          }`}
        >
          <ArrowLeftRight className="size-4" />
        </button>
      </div>
      {c.rationale && <DetailLead>{c.rationale}</DetailLead>}
      {c.created_at && (
        <DetailMeta>Proposed {fmtDateTime(c.created_at)}</DetailMeta>
      )}
      <div className="flex gap-2 pt-2">
        <Button size="sm" disabled={busy} onClick={() => onConfirm(c, swapped)}>
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
  );
}

function detailOf(
  c: AliasCandidateModel,
  onConfirm: (c: AliasCandidateModel, swapped: boolean) => void,
  onReject: (id: string) => void,
  busyID: string | null,
): RowDetail {
  return {
    title: "Alias suggestion",
    description: `${(c.similarity * 100).toFixed(1)}% similar`,
    body: (
      <CandidateDetail
        c={c}
        onConfirm={onConfirm}
        onReject={onReject}
        busy={busyID === c.id}
      />
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
    (c: AliasCandidateModel, swapped: boolean) => {
      if (!swapped) {
        run(c.id, confirmAliasCandidate);
      } else {
        run(c.id, async (id) => {
          await rejectAliasCandidate(id);
          await createAlias({ alias_uri: c.canonical_uri, canonical_uri: c.alias_uri });
        });
      }
    },
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
