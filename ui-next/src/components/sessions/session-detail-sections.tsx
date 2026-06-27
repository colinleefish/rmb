import { CategoryBadge } from "@/components/category-badge";
import { pick, truncate } from "@/lib/format";
import type { AtomModel, SceneModel } from "@/lib/types";

export function AtomsSection({ atoms }: { atoms: AtomModel[] }) {
  if (!atoms.length)
    return <p className="text-muted-foreground text-sm">No atoms yet.</p>;
  return (
    <ul className="divide-border/40 flex flex-col divide-y">
      {atoms.map((a, i) => {
        const scene = pick(a, "SceneName", "scene_name");
        const slug = pick(a, "Slug", "slug");
        const topic = [scene, slug].filter(Boolean).join(" · ");
        return (
          <li
            key={pick(a, "URI", "uri") ?? i}
            className="flex flex-col gap-1.5 py-4"
          >
            <div className="flex flex-wrap items-center gap-2">
              <CategoryBadge category={pick(a, "Category", "category")} />
              {topic && (
                <span className="text-muted-foreground text-xs">{topic}</span>
              )}
            </div>
            <p className="text-foreground/90 text-sm leading-relaxed">
              {pick(a, "Content", "content")}
            </p>
          </li>
        );
      })}
    </ul>
  );
}

export function ScenesSection({ scenes }: { scenes: SceneModel[] }) {
  if (!scenes.length)
    return <p className="text-muted-foreground text-sm">No scenes yet.</p>;
  return (
    <ul className="divide-border/40 flex flex-col divide-y">
      {scenes.map((s, i) => {
        const name = pick(s, "DisplayName", "display_name") ?? "Scene";
        const summary =
          pick(s, "Abstract", "abstract") ??
          truncate(pick(s, "Body", "body"), 160);
        return (
          <li
            key={pick(s, "URI", "uri") ?? i}
            className="flex flex-col gap-1 py-4"
          >
            <span className="text-foreground text-sm font-medium">{name}</span>
            <p className="text-muted-foreground text-sm leading-relaxed">
              {summary || "—"}
            </p>
          </li>
        );
      })}
    </ul>
  );
}
