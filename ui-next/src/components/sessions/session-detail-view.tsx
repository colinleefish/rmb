"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";

import { useSetPageHeaderTitle } from "@/components/page-header-context";
import { SessionDetailHero } from "@/components/sessions/session-detail-hero";
import {
  parseSessionDetailTab,
  type SessionDetailTab,
} from "@/components/sessions/session-detail-types";
import {
  AtomsSection,
  ScenesSection,
} from "@/components/sessions/session-detail-sections";
import { TurnsSection } from "@/components/sessions/turn-messages";
import { getSession } from "@/lib/api";
import {
  sessionDisplayTitle,
  shortKey,
} from "@/lib/format";
import { sessionDetailHref } from "@/lib/session-routes";
import type { SessionDetail } from "@/lib/types";

export type { SessionDetailTab } from "@/components/sessions/session-detail-types";

export function SessionDetailView({ sessionKey }: { sessionKey: string }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const tab = parseSessionDetailTab(searchParams.get("tab"));

  const [detail, setDetail] = useState<SessionDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    setDetail(null);
    getSession(sessionKey)
      .then(setDetail)
      .catch((err: Error) => setError(err.message))
      .finally(() => setLoading(false));
  }, [sessionKey]);

  const setTab = (next: SessionDetailTab) => {
    router.replace(sessionDetailHref(sessionKey, next), { scroll: false });
  };

  const session = detail?.session;
  const heading = session
    ? sessionDisplayTitle(session, 120)
    : sessionKey;
  const breadcrumbTitle = session
    ? sessionDisplayTitle(session, 48)
    : shortKey(sessionKey);
  useSetPageHeaderTitle(breadcrumbTitle);

  const fullAbstract = session?.abstract?.trim() ?? null;
  const abstract =
    fullAbstract &&
    (fullAbstract.includes("\n") || fullAbstract.length > 120)
      ? fullAbstract
      : null;

  return (
    <div className="flex w-full flex-col gap-6">
      <div className="sticky top-14 z-[5] -mx-4 border-b bg-background/95 px-4 pt-1 pb-0 backdrop-blur-sm md:-mx-6 md:px-6">
        <SessionDetailHero
            loading={loading}
            title={heading}
            abstract={abstract}
            session={session ?? null}
            detail={detail}
            tab={tab}
            onTabChange={setTab}
          />
      </div>

      <div className="min-w-0">
        {error ? (
          <p className="text-destructive text-sm">Failed to load: {error}</p>
        ) : loading || !detail ? null : (
          <>
            {tab === "turns" && <TurnsSection turns={detail.turns} />}
            {tab === "atoms" && <AtomsSection atoms={detail.atoms} />}
            {tab === "scenes" && <ScenesSection scenes={detail.scenes} />}
          </>
        )}
      </div>
    </div>
  );
}
