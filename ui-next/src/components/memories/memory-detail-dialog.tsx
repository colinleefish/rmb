"use client";

import { useCallback, useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";

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
import { CategoryBadge } from "@/components/category-badge";
import { OutlineBadge } from "@/components/detail";
import {
  createAssertion,
  listAssertions,
  retractAssertion,
} from "@/lib/api";
import { fmtDateTime, pick } from "@/lib/format";
import type { AssertionModel, MemoryModel } from "@/lib/types";

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-muted-foreground mb-2 text-xs font-semibold tracking-wider uppercase">
      {children}
    </h3>
  );
}

function AssertionsSection({
  assertions,
  loading,
  error,
  onRetract,
  retractingURI,
}: {
  assertions: AssertionModel[];
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
  if (assertions.length === 0)
    return (
      <p className="text-muted-foreground text-sm">
        No corrections attached to this memory.
      </p>
    );
  return (
    <div className="flex flex-col gap-2">
      {assertions.map((a) => (
        <div
          key={a.uri}
          className="bg-muted/30 flex flex-col gap-1.5 rounded-lg border p-3"
        >
          <div className="flex items-center gap-2">
            <OutlineBadge>{a.kind}</OutlineBadge>
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

function AddAssertionForm({
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

export function MemoryDetailDialog({
  memory,
  onOpenChange,
}: {
  memory: MemoryModel | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [assertions, setAssertions] = useState<AssertionModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [retractingURI, setRetractingURI] = useState<string | null>(null);

  const memoryURI = memory ? pick(memory, "URI", "uri") : null;

  const reload = useCallback(() => {
    if (!memoryURI) return;
    setLoading(true);
    setError(null);
    listAssertions(memoryURI)
      .then(setAssertions)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [memoryURI]);

  useEffect(() => {
    if (!memoryURI) return;
    setAssertions([]);
    setSubmitError(null);
    reload();
  }, [memoryURI, reload]);

  const handleAdd = useCallback(
    (statement: string) => {
      if (!memoryURI) return;
      setSubmitting(true);
      setSubmitError(null);
      createAssertion({ statement, target_uris: [memoryURI] })
        .then(() => reload())
        .catch((err: Error) => setSubmitError(err.message))
        .finally(() => setSubmitting(false));
    },
    [memoryURI, reload],
  );

  const handleRetract = useCallback(
    (uri: string) => {
      setRetractingURI(uri);
      retractAssertion(uri)
        .then(() => reload())
        .catch((err: Error) => setError(err.message))
        .finally(() => setRetractingURI(null));
    },
    [reload],
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
            <SectionTitle>Corrections ({assertions.length})</SectionTitle>
            <AssertionsSection
              assertions={assertions}
              loading={loading}
              error={error}
              onRetract={handleRetract}
              retractingURI={retractingURI}
            />
          </section>

          <Separator />

          <section>
            <SectionTitle>Add correction</SectionTitle>
            <AddAssertionForm
              onAdd={handleAdd}
              submitting={submitting}
              submitError={submitError}
            />
          </section>
        </div>
      </DialogContent>
    </Dialog>
  );
}
