import { SessionsTable } from "@/components/sessions/sessions-table";

export const metadata = {
  title: "Sessions — MyPast Observer",
};

export default function SessionsPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Sessions</h1>
        <p className="text-muted-foreground text-sm">
          Agent conversations and the turns captured from each one. Click a row
          to inspect its pipeline, turns, atoms, and scenes.
        </p>
      </div>
      <SessionsTable />
    </div>
  );
}
