import { AliasesView } from "@/components/aliases/aliases-view";

export const metadata = { title: "Aliases — MyPast Observer" };

export default function AliasesPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Aliases</h1>
        <p className="text-muted-foreground text-sm">
          Declare that two memory slugs are the same entity. Aliases fold into
          the canonical at distillation and in search results. Click a row for
          details or retract.
        </p>
      </div>
      <AliasesView />
    </div>
  );
}
