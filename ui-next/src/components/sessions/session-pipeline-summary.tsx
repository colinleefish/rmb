import { statusTone } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { SessionRow } from "@/lib/types";

const TONE_TEXT: Record<ReturnType<typeof statusTone>, string> = {
  neutral: "text-muted-foreground",
  success: "text-emerald-700",
  warning: "text-amber-700",
  destructive: "text-destructive",
  info: "text-sky-700",
};

function StagePill({
  tier,
  status,
}: {
  tier: string;
  status: string | null | undefined;
}) {
  const label = status?.trim() || "idle";
  const tone = statusTone(label);
  return (
    <div className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
      <span className="font-mono font-medium">{tier}</span>
      <span
        className={cn("capitalize tabular-nums", TONE_TEXT[tone])}
      >
        {label}
      </span>
    </div>
  );
}

export function SessionPipelineSummary({
  session,
  layout = "stacked",
}: {
  session: Pick<
    SessionRow,
    | "t1_status"
    | "t2_status"
    | "t3_status"
    | "turn_count"
    | "atom_count"
    | "scene_count"
  >;
  layout?: "stacked" | "inline" | "stages-only";
}) {
  const counts = (
    <p className="text-muted-foreground text-xs tabular-nums">
      <span className="text-foreground/80 font-medium">
        {session.turn_count ?? 0}
      </span>{" "}
      turns
      <span className="text-muted-foreground/60 mx-1.5">·</span>
      <span className="text-foreground/80 font-medium">
        {session.atom_count ?? 0}
      </span>{" "}
      atoms
      <span className="text-muted-foreground/60 mx-1.5">·</span>
      <span className="text-foreground/80 font-medium">
        {session.scene_count ?? 0}
      </span>{" "}
      scenes
    </p>
  );

  const stages = (
    <div className="flex flex-wrap items-center gap-1">
      <StagePill tier="T1" status={session.t1_status} />
      <StagePill tier="T2" status={session.t2_status} />
      <StagePill tier="T3" status={session.t3_status} />
    </div>
  );

  if (layout === "stages-only") {
    return stages;
  }

  if (layout === "inline") {
    return (
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        {stages}
        {counts}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      {stages}
      {counts}
    </div>
  );
}
