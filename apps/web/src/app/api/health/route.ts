import { apiHealthSchema } from "@/lib/health";

const DEFAULT_API_URL = "http://127.0.0.1:8080";
const HEALTH_TIMEOUT_MS = 3_000;

function apiHealthUrl() {
  const baseUrl = process.env.API_INTERNAL_URL?.trim() || DEFAULT_API_URL;
  return new URL("/api/v1/health", baseUrl);
}

export async function GET() {
  try {
    const response = await fetch(apiHealthUrl(), {
      cache: "no-store",
      headers: { Accept: "application/json" },
      signal: AbortSignal.timeout(HEALTH_TIMEOUT_MS),
    });
    const payload: unknown = await response.json();
    const parsed = apiHealthSchema.safeParse(payload);

    if (!response.ok || !parsed.success) {
      return Response.json(
        { connected: false, error: "The API health check did not pass." },
        { status: 503, headers: { "Cache-Control": "no-store" } },
      );
    }

    const upstreamRequestId = response.headers.get("x-request-id");
    const headers = new Headers({ "Cache-Control": "no-store" });
    if (upstreamRequestId) {
      headers.set("x-upstream-request-id", upstreamRequestId);
    }

    return Response.json({ connected: true, api: parsed.data }, { status: 200, headers });
  } catch {
    return Response.json(
      { connected: false, error: "The API could not be reached." },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}
