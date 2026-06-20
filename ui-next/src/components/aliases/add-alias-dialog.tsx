"use client";

import { useState } from "react";
import { Plus } from "lucide-react";

import { MemoryUriInput } from "@/components/aliases/memory-uri-input";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { createAlias } from "@/lib/api";

const noteInputClass =
  "border-input placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 w-full rounded-lg border bg-transparent px-2.5 py-2 text-xs transition-colors outline-none focus-visible:ring-3";

export function AddAliasDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: () => void;
}) {
  const [aliasURI, setAliasURI] = useState("");
  const [canonicalURI, setCanonicalURI] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setAliasURI("");
    setCanonicalURI("");
    setNote("");
    setError(null);
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) reset();
    onOpenChange(next);
  };

  const handleSubmit = () => {
    const alias = aliasURI.trim();
    const canonical = canonicalURI.trim();
    if (!alias || !canonical) return;
    setSubmitting(true);
    setError(null);
    createAlias({ alias_uri: alias, canonical_uri: canonical, note: note.trim() })
      .then(() => {
        reset();
        onOpenChange(false);
        onCreated();
      })
      .catch((err: Error) => setError(err.message))
      .finally(() => setSubmitting(false));
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Create alias</DialogTitle>
          <DialogDescription>
            Declare the alias URI (redundant slug) to be the same entity as the
            canonical URI. Both must be preferences or entities in the same
            category. Start typing to see memory suggestions.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5 text-xs">
            <span className="text-muted-foreground font-medium">Alias URI</span>
            <MemoryUriInput
              value={aliasURI}
              onChange={setAliasURI}
              placeholder="mem9://entities/aliyun-rds-instance"
              excludeURIs={[canonicalURI]}
            />
          </label>
          <label className="flex flex-col gap-1.5 text-xs">
            <span className="text-muted-foreground font-medium">Canonical URI</span>
            <MemoryUriInput
              value={canonicalURI}
              onChange={setCanonicalURI}
              placeholder="mem9://entities/aliyun-rds"
              excludeURIs={[aliasURI]}
            />
          </label>
          <label className="flex flex-col gap-1.5 text-xs">
            <span className="text-muted-foreground font-medium">Note (optional)</span>
            <input
              className={noteInputClass}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="same DB, instance vs prod-role"
            />
          </label>
          {error && <p className="text-destructive text-sm">{error}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleSubmit}
              disabled={submitting || !aliasURI.trim() || !canonicalURI.trim()}
            >
              <Plus />
              {submitting ? "Creating…" : "Create alias"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
