"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  ArrowRight,
  ArrowUpRight,
  Atom,
  Boxes,
  Brain,
  ListChecks,
  MessagesSquare,
  ScrollText,
  ShieldCheck,
  type LucideIcon,
} from "lucide-react";

import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getOverview } from "@/lib/api";
import type { OverviewCounts } from "@/lib/types";

type Stat = {
  key: keyof OverviewCounts;
  label: string;
  icon: LucideIcon;
  hint: string;
  href?: string;
};

type StatGroup = {
  id: string;
  title: string;
  caption: string;
  stats: Stat[];
};

const GROUPS: StatGroup[] = [
  {
    id: "per-session",
    title: "Per session",
    caption: "Raw signal captured inside a single conversation.",
    stats: [
      {
        key: "sessions",
        label: "Sessions",
        icon: MessagesSquare,
        hint: "Agent conversations",
        href: "/sessions",
      },
      { key: "turns", label: "Turns", icon: ScrollText, hint: "Message exchanges" },
      { key: "atoms", label: "Atoms", icon: Atom, hint: "Typed facts" },
      { key: "scenes", label: "Scenes", icon: Boxes, hint: "Activity spans" },
    ],
  },
  {
    id: "across-sessions",
    title: "Across sessions",
    caption: "Durable knowledge rolled up from many conversations.",
    stats: [
      {
        key: "memories",
        label: "Memories",
        icon: Brain,
        hint: "Long-term memory",
        href: "/memories",
      },
      {
        key: "corrections",
        label: "Corrections",
        icon: ShieldCheck,
        hint: "Human overrides",
        href: "/corrections",
      },
    ],
  },
  {
    id: "workers",
    title: "Workers",
    caption: "Background jobs advancing the pipeline.",
    stats: [
      {
        key: "tasks",
        label: "Tasks",
        icon: ListChecks,
        hint: "Queued & running",
        href: "/tasks",
      },
    ],
  },
];

const PYRAMID = [
  { tier: "session", name: "sessions", sub: "one agent conversation" },
  { tier: "T1", name: "atoms", sub: "typed facts" },
  { tier: "T2", name: "scenes", sub: "what we were doing" },
  { tier: "T3", name: "memories", sub: "profile · preferences · entities · events" },
];

function StatCard({ stat, value }: { stat: Stat; value: number | undefined }) {
  const Icon = stat.icon;
  const inner = (
    <Card className="group relative h-full overflow-hidden transition-colors hover:border-foreground/25">
      <CardContent className="flex h-full flex-col gap-4">
        <div className="flex items-start justify-between">
          <span className="flex size-9 items-center justify-center rounded-lg border bg-muted/40 text-foreground/70 transition-colors group-hover:text-foreground">
            <Icon className="size-[18px]" aria-hidden />
          </span>
          {stat.href && (
            <ArrowUpRight className="size-4 text-muted-foreground/0 transition-colors group-hover:text-muted-foreground" />
          )}
        </div>
        <div className="flex flex-col gap-0.5">
          {value === undefined ? (
            <Skeleton className="h-9 w-16" />
          ) : (
            <span className="text-4xl font-semibold leading-none tracking-tight tabular-nums">
              {value.toLocaleString()}
            </span>
          )}
          <span className="text-sm font-medium text-foreground">{stat.label}</span>
          <span className="text-xs text-muted-foreground">{stat.hint}</span>
        </div>
      </CardContent>
    </Card>
  );
  return stat.href ? (
    <Link href={stat.href} className="block">
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
    <div className="flex flex-col gap-8">
      {error && (
        <p className="text-destructive text-sm">Failed to load: {error}</p>
      )}

      {GROUPS.map((group) => (
        <section key={group.id} className="flex flex-col gap-3">
          <div className="flex flex-col gap-0.5">
            <h2 className="text-sm font-semibold tracking-tight">
              {group.title}
            </h2>
            <p className="text-muted-foreground text-xs">{group.caption}</p>
          </div>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
            {group.stats.map((stat) => (
              <StatCard
                key={stat.key}
                stat={stat}
                value={counts?.[stat.key]}
              />
            ))}
          </div>
        </section>
      ))}

      <Card className="bg-muted/30">
        <CardContent className="flex flex-col gap-5">
          <div className="flex flex-col gap-0.5">
            <h2 className="text-sm font-semibold tracking-tight">
              The distillation pyramid
            </h2>
            <p className="text-muted-foreground text-sm">
              How raw conversation turns are refined into long-term memory.
            </p>
          </div>
          <div className="flex flex-col items-stretch gap-2 sm:flex-row sm:items-center">
            {PYRAMID.map((step, i) => (
              <div key={step.name} className="contents">
                <div className="flex flex-1 flex-col gap-1 rounded-lg border bg-card px-4 py-3">
                  <span className="text-muted-foreground font-mono text-[10px] uppercase tracking-wider">
                    {step.tier}
                  </span>
                  <span className="text-foreground text-sm font-semibold">
                    {step.name}
                  </span>
                  <span className="text-muted-foreground text-xs leading-relaxed">
                    {step.sub}
                  </span>
                </div>
                {i < PYRAMID.length - 1 && (
                  <ArrowRight className="text-muted-foreground/60 mx-auto size-4 shrink-0 rotate-90 sm:mx-0 sm:rotate-0" />
                )}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
