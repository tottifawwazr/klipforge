import { NextRequest, NextResponse } from "next/server";
import { isSameOrigin } from "@/lib/csrf";

const ACCESS_COOKIE = "klipforge_access";
const REFRESH_COOKIE = "klipforge_refresh_token";
const allowedAuth = new Set(["login", "register", "refresh", "logout", "logout-all", "me"]);
export const dynamic = "force-dynamic";
async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  if (
    !path.length ||
    path.some((segment) => !/^[a-zA-Z0-9_-]+$/.test(segment)) ||
    (path[0] === "auth" && (path.length !== 2 || !allowedAuth.has(path[1])))
  ) {
    return NextResponse.json({ error: { message: "Endpoint not found." } }, { status: 404 });
  }
  if (!["GET", "HEAD"].includes(request.method)) {
    const origin = request.headers.get("origin");
    if (!isSameOrigin(origin, request.headers.get("host")))
      return NextResponse.json({ error: { message: "Request origin is not allowed." } }, { status: 403 });
    const declaredLength = Number(request.headers.get("content-length") || 0);
    if (declaredLength > 1_048_576)
      return NextResponse.json({ error: { message: "Request is too large." } }, { status: 413 });
  }
  const headers = new Headers({ Accept: "application/json" });
  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  const refresh = request.cookies.get(REFRESH_COOKIE)?.value;
  if (access) headers.set("Authorization", `Bearer ${access}`);
  const key = request.headers.get("Idempotency-Key");
  if (key) headers.set("Idempotency-Key", key);
  if (refresh && path[0] === "auth") headers.set("Cookie", `${REFRESH_COOKIE}=${refresh}`);
  let body: string | undefined;
  if (!["GET", "HEAD"].includes(request.method)) {
    body = await request.text();
    if (new TextEncoder().encode(body).length > 1_048_576)
      return NextResponse.json({ error: { message: "Request is too large." } }, { status: 413 });
    if (body) headers.set("Content-Type", "application/json");
  }
  try {
    const base = process.env.API_INTERNAL_URL?.trim() || "http://127.0.0.1:8080";
    const url = new URL(`/api/v1/${path.join("/")}${request.nextUrl.search}`, base);
    const upstream = await fetch(url, {
      method: request.method,
      headers,
      body: body || undefined,
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.timeout(12_000),
    });
    const noBody = upstream.status === 204;
    const payload = noBody ? {} : await upstream.json();
    const token = typeof payload.access_token === "string" ? payload.access_token : undefined;
    delete payload.access_token;
    const response = noBody
      ? new NextResponse(null, { status: 204, headers: { "Cache-Control": "no-store", Vary: "Cookie" } })
      : NextResponse.json(payload, {
          status: upstream.status,
          headers: { "Cache-Control": "no-store", Vary: "Cookie" },
        });
    const secure = request.nextUrl.protocol === "https:";
    if (token && upstream.ok)
      response.cookies.set(ACCESS_COOKIE, token, {
        httpOnly: true,
        secure,
        sameSite: "lax",
        path: "/",
        maxAge: Math.min(Number(payload.expires_in) || 900, 3600),
      });
    for (const cookie of upstream.headers.getSetCookie()) {
      const match = cookie.match(/^klipforge_refresh_token=([^;]*)/);
      if (!match) continue;
      const maxAge = cookie.match(/;\s*Max-Age=(-?\d+)/i);
      response.cookies.set(REFRESH_COOKIE, match[1], {
        httpOnly: true,
        secure,
        sameSite: "lax",
        path: "/api/backend/auth",
        maxAge: maxAge ? Number(maxAge[1]) : 2_592_000,
      });
    }
    if (
      path[0] === "auth" &&
      (["logout", "logout-all"].includes(path[1]) || (path[1] === "refresh" && upstream.status === 401))
    ) {
      response.cookies.set(ACCESS_COOKIE, "", {
        httpOnly: true,
        secure,
        sameSite: "lax",
        path: "/",
        maxAge: 0,
      });
      response.cookies.set(REFRESH_COOKIE, "", {
        httpOnly: true,
        secure,
        sameSite: "lax",
        path: "/api/backend/auth",
        maxAge: 0,
      });
    }
    return response;
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "API_UNAVAILABLE",
          message:
            "The API is unavailable. Check that the local API, PostgreSQL, and Redis are running, then try again.",
        },
      },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}
export { proxy as GET, proxy as POST, proxy as PATCH, proxy as PUT, proxy as DELETE };
