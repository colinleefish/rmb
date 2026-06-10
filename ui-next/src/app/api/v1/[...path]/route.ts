import { NextRequest, NextResponse } from "next/server";

// Server-side proxy for /api/v1/* → the mypast Go API.
//
// Why a route handler instead of next.config rewrites: the upstream may require
// Basic auth, and rewrites cannot add request headers. This keeps credentials on
// the server (never shipped to the browser) and avoids CORS, since the browser
// only ever talks to this same-origin endpoint.
const API_BASE = process.env.MYPAST_API_URL ?? "http://localhost:8080";
const API_USER = process.env.MYPAST_API_USER ?? "";
const API_PASS = process.env.MYPAST_API_PASS ?? "";

function authHeader(): Record<string, string> {
  if (!API_USER || !API_PASS) return {};
  const token = Buffer.from(`${API_USER}:${API_PASS}`).toString("base64");
  return { Authorization: `Basic ${token}` };
}

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  const { path } = await params;
  const target = `${API_BASE}/api/v1/${path.map(encodeURIComponent).join("/")}${request.nextUrl.search}`;

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      headers: { Accept: "application/json", ...authHeader() },
      cache: "no-store",
    });
  } catch (err) {
    return NextResponse.json(
      { error: `cannot reach mypast API at ${API_BASE}: ${(err as Error).message}` },
      { status: 502 },
    );
  }

  const body = await upstream.text();
  return new NextResponse(body, {
    status: upstream.status,
    headers: {
      "content-type": upstream.headers.get("content-type") ?? "application/json",
    },
  });
}
