import { describe, expect, it, vi } from "vitest";
import { getApiHealth } from "./health";

const healthyApi = {
  status: "ok",
  service: "klipforge-api",
  version: "dev",
  timestamp: "2026-07-11T03:00:00Z",
  checks: {
    postgres: { status: "up", latency_ms: 2 },
    redis: { status: "up", latency_ms: 1 },
  },
};

describe("getApiHealth", () => {
  it("returns a validated upstream health response", async () => {
    const fetcher = vi.fn(async () =>
      Response.json({ connected: true, api: healthyApi }, { status: 200 }),
    ) as unknown as typeof fetch;

    await expect(getApiHealth(fetcher)).resolves.toMatchObject({ service: "klipforge-api", status: "ok" });
  });

  it("rejects an unavailable upstream response", async () => {
    const fetcher = vi.fn(async () =>
      Response.json({ connected: false, error: "The API could not be reached." }, { status: 503 }),
    ) as unknown as typeof fetch;

    await expect(getApiHealth(fetcher)).rejects.toThrow("The API could not be reached.");
  });
});
