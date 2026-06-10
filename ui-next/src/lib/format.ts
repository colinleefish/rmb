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

export function truncate(value: string | null | undefined, max: number): string {
  if (!value) return "";
  const t = value.trim();
  return t.length <= max ? t : `${t.slice(0, max - 1)}…`;
}

export function shortKey(key: string | null | undefined, len = 8): string {
  if (!key) return "—";
  return key.length > len ? `${key.slice(0, len)}…` : key;
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
