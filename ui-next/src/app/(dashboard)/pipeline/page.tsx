import { PipelineTable } from "@/components/pipeline/pipeline-table";

export const metadata = { title: "Pipeline — Mem9 Observer" };

export default function PipelinePage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Pipeline</h1>
        <p className="text-muted-foreground text-sm">
          Per-session worker state across T1/T2/T3. Click a row to open the
          session.
        </p>
      </div>
      <PipelineTable />
    </div>
  );
}
