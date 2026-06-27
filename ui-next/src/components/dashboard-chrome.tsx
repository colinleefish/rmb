"use client";

import { Suspense, type ReactNode } from "react";

import { AppSidebar } from "@/components/app-sidebar";
import { DashboardHeader } from "@/components/dashboard-header";
import { PageHeaderProvider } from "@/components/page-header-context";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";

function HeaderFallback() {
  return (
    <header className="bg-background/80 sticky top-0 z-10 flex h-14 shrink-0 items-center gap-2 border-b px-4 backdrop-blur">
      <Skeleton className="h-4 w-32" />
    </header>
  );
}

export function DashboardChrome({ children }: { children: ReactNode }) {
  return (
    <PageHeaderProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset>
          <Suspense fallback={<HeaderFallback />}>
            <DashboardHeader />
          </Suspense>
          <div className="flex-1 p-4 md:p-6">{children}</div>
        </SidebarInset>
      </SidebarProvider>
    </PageHeaderProvider>
  );
}
