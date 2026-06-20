import { ScenesTable } from "@/components/scenes/scenes-table";

export const metadata = { title: "Scenes — RMB Observer" };

export default function ScenesPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Scenes</h1>
        <p className="text-muted-foreground text-sm">
          Session-level summaries of what was going on (T2). Click a row for
          details.
        </p>
      </div>
      <ScenesTable />
    </div>
  );
}
