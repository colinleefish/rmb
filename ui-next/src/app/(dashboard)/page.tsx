import { OverviewView } from "@/components/overview/overview-view";

export const metadata = { title: "Overview — Mem9 Observer" };

export default function HomePage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-muted-foreground text-sm">
          Row counts across the memory pipeline.
        </p>
      </div>
      <OverviewView />
    </div>
  );
}
