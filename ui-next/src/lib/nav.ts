import {
  Boxes,
  GitBranch,
  LayoutDashboard,
  ListTodo,
  MessagesSquare,
  ShieldCheck,
  Sparkles,
  Atom,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  id: string;
  label: string;
  href: string;
  icon: LucideIcon;
  /** Key into Overview.counts, for the sidebar badge. */
  countKey?: string;
  enabled: boolean;
}

// The mypast distillation pyramid: sessions → turns → atoms → scenes → memories.
export const NAV_ITEMS: NavItem[] = [
  { id: "overview", label: "Overview", href: "/", icon: LayoutDashboard, enabled: true },
  { id: "sessions", label: "Sessions", href: "/sessions", icon: MessagesSquare, countKey: "sessions", enabled: true },
  { id: "atoms", label: "Atoms", href: "/atoms", icon: Atom, countKey: "atoms", enabled: true },
  { id: "scenes", label: "Scenes", href: "/scenes", icon: Boxes, countKey: "scenes", enabled: true },
  { id: "memories", label: "Memories", href: "/memories", icon: Sparkles, countKey: "memories", enabled: true },
  { id: "corrections", label: "Corrections", href: "/corrections", icon: ShieldCheck, countKey: "corrections", enabled: true },
  { id: "pipeline", label: "Pipeline", href: "/pipeline", icon: GitBranch, countKey: "pipeline_states", enabled: true },
  { id: "tasks", label: "Tasks", href: "/tasks", icon: ListTodo, countKey: "tasks", enabled: true },
];
