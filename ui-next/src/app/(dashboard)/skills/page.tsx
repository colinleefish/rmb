import { SkillsTable } from "@/components/skills/skills-table";

export const metadata = { title: "Skills — RMB Observer" };

export default function SkillsPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Skills</h1>
        <p className="text-muted-foreground text-sm">
          Curated agent playbooks. Click a skill to browse SKILL.md and bundled
          files.
        </p>
      </div>
      <SkillsTable />
    </div>
  );
}
