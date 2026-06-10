"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowRight } from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getOverview } from "@/lib/api";
import type { OverviewCounts } from "@/lib/types";

const STATS: Array<{ key: keyof OverviewCounts; label: string; href?: string }> = [
  { key: "sessions", label: "Sessions", href: "/sessions" },
  { key: "turns", label: "Turns" },
  { key: "atoms", label: "Atoms", href: "/atoms" },
  { key: "scenes", label: "Scenes", href: "/scenes" },
  { key: "memories", label: "Memories", href: "/memories" },
  { key: "pipeline_states", label: "Pipeline", href: "/pipeline" },
  { key: "tasks", label: "Tasks", href: "/tasks" },
];

const PYRAMID = [
  { tier: "session", name: "sessions", sub: "one agent conversation" },
  { tier: "T1", name: "atoms", sub: "typed facts" },
  { tier: "T2", name: "scenes", sub: "what we were doing" },
  { tier: "T3", name: "memories", sub: "profile · preferences · entities · events" },
];

function StatCard({
  label,
  value,
  href,
}: {
  label: string;
  value: number | undefined;
  href?: string;
}) {
  const inner = (
    <Card className="transition-colors hover:border-foreground/20">
      <CardContent className="flex flex-col gap-1">
        <span className="text-muted-foreground text-sm">{label}</span>
        {value === undefined ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <span className="text-3xl font-semibold tabular-nums">
            {value.toLocaleString()}
          </span>
        )}
      </CardContent>
    </Card>
  );
  return href ? (
    <Link href={href} className="block">
      {inner}
    </Link>
  ) : (
    inner
  );
}

export function OverviewView() {
  const [counts, setCounts] = useState<OverviewCounts | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getOverview()
      .then((o) => setCounts(o.counts))
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <div className="flex flex-col gap-6">
      {error && (
        <p className="text-destructive text-sm">Failed to load: {error}</p>
      )}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        {STATS.map((s) => (
          <StatCard
            key={s.key}
            label={s.label}
            value={counts?.[s.key]}
            href={s.href}
          />
        ))}
      </div>

      <Card>
        <CardContent className="flex flex-col gap-4">
          <div>
            <h2 className="text-sm font-semibold">The distillation pyramid</h2>
            <p className="text-muted-foreground text-sm">
              How raw conversation turns are refined into long-term memory.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {PYRAMID.map((step, i) => (
              <div key={step.name} className="flex items-center gap-2">
                <div className="bg-muted/50 flex flex-col rounded-lg border px-3 py-2">
                  <span className="text-muted-foreground font-mono text-[10px] uppercase">
                    {step.tier}
                  </span>
                  <span className="text-foreground text-sm font-medium">
                    {step.name}
                  </span>
                  <span className="text-muted-foreground text-xs">
                    {step.sub}
                  </span>
                </div>
                {i < PYRAMID.length - 1 && (
                  <ArrowRight className="text-muted-foreground size-4 shrink-0" />
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
