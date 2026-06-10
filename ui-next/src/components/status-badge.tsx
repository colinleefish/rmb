import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { statusTone, type Tone } from "@/lib/format";

const TONE_CLASS: Record<Tone, string> = {
  neutral: "border-border bg-muted text-muted-foreground",
  success: "border-emerald-600/30 bg-emerald-500/10 text-emerald-700",
  warning: "border-amber-600/30 bg-amber-500/10 text-amber-700",
  destructive: "border-destructive/40 bg-destructive/10 text-destructive",
  info: "border-sky-600/30 bg-sky-500/10 text-sky-700",
};

export function StatusBadge({
  status,
  className,
}: {
  status: string | null | undefined;
  className?: string;
}) {
  const tone = statusTone(status);
  return (
    <Badge
      variant="outline"
      className={cn("font-medium capitalize", TONE_CLASS[tone], className)}
    >
      {status ?? "unknown"}
    </Badge>
  );
}
