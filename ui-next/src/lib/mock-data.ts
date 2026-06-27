// Mock dataset used as a fallback when the rmb Go backend is unreachable.
//
// The proxy route handler (app/api/v1/[...path]/route.ts) calls resolveMock()
// when it cannot reach the upstream API, so the dashboard still renders with
// representative data during local/offline development. Shapes mirror lib/types.

import type {
  AtomModel,
  CorrectionModel,
  MemoryModel,
  Overview,
  PipelineRow,
  SceneModel,
  SessionDetail,
  SessionRow,
  TaskModel,
  TurnRow,
} from "@/lib/types";

function iso(daysAgo: number, hour = 9, minute = 0): string {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() - daysAgo);
  d.setUTCHours(hour, minute, 0, 0);
  return d.toISOString();
}

function jsonl(messages: { role: string; content: string }[]): string {
  return messages.map((m) => JSON.stringify(m)).join("\n");
}

const SESSIONS: SessionRow[] = [
  {
    id: "ses_01HX2A",
    session_key: "a1b2c3d4e5f6",
    scope_key: "workspace/rmb",
    title: "Designing the memory pipeline",
    status: "ready",
    abstract:
      "Walked through the three-tier pipeline (T1 ingest, T2 summarize, T3 consolidate) and how atoms roll up into scenes and long-term memories.",
    turn_count: 12,
    atom_count: 24,
    scene_count: 5,
    t1_status: "done",
    t2_status: "done",
    t3_status: "done",
    uri: "rmb://session/a1b2c3d4e5f6",
    created_at: iso(6, 14, 12),
    updated_at: iso(1, 10, 30),
    last_turn_at: iso(1, 10, 30),
  },
  {
    id: "ses_01HX2B",
    session_key: "b2c3d4e5f6a1",
    scope_key: "workspace/rmb",
    title: "Debugging correction retraction",
    status: "running",
    abstract:
      "Investigated why retracted corrections were still influencing T3 consolidation. Narrowed it to a stale cache key.",
    turn_count: 8,
    atom_count: 15,
    scene_count: 3,
    t1_status: "done",
    t2_status: "done",
    t3_status: "running",
    uri: "rmb://session/b2c3d4e5f6a1",
    created_at: iso(4, 9, 5),
    updated_at: iso(0, 8, 45),
    last_turn_at: iso(0, 8, 45),
  },
  {
    id: "ses_01HX2C",
    session_key: "c3d4e5f6a1b2",
    scope_key: "workspace/personal",
    title: null,
    status: "pending",
    abstract:
      "What's the difference between an atom and a scene in this system? Trying to model my note-taking around it.",
    turn_count: 3,
    atom_count: 4,
    scene_count: 0,
    t1_status: "done",
    t2_status: "pending",
    t3_status: "idle",
    uri: "rmb://session/c3d4e5f6a1b2",
    created_at: iso(2, 16, 40),
    updated_at: iso(2, 17, 2),
    last_turn_at: iso(2, 17, 2),
  },
  {
    id: "ses_01HX2D",
    session_key: "d4e5f6a1b2c3",
    scope_key: "workspace/rmb",
    title: "Weekly planning sync",
    status: "ready",
    abstract:
      "Captured priorities for the week: ship the browse UI, fix pipeline backpressure, and draft the corrections API docs.",
    turn_count: 18,
    atom_count: 31,
    scene_count: 7,
    t1_status: "done",
    t2_status: "done",
    t3_status: "done",
    uri: "rmb://session/d4e5f6a1b2c3",
    created_at: iso(9, 11, 0),
    updated_at: iso(3, 13, 20),
    last_turn_at: iso(3, 13, 20),
  },
  {
    id: "ses_01HX2E",
    session_key: "e5f6a1b2c3d4",
    scope_key: null,
    title: "Failed ingest investigation",
    status: "failed",
    abstract:
      "A malformed JSONL turn crashed the T1 ingester. Logged the parse error and added a defensive guard.",
    turn_count: 5,
    atom_count: 2,
    scene_count: 0,
    t1_status: "failed",
    t2_status: "idle",
    t3_status: "idle",
    uri: "rmb://session/e5f6a1b2c3d4",
    created_at: iso(7, 20, 15),
    updated_at: iso(7, 20, 22),
    last_turn_at: iso(7, 20, 22),
  },
  {
    id: "ses_01HX2F",
    session_key: "f6a1b2c3d4e5",
    scope_key: "workspace/personal",
    title: "Reading notes: distributed systems",
    status: "ready",
    abstract:
      "Notes on consensus, quorum reads, and how eventual consistency maps onto the memory consolidation model.",
    turn_count: 22,
    atom_count: 40,
    scene_count: 9,
    t1_status: "done",
    t2_status: "done",
    t3_status: "done",
    uri: "rmb://session/f6a1b2c3d4e5",
    created_at: iso(12, 8, 30),
    updated_at: iso(5, 19, 10),
    last_turn_at: iso(5, 19, 10),
  },
];

const TURNS_BY_KEY: Record<string, TurnRow[]> = {
  a1b2c3d4e5f6: [
    {
      id: "turn_a1_0",
      turn_index: 0,
      uri: "rmb://session/a1b2c3d4e5f6/turn/0",
      messages_jsonl: jsonl([
        {
          role: "user",
          content: "Can you walk me through how the memory pipeline works end to end?",
        },
        {
          role: "assistant",
          content:
            "Sure. The pipeline has three tiers. T1 ingests raw turns and extracts atoms — small, atomic facts. T2 summarizes related atoms into scenes. T3 consolidates scenes into durable memories that survive across sessions.",
        },
      ]),
      created_at: iso(6, 14, 12),
      updated_at: iso(6, 14, 12),
    },
    {
      id: "turn_a1_1",
      turn_index: 1,
      uri: "rmb://session/a1b2c3d4e5f6/turn/1",
      messages_jsonl: jsonl([
        {
          role: "user",
          content: "What triggers T3 consolidation?",
        },
        {
          role: "assistant",
          content:
            "T3 runs once a session crosses the warmup threshold — by default 10 turns — or when a session is explicitly closed. It deduplicates overlapping scenes and promotes the high-priority ones to long-term memory.",
        },
      ]),
      created_at: iso(6, 14, 18),
      updated_at: iso(6, 14, 18),
    },
    {
      id: "turn_a1_2",
      turn_index: 2,
      uri: "rmb://session/a1b2c3d4e5f6/turn/2",
      messages_jsonl: jsonl([
        {
          role: "user",
          content: "How do corrections fit in?",
        },
        {
          role: "assistant",
          content:
            "Corrections are user-supplied statements that override or retract derived facts. They're applied during T3 so the consolidated memory reflects the truth, not the raw transcript.",
        },
      ]),
      created_at: iso(1, 10, 30),
      updated_at: iso(1, 10, 30),
    },
  ],
  b2c3d4e5f6a1: [
    {
      id: "turn_b2_0",
      turn_index: 0,
      uri: "rmb://session/b2c3d4e5f6a1/turn/0",
      messages_jsonl: jsonl([
        {
          role: "user",
          content:
            "A correction I retracted yesterday is still showing up in the consolidated memory. Why?",
        },
        {
          role: "assistant",
          content:
            "That points to a stale cache. T3 memoizes the resolved correction set per scope. If retraction doesn't bust that key, the old statement lingers until the next cold start.",
        },
      ]),
      created_at: iso(4, 9, 5),
      updated_at: iso(4, 9, 5),
    },
    {
      id: "turn_b2_1",
      turn_index: 1,
      uri: "rmb://session/b2c3d4e5f6a1/turn/1",
      messages_jsonl: jsonl([
        {
          role: "user",
          content: "Where is that cache key built?",
        },
        {
          role: "assistant",
          content:
            "In resolveCorrections() — it hashes only the scope key, not the correction version. Adding the max updated_at to the hash will invalidate it on retraction.",
        },
      ]),
      created_at: iso(0, 8, 45),
      updated_at: iso(0, 8, 45),
    },
  ],
};

const ATOMS: AtomModel[] = [
  {
    URI: "rmb://atom/0001",
    Category: "fact",
    Priority: 3,
    SceneName: "Pipeline architecture",
    Slug: "three-tier-pipeline",
    Content:
      "The memory pipeline has three tiers: T1 (ingest atoms), T2 (summarize scenes), T3 (consolidate memories).",
    CreatedAt: iso(6, 14, 13),
  },
  {
    URI: "rmb://atom/0002",
    Category: "preference",
    Priority: 2,
    SceneName: "Pipeline architecture",
    Slug: "warmup-threshold",
    Content: "Default warmup threshold before T3 runs is 10 turns.",
    CreatedAt: iso(6, 14, 19),
  },
  {
    URI: "rmb://atom/0003",
    Category: "decision",
    Priority: 4,
    SceneName: "Corrections",
    Slug: "corrections-applied-at-t3",
    Content: "Corrections are applied during T3 consolidation, not at ingest time.",
    CreatedAt: iso(1, 10, 31),
  },
  {
    URI: "rmb://atom/0004",
    Category: "fact",
    Priority: 1,
    SceneName: null,
    Slug: null,
    Content: "Atoms are small, atomic facts extracted from a single turn.",
    CreatedAt: iso(2, 16, 41),
  },
  {
    URI: "rmb://atom/0005",
    Category: "issue",
    Priority: 5,
    SceneName: "Corrections",
    Slug: "stale-correction-cache",
    Content:
      "Retracted corrections persist because the T3 cache key hashes only the scope, not the correction version.",
    CreatedAt: iso(0, 8, 46),
  },
  {
    URI: "rmb://atom/0006",
    Category: "fact",
    Priority: 2,
    SceneName: "Distributed systems notes",
    Slug: "quorum-reads",
    Content: "A quorum read requires R + W > N to guarantee read-your-writes.",
    CreatedAt: iso(5, 19, 11),
  },
];

const SCENES: SceneModel[] = [
  {
    URI: "rmb://scene/pipeline-architecture",
    DisplayName: "Pipeline architecture",
    Abstract: "How the three-tier memory pipeline is structured and triggered.",
    Body: "T1 ingests raw turns and extracts atoms. T2 groups related atoms into scenes with a short abstract. T3 consolidates scenes into durable memories once a session crosses the warmup threshold or is closed.",
    UpdatedAt: iso(1, 10, 32),
  },
  {
    URI: "rmb://scene/corrections",
    DisplayName: "Corrections",
    Abstract: "User-supplied overrides applied during consolidation.",
    Body: "Corrections are statements that retract or override derived facts. They are resolved per scope and applied at T3 so consolidated memory reflects user intent.",
    UpdatedAt: iso(0, 8, 47),
  },
  {
    URI: "rmb://scene/distributed-systems-notes",
    DisplayName: "Distributed systems notes",
    Abstract: "Reading notes on consensus and consistency models.",
    Body: "Covers quorum reads/writes, eventual consistency, and how consolidation resembles anti-entropy repair in Dynamo-style systems.",
    UpdatedAt: iso(5, 19, 12),
  },
];

const MEMORIES: MemoryModel[] = [
  {
    ID: "mem_0001",
    URI: "rmb://memory/pipeline-overview",
    Category: "knowledge",
    Slug: "pipeline-overview",
    Version: 3,
    Abstract: "The rmb memory pipeline is a three-tier system.",
    Body: "T1 extracts atoms from turns, T2 summarizes atoms into scenes, and T3 consolidates scenes into long-term memories. Corrections are applied at T3.",
    CreatedAt: iso(6, 14, 30),
    UpdatedAt: iso(1, 10, 33),
  },
  {
    ID: "mem_0002",
    URI: "rmb://memory/user-preferences",
    Category: "preference",
    Slug: "user-preferences",
    Version: 1,
    Abstract: "User prefers concise, technical explanations.",
    Body: "When explaining systems, lead with the architecture and keep prose tight. Avoid restating the question.",
    CreatedAt: iso(9, 11, 5),
    UpdatedAt: iso(3, 13, 21),
  },
  {
    ID: "mem_0003",
    URI: "rmb://memory/known-issues",
    Category: "issue",
    Slug: "known-issues",
    Version: 2,
    Abstract: "Open issue: stale correction cache in T3.",
    Body: "Retracted corrections can persist until cold start because the cache key omits correction version. Fix in progress.",
    CreatedAt: iso(4, 9, 10),
    UpdatedAt: iso(0, 8, 48),
  },
];

const TASKS: TaskModel[] = [
  {
    ID: "task_0001",
    Kind: "t3_consolidate",
    Status: "running",
    Progress: 62,
    ResultURI: null,
    Error: null,
    SessionID: "ses_01HX2B",
    CreatedAt: iso(0, 8, 50),
    UpdatedAt: iso(0, 9, 2),
  },
  {
    ID: "task_0002",
    Kind: "t2_summarize",
    Status: "done",
    Progress: 100,
    ResultURI: "rmb://scene/pipeline-architecture",
    Error: null,
    SessionID: "ses_01HX2A",
    CreatedAt: iso(1, 10, 20),
    UpdatedAt: iso(1, 10, 32),
  },
  {
    ID: "task_0003",
    Kind: "t1_ingest",
    Status: "failed",
    Progress: 18,
    ResultURI: null,
    Error: "invalid JSONL at line 4: unexpected end of input",
    SessionID: "ses_01HX2E",
    CreatedAt: iso(7, 20, 18),
    UpdatedAt: iso(7, 20, 22),
  },
  {
    ID: "task_0004",
    Kind: "t3_consolidate",
    Status: "queued",
    Progress: 0,
    ResultURI: null,
    Error: null,
    SessionID: "ses_01HX2C",
    CreatedAt: iso(2, 17, 5),
    UpdatedAt: iso(2, 17, 5),
  },
];

const CORRECTIONS: CorrectionModel[] = [
  {
    uri: "rmb://correction/0001",
    statement: "The warmup threshold is configurable per workspace, not global.",
    target_uris: ["rmb://atom/0002", "rmb://memory/pipeline-overview"],
    created_at: iso(3, 12, 0),
  },
  {
    uri: "rmb://correction/0002",
    statement: "Atoms can belong to multiple scenes, not just one.",
    target_uris: ["rmb://atom/0004"],
    created_at: iso(1, 15, 30),
  },
  {
    uri: "rmb://correction/0003",
    statement: "Corrections are also re-applied on session reopen.",
    target_uris: ["rmb://scene/corrections"],
    created_at: iso(0, 9, 15),
  },
];

const PIPELINE_ROWS: PipelineRow[] = SESSIONS.map((s) => ({
  T1Status: s.t1_status ?? "idle",
  T2Status: s.t2_status ?? "idle",
  T3Status: s.t3_status ?? "idle",
  WarmupThreshold: 10,
  session_key: s.session_key,
  session_uri: s.uri,
}));

const OVERVIEW: Overview = {
  counts: {
    sessions: SESSIONS.length,
    turns: SESSIONS.reduce((n, s) => n + s.turn_count, 0),
    atoms: ATOMS.length,
    scenes: SCENES.length,
    memories: MEMORIES.length,
    pipeline_states: PIPELINE_ROWS.length,
    tasks: TASKS.length,
    corrections: CORRECTIONS.length,
  },
};

function paginate<T>(items: T[], params: URLSearchParams) {
  const limit = Number(params.get("limit") ?? items.length);
  const offset = Number(params.get("offset") ?? 0);
  const q = params.get("q")?.toLowerCase().trim();
  let filtered = items;
  if (q) {
    filtered = items.filter((it) =>
      JSON.stringify(it).toLowerCase().includes(q),
    );
  }
  return {
    items: filtered.slice(offset, offset + limit),
    total: filtered.length,
  };
}

/**
 * Resolve a mock response for an `/api/v1/<path>` request.
 * Returns null when no mock exists for the route (caller should surface an error).
 */
export function resolveMock(
  method: string,
  pathSegments: string[],
  params: URLSearchParams,
): unknown | null {
  const path = pathSegments.join("/");

  if (method === "GET") {
    if (path === "browse/overview") return OVERVIEW;
    if (path === "browse/sessions") return paginate(SESSIONS, params);
    if (path === "browse/pipeline-state") return paginate(PIPELINE_ROWS, params);
    if (path === "browse/atoms") return paginate(ATOMS, params);
    if (path === "browse/scenes") return paginate(SCENES, params);
    if (path === "browse/memories") return paginate(MEMORIES, params);
    if (path === "browse/tasks") return paginate(TASKS, params);
    if (path === "corrections") {
      const target = params.get("target");
      const filtered = target
        ? CORRECTIONS.filter((c) => c.target_uris.includes(target))
        : CORRECTIONS;
      return paginate(filtered, params);
    }
    if (path.startsWith("browse/sessions/")) {
      const key = decodeURIComponent(pathSegments[pathSegments.length - 1]);
      const session = SESSIONS.find((s) => s.session_key === key);
      if (!session) return null;
      const detail: SessionDetail = {
        session,
        turns: TURNS_BY_KEY[key] ?? [],
        pipeline_state: {
          T1Status: session.t1_status ?? "idle",
          T2Status: session.t2_status ?? "idle",
          T3Status: session.t3_status ?? "idle",
          WarmupThreshold: 10,
        },
        atoms: ATOMS.filter((a) => a.SceneName != null).slice(0, 4),
        scenes: SCENES.slice(0, 2),
      };
      return detail;
    }
  }

  if (method === "POST" && path === "corrections") {
    return { uri: `rmb://correction/${Date.now()}`, target_uris: [] };
  }
  if (method === "DELETE" && path === "corrections") {
    return { uri: params.get("uri") ?? "", retracted: true };
  }

  return null;
}
