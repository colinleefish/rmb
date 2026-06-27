"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { SessionPipelineSummary } from "@/components/sessions/session-pipeline-summary";
import {
  SESSION_DETAIL_TABS,
  type SessionDetailTab,
} from "@/components/sessions/session-detail-types";
import { fmtDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { SessionDetail } from "@/lib/types";
import type { SessionRow } from "@/lib/types";

export function SessionDetailHero({
  loading,
  title,
  abstract,
  session,
  detail,
  tab,
  onTabChange,
}: {
  loading: boolean;
  title: string;
  abstract: string | null;
  session: SessionRow | null;
  detail: SessionDetail | null;
  tab: SessionDetailTab;
  onTabChange: (tab: SessionDetailTab) => void;
}) {
  if (loading && !session) {
    return (
      <div className="space-y-3 pb-4">
        <Skeleton className="h-8 w-2/3 max-w-lg" />
        <Skeleton className="h-16 w-full max-w-2xl" />
        <Skeleton className="h-4 w-full max-w-md" />
      </div>
    );
  }

  const uri =
    session?.uri ?? (session ? `rmb://sessions/${session.session_key}` : "");

  return (
    <div className="flex flex-col gap-4">
      <div className="space-y-2.5">
        <h1 className="text-xl font-semibold tracking-tight text-balance sm:text-2xl">
          {title}
        </h1>
        {abstract && (
          <p className="text-muted-foreground max-w-3xl text-sm leading-relaxed">
            {abstract}
          </p>
        )}
        {(session?.created_at || session?.last_turn_at) && (
          <div className="text-muted-foreground flex flex-wrap items-center gap-x-2 gap-y-1 text-xs tabular-nums">
            {session.created_at && (
              <span>Created {fmtDateTime(session.created_at)}</span>
            )}
            {session.created_at && session.last_turn_at && <span>·</span>}
            {session.last_turn_at && (
              <span>Last turn {fmtDateTime(session.last_turn_at)}</span>
            )}
          </div>
        )}
        {uri && (
          <p className="text-muted-foreground font-mono text-[11px] leading-relaxed break-all select-all">
            {uri}
          </p>
        )}
        {session && <SessionPipelineSummary session={session} layout="stages-only" />}
      </div>

      {detail && (
        <nav
          aria-label="Session sections"
          className="-mb-px flex gap-6"
          role="tablist"
        >
          {SESSION_DETAIL_TABS.map(({ id, label, count }) => {
            const active = tab === id;
            const n = count(detail);
            return (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => onTabChange(id)}
                className={cn(
                  "inline-flex items-center gap-2 border-b-2 pb-2.5 text-sm transition-colors",
                  active
                    ? "border-foreground text-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground border-transparent",
                )}
              >
                {label}
                <span className="font-mono text-xs tabular-nums opacity-70">
                  {n}
                </span>
              </button>
            );
          })}
        </nav>
      )}
    </div>
  );
}
