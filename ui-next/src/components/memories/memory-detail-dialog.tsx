"use client";

import { useCallback, useEffect, useState } from "react";
import { ArrowRight, Plus, Trash2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { MemoryUriInput } from "@/components/aliases/memory-uri-input";
import { CategoryBadge } from "@/components/category-badge";
import { OutlineBadge } from "@/components/detail";
import {
  createAlias,
  createCorrection,
  listAliases,
  listCorrections,
  retractAlias,
  retractCorrection,
} from "@/lib/api";
import { fmtDateTime, pick } from "@/lib/format";
import type { AliasModel, CorrectionModel, MemoryModel } from "@/lib/types";

function isMergeableCategory(category: string | null | undefined): boolean {
  return category === "entities" || category === "preferences";
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-muted-foreground mb-2 text-xs font-semibold tracking-wider uppercase">
      {children}
    </h3>
  );
}

function CorrectionsSection({
  corrections,
  loading,
  error,
  onRetract,
  retractingURI,
}: {
  corrections: CorrectionModel[];
  loading: boolean;
  error: string | null;
  onRetract: (uri: string) => void;
  retractingURI: string | null;
}) {
  if (loading)
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-12 w-full" />
        <Skeleton className="h-12 w-full" />
      </div>
    );
  if (error)
    return <p className="text-destructive text-sm">Failed to load: {error}</p>;
  if (corrections.length === 0)
    return (
      <p className="text-muted-foreground text-sm">
        No corrections attached to this memory.
      </p>
    );
  return (
    <div className="flex flex-col gap-2">
      {corrections.map((a) => (
        <div
          key={a.uri}
          className="bg-muted/30 flex flex-col gap-1.5 rounded-lg border p-3"
        >
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground ml-auto text-xs">
              {fmtDateTime(a.created_at)}
            </span>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Retract"
              disabled={retractingURI === a.uri}
              onClick={() => onRetract(a.uri)}
            >
              <Trash2 />
            </Button>
          </div>
          <p className="text-foreground/90 text-sm whitespace-pre-wrap">
            {a.statement}
          </p>
          <p className="text-muted-foreground font-mono text-[10px] break-all">
            {a.uri}
          </p>
        </div>
      ))}
    </div>
  );
}

function AddCorrectionForm({
  onAdd,
  submitting,
  submitError,
}: {
  onAdd: (statement: string) => void;
  submitting: boolean;
  submitError: string | null;
}) {
  const [statement, setStatement] = useState("");

  const handleSubmit = () => {
    const trimmed = statement.trim();
    if (!trimmed) return;
    onAdd(trimmed);
    setStatement("");
  };

  return (
    <div className="flex flex-col gap-2">
      <textarea
        value={statement}
        onChange={(e) => setStatement(e.target.value)}
        placeholder="Write a human correction that overlays this memory…"
        rows={3}
        className="border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 w-full resize-y rounded-lg border bg-transparent px-2.5 py-2 text-sm transition-colors outline-none focus-visible:ring-3"
      />
      {submitError && <p className="text-destructive text-sm">{submitError}</p>}
      <div className="flex justify-end">
        <Button
          size="sm"
          onClick={handleSubmit}
          disabled={submitting || statement.trim() === ""}
        >
          <Plus />
          {submitting ? "Adding…" : "Add correction"}
        </Button>
      </div>
    </div>
  );
}

function AliasesSection({
  aliases,
  memoryURI,
  loading,
  error,
  onRetract,
  retractingURI,
}: {
  aliases: AliasModel[];
  memoryURI: string;
  loading: boolean;
  error: string | null;
  onRetract: (uri: string) => void;
  retractingURI: string | null;
}) {
  if (loading)
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-12 w-full" />
      </div>
    );
  if (error)
    return <p className="text-destructive text-sm">Failed to load: {error}</p>;

  const asAlias = aliases.filter((a) => a.alias_uri === memoryURI);
  const pointingHere = aliases.filter((a) => a.canonical_uri === memoryURI);

  if (asAlias.length === 0 && pointingHere.length === 0)
    return (
      <p className="text-muted-foreground text-sm">
        No alias relationships for this memory.
      </p>
    );

  return (
    <div className="flex flex-col gap-3">
      {asAlias.map((a) => (
        <div
          key={a.uri}
          className="bg-muted/30 flex flex-col gap-1.5 rounded-lg border p-3"
        >
          <p className="text-foreground text-sm font-medium">
            This memory is an alias
          </p>
          <div className="flex flex-wrap items-center gap-2 font-mono text-[10px] break-all">
            <span className="text-muted-foreground">{a.alias_uri}</span>
            <ArrowRight className="text-muted-foreground size-3 shrink-0" />
            <span>{a.canonical_uri}</span>
          </div>
          {a.note && (
            <p className="text-muted-foreground text-xs">{a.note}</p>
          )}
          <div className="flex items-center justify-between gap-2">
            <span className="text-muted-foreground text-xs">
              {fmtDateTime(a.created_at)}
            </span>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Retract alias"
              disabled={retractingURI === a.uri}
              onClick={() => onRetract(a.uri)}
            >
              <Trash2 />
            </Button>
          </div>
        </div>
      ))}
      {pointingHere.map((a) => (
        <div
          key={a.uri}
          className="bg-muted/30 flex flex-col gap-1.5 rounded-lg border p-3"
        >
          <p className="text-foreground text-sm font-medium">
            Alias pointing here
          </p>
          <div className="flex flex-wrap items-center gap-2 font-mono text-[10px] break-all">
            <span className="text-muted-foreground">{a.alias_uri}</span>
            <ArrowRight className="text-muted-foreground size-3 shrink-0" />
            <span>{a.canonical_uri}</span>
          </div>
          {a.note && (
            <p className="text-muted-foreground text-xs">{a.note}</p>
          )}
          <div className="flex items-center justify-between gap-2">
            <span className="text-muted-foreground text-xs">
              {fmtDateTime(a.created_at)}
            </span>
            <Button
              variant="ghost"
              size="icon-xs"
              aria-label="Retract alias"
              disabled={retractingURI === a.uri}
              onClick={() => onRetract(a.uri)}
            >
              <Trash2 />
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}

const noteInputClass =
  "border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 w-full rounded-lg border bg-transparent px-2.5 py-2 text-xs transition-colors outline-none focus-visible:ring-3";

function MarkAsAliasForm({
  memoryURI,
  onCreate,
  submitting,
  submitError,
}: {
  memoryURI: string;
  onCreate: (canonicalURI: string, note: string) => void;
  submitting: boolean;
  submitError: string | null;
}) {
  const [canonicalURI, setCanonicalURI] = useState("");
  const [note, setNote] = useState("");

  const handleSubmit = () => {
    const canonical = canonicalURI.trim();
    if (!canonical) return;
    onCreate(canonical, note.trim());
    setCanonicalURI("");
    setNote("");
  };

  return (
    <div className="flex flex-col gap-2">
      <p className="text-muted-foreground text-xs">
        Mark <span className="font-mono">{memoryURI}</span> as an alias of another
        memory (the canonical).
      </p>
      <MemoryUriInput
        value={canonicalURI}
        onChange={setCanonicalURI}
        placeholder="mem9://entities/aliyun-rds"
        excludeURIs={[memoryURI]}
      />
      <input
        className={noteInputClass}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="Note (optional)"
      />
      {submitError && <p className="text-destructive text-sm">{submitError}</p>}
      <div className="flex justify-end">
        <Button
          size="sm"
          variant="outline"
          onClick={handleSubmit}
          disabled={submitting || !canonicalURI.trim()}
        >
          <Plus />
          {submitting ? "Creating…" : "Mark as alias"}
        </Button>
      </div>
    </div>
  );
}

export function MemoryDetailDialog({
  memory,
  onOpenChange,
}: {
  memory: MemoryModel | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [corrections, setCorrections] = useState<CorrectionModel[]>([]);
  const [aliases, setAliases] = useState<AliasModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [aliasLoading, setAliasLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [aliasError, setAliasError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [aliasSubmitting, setAliasSubmitting] = useState(false);
  const [aliasSubmitError, setAliasSubmitError] = useState<string | null>(null);
  const [retractingURI, setRetractingURI] = useState<string | null>(null);
  const [retractingAliasURI, setRetractingAliasURI] = useState<string | null>(
    null,
  );

  const memoryURI = memory ? pick(memory, "URI", "uri") : null;
  const category = memory ? pick(memory, "Category", "category") : null;
  const mergeable = isMergeableCategory(category);
  const isAlias =
    mergeable && aliases.some((a) => a.alias_uri === memoryURI);

  const reload = useCallback(() => {
    if (!memoryURI) return;
    setLoading(true);
    setError(null);
    listCorrections(memoryURI)
      .then(setCorrections)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [memoryURI]);

  const reloadAliases = useCallback(() => {
    if (!memoryURI || !mergeable) return;
    setAliasLoading(true);
    setAliasError(null);
    listAliases(memoryURI)
      .then(setAliases)
      .catch((err: Error) => setAliasError(err.message))
      .finally(() => setAliasLoading(false));
  }, [memoryURI, mergeable]);

  useEffect(() => {
    if (!memoryURI) return;
    setCorrections([]);
    setAliases([]);
    setSubmitError(null);
    setAliasSubmitError(null);
    reload();
    reloadAliases();
  }, [memoryURI, reload, reloadAliases]);

  const handleAdd = useCallback(
    (statement: string) => {
      if (!memoryURI) return;
      setSubmitting(true);
      setSubmitError(null);
      createCorrection({ statement, target_uris: [memoryURI] })
        .then(() => reload())
        .catch((err: Error) => setSubmitError(err.message))
        .finally(() => setSubmitting(false));
    },
    [memoryURI, reload],
  );

  const handleRetract = useCallback(
    (uri: string) => {
      setRetractingURI(uri);
      retractCorrection(uri)
        .then(() => reload())
        .catch((err: Error) => setError(err.message))
        .finally(() => setRetractingURI(null));
    },
    [reload],
  );

  const handleCreateAlias = useCallback(
    (aliasURI: string, canonicalURI: string, note: string) => {
      setAliasSubmitting(true);
      setAliasSubmitError(null);
      createAlias({ alias_uri: aliasURI, canonical_uri: canonicalURI, note })
        .then(() => reloadAliases())
        .catch((err: Error) => setAliasSubmitError(err.message))
        .finally(() => setAliasSubmitting(false));
    },
    [reloadAliases],
  );

  const handleRetractAlias = useCallback(
    (uri: string) => {
      setRetractingAliasURI(uri);
      retractAlias(uri)
        .then(() => reloadAliases())
        .catch((err: Error) => setAliasError(err.message))
        .finally(() => setRetractingAliasURI(null));
    },
    [reloadAliases],
  );

  const open = memory != null;
  const abstract = memory ? pick(memory, "Abstract", "abstract") : null;
  const body = memory ? pick(memory, "Body", "body") : null;
  const version = memory ? pick<number>(memory, "Version", "version") : null;
  const updated = memory ? pick(memory, "UpdatedAt", "updated_at") : null;
  const title = memory
    ? (pick(memory, "Slug", "slug") ??
      pick(memory, "Category", "category") ??
      "Memory")
    : "Memory";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-2xl">
        <DialogHeader className="border-b p-4">
          <DialogTitle className="pr-8">{title}</DialogTitle>
          <DialogDescription className="font-mono text-xs break-all">
            {memoryURI}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5 overflow-y-auto p-4">
          <section className="flex flex-col gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <CategoryBadge category={memory ? pick(memory, "Category", "category") : null} />
              {version != null && <OutlineBadge>v{version}</OutlineBadge>}
            </div>
            {abstract && (
              <p className="text-foreground text-sm font-medium whitespace-pre-wrap">
                {abstract}
              </p>
            )}
            {body && (
              <p className="text-foreground/90 text-sm whitespace-pre-wrap">
                {body}
              </p>
            )}
            {updated && (
              <p className="text-muted-foreground text-xs">
                Updated {fmtDateTime(updated)}
              </p>
            )}
          </section>

          <Separator />

          <section>
            <SectionTitle>Corrections ({corrections.length})</SectionTitle>
            <CorrectionsSection
              corrections={corrections}
              loading={loading}
              error={error}
              onRetract={handleRetract}
              retractingURI={retractingURI}
            />
          </section>

          <Separator />

          <section>
            <SectionTitle>Add correction</SectionTitle>
            <AddCorrectionForm
              onAdd={handleAdd}
              submitting={submitting}
              submitError={submitError}
            />
          </section>

          {mergeable && (
            <>
              <Separator />

              <section>
                <SectionTitle>Aliases ({aliases.length})</SectionTitle>
                <AliasesSection
                  aliases={aliases}
                  memoryURI={memoryURI ?? ""}
                  loading={aliasLoading}
                  error={aliasError}
                  onRetract={handleRetractAlias}
                  retractingURI={retractingAliasURI}
                />
              </section>

              {!isAlias && (
                <>
                  <Separator />
                  <section>
                    <SectionTitle>Mark as alias</SectionTitle>
                    <MarkAsAliasForm
                      memoryURI={memoryURI ?? ""}
                      onCreate={(canonical, note) =>
                        handleCreateAlias(memoryURI ?? "", canonical, note)
                      }
                      submitting={aliasSubmitting}
                      submitError={aliasSubmitError}
                    />
                  </section>
                </>
              )}
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
