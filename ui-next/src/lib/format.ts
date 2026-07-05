import type { ChatMessage } from "@/lib/types";

/** Read the first present, non-empty value across PascalCase/snake_case keys. */
export function pick<T = string>(obj: unknown, ...keys: string[]): T | null {
  if (!obj || typeof obj !== "object") return null;
  const record = obj as Record<string, unknown>;
  for (const key of keys) {
    const value = record[key];
    if (value !== null && value !== undefined && value !== "") {
      return value as T;
    }
  }
  return null;
}

export function fmtDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

export function fmtDateShort(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function fmtRelative(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const diffMs = Date.now() - d.getTime();
  const sec = Math.round(diffMs / 1000);
  const min = Math.round(sec / 60);
  const hr = Math.round(min / 60);
  const day = Math.round(hr / 24);
  if (sec < 60) return "just now";
  if (min < 60) return `${min}m ago`;
  if (hr < 24) return `${hr}h ago`;
  if (day < 30) return `${day}d ago`;
  return fmtDateShort(iso);
}

export function truncate(value: string | null | undefined, max: number): string {
  if (!value) return "";
  const t = value.trim();
  return t.length <= max ? t : `${t.slice(0, max - 1)}…`;
}

export function shortKey(key: string | null | undefined, len = 8): string {
  if (!key) return "—";
  return key.length > len ? `${key.slice(0, len)}…` : key;
}

/** Human label for a session row: T2 abstract, else short session key. */
export function sessionDisplayTitle(
  session: {
    abstract?: string | null;
    session_key: string;
  },
  maxLen = 72,
): string {
  const abstract = session.abstract?.trim();
  if (abstract) {
    const firstLine = abstract.split(/\n+/)[0]?.trim() || abstract;
    return truncate(firstLine, maxLen);
  }
  return shortKey(session.session_key) || "Untitled session";
}

export function sessionHasSummary(
  session: { abstract?: string | null },
): boolean {
  return Boolean(session.abstract?.trim());
}

export function parseJSONL(raw: string | null | undefined): ChatMessage[] {
  if (!raw) return [];
  return raw
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => {
      try {
        return JSON.parse(line) as ChatMessage;
      } catch {
        return { role: "?", content: line };
      }
    });
}

/** Strip Cursor-style user_query wrappers from archived turn text. */
export function stripUserQueryTags(text: string): string {
  return text
    .replace(/^\s*<user_query>\s*/i, "")
    .replace(/\s*<\/user_query>\s*$/i, "")
    .trim();
}

/** Split assistant paraphrase lead-in from the actual reply body. */
export function formatTurnMessage(
  role: string | undefined,
  content: string | undefined,
): { aside: string | null; body: string } {
  const normalizedRole = (role ?? "").toLowerCase();
  let body = stripUserQueryTags(content ?? "").trim();
  body = body.replace(/\n*\[REDACTED\]\s*$/gi, "").trim();

  if (!body) return { aside: null, body: "" };

  if (normalizedRole === "assistant") {
    const match = body.match(/^(You're saying[^\n]+)\n\n([\s\S]+)$/);
    if (match) {
      return { aside: match[1].trim(), body: match[2].trim() };
    }
  }

  return { aside: null, body };
}

export function turnMessagePreview(messages: ChatMessage[]): string {
  const user = messages.find((m) => (m.role ?? "").toLowerCase() === "user");
  const text = formatTurnMessage(user?.role, user?.content).body;
  return truncate(text.replace(/\s+/g, " "), 96) || "Empty turn";
}

export function turnRoleLabel(role: string | undefined): string {
  switch ((role ?? "").toLowerCase()) {
    case "user":
      return "You";
    case "assistant":
      return "Assistant";
    case "system":
      return "System";
    case "tool":
      return "Tool";
    default:
      return role?.trim() || "Message";
  }
}

export type Tone = "neutral" | "success" | "warning" | "destructive" | "info";

/** Map a pipeline/session/turn status string to a badge tone. */
export function statusTone(status: string | null | undefined): Tone {
  const s = (status ?? "").toLowerCase();
  if (s === "failed" || s === "error") return "destructive";
  if (s === "done" || s === "complete" || s === "completed" || s === "ready")
    return "success";
  if (s === "running" || s === "pending" || s === "queued") return "warning";
  if (s === "idle") return "neutral";
  return "info";
}
