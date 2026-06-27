export interface BreadcrumbItem {
  label: string;
  href?: string;
}

const STATIC_ROUTES: Record<string, string> = {
  "/": "Overview",
  "/sessions": "Sessions",
  "/memories": "Memories",
  "/corrections": "Corrections",
  "/tasks": "Tasks",
};

const SESSION_TAB_LABELS: Record<string, string> = {
  turns: "Turns",
  atoms: "Atoms",
  scenes: "Scenes",
};

export function buildBreadcrumbs(
  pathname: string,
  searchParams: URLSearchParams,
  dynamicTitle?: string,
): BreadcrumbItem[] {
  if (pathname === "/" || pathname === "") {
    return [{ label: "Overview" }];
  }

  const staticLabel = STATIC_ROUTES[pathname];
  if (staticLabel) {
    return [{ label: staticLabel }];
  }

  const sessionMatch = pathname.match(/^\/sessions\/([^/]+)$/);
  if (sessionMatch) {
    const key = decodeURIComponent(sessionMatch[1]);
    const tab = searchParams.get("tab");
    const hasTab = tab && tab !== "turns" && SESSION_TAB_LABELS[tab];
    const sessionHref = `/sessions/${encodeURIComponent(key)}`;
    const title =
      dynamicTitle ??
      (key.length > 8 ? `${key.slice(0, 8)}…` : key);

    const crumbs: BreadcrumbItem[] = [
      { label: "Sessions", href: "/sessions" },
      {
        label: title,
        href: hasTab ? sessionHref : undefined,
      },
    ];

    if (hasTab) {
      crumbs.push({ label: SESSION_TAB_LABELS[tab] });
    }

    return crumbs;
  }

  const segment = pathname.split("/").filter(Boolean).pop() ?? "Page";
  return [
    {
      label: segment.charAt(0).toUpperCase() + segment.slice(1),
    },
  ];
}
