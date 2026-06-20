"use client";

import { useCallback, useMemo, useState } from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { ArrowRight, Plus, Trash2 } from "lucide-react";

import { DataTable, SortButton, type RowDetail } from "@/components/data-table";
import {
  DetailLead,
  DetailMeta,
  DetailUri,
} from "@/components/detail";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { createAlias, listAliases, retractAlias } from "@/lib/api";
import { fmtDateShort, fmtDateTime, truncate } from "@/lib/format";
import type { AliasModel } from "@/lib/types";

function AddAliasForm({ onCreated }: { onCreated: () => void }) {
  const [aliasURI, setAliasURI] = useState("");
  const [canonicalURI, setCanonicalURI] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = () => {
    const alias = aliasURI.trim();
    const canonical = canonicalURI.trim();
    if (!alias || !canonical) return;
    setSubmitting(true);
    setError(null);
    createAlias({ alias_uri: alias, canonical_uri: canonical, note: note.trim() })
      .then(() => {
        setAliasURI("");
        setCanonicalURI("");
        setNote("");
        onCreated();
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setSubmitting(false));
  };

  const inputClass =
    "border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 w-full rounded-lg border bg-transparent px-2.5 py-2 font-mono text-xs transition-colors outline-none focus-visible:ring-3";

  return (
    <Card className="flex flex-col gap-3 p-4">
      <h2 className="text-sm font-semibold">Create alias</h2>
      <p className="text-muted-foreground text-xs">
        Declare the alias URI (redundant slug) to be the same entity as the
        canonical URI. Both must be <code className="text-xs">preferences</code>{" "}
        or <code className="text-xs">entities</code> memories in the same category.
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        <label className="flex flex-col gap-1.5 text-xs">
          <span className="text-muted-foreground font-medium">Alias URI</span>
          <input
            className={inputClass}
            value={aliasURI}
            onChange={(e) => setAliasURI(e.target.value)}
            placeholder="mypast://entities/aliyun-rds-instance"
          />
        </label>
        <label className="flex flex-col gap-1.5 text-xs">
          <span className="text-muted-foreground font-medium">Canonical URI</span>
          <input
            className={inputClass}
            value={canonicalURI}
            onChange={(e) => setCanonicalURI(e.target.value)}
            placeholder="mypast://entities/aliyun-rds"
          />
        </label>
      </div>
      <label className="flex flex-col gap-1.5 text-xs">
        <span className="text-muted-foreground font-medium">Note (optional)</span>
        <input
          className={inputClass}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="same DB, instance vs prod-role"
        />
      </label>
      {error && <p className="text-destructive text-sm">{error}</p>}
      <div className="flex justify-end">
        <Button
          size="sm"
          onClick={handleSubmit}
          disabled={submitting || !aliasURI.trim() || !canonicalURI.trim()}
        >
          <Plus />
          {submitting ? "Creating…" : "Create alias"}
        </Button>
      </div>
    </Card>
  );
}

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
      <AddAliasForm onCreated={reload} />
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
      />
    </div>
  );
}
