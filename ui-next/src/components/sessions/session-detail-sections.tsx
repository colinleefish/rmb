import { Atom, Layers } from "lucide-react";

import { CategoryBadge } from "@/components/category-badge";
import { pick, truncate } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { AtomModel, SceneModel } from "@/lib/types";

const UNGROUPED = "__ungrouped__";

/** A compact 1–5 priority meter rendered as stacked bars. */
function PriorityMeter({ priority }: { priority: number | null }) {
  if (!priority || priority < 1) return null;
  const level = Math.min(priority, 5);
  return (
    <span
      className="flex items-center gap-1"
      title={`Priority ${level} of 5`}
      aria-label={`Priority ${level} of 5`}
    >
      <span className="text-muted-foreground font-mono text-[10px] tabular-nums">
        P{level}
      </span>
      <span className="flex items-end gap-0.5">
        {Array.from({ length: 5 }).map((_, i) => (
          <span
            key={i}
            className={cn(
              "w-0.5 rounded-full",
              i < level ? "bg-foreground/70" : "bg-border",
            )}
            style={{ height: `${5 + i * 2}px` }}
          />
        ))}
      </span>
    </span>
  );
}

export function AtomsSection({ atoms }: { atoms: AtomModel[] }) {
  if (!atoms.length)
    return <p className="text-muted-foreground text-sm">No atoms yet.</p>;

  // Group atoms under the scene they roll up into; loose atoms go last.
  const groups = new Map<string, AtomModel[]>();
  for (const a of atoms) {
    const scene = pick(a, "SceneName", "scene_name") ?? UNGROUPED;
    const list = groups.get(scene) ?? [];
    list.push(a);
    groups.set(scene, list);
  }
  const orderedKeys = [...groups.keys()].sort((a, b) => {
    if (a === UNGROUPED) return 1;
    if (b === UNGROUPED) return -1;
    return a.localeCompare(b);
  });

  return (
    <div className="flex flex-col gap-8">
      {orderedKeys.map((key) => {
        const items = groups.get(key)!;
        const isUngrouped = key === UNGROUPED;
        return (
          <section key={key} className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <Layers className="text-muted-foreground size-3.5" />
              <h3 className="text-foreground text-sm font-semibold">
                {isUngrouped ? "Unassigned" : key}
              </h3>
              <span className="text-muted-foreground font-mono text-xs">
                {items.length}
              </span>
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
              {items.map((a, i) => (
                <article
                  key={pick(a, "URI", "uri") ?? i}
                  className="bg-card hover:border-foreground/20 flex flex-col gap-3 rounded-lg border p-4 transition-colors"
                >
                  <div className="flex items-center justify-between gap-2">
                    <CategoryBadge category={pick(a, "Category", "category")} />
                    <PriorityMeter
                      priority={pick<number>(a, "Priority", "priority")}
                    />
                  </div>
                  <p className="text-foreground/90 text-sm leading-relaxed">
                    {pick(a, "Content", "content")}
                  </p>
                  {pick(a, "Slug", "slug") && (
                    <span className="text-muted-foreground mt-auto inline-flex items-center gap-1 font-mono text-[11px]">
                      <Atom className="size-3" />
                      {pick(a, "Slug", "slug")}
                    </span>
                  )}
                </article>
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

export function ScenesSection({ scenes }: { scenes: SceneModel[] }) {
  if (!scenes.length)
    return <p className="text-muted-foreground text-sm">No scenes yet.</p>;
  return (
    <div className="flex flex-col gap-4">
      {scenes.map((s, i) => {
        const name = pick(s, "DisplayName", "display_name") ?? "Scene";
        const abstract = pick(s, "Abstract", "abstract");
        const body = pick(s, "Body", "body");
        return (
          <article
            key={pick(s, "URI", "uri") ?? i}
            className="bg-card flex flex-col gap-3 rounded-xl border p-5 sm:flex-row sm:gap-5"
          >
            <div className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-lg">
              <Layers className="size-4" />
            </div>
            <div className="flex min-w-0 flex-col gap-2">
              <div className="flex flex-col gap-0.5">
                <h3 className="text-foreground text-base font-semibold leading-snug">
                  {name}
                </h3>
                {abstract && (
                  <p className="text-muted-foreground text-sm leading-relaxed">
                    {abstract}
                  </p>
                )}
              </div>
              {body && (
                <p className="text-foreground/80 border-border/60 border-l-2 pl-3 text-sm leading-relaxed">
                  {truncate(body, 320)}
                </p>
              )}
            </div>
          </article>
        );
      })}
    </div>
  );
}
