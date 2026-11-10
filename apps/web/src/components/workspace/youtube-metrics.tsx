"use client";
import { useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, type User } from "@/lib/api";
type Index = {
  configured: boolean;
  videos: { id: string; campaign_title: string; content_url: string }[];
  cache_seconds: number;
};
type Snapshot = {
  source: "YOUTUBE_DATA_API";
  video_id: string;
  views: number;
  likes: number | null;
  comments: number | null;
  fetched_at: string;
  expires_at: string;
  scope: string;
  used_for_payouts: false;
};
export function YouTubeMetrics({ user, campaignID }: { user: User; campaignID: string }) {
  const client = useQueryClient();
  const [selected, setSelected] = useState("");
  const [requested, setRequested] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const index = useQuery({
    queryKey: ["youtube-index", user.id, campaignID],
    queryFn: () =>
      api<Index>(`integrations/youtube${campaignID ? `?campaign_id=${encodeURIComponent(campaignID)}` : ""}`),
    retry: false,
  });
  const key = ["youtube-public", user.id, requested];
  const snapshot = useQuery({
    queryKey: key,
    queryFn: () => api<{ metrics: Snapshot }>(`integrations/youtube/submissions/${requested}`),
    enabled: !!requested && !!index.data?.configured,
    retry: false,
  });
  const metrics = snapshot.data?.metrics;
  function load(event: FormEvent) {
    event.preventDefault();
    setError("");
    if (selected === requested) void snapshot.refetch();
    else setRequested(selected);
  }
  async function refresh() {
    if (refreshing || !requested) return;
    setRefreshing(true);
    setError("");
    try {
      const data = await api<{ metrics: Snapshot }>(`admin/youtube/submissions/${requested}/refresh`, {
        method: "POST",
        body: "{}",
      });
      client.setQueryData(key, data);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Public counters could not be refreshed.");
    } finally {
      setRefreshing(false);
    }
  }
  return (
    <section className="kf-panel mt-6" aria-labelledby="youtube-heading">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="youtube-heading" className="text-lg font-semibold">
          YouTube public metrics
        </h2>
        <span className="rounded-md border border-border bg-surface-muted px-2 py-1 text-xs font-semibold">
          {index.isPending
            ? "Checking connection"
            : index.error
              ? "Connection unavailable"
              : index.data?.configured
                ? "API key configured"
                : "Not connected"}
        </span>
      </div>
      <p className="mt-3 text-sm leading-6 text-foreground-secondary">
        Read-only lifetime counters for approved YouTube clips. These are separate from simulated campaign
        analytics and payout calculations. Public statistics do not verify creator ownership.
      </p>
      {index.isPending ? (
        <p role="status" className="mt-4 text-sm">
          Checking YouTube setup...
        </p>
      ) : index.error ? (
        <p role="alert" className="mt-4 text-sm text-danger">
          {index.error instanceof Error ? index.error.message : "Could not load YouTube setup."}
        </p>
      ) : !index.data?.configured ? (
        <div className="kf-notice mt-4">
          <strong>YouTube is not connected yet.</strong> An admin needs to add a server API key before real
          counters can load. No live values are shown until a request succeeds.
          {user.role === "ADMIN" && (
            <details className="mt-3 text-sm">
              <summary className="min-h-11 cursor-pointer py-3 font-semibold">Set up YouTube access</summary>
              <p className="mt-2 leading-6">
                Enable YouTube Data API v3 in your Google Cloud project, create a restricted API key, set
                YOUTUBE_API_KEY in the API server environment, then restart the API. Keep the key on the
                server.
              </p>
              <a
                className="mt-3 inline-block py-2 font-semibold underline"
                href="https://developers.google.com/youtube/v3/guides/authentication"
                target="_blank"
                rel="noopener noreferrer"
              >
                Read the official setup guide
              </a>
            </details>
          )}
        </div>
      ) : (
        <>
          {!index.data.videos.length ? (
            <p className="mt-4 text-sm text-foreground-secondary">
              No approved YouTube clips are available in this selection. Submit a YouTube video and have it
              reviewed first. This view lists up to 20 recent approved clips.
            </p>
          ) : (
            <form onSubmit={load} className="mt-5 flex flex-wrap items-end gap-3">
              <div className="min-w-0 flex-1">
                <label htmlFor="youtube-submission" className="text-sm font-medium">
                  Approved YouTube clip
                </label>
                <select
                  id="youtube-submission"
                  required
                  value={selected}
                  disabled={refreshing || snapshot.isFetching}
                  onChange={(event) => {
                    setSelected(event.target.value);
                    setRequested("");
                    setError("");
                  }}
                  className="kf-input mt-2"
                >
                  <option value="">Choose a clip</option>
                  {index.data.videos.map((video) => (
                    <option key={video.id} value={video.id}>
                      {video.campaign_title} / {video.content_url}
                    </option>
                  ))}
                </select>
              </div>
              <button disabled={!selected || snapshot.isFetching || refreshing} className="kf-button">
                {snapshot.isFetching ? "Loading counters..." : "Load public counters"}
              </button>
            </form>
          )}
          {(error || snapshot.error) && (
            <p
              role="alert"
              className="mt-4 rounded-xl border border-danger-border bg-danger-subtle p-3 text-sm text-danger-on-subtle"
            >
              {error ||
                (snapshot.error instanceof Error
                  ? snapshot.error.message
                  : "YouTube counters are unavailable.")}
            </p>
          )}
          {metrics && (
            <div className="mt-5">
              <div className="grid gap-3 sm:grid-cols-3">
                {[
                  ["Views", metrics.views],
                  ["Likes", metrics.likes],
                  ["Comments", metrics.comments],
                ].map(([label, value]) => (
                  <div key={String(label)} className="rounded-xl border border-border bg-surface-subtle p-4">
                    <h3 className="text-sm text-foreground-secondary">{label}</h3>
                    <p className="mt-2 text-xl font-semibold tabular-nums">
                      {value === null ? "Not provided" : new Intl.NumberFormat("en-US").format(Number(value))}
                    </p>
                  </div>
                ))}
              </div>
              <p className="mt-4 text-xs leading-5 text-foreground-muted">
                Source: YouTube Data API. Retrieved {new Date(metrics.fetched_at).toLocaleString()}; cached
                until {new Date(metrics.expires_at).toLocaleString()}. Shares and saves are not supplied by
                this endpoint.
              </p>
              <div className="mt-3 flex flex-wrap gap-3">
                <a
                  href={`https://www.youtube.com/watch?v=${metrics.video_id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="kf-button kf-button-secondary"
                >
                  Open video on YouTube
                </a>
                {user.role === "ADMIN" && (
                  <button
                    disabled={refreshing || snapshot.isFetching}
                    className="kf-button kf-button-secondary"
                    onClick={() => void refresh()}
                  >
                    {refreshing ? "Refreshing..." : "Refresh from YouTube"}
                  </button>
                )}
              </div>
              <p className="mt-3 text-xs leading-5 text-foreground-muted">
                Results are cached for 15 minutes. Admin refresh requests are limited to once per video per
                minute; the server also caps external requests per day.
              </p>
            </div>
          )}
        </>
      )}
    </section>
  );
}
