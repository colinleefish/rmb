import { TasksTable } from "@/components/tasks/tasks-table";

export const metadata = { title: "Tasks — MyPast Observer" };

export default function TasksPage() {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Tasks</h1>
        <p className="text-muted-foreground text-sm">
          Async worker runs and their status. Click a row for details.
        </p>
      </div>
      <TasksTable />
    </div>
  );
}
