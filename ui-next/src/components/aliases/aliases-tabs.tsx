"use client";

import { useState } from "react";

import { AliasCandidatesView } from "@/components/aliases/alias-candidates-view";
import { AliasesView } from "@/components/aliases/aliases-view";
import { Button } from "@/components/ui/button";

type Tab = "active" | "suggestions";

export function AliasesTabs() {
  const [tab, setTab] = useState<Tab>("active");

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-muted/50 inline-flex w-fit gap-1 rounded-lg p-1">
        <Button
          variant={tab === "active" ? "default" : "ghost"}
          size="sm"
          onClick={() => setTab("active")}
        >
          Active aliases
        </Button>
        <Button
          variant={tab === "suggestions" ? "default" : "ghost"}
          size="sm"
          onClick={() => setTab("suggestions")}
        >
          Suggestions
        </Button>
      </div>
      {tab === "active" ? <AliasesView /> : <AliasCandidatesView />}
    </div>
  );
}
