// @vitest-environment node
import { afterEach, describe, expect, it, vi } from "vitest";
import { NextRequest } from "next/server";
import { POST } from "./route";
afterEach(() => vi.unstubAllGlobals());
const context = { params: Promise.resolve({ path: ["auth", "login"] }) };
describe("authenticated browser API bridge", () => {
  it("keeps access and refresh tokens in HttpOnly cookies and out of JSON", async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({ access_token: "fixture-access-token", expires_in: 900, user: { role: "BRAND" } }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json",
            "Set-Cookie":
              "klipforge_refresh_token=fixture-refresh-token; Path=/api/v1/auth; Max-Age=3600; HttpOnly",
          },
        },
      ),
    );
    vi.stubGlobal("fetch", fetcher);
    const request = new NextRequest("http://localhost:3002/api/backend/auth/login", {
      method: "POST",
      headers: {
        host: "127.0.0.1:3002",
        origin: "http://127.0.0.1:3002",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ email: "fixture@example.test", password: "Fixture123!" }),
    });
    const result = await POST(request, context);
    const json = await result.json();
    expect(result.status).toBe(200);
    expect(json).not.toHaveProperty("access_token");
    expect(result.cookies.get("klipforge_access")?.value).toBe("fixture-access-token");
    expect(result.cookies.get("klipforge_refresh_token")?.value).toBe("fixture-refresh-token");
    for (const cookie of result.headers.getSetCookie()) {
      expect(cookie).toContain("HttpOnly");
      expect(cookie).toContain("SameSite=lax");
    }
    expect(result.headers.get("cache-control")).toBe("no-store");
  });
  it("rejects cross-site mutation before contacting the API", async () => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const request = new NextRequest("http://localhost:3002/api/backend/auth/login", {
      method: "POST",
      headers: { host: "localhost:3002", origin: "https://attacker.example" },
      body: "{}",
    });
    const result = await POST(request, context);
    expect(result.status).toBe(403);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("rejects oversized declared bodies before proxying", async () => {
    const fetcher = vi.fn();
    vi.stubGlobal("fetch", fetcher);
    const request = new NextRequest("http://localhost:3002/api/backend/auth/login", {
      method: "POST",
      headers: { host: "localhost:3002", origin: "http://localhost:3002", "content-length": "1048577" },
      body: "{}",
    });
    expect((await POST(request, context)).status).toBe(413);
    expect(fetcher).not.toHaveBeenCalled();
  });
  it("preserves an empty logout response and clears both session cookies", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 204 })));
    const request = new NextRequest("http://localhost:3002/api/backend/auth/logout", {
      method: "POST",
      headers: { host: "localhost:3002", origin: "http://localhost:3002" },
    });
    const result = await POST(request, { params: Promise.resolve({ path: ["auth", "logout"] }) });
    expect(result.status).toBe(204);
    expect(await result.text()).toBe("");
    expect(result.cookies.get("klipforge_access")?.value).toBe("");
    expect(result.cookies.get("klipforge_refresh_token")?.value).toBe("");
    expect(result.headers.getSetCookie().every((cookie) => cookie.includes("Max-Age=0"))).toBe(true);
  });
});
