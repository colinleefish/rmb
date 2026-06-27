import { SessionsTable } from "@/components/sessions/sessions-table";

export const metadata = {
  title: "Sessions — RMB Observer",
};

export default function SessionsPage() {
  return (
    <div className="flex w-full flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Sessions</h1>
        <p className="text-muted-foreground text-sm max-w-[65ch]">
          Each row is one agent conversation. Distillation shows worker status
          (T1–T3) plus turn, atom, and scene counts. Updated is when the latest
          turn was inserted. Click a row to open the full session page.
        </p>
      </div>
      <SessionsTable />
    </div>
  );
}
