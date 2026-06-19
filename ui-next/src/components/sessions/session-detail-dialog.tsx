"use client";

import { useEffect, useState } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/status-badge";
import { CategoryBadge } from "@/components/category-badge";
import { getSession } from "@/lib/api";
import { fmtDateTime, parseJSONL, pick, truncate } from "@/lib/format";
import type {
  AtomModel,
  PipelineStateModel,
  SceneModel,
  SessionDetail,
  TurnRow,
} from "@/lib/types";

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="text-muted-foreground mb-2 text-xs font-semibold tracking-wider uppercase">
      {children}
    </h3>
  );
}

function PipelineSection({ ps }: { ps: PipelineStateModel | null }) {
  if (!ps) return <p className="text-muted-foreground text-sm">No pipeline state.</p>;
  const stages: Array<[string, string | null]> = [
    ["T1 · atoms", pick(ps, "T1Status", "t1_status")],
    ["T2 · scenes", pick(ps, "T2Status", "t2_status")],
    ["T3 · memories", pick(ps, "T3Status", "t3_status")],
  ];
  const warmup = pick<number>(ps, "WarmupThreshold", "warmup_threshold");
  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-3 gap-2">
        {stages.map(([label, status]) => (
          <div
            key={label}
            className="bg-muted/40 flex flex-col gap-1.5 rounded-lg border p-3"
          >
            <span className="text-muted-foreground text-xs">{label}</span>
            <StatusBadge status={status} />
          </div>
        ))}
      </div>
      {warmup != null && (
        <p className="text-muted-foreground text-xs">
          warmup threshold: <span className="font-mono">{warmup}</span>
        </p>
      )}
    </div>
  );
}

function TurnsSection({ turns }: { turns: TurnRow[] }) {
  if (!turns.length)
    return <p className="text-muted-foreground text-sm">No turns yet.</p>;
  return (
    <div className="flex flex-col gap-2">
      {turns.map((turn) => {
        const messages = parseJSONL(turn.messages_jsonl);
        return (
          <details
            key={turn.id}
            className="bg-muted/30 group rounded-lg border px-3 py-2"
          >
            <summary className="flex cursor-pointer list-none items-center gap-2 text-sm">
              <span className="font-mono text-xs">#{turn.turn_index}</span>
              <span className="text-muted-foreground ml-auto text-xs">
                {fmtDateTime(turn.created_at)}
              </span>
            </summary>
            <div className="mt-2 flex flex-col gap-2">
              {messages.length === 0 ? (
                <p className="text-muted-foreground text-xs">No messages.</p>
              ) : (
                messages.map((m, i) => (
                  <div key={i} className="rounded-md border bg-background/60 p-2">
                    <span className="text-muted-foreground mb-1 block font-mono text-[10px] uppercase">
                      {m.role ?? "?"}
                    </span>
                    <p className="text-foreground/90 text-sm whitespace-pre-wrap">
                      {truncate(m.content ?? "", 1200)}
                    </p>
                  </div>
                ))
              )}
            </div>
          </details>
        );
      })}
    </div>
  );
}

function AtomsSection({ atoms }: { atoms: AtomModel[] }) {
  if (!atoms.length)
    return <p className="text-muted-foreground text-sm">No atoms yet.</p>;
  return (
    <div className="flex flex-col gap-2">
      {atoms.map((a, i) => {
        const scene = pick(a, "SceneName", "scene_name");
        const slug = pick(a, "Slug", "slug");
        const topic = [scene, slug].filter(Boolean).join(" · ");
        return (
          <div
            key={pick(a, "URI", "uri") ?? i}
            className="bg-muted/30 flex flex-col gap-1.5 rounded-lg border p-3"
          >
            <div className="flex items-center gap-2">
              <CategoryBadge category={pick(a, "Category", "category")} />
              {topic && (
                <span className="text-muted-foreground text-xs">{topic}</span>
              )}
            </div>
            <p className="text-foreground/90 text-sm">
              {pick(a, "Content", "content")}
            </p>
          </div>
        );
      })}
    </div>
  );
}

function ScenesSection({ scenes }: { scenes: SceneModel[] }) {
  if (!scenes.length)
    return <p className="text-muted-foreground text-sm">No scenes yet.</p>;
  return (
    <div className="flex flex-col gap-2">
      {scenes.map((s, i) => {
        const name = pick(s, "DisplayName", "display_name") ?? "Scene";
        const summary =
          pick(s, "Abstract", "abstract") ??
          truncate(pick(s, "Body", "body"), 160);
        return (
          <div
            key={pick(s, "URI", "uri") ?? i}
            className="bg-muted/30 flex flex-col gap-1 rounded-lg border p-3"
          >
            <span className="text-foreground text-sm font-medium">{name}</span>
            <p className="text-muted-foreground text-sm">{summary || "—"}</p>
          </div>
        );
      })}
    </div>
  );
}

export function SessionDetailDialog({
  sessionKey,
  onOpenChange,
}: {
  sessionKey: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const [detail, setDetail] = useState<SessionDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!sessionKey) return;
    setLoading(true);
    setError(null);
    setDetail(null);
    getSession(sessionKey)
      .then(setDetail)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [sessionKey]);

  const open = sessionKey != null;
  const session = detail?.session;
  const title = session?.title?.trim() || session?.session_key || "Session";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[85vh] flex-col gap-0 p-0 sm:max-w-2xl">
        <DialogHeader className="border-b p-4">
          <DialogTitle className="pr-8">
            {loading && !detail ? <Skeleton className="h-5 w-48" /> : title}
          </DialogTitle>
          <DialogDescription className="font-mono text-xs break-all">
            {detail?.session.uri ?? sessionKey ?? ""}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5 overflow-y-auto p-4">
          {error ? (
            <p className="text-destructive text-sm">Failed to load: {error}</p>
          ) : loading || !detail ? (
            <div className="flex flex-col gap-3">
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-32 w-full" />
            </div>
          ) : (
            <>
              <section>
                <SectionTitle>Pipeline</SectionTitle>
                <PipelineSection ps={detail.pipeline_state} />
              </section>
              <Separator />
              <section>
                <SectionTitle>Turns ({detail.turns.length})</SectionTitle>
                <TurnsSection turns={detail.turns} />
              </section>
              <Separator />
              <section>
                <SectionTitle>Atoms ({detail.atoms.length})</SectionTitle>
                <AtomsSection atoms={detail.atoms} />
              </section>
              <Separator />
              <section>
                <SectionTitle>Scenes ({detail.scenes.length})</SectionTitle>
                <ScenesSection scenes={detail.scenes} />
              </section>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
