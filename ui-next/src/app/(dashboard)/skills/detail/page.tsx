import { Suspense } from "react";

import { SkillDetailRoute } from "@/components/skills/skill-detail-route";
import { Skeleton } from "@/components/ui/skeleton";

export const metadata = {
  title: "Skill — Skills — RMB Observer",
};

function SkillDetailFallback() {
  return (
    <div className="flex flex-col gap-4">
      <Skeleton className="h-8 w-64" />
      <Skeleton className="h-20 w-full" />
    </div>
  );
}

export default function SkillDetailPage() {
  return (
    <Suspense fallback={<SkillDetailFallback />}>
      <SkillDetailRoute />
    </Suspense>
  );
}
