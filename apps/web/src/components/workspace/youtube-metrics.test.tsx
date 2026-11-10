import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { YouTubeMetrics } from "./youtube-metrics";
import type { User } from "@/lib/api";
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
const user: User = {
  id: "actor",
  email: "brand@example.test",
  display_name: "Brand",
  role: "BRAND",
  is_active: true,
};
function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  render(
    <QueryClientProvider client={client}>
      <YouTubeMetrics user={user} campaignID="" />
    </QueryClientProvider>,
  );
}
describe("public YouTube metrics", () => {
  it("does not claim a connection or fetch counters without a server key", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify({ configured: false, videos: [], cache_seconds: 900 }), { status: 200 }),
      );
    vi.stubGlobal("fetch", fetcher);
    mount();
    expect(await screen.findByText("YouTube is not connected yet.", { exact: true })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Load public counters" })).not.toBeInTheDocument();
    expect(screen.queryByText("Set up YouTube access", { exact: true })).not.toBeInTheDocument();
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
  it("keeps missing counters distinct from a real zero and shows provenance", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async (input: RequestInfo | URL) =>
          new Response(
            JSON.stringify(
              String(input).includes("/submissions/")
                ? {
                    metrics: {
                      source: "YOUTUBE_DATA_API",
                      video_id: "abcdefghijk",
                      views: 1024,
                      likes: 0,
                      comments: null,
                      fetched_at: "2026-11-10T12:00:00Z",
                      expires_at: "2026-11-10T12:15:00Z",
                      scope: "PUBLIC_LIFETIME_COUNTERS",
                      used_for_payouts: false,
                    },
                  }
                : {
                    configured: true,
                    videos: [
                      {
                        id: "approved",
                        campaign_title: "Approved campaign",
                        content_url: "https://youtu.be/abcdefghijk",
                      },
                    ],
                    cache_seconds: 900,
                  },
            ),
            { status: 200 },
          ),
      ),
    );
    mount();
    const controls = userEvent.setup();
    await controls.selectOptions(await screen.findByLabelText("Approved YouTube clip"), "approved");
    await controls.click(screen.getByRole("button", { name: "Load public counters" }));
    expect(await screen.findByText("1,024", { exact: true })).toBeVisible();
    expect(screen.getByText("0", { exact: true })).toBeVisible();
    expect(screen.getByText("Not provided", { exact: true })).toBeVisible();
    expect(screen.getByText(/Source: YouTube Data API/)).toBeVisible();
    expect(screen.getByText(/separate from simulated campaign analytics and payout/)).toBeVisible();
    expect(screen.queryByRole("button", { name: "Refresh from YouTube" })).not.toBeInTheDocument();
  });
});
