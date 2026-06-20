"use client";

import { useCallback, useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { ArrowRight, Plus, Trash2 } from "lucide-react";

import { AddAliasDialog } from "@/components/aliases/add-alias-dialog";
import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import {
  DetailLead,
  DetailMeta,
  DetailUri,
} from "@/components/detail";
import { Button } from "@/components/ui/button";
import { listAliases, retractAlias } from "@/lib/api";
import { fmtDateShort, fmtDateTime, truncate } from "@/lib/format";
import type { AliasModel } from "@/lib/types";

function detailOf(
  a: AliasModel,
  onRetract: (uri: string) => void,
  retractingURI: string | null,
): RowDetail {
  return {
    title: "Alias",
    description: a.uri,
    body: (
      <>
        <div className="flex flex-wrap items-center gap-2 font-mono text-xs">
          <DetailUri>{a.alias_uri}</DetailUri>
          <ArrowRight className="text-muted-foreground size-3.5 shrink-0" />
          <DetailUri>{a.canonical_uri}</DetailUri>
        </div>
        {a.note && <DetailLead>{a.note}</DetailLead>}
        {a.created_at && (
          <DetailMeta>Created {fmtDateTime(a.created_at)}</DetailMeta>
        )}
        <div className="pt-2">
          <Button
            variant="outline"
            size="sm"
            disabled={retractingURI === a.uri}
            onClick={() => onRetract(a.uri)}
          >
            <Trash2 />
            {retractingURI === a.uri ? "Retracting…" : "Retract alias"}
          </Button>
        </div>
      </>
    ),
  };
}

export function AliasesView() {
  const [reloadKey, setReloadKey] = useState(0);
  const [addOpen, setAddOpen] = useState(false);
  const [retractingURI, setRetractingURI] = useState<string | null>(null);
  const [retractError, setRetractError] = useState<string | null>(null);

  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  const handleRetract = useCallback(
    (uri: string) => {
      setRetractingURI(uri);
      setRetractError(null);
      retractAlias(uri)
        .then(() => reload())
        .catch((err: Error) => setRetractError(err.message))
        .finally(() => setRetractingURI(null));
    },
    [reload],
  );

  const columns = useMemo<ColumnDef<AliasModel>[]>(
    () => [
      {
        id: "alias",
        accessorFn: (a) => a.alias_uri,
        header: "Alias",
        cell: ({ row }) => (
          <span className="text-muted-foreground font-mono text-xs">
            {truncate(row.original.alias_uri, 52)}
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
        accessorFn: (a) => a.canonical_uri,
        header: "Canonical",
        cell: ({ row }) => (
          <span className="text-foreground font-mono text-xs">
            {truncate(row.original.canonical_uri, 52)}
          </span>
        ),
      },
      {
        id: "note",
        accessorFn: (a) => a.note ?? "",
        header: "Note",
        cell: ({ row }) => (
          <p className="text-muted-foreground line-clamp-2 max-w-xs text-sm">
            {row.original.note || "—"}
          </p>
        ),
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
    <div className="flex flex-col gap-6">
      <AddAliasDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        onCreated={reload}
      />
      {retractError && (
        <p className="text-destructive text-sm">Retract failed: {retractError}</p>
      )}
      <DataTable
        key={reloadKey}
        load={() => listAliases()}
        columns={columns}
        searchText={(a) =>
          [a.alias_uri, a.canonical_uri, a.note, a.uri]
            .filter(Boolean)
            .join(" ")
        }
        searchPlaceholder="Search aliases…"
        emptyMessage="No aliases yet."
        initialSorting={[{ id: "created", desc: true }]}
        renderDetail={(a) => detailOf(a, handleRetract, retractingURI)}
        toolbarActions={
          <Button size="sm" onClick={() => setAddOpen(true)}>
            <Plus />
            Add alias
          </Button>
        }
      />
    </div>
  );
}
