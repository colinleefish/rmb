import { Atom, Clapperboard, MessagesSquare } from "lucide-react";

import { statusTone, type Tone } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { SessionRow } from "@/lib/types";

const TONE_CHIP: Record<Tone, string> = {
  neutral: "border-border bg-muted text-muted-foreground",
  success: "border-emerald-200 bg-emerald-50 text-emerald-700",
  warning: "border-amber-200 bg-amber-50 text-amber-700",
  destructive: "border-destructive/20 bg-destructive/10 text-destructive",
  info: "border-sky-200 bg-sky-50 text-sky-700",
};

const TONE_DOT: Record<Tone, string> = {
  neutral: "bg-muted-foreground/40",
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  destructive: "bg-destructive",
  info: "bg-sky-500",
};

function StageChip({
  tier,
  status,
}: {
  tier: string;
  status: string | null | undefined;
}) {
  const label = status?.trim() || "idle";
  const tone = statusTone(label);
  const active = tone === "warning"; // running / pending / queued

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-1.5 py-0.5 text-xs font-medium",
        TONE_CHIP[tone],
      )}
    >
      <span className="relative flex size-1.5">
        {active && (
          <span
            className={cn(
              "absolute inline-flex size-full animate-ping rounded-full opacity-75",
              TONE_DOT[tone],
            )}
          />
        )}
        <span
          className={cn(
            "relative inline-flex size-1.5 rounded-full",
            TONE_DOT[tone],
          )}
        />
      </span>
      <span className="font-mono text-[10px] tracking-wide opacity-70">
        {tier}
      </span>
      <span className="capitalize tabular-nums">{label}</span>
    </span>
  );
}

function Count({
  icon: Icon,
  value,
  label,
}: {
  icon: typeof Atom;
  value: number;
  label: string;
}) {
  return (
    <span className="text-muted-foreground inline-flex items-center gap-1 text-xs tabular-nums">
      <Icon className="size-3.5 opacity-60" aria-hidden />
      <span className="text-foreground/80 font-medium">{value}</span>
      <span className="sr-only">{label}</span>
    </span>
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
    <div className="flex items-center gap-3">
      <Count icon={MessagesSquare} value={session.turn_count ?? 0} label="turns" />
      <Count icon={Atom} value={session.atom_count ?? 0} label="atoms" />
      <Count icon={Clapperboard} value={session.scene_count ?? 0} label="scenes" />
    </div>
  );

  const stages = (
    <div className="flex flex-wrap items-center gap-1">
      <StageChip tier="T1" status={session.t1_status} />
      <StageChip tier="T2" status={session.t2_status} />
      <StageChip tier="T3" status={session.t3_status} />
    </div>
  );

  if (layout === "stages-only") {
    return stages;
  }

  if (layout === "inline") {
    return (
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
        {stages}
        {counts}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {stages}
      {counts}
    </div>
  );
}
