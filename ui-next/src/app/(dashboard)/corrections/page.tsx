import { CorrectionsTable } from "@/components/corrections/corrections-table";

export const metadata = { title: "Corrections — MyPast Observer" };

export default function CorrectionsPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Corrections</h1>
        <p className="text-muted-foreground text-sm">
          Human corrections that overlay distilled memory. They always win over
          the machine-distilled fact. Click a row for details.
        </p>
      </div>
      <CorrectionsTable />
    </div>
  );
}
