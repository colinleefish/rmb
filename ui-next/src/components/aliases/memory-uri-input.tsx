"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";

import { pageMemories } from "@/lib/api";
import { pick, truncate } from "@/lib/format";
import type { MemoryModel } from "@/lib/types";

function useDebounced(value: string, ms: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

function isMergeable(m: MemoryModel): boolean {
  const cat = pick(m, "Category", "category");
  return cat === "entities" || cat === "preferences";
}

function memoryURI(m: MemoryModel): string {
  return pick(m, "URI", "uri") ?? "";
}

function memoryLabel(m: MemoryModel): string {
  const slug = pick(m, "Slug", "slug");
  const abstract = pick(m, "Abstract", "abstract");
  if (slug) return slug;
  return truncate(abstract ?? pick(m, "Body", "body"), 48) || memoryURI(m);
}

export function MemoryUriInput({
  value,
  onChange,
  placeholder,
  excludeURIs = [],
  disabled,
}: {
  value: string;
  onChange: (uri: string) => void;
  placeholder?: string;
  excludeURIs?: string[];
  disabled?: boolean;
}) {
  const listId = useId();
  const wrapperRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [suggestions, setSuggestions] = useState<MemoryModel[]>([]);
  const debounced = useDebounced(value, 250);
  const excludeKey = excludeURIs.filter(Boolean).join("\0");

  const loadSuggestions = useCallback(
    (query: string) => {
      const excluded = new Set(excludeURIs.filter(Boolean));
      setLoading(true);
      pageMemories({
        limit: 12,
        offset: 0,
        q: query.trim() || undefined,
        sort: "updated_at",
        order: "desc",
      })
        .then(({ items }) => {
          const filtered = items.filter((m) => {
            const uri = memoryURI(m);
            return uri && isMergeable(m) && !excluded.has(uri);
          });
          setSuggestions(filtered.slice(0, 8));
        })
        .catch(() => setSuggestions([]))
        .finally(() => setLoading(false));
    },
    [excludeKey, excludeURIs],
  );

  useEffect(() => {
    if (!open) return;
    loadSuggestions(debounced);
  }, [open, debounced, loadSuggestions]);

  useEffect(() => {
    const onDocClick = (e: MouseEvent) => {
      if (!wrapperRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, []);

  const inputClass =
    "border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 w-full rounded-lg border bg-transparent px-2.5 py-2 font-mono text-xs transition-colors outline-none focus-visible:ring-3";

  return (
    <div ref={wrapperRef} className="relative">
      <input
        className={inputClass}
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        role="combobox"
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        onChange={(e) => {
          onChange(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
      />
      {open && (loading || suggestions.length > 0) && (
        <ul
          id={listId}
          role="listbox"
          className="bg-popover text-popover-foreground absolute z-50 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border p-1 shadow-md"
        >
          {loading && suggestions.length === 0 ? (
            <li className="text-muted-foreground px-2.5 py-2 text-xs">
              Searching…
            </li>
          ) : (
            suggestions.map((m) => {
              const uri = memoryURI(m);
              const cat = pick(m, "Category", "category");
              return (
                <li key={uri}>
                  <button
                    type="button"
                    role="option"
                    className="hover:bg-muted flex w-full flex-col gap-0.5 rounded-md px-2.5 py-2 text-left"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => {
                      onChange(uri);
                      setOpen(false);
                    }}
                  >
                    <span className="font-mono text-[11px] break-all">{uri}</span>
                    <span className="text-muted-foreground text-[10px]">
                      {cat}
                      {memoryLabel(m) !== uri ? ` · ${memoryLabel(m)}` : ""}
                    </span>
                  </button>
                </li>
              );
            })
          )}
        </ul>
      )}
    </div>
  );
}
