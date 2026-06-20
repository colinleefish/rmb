// Types mirroring the mypast Go API.
//
// Note: the `browse` wrapper structs (SessionRow, TurnRow, Overview) carry
// explicit snake_case JSON tags, whereas raw GORM models (Atom, Scene,
// PipelineState) are marshalled with Go's default PascalCase field names.
// Detail rendering reads both cases defensively via `pick()` in lib/format.

export interface OverviewCounts {
  sessions: number;
  turns: number;
  atoms: number;
  scenes: number;
  memories: number;
  pipeline_states: number;
  tasks: number;
  corrections: number;
}

export interface Overview {
  counts: OverviewCounts;
}

export interface SessionRow {
  id: string;
  session_key: string;
  scope_key: string | null;
  title: string | null;
  status: string;
  abstract: string | null;
  turn_count: number;
  uri: string;
  created_at: string;
  updated_at: string;
}

export interface TurnRow {
  id: string;
  turn_index: number;
  uri: string;
  messages_jsonl: string;
  created_at: string;
  updated_at: string;
}

// Raw GORM models are marshalled with PascalCase keys (no JSON tags).
// All fields optional/loose on purpose; read via pick() defensively.
export interface AtomModel {
  URI?: string;
  Category?: string;
  Priority?: number;
  SceneName?: string | null;
  Slug?: string | null;
  Content?: string;
  CreatedAt?: string;
}

export interface SceneModel {
  URI?: string;
  DisplayName?: string | null;
  Abstract?: string | null;
  Body?: string | null;
  UpdatedAt?: string;
}

export interface MemoryModel {
  ID?: string;
  URI?: string;
  Category?: string;
  Slug?: string | null;
  Version?: number;
  Abstract?: string | null;
  Body?: string | null;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface TaskModel {
  ID?: string;
  Kind?: string;
  Status?: string;
  Progress?: number;
  ResultURI?: string | null;
  Error?: string | null;
  SessionID?: string | null;
  CreatedAt?: string;
  UpdatedAt?: string;
}

// The corrections API returns explicit snake_case JSON (see handler/correction.go).
export interface CorrectionModel {
  uri: string;
  statement: string;
  target_uris: string[];
  created_at: string;
}

// The aliases API returns explicit snake_case JSON (see handler/alias.go).
export interface AliasModel {
  uri: string;
  alias_uri: string;
  canonical_uri: string;
  note: string;
  created_at: string;
}

export interface PipelineStateModel {
  T1Status?: string;
  T2Status?: string;
  T3Status?: string;
  WarmupThreshold?: number;
}

// pipeline-state list rows embed the PascalCase model plus snake_case joins.
export interface PipelineRow extends PipelineStateModel {
  session_key?: string;
  session_uri?: string;
}

export interface SessionDetail {
  session: SessionRow;
  turns: TurnRow[];
  pipeline_state: PipelineStateModel | null;
  atoms: AtomModel[];
  scenes: SceneModel[];
}

export interface ChatMessage {
  role?: string;
  content?: string;
  [key: string]: unknown;
}
