"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";

import { useSetPageHeaderTitle } from "@/components/page-header-context";
import { SkillFileTree } from "@/components/skills/skill-file-tree";
import { SkillFileViewer } from "@/components/skills/skill-file-viewer";
import { getSkill } from "@/lib/api";
import { fmtRelative } from "@/lib/format";
import { skillDetailHref } from "@/lib/skill-routes";
import type { SkillDetail } from "@/lib/types";

function defaultFile(detail: SkillDetail | null): string | null {
  if (!detail) return null;
  if (detail.files["SKILL.md"]) return "SKILL.md";
  const paths = Object.keys(detail.files).sort();
  return paths[0] ?? null;
}

export function SkillDetailView({ name }: { name: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const fileParam = searchParams.get("file");

  const [detail, setDetail] = useState<SkillDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    setDetail(null);
    getSkill(name)
      .then(setDetail)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [name]);

  const selectedFile = useMemo(() => {
    if (fileParam && detail?.files[fileParam] !== undefined) return fileParam;
    return defaultFile(detail);
  }, [fileParam, detail]);

  useSetPageHeaderTitle(detail?.skill.name ?? name);

  const setFile = (path: string) => {
    router.replace(skillDetailHref(name, path), { scroll: false });
  };

  if (error) {
    return <p className="text-destructive text-sm">Failed to load: {error}</p>;
  }

  if (loading || !detail) {
    return <p className="text-muted-foreground text-sm">Loading skill…</p>;
  }

  const content = selectedFile ? (detail.files[selectedFile] ?? "") : "";

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="space-y-2 border-b pb-4">
        <h1 className="text-2xl font-semibold tracking-tight">
          {detail.skill.name}
        </h1>
        <p className="text-muted-foreground max-w-3xl text-sm">
          {detail.skill.description}
        </p>
        <div className="text-muted-foreground flex flex-wrap gap-4 text-xs">
          <span>v{detail.skill.version}</span>
          <span>updated {fmtRelative(detail.skill.updated_at)}</span>
          <span className="font-mono">{detail.skill.uri}</span>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[240px_minmax(0,1fr)]">
        <div className="rounded-md border p-2">
          <SkillFileTree
            tree={detail.tree}
            selected={selectedFile}
            onSelect={setFile}
          />
        </div>
        <SkillFileViewer path={selectedFile} content={content} />
      </div>
    </div>
  );
}
