import type { SessionDetail } from "@/lib/types";

export type SessionDetailTab = "turns" | "atoms" | "scenes";

export const SESSION_DETAIL_TABS: Array<{
  id: SessionDetailTab;
  label: string;
  count: (detail: SessionDetail) => number;
}> = [
  { id: "turns", label: "Turns", count: (d) => d.turns.length },
  { id: "atoms", label: "Atoms", count: (d) => d.atoms.length },
  { id: "scenes", label: "Scenes", count: (d) => d.scenes.length },
];

export function parseSessionDetailTab(value: string | null): SessionDetailTab {
  if (value === "atoms" || value === "scenes") return value;
  return "turns";
}
