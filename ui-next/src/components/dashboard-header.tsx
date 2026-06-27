"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { ChevronRight } from "lucide-react";

import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { usePageHeader } from "@/components/page-header-context";
import { buildBreadcrumbs } from "@/lib/breadcrumbs";
import { cn } from "@/lib/utils";

export function DashboardHeader() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { override } = usePageHeader();

  const crumbs = buildBreadcrumbs(
    pathname,
    searchParams,
    override?.title,
  );

  return (
    <header className="bg-background/80 sticky top-0 z-10 flex h-14 shrink-0 items-center gap-2 border-b px-4 backdrop-blur">
      <SidebarTrigger className="-ml-1" />
      <Separator
        orientation="vertical"
        className="mr-1 data-[orientation=vertical]:h-4"
      />
      <nav aria-label="Breadcrumb" className="min-w-0 flex-1">
        <ol className="flex min-w-0 items-center gap-1 text-sm">
          {crumbs.map((crumb, index) => {
            const isLast = index === crumbs.length - 1;
            return (
              <li
                key={`${crumb.label}-${index}`}
                className="flex min-w-0 items-center gap-1"
              >
                {index > 0 && (
                  <ChevronRight
                    className="text-muted-foreground size-3.5 shrink-0"
                    aria-hidden
                  />
                )}
                {crumb.href && !isLast ? (
                  <Link
                    href={crumb.href}
                    className="text-muted-foreground hover:text-foreground truncate transition-colors"
                  >
                    {crumb.label}
                  </Link>
                ) : (
                  <span
                    className={cn(
                      "truncate",
                      isLast
                        ? "text-foreground font-medium"
                        : "text-muted-foreground",
                    )}
                    aria-current={isLast ? "page" : undefined}
                  >
                    {crumb.label}
                  </span>
                )}
              </li>
            );
          })}
        </ol>
      </nav>
    </header>
  );
}
