import type {
  CorrectionModel,
  AtomModel,
  MemoryModel,
  Overview,
  PipelineRow,
  SceneModel,
  SessionDetail,
  SessionRow,
  TaskModel,
  SkillDetail,
  SkillRow,
} from "@/lib/types";

// Proxied to the Go backend via the App Router handler at app/api/v1/[...path]/route.ts.
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

async function apiSend<T>(
  method: "POST" | "DELETE",
  path: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      Accept: "application/json",
      ...(body != null ? { "Content-Type": "application/json" } : {}),
    },
    body: body != null ? JSON.stringify(body) : undefined,
  });
  const data = (await res.json().catch(() => ({}))) as T | { error?: string };
  if (!res.ok) {
    const message =
      (data as { error?: string }).error ?? res.statusText ?? "request failed";
    throw new Error(message);
  }
  return data as T;
}

export function getOverview(): Promise<Overview> {
  return apiGet<Overview>("/browse/overview");
}

export function getSession(sessionKey: string): Promise<SessionDetail> {
  return apiGet<SessionDetail>(
    `/browse/sessions/${encodeURIComponent(sessionKey)}`,
  );
}

export function getSkill(name: string): Promise<SkillDetail> {
  return apiGet<SkillDetail>(`/browse/skills/${encodeURIComponent(name)}`);
}

// Server-side pagination contract for the browse list endpoints.
export interface PageRequest {
  limit: number;
  offset: number;
  q?: string;
  sort?: string;
  order?: "asc" | "desc";
}

export interface Page<T> {
  items: T[];
  total: number;
}

async function listPage<T>(path: string, req: PageRequest): Promise<Page<T>> {
  const params = new URLSearchParams({
    limit: String(req.limit),
    offset: String(req.offset),
  });
  if (req.q) params.set("q", req.q);
  if (req.sort) {
    params.set("sort", req.sort);
    params.set("order", req.order ?? "desc");
  }
  const { items, total } = await apiGet<{ items: T[]; total: number }>(
    `${path}?${params.toString()}`,
  );
  return { items: items ?? [], total: total ?? 0 };
}

export const pageSessions = (req: PageRequest) =>
  listPage<SessionRow>("/browse/sessions", req);
export const pagePipelineStates = (req: PageRequest) =>
  listPage<PipelineRow>("/browse/pipeline-state", req);
export const pageCorrections = (req: PageRequest, target?: string) => {
  const params = new URLSearchParams({
    limit: String(req.limit),
    offset: String(req.offset),
  });
  if (req.q) params.set("q", req.q);
  if (req.sort) {
    params.set("sort", req.sort);
    params.set("order", req.order ?? "desc");
  }
  if (target) params.set("target", target);
  return apiGet<{ items: CorrectionModel[]; total: number }>(
    `/corrections?${params.toString()}`,
  ).then(({ items, total }) => ({
    items: items ?? [],
    total: total ?? 0,
  }));
};

export const pageAtoms = (req: PageRequest) =>
  listPage<AtomModel>("/browse/atoms", req);
export const pageScenes = (req: PageRequest) =>
  listPage<SceneModel>("/browse/scenes", req);
export const pageMemories = (req: PageRequest) =>
  listPage<MemoryModel>("/browse/memories", req);
export const pageSkills = (req: PageRequest) =>
  listPage<SkillRow>("/browse/skills", req);
export const pageTasks = (req: PageRequest) =>
  listPage<TaskModel>("/browse/tasks", req);

export async function listSessions(): Promise<SessionRow[]> {
  const { items } = await pageSessions({ limit: 200, offset: 0 });
  return items;
}

export async function listCorrections(target?: string): Promise<CorrectionModel[]> {
  const { items } = await pageCorrections({ limit: 200, offset: 0 }, target);
  return items;
}

export function createCorrection(input: {
  statement: string;
  target_uris: string[];
}): Promise<{ uri: string; target_uris: string[] }> {
  return apiSend("POST", "/corrections", {
    target_uris: input.target_uris,
    statement: input.statement,
  });
}

export function retractCorrection(
  uri: string,
): Promise<{ uri: string; retracted: boolean }> {
  return apiSend("DELETE", `/corrections?uri=${encodeURIComponent(uri)}`);
}
