"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Brain } from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import { getOverview } from "@/lib/api";
import type { OverviewCounts } from "@/lib/types";
import { NAV_GROUPS, type NavItem } from "@/lib/nav";

function NavMenuItem({
  item,
  pathname,
  counts,
}: {
  item: NavItem;
  pathname: string;
  counts: OverviewCounts | null;
}) {
  const isActive =
    item.href === "/"
      ? pathname === "/"
      : pathname === item.href || pathname.startsWith(`${item.href}/`);
  const count =
    item.countKey && counts
      ? counts[item.countKey as keyof OverviewCounts]
      : undefined;

  if (!item.enabled) {
    return (
      <SidebarMenuItem>
        <SidebarMenuButton
          disabled
          tooltip={`${item.label} — coming soon`}
          className="opacity-50"
        >
          <item.icon />
          <span className="flex flex-1 items-center gap-2">
            <span>{item.label}</span>
            {item.tier && (
              <span className="text-muted-foreground font-mono text-[10px]">
                {item.tier}
              </span>
            )}
          </span>
        </SidebarMenuButton>
        <SidebarMenuBadge className="text-[10px] tracking-wide uppercase">
          soon
        </SidebarMenuBadge>
      </SidebarMenuItem>
    );
  }

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<Link href={item.href} />}
        isActive={isActive}
        tooltip={
          item.tier ? `${item.label} (${item.tier})` : item.label
        }
      >
        <item.icon />
        <span className="flex flex-1 items-center gap-2">
          <span>{item.label}</span>
          {item.tier && (
            <span className="text-muted-foreground font-mono text-[10px] group-data-[collapsible=icon]:hidden">
              {item.tier}
            </span>
          )}
        </span>
      </SidebarMenuButton>
      {count !== undefined && <SidebarMenuBadge>{count}</SidebarMenuBadge>}
    </SidebarMenuItem>
  );
}

export function AppSidebar() {
  const pathname = usePathname();
  const [counts, setCounts] = useState<OverviewCounts | null>(null);

  useEffect(() => {
    let active = true;
    getOverview()
      .then((o) => active && setCounts(o.counts))
      .catch(() => {
        /* badge counts are best-effort */
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-2 px-2 py-1.5 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <Brain className="size-4" />
          </div>
          <div className="grid leading-tight group-data-[collapsible=icon]:hidden">
            <span className="text-sm font-semibold">RMB</span>
            <span className="text-muted-foreground text-xs">Observer</span>
          </div>
        </div>
      </SidebarHeader>

      <SidebarContent>
        {NAV_GROUPS.map((group) => (
          <SidebarGroup key={group.id}>
            <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
            {group.hint && (
              <p className="text-muted-foreground px-2 pb-1.5 text-[11px] leading-snug group-data-[collapsible=icon]:hidden">
                {group.hint}
              </p>
            )}
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => (
                  <NavMenuItem
                    key={item.id}
                    item={item}
                    pathname={pathname}
                    counts={counts}
                  />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarRail />
    </Sidebar>
  );
}
