import type { SessionDetailTab } from "@/components/sessions/session-detail-types";
import { sessionDetailHref, sessionKeyFromSearchParams } from "@/lib/session-routes";
import { skillDetailHref, skillNameFromSearchParams } from "@/lib/skill-routes";

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

const STATIC_ROUTES: Record<string, string> = {
  "/": "Overview",
  "/sessions": "Sessions",
  "/sessions/detail": "Sessions",
  "/memories": "Memories",
  "/skills": "Skills",
  "/skills/detail": "Skills",
  "/corrections": "Corrections",
  "/tasks": "Tasks",
};

const SESSION_TAB_LABELS: Record<string, string> = {
  turns: "Turns",
  atoms: "Atoms",
  scenes: "Scenes",
};

function sessionDetailCrumbs(
  key: string,
  searchParams: URLSearchParams,
  dynamicTitle?: string,
): BreadcrumbItem[] {
  const tab = searchParams.get("tab") as SessionDetailTab | null;
  const hasTab = tab && tab !== "turns" && SESSION_TAB_LABELS[tab];
  const title =
    dynamicTitle ?? (key.length > 8 ? `${key.slice(0, 8)}…` : key);

  const crumbs: BreadcrumbItem[] = [
    { label: "Sessions", href: "/sessions" },
    {
      label: title,
      href: hasTab ? sessionDetailHref(key) : undefined,
    },
  ];

  if (hasTab) {
    crumbs.push({ label: SESSION_TAB_LABELS[tab] });
  }

  return crumbs;
}

export function buildBreadcrumbs(
  pathname: string,
  searchParams: URLSearchParams,
  dynamicTitle?: string,
): BreadcrumbItem[] {
  if (pathname === "/" || pathname === "") {
    return [{ label: "Overview" }];
  }

  if (pathname === "/sessions/detail") {
    const key = sessionKeyFromSearchParams(searchParams);
    if (!key) {
      return [{ label: "Sessions", href: "/sessions" }, { label: "Session" }];
    }
    return sessionDetailCrumbs(key, searchParams, dynamicTitle);
  }

  if (pathname === "/skills/detail") {
    const name = skillNameFromSearchParams(searchParams);
    if (!name) {
      return [{ label: "Skills", href: "/skills" }, { label: "Skill" }];
    }
    return [
      { label: "Skills", href: "/skills" },
      {
        label: dynamicTitle ?? name,
        href: skillDetailHref(name),
      },
    ];
  }

  const staticLabel = STATIC_ROUTES[pathname];
  if (staticLabel) {
    return [{ label: staticLabel }];
  }

  const segment = pathname.split("/").filter(Boolean).pop() ?? "Page";
  return [
    {
      label: segment.charAt(0).toUpperCase() + segment.slice(1),
    },
  ];
}
