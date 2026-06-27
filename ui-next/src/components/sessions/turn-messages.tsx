import {
  fmtDateTime,
  formatTurnMessage,
  turnMessagePreview,
  turnRoleLabel,
  parseJSONL,
} from "@/lib/format";
import { cn } from "@/lib/utils";
import type { ChatMessage, TurnRow } from "@/lib/types";

function TurnMessage({ message }: { message: ChatMessage }) {
  const role = message.role;
  const { aside, body } = formatTurnMessage(role, message.content);
  if (!aside && !body) return null;

  const normalizedRole = (role ?? "").toLowerCase();
  const isUser = normalizedRole === "user";
  const isSystem = normalizedRole === "system" || normalizedRole === "tool";

  return (
    <div className="flex flex-col gap-1.5">
      <span
        className={cn(
          "text-xs font-medium tracking-wide",
          isUser && "text-sky-700 dark:text-sky-400",
          normalizedRole === "assistant" && "text-foreground/80",
          isSystem && "text-muted-foreground",
        )}
      >
        {turnRoleLabel(role)}
      </span>
      {aside && (
        <p className="text-muted-foreground pl-3 text-xs leading-relaxed italic">
          {aside}
        </p>
      )}
      {body && (
        <div
          className={cn(
            "rounded-lg text-sm leading-relaxed whitespace-pre-wrap",
            isUser && "bg-muted/60 text-foreground border-border/60 border px-3.5 py-2.5",
            normalizedRole === "assistant" && "text-foreground/90",
            isSystem && "text-muted-foreground bg-muted/40 px-3 py-2 font-mono text-xs",
          )}
        >
          {body}
        </div>
      )}
    </div>
  );
}

export function TurnsSection({ turns }: { turns: TurnRow[] }) {
  if (!turns.length)
    return <p className="text-muted-foreground text-sm">No turns yet.</p>;

  return (
    <ol className="flex flex-col">
      {turns.map((turn, idx) => {
        const messages = parseJSONL(turn.messages_jsonl);
        const preview = turnMessagePreview(messages);
        const isLast = idx === turns.length - 1;

        return (
          <li key={turn.id} className="flex gap-4">
            {/* Timeline rail: numbered marker + connector line */}
            <div className="flex flex-col items-center">
              <span className="bg-background text-muted-foreground ring-border flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-medium tabular-nums ring-1">
                {turn.turn_index}
              </span>
              {!isLast && <span className="bg-border w-px flex-1" />}
            </div>

            <article className={cn("min-w-0 flex-1", isLast ? "pb-1" : "pb-8")}>
              <header className="mb-3 flex items-baseline justify-between gap-3">
                <span className="text-muted-foreground min-w-0 flex-1 truncate text-sm">
                  {preview}
                </span>
                <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                  {fmtDateTime(turn.created_at)}
                </span>
              </header>
              <div className="flex flex-col gap-4">
                {messages.length === 0 ? (
                  <p className="text-muted-foreground text-sm">No messages.</p>
                ) : (
                  messages.map((message, i) => (
                    <TurnMessage key={i} message={message} />
                  ))
                )}
              </div>
            </article>
          </li>
        );
      })}
    </ol>
  );
}
