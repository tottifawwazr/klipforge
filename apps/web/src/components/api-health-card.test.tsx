import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ApiHealthView } from "./api-health-card";

const healthyApi = {
  status: "ok" as const,
  service: "klipforge-api" as const,
  version: "dev",
  timestamp: "2026-07-11T03:00:00Z",
  checks: {
    postgres: { status: "up" as const, latency_ms: 2 },
    redis: { status: "up" as const, latency_ms: 1 },
  },
};

describe("ApiHealthView", () => {
  it("renders verified dependencies when the API is healthy", () => {
    render(<ApiHealthView state="connected" health={healthyApi} />);

    expect(screen.getByText("API connection verified")).toBeInTheDocument();
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.getByText("redis")).toBeInTheDocument();
  });

  it("allows an unavailable connection to be retried", () => {
    const onRetry = vi.fn();
    render(<ApiHealthView state="unavailable" onRetry={onRetry} />);

    fireEvent.click(screen.getByRole("button", { name: "Retry check" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });
});
