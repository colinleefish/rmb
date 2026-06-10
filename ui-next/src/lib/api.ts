import type {
  AtomModel,
  MemoryModel,
  Overview,
  PipelineRow,
  SceneModel,
  SessionDetail,
  SessionRow,
  TaskModel,
} from "@/lib/types";

// Proxied to the Go backend via next.config rewrites (see next.config.ts).
const API = "/api/v1";

async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    headers: { Accept: "application/json" },
  });
  const body = (await res.json().catch(() => ({}))) as
    | T
    | { error?: string };
  if (!res.ok) {
    const message =
      (body as { error?: string }).error ?? res.statusText ?? "request failed";
    throw new Error(message);
  }
  return body as T;
}

export function getOverview(): Promise<Overview> {
  return apiGet<Overview>("/browse/overview");
}

export async function listSessions(): Promise<SessionRow[]> {
  const { items } = await apiGet<{ items: SessionRow[] }>("/browse/sessions");
  return items ?? [];
}

export function getSession(sessionKey: string): Promise<SessionDetail> {
  return apiGet<SessionDetail>(
    `/browse/sessions/${encodeURIComponent(sessionKey)}`,
  );
}

async function listItems<T>(path: string): Promise<T[]> {
  const { items } = await apiGet<{ items: T[] }>(path);
  return items ?? [];
}

export const listAtoms = () => listItems<AtomModel>("/browse/atoms");
export const listScenes = () => listItems<SceneModel>("/browse/scenes");
export const listMemories = () => listItems<MemoryModel>("/browse/memories");
export const listPipelineStates = () =>
  listItems<PipelineRow>("/browse/pipeline-state");
export const listTasks = () => listItems<TaskModel>("/browse/tasks");
