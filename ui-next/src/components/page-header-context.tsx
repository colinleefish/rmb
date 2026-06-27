"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

interface PageHeaderOverride {
  title?: string;
}

interface PageHeaderContextValue {
  override: PageHeaderOverride | null;
  setOverride: (override: PageHeaderOverride | null) => void;
}

const PageHeaderContext = createContext<PageHeaderContextValue | null>(null);

export function PageHeaderProvider({ children }: { children: ReactNode }) {
  const [override, setOverrideState] = useState<PageHeaderOverride | null>(
    null,
  );
  const setOverride = useCallback(
    (next: PageHeaderOverride | null) => setOverrideState(next),
    [],
  );
  const value = useMemo(
    () => ({ override, setOverride }),
    [override, setOverride],
  );

  return (
    <PageHeaderContext.Provider value={value}>
      {children}
    </PageHeaderContext.Provider>
  );
}

export function usePageHeader() {
  const ctx = useContext(PageHeaderContext);
  if (!ctx) {
    throw new Error("usePageHeader must be used within PageHeaderProvider");
  }
  return ctx;
}

/** Set a dynamic breadcrumb leaf (e.g. session title). Clears on unmount. */
export function useSetPageHeaderTitle(title: string | undefined) {
  const { setOverride } = usePageHeader();

  useEffect(() => {
    if (title) setOverride({ title });
    else setOverride(null);
    return () => setOverride(null);
  }, [title, setOverride]);
}
