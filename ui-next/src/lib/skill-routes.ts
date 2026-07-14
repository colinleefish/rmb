/** Skill detail URL (static-export friendly). */
export function skillDetailHref(name: string, file?: string): string {
  const params = new URLSearchParams();
  params.set("name", name);
  if (file) params.set("file", file);
  return `/skills/detail?${params.toString()}`;
}

export function skillNameFromSearchParams(
  searchParams: URLSearchParams,
): string | null {
  const name = searchParams.get("name")?.trim();
  return name || null;
}

export function skillFileFromSearchParams(
  searchParams: URLSearchParams,
): string | null {
  const file = searchParams.get("file")?.trim();
  return file || null;
}
