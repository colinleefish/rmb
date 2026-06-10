import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const CATEGORY_CLASS: Record<string, string> = {
  profile: "border-violet-600/30 bg-violet-500/10 text-violet-700",
  preferences: "border-sky-600/30 bg-sky-500/10 text-sky-700",
  entities: "border-emerald-600/30 bg-emerald-500/10 text-emerald-700",
  events: "border-amber-600/30 bg-amber-500/10 text-amber-700",
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
