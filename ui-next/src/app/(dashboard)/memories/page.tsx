import { MemoriesTable } from "@/components/memories/memories-table";

export const metadata = { title: "Memories — Mem9 Observer" };

export default function MemoriesPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Memories</h1>
        <p className="text-muted-foreground text-sm">
          Long-term knowledge: profile, preferences, entities, events (T3).
          Click a row for details.
        </p>
      </div>
      <MemoriesTable />
    </div>
  );
}
