import {
  LayoutDashboard,
  ListTodo,
  MessagesSquare,
  ShieldCheck,
  Sparkles,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  id: string;
  label: string;
  href: string;
  icon: LucideIcon;
  /** Pipeline tier label shown beside the item (T1, T2, …). */
  tier?: string;
  /** Key into Overview.counts, for the sidebar badge. */
  countKey?: string;
  enabled: boolean;
}

export interface NavGroup {
  id: string;
  label: string;
  /** Shown under the group label when the sidebar is expanded. */
  hint?: string;
  items: NavItem[];
}

/** Sidebar navigation mirrors the distillation pyramid. */
export const NAV_GROUPS: NavGroup[] = [
  {
    id: "home",
    label: "Home",
    items: [
      {
        id: "overview",
        label: "Overview",
        href: "/",
        icon: LayoutDashboard,
        enabled: true,
      },
    ],
  },
  {
    id: "session",
    label: "Per session",
    hint: "Turns, pipeline workers, atoms, and scenes all live under a session.",
    items: [
      {
        id: "sessions",
        label: "Sessions",
        href: "/sessions",
        icon: MessagesSquare,
        tier: "T0–T2",
        countKey: "sessions",
        enabled: true,
      },
    ],
  },
  {
    id: "global",
    label: "Across sessions",
    hint: "Long-term memory rolled up from many conversations.",
    items: [
      {
        id: "memories",
        label: "Memories",
        href: "/memories",
        icon: Sparkles,
        tier: "T3",
        countKey: "memories",
        enabled: true,
      },
      {
        id: "corrections",
        label: "Corrections",
        href: "/corrections",
        icon: ShieldCheck,
        countKey: "corrections",
        enabled: true,
      },
    ],
  },
  {
    id: "ops",
    label: "Workers",
    hint: "Background jobs that advance the pipeline.",
    items: [
      {
        id: "tasks",
        label: "Tasks",
        href: "/tasks",
        icon: ListTodo,
        countKey: "tasks",
        enabled: true,
      },
    ],
  },
];

/** Flat list for callers that still need every item. */
export const NAV_ITEMS: NavItem[] = NAV_GROUPS.flatMap((g) => g.items);
