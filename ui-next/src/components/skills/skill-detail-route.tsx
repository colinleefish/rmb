"use client";

import { useSearchParams } from "next/navigation";

import { SkillDetailView } from "@/components/skills/skill-detail-view";
import { skillNameFromSearchParams } from "@/lib/skill-routes";

export function SkillDetailRoute() {
  const searchParams = useSearchParams();
  const name = skillNameFromSearchParams(searchParams);

  if (!name) {
    return (
      <p className="text-muted-foreground text-sm">
        Missing skill name. Open a skill from the catalog.
      </p>
    );
  }

  return <SkillDetailView name={name} />;
}
