import type {
  AliasModel,
  AliasCandidateModel,
  CorrectionModel,
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

export const pageAtoms = (req: PageRequest) =>
  listPage<AtomModel>("/browse/atoms", req);
export const pageScenes = (req: PageRequest) =>
  listPage<SceneModel>("/browse/scenes", req);
export const pageMemories = (req: PageRequest) =>
  listPage<MemoryModel>("/browse/memories", req);
export const pageTasks = (req: PageRequest) =>
  listPage<TaskModel>("/browse/tasks", req);

export const listPipelineStates = () =>
  listItems<PipelineRow>("/browse/pipeline-state");

// Corrections: human corrections that overlay distilled memory.
// `target` filters to corrections attached to a single memory URI.
export const listCorrections = (target?: string) =>
  listItems<CorrectionModel>(
    target
      ? `/corrections?target=${encodeURIComponent(target)}`
      : "/corrections",
  );

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

// Aliases: declare one memory URI is the same entity as another (preferences/entities).
// `uri` filters to aliases where it appears on either side (alias or canonical).
export const listAliases = (uri?: string) =>
  listItems<AliasModel>(
    uri ? `/aliases?uri=${encodeURIComponent(uri)}` : "/aliases",
  );

export function createAlias(input: {
  alias_uri: string;
  canonical_uri: string;
  note?: string;
}): Promise<{ uri: string; alias_uri: string; canonical_uri: string }> {
  return apiSend("POST", "/aliases", {
    alias_uri: input.alias_uri,
    canonical_uri: input.canonical_uri,
    note: input.note ?? "",
  });
}

export function retractAlias(
  uri: string,
): Promise<{ uri: string; retracted: boolean }> {
  return apiSend("DELETE", `/aliases?uri=${encodeURIComponent(uri)}`);
}

// Alias candidates: machine-proposed pairs from the suggest worker, awaiting
// human confirmation. `status` filters by pending|confirmed|rejected|all.
export const listAliasCandidates = (status?: string) =>
  listItems<AliasCandidateModel>(
    status
      ? `/alias-candidates?status=${encodeURIComponent(status)}`
      : "/alias-candidates",
  );

export function confirmAliasCandidate(
  id: string,
): Promise<{ uri: string; alias_uri: string; canonical_uri: string }> {
  return apiSend("POST", "/alias-candidates/confirm", { id });
}

export function rejectAliasCandidate(
  id: string,
): Promise<{ id: string; rejected: boolean }> {
  return apiSend("POST", "/alias-candidates/reject", { id });
}
