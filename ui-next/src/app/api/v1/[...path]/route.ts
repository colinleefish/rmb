import { NextRequest, NextResponse } from "next/server";

// Server-side proxy for /api/v1/* → the mem9 Go API.
//
// Why a route handler instead of next.config rewrites: the upstream may require
// Basic auth, and rewrites cannot add request headers. This keeps credentials on
// the server (never shipped to the browser) and avoids CORS, since the browser
// only ever talks to this same-origin endpoint.
const API_BASE = process.env.MEM9_API_URL ?? "http://localhost:8080";
const API_USER = process.env.MEM9_API_USER ?? "";
const API_PASS = process.env.MEM9_API_PASS ?? "";

function authHeader(): Record<string, string> {
  if (!API_USER || !API_PASS) return {};
  const token = Buffer.from(`${API_USER}:${API_PASS}`).toString("base64");
  return { Authorization: `Basic ${token}` };
}

async function forward(
  request: NextRequest,
  params: Promise<{ path: string[] }>,
) {
  const { path } = await params;
  const target = `${API_BASE}/api/v1/${path.map(encodeURIComponent).join("/")}${request.nextUrl.search}`;

  // Forward the body verbatim for mutating methods so the upstream sees the
  // same JSON the browser sent (e.g. correction create payloads).
  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  const reqBody = hasBody ? await request.text() : undefined;

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method: request.method,
      headers: {
        Accept: "application/json",
        ...(reqBody
          ? {
              "Content-Type":
                request.headers.get("content-type") ?? "application/json",
            }
          : {}),
        ...authHeader(),
      },
      body: reqBody,
      cache: "no-store",
    });
  } catch (err) {
    return NextResponse.json(
      { error: `cannot reach mem9 API at ${API_BASE}: ${(err as Error).message}` },
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

export function GET(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return forward(request, params);
}

export function POST(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return forward(request, params);
}

export function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  return forward(request, params);
}
