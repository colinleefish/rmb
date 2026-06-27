import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const CATEGORY_CLASS: Record<string, string> = {
  profile: "border-violet-600/30 bg-violet-500/10 text-violet-700 dark:text-violet-300",
  preferences: "border-violet-600/30 bg-violet-500/10 text-violet-700 dark:text-violet-300",
  preference: "border-violet-600/30 bg-violet-500/10 text-violet-700 dark:text-violet-300",
  entities: "border-emerald-600/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  decision: "border-emerald-600/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  events: "border-amber-600/30 bg-amber-500/10 text-amber-700 dark:text-amber-300",
  fact: "border-sky-600/30 bg-sky-500/10 text-sky-700 dark:text-sky-300",
  knowledge: "border-sky-600/30 bg-sky-500/10 text-sky-700 dark:text-sky-300",
  issue: "border-rose-600/30 bg-rose-500/10 text-rose-700 dark:text-rose-300",
};

export function CategoryBadge({
  category,
  className,
}: {
  category: string | null | undefined;
  className?: string;
}) {
  const key = (category ?? "").toLowerCase();
  return (
    <Badge
      variant="outline"
      className={cn("font-medium", CATEGORY_CLASS[key] ?? "border-border", className)}
    >
      {category ?? "—"}
    </Badge>
  );
}
