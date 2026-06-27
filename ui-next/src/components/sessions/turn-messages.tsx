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
            "rounded-md px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap",
            isUser && "bg-muted/50 text-foreground",
            normalizedRole === "assistant" && "text-foreground/90",
            isSystem && "text-muted-foreground font-mono text-xs",
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
    <div className="divide-border/40 flex flex-col divide-y">
      {turns.map((turn) => {
        const messages = parseJSONL(turn.messages_jsonl);
        const preview = turnMessagePreview(messages);

        return (
          <article key={turn.id} className="py-6">
            <header className="mb-4 flex items-baseline gap-3 text-sm">
              <span className="text-foreground font-mono text-xs tabular-nums">
                #{turn.turn_index}
              </span>
              <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
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
        );
      })}
    </div>
  );
}
