import { AtomsTable } from "@/components/atoms/atoms-table";

export const metadata = { title: "Atoms — MyPast Observer" };

export default function AtomsPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Atoms</h1>
        <p className="text-muted-foreground text-sm">
          Typed facts extracted from chat turns (T1). Click a row for details.
        </p>
      </div>
      <AtomsTable />
    </div>
  );
}
