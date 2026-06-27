import { Suspense } from "react";

import { SessionDetailView } from "@/components/sessions/session-detail-view";
import { Skeleton } from "@/components/ui/skeleton";

export async function generateMetadata({
  params,
}: {
  params: Promise<{ sessionKey: string }>;
}) {
  const { sessionKey } = await params;
  return {
    title: `${sessionKey} — Sessions — RMB Observer`,
  };
}

function SessionDetailFallback() {
  return (
    <div className="flex flex-col gap-4">
      <Skeleton className="h-5 w-24" />
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-20 w-full" />
    </div>
  );
}

export default async function SessionDetailPage({
  params,
}: {
  params: Promise<{ sessionKey: string }>;
}) {
  const { sessionKey } = await params;
  return (
    <Suspense fallback={<SessionDetailFallback />}>
      <SessionDetailView sessionKey={sessionKey} />
    </Suspense>
  );
}
