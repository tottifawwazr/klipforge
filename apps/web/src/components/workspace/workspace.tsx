"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BarChart3,
  BookOpen,
  ClipboardCheck,
  LayoutDashboard,
  LogOut,
  Megaphone,
  Plus,
  RefreshCw,
  ArrowUpRight,
  LoaderCircle,
} from "lucide-react";
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { SubmissionEditor } from "@/components/workspace/submission-editor";
import { YouTubeMetrics } from "@/components/workspace/youtube-metrics";
import { ActionDialog, type Decision } from "@/components/workspace/action-dialog";
import { statusHelp, platformForVideo } from "@/lib/workflow";
import { NextStepCard, WorkspaceGuide } from "@/components/workspace/workspace-guide";
import { nextStepFor } from "@/lib/getting-started";
import { CampaignForm } from "@/components/workspace/campaign-form";
import { Leaderboard } from "@/components/workspace/leaderboard";
import { OperationsPanel } from "@/components/workspace/operations-panel";
import { Wallet, Bell, History } from "lucide-react";
import { Wordmark } from "@/components/brand/wordmark";
import { api, APIError, type User } from "@/lib/api";

type Campaign = {
  id: string;
  title: string;
  description: string;
  brief: string;
  status: string;
  platforms: string[];
  start_date: string;
  end_date: string;
  total_budget?: string;
  remaining_budget?: string;
  currency_code?: string;
  requirements: { type: string; description: string }[];
};
type Submission = {
  id: string;
  campaign_id: string;
  campaign_title?: string;
  clipper_display_name?: string;
  content_url: string;
  platform: string;
  status: string;
  caption: string;
  reason?: string;
  rejection_reason?: string;
};
type Overview = {
  metrics_mode: string;
  campaigns: number;
  submissions: number;
  approved_submissions: number;
  metrics: { views: number; likes: number; comments: number; shares: number; saves: number };
  engagement_rate: number;
  last_updated: string | null;
  timeline: { date: string; views: number; engagements: number }[];
};
const tabs = [
  { id: "overview", label: "Overview", Icon: LayoutDashboard },
  { id: "guide", label: "How it works", Icon: BookOpen },
  { id: "campaigns", label: "Campaigns", Icon: Megaphone },
  { id: "submissions", label: "Submissions", Icon: ClipboardCheck },
  { id: "analytics", label: "Analytics", Icon: BarChart3 },
];
tabs.push(
  { id: "payouts", label: "Payouts", Icon: Wallet },
  { id: "notifications", label: "Notifications", Icon: Bell },
  { id: "audit", label: "Audit", Icon: History },
);
const titles: Record<string, string> = {
  guide: "How KlipForge works",
  payouts: "Payout records",
  notifications: "Your notifications",
  audit: "Audit activity",
  overview: "Your workspace, at a glance",
  campaigns: "Campaigns",
  submissions: "Submission workspace",
  analytics: "Content performance",
};
const number = (value: number) => new Intl.NumberFormat("en-US").format(value);
function ErrorNotice({ error, retry }: { error: unknown; retry?: () => void }) {
  return (
    <div
      role="alert"
      className="rounded-xl border border-danger-border bg-danger-subtle p-4 text-sm text-danger-on-subtle"
    >
      <p>{error instanceof Error ? error.message : "Something went wrong. Please try again."}</p>
      {retry && (
        <button onClick={retry} className="mt-3 font-semibold underline">
          Try again
        </button>
      )}
    </div>
  );
}
function Status({ value }: { value: string }) {
  return (
    <span className="inline-flex rounded-md border border-border bg-surface-muted px-2 py-1 text-xs font-semibold text-foreground-secondary">
      {value.replaceAll("_", " ")}
    </span>
  );
}
function Loading() {
  return (
    <p role="status" className="flex items-center gap-3 py-12 text-sm text-foreground-secondary">
      <LoaderCircle size={18} className="animate-spin" aria-hidden="true" />
      Loading workspace...
    </p>
  );
}

export function Workspace() {
  const router = useRouter();
  const client = useQueryClient();
  const [decision, setDecision] = useState<Decision>();
  const [tab, setTab] = useState("overview");
  const [reviewStatus, setReviewStatus] = useState("");
  const [selected, setSelected] = useState("");
  const [creating, setCreating] = useState(false);
  const [demo, setDemo] = useState(false);
  const [editing, setEditing] = useState<Campaign>();
  const [editingSubmission, setEditingSubmission] = useState<Submission>();
  const [notice, setNotice] = useState("");
  const [actionError, setActionError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const session = useQuery({
    queryKey: ["session"],
    queryFn: () => api<{ user: User }>("auth/me"),
    retry: false,
  });
  const user = session.data?.user;
  useEffect(() => {
    if (session.error instanceof APIError && session.error.status === 401) router.replace("/login");
  }, [session.error, router]);
  const campaigns = useQuery({
    queryKey: ["campaigns", user?.id],
    queryFn: () =>
      api<{ campaigns: Campaign[] }>(
        `${user?.role === "BRAND" ? "brand/" : user?.role === "ADMIN" ? "admin/" : ""}campaigns?limit=100`,
      ),
    enabled: !!user,
  });
  const analytics = useQuery({
    queryKey: ["analytics", user?.id, tab === "analytics" ? selected : ""],
    queryFn: () =>
      api<{ analytics: Overview }>(
        tab === "analytics" && selected ? `campaigns/${selected}/analytics` : "analytics",
      ),
    enabled: !!user && (tab === "overview" || tab === "analytics"),
  });
  const submissionPath =
    user?.role === "CLIPPER"
      ? `clipper/submissions?limit=100${selected ? "&campaign_id=" + selected : ""}`
      : user?.role === "ADMIN"
        ? `admin/moderation/submissions?limit=100${reviewStatus ? "&status=" + reviewStatus : ""}`
        : selected
          ? `brand/campaigns/${selected}/moderation/submissions?limit=100${reviewStatus ? "&status=" + reviewStatus : ""}`
          : "";
  const submissions = useQuery({
    queryKey: ["submissions", user?.id, selected, reviewStatus],
    queryFn: () => api<{ submissions: Submission[] }>(submissionPath),
    enabled: !!user && tab === "submissions" && !!submissionPath,
  });
  const guideSubmissions = useQuery({
    queryKey: ["onboarding-submissions", user?.id],
    queryFn: () => api<{ submissions: Submission[] }>("clipper/submissions?limit=100"),
    enabled: user?.role === "CLIPPER" && tab === "overview",
  });
  async function mutate(path: string, body: unknown = {}) {
    if (busy) return "Another action is still saving. Please wait.";
    setBusy(true);
    setActionError(undefined);
    setNotice("");
    try {
      await api(path, { method: "POST", body: JSON.stringify(body) });
      const action = path.split("/").at(-1);
      setNotice(
        path.includes("/submissions/") && (action === "approve" || action === "reject")
          ? `Submission ${action === "approve" ? "approved" : "rejected"}. Choose ${action === "approve" ? "Approved" : "Rejected"} in Review status to find it again.`
          : action === "flag"
            ? "Submission flagged for further review. It remains in the review queue."
            : "Changes saved.",
      );
      await client.invalidateQueries({ predicate: (q) => q.queryKey[0] !== "session" });
    } catch (error) {
      setActionError(error);
      return error instanceof Error ? error.message : "The action could not be saved.";
    } finally {
      setBusy(false);
    }
  }
  async function logout() {
    try {
      await api("auth/logout", { method: "POST" });
      client.clear();
      router.replace("/login");
    } catch (error) {
      setActionError(error);
    }
  }
  if (!user)
    return (
      <main id="main-content" className="mx-auto w-full max-w-2xl p-8">
        <Wordmark href="/" />
        {session.error ? (
          <div className="mt-10">
            <ErrorNotice error={session.error} retry={() => void session.refetch()} />
          </div>
        ) : (
          <Loading />
        )}
      </main>
    );
  const list = campaigns.data?.campaigns || [];
  const data = analytics.data?.analytics;
  const changeTab = (id: string) => {
    setTab(id);
    setNotice("");
    setActionError(undefined);
    setCreating(false);
  };
  const nextStep = nextStepFor(user.role, list, guideSubmissions.data?.submissions ?? []);
  const openNextStep = () => {
    changeTab(nextStep.view);
    setSelected(
      user.role === "CLIPPER" && !list.some((c) => c.id === nextStep.campaignID)
        ? ""
        : (nextStep.campaignID ?? ""),
    );
    setDemo(false);
    setEditingSubmission(undefined);
    if (nextStep.editor) {
      setEditing(nextStep.editor === "existing" ? list.find((c) => c.id === nextStep.campaignID) : undefined);
      setCreating(true);
    }
  };
  const startPractice = () => {
    changeTab(user.role === "ADMIN" ? "submissions" : "campaigns");
    setSelected("");
    setEditingSubmission(undefined);
    if (user.role === "BRAND") {
      setEditing(undefined);
      setDemo(true);
      setCreating(true);
    }
  };
  return (
    <div className="min-h-dvh bg-canvas">
      {decision && <ActionDialog decision={decision} onClose={() => setDecision(undefined)} />}
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex min-h-18 max-w-7xl items-center justify-between gap-4 px-5 sm:px-8">
          <Wordmark href="/" />
          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-foreground-secondary sm:block">{user.display_name}</span>
            <Status value={user.role} />
            <button
              onClick={() => void logout()}
              aria-label="Sign out"
              className="rounded-xl p-3 text-foreground-secondary hover:bg-surface-muted"
            >
              <LogOut size={20} aria-hidden="true" />
            </button>
          </div>
        </div>
      </header>
      <div className="mx-auto grid max-w-7xl grid-cols-1 lg:grid-cols-[216px_minmax(0,1fr)]">
        <aside className="min-w-0 border-b border-border px-3 py-4 lg:min-h-[calc(100dvh-72px)] lg:border-b-0 lg:border-r lg:px-5 lg:py-8">
          <p className="mb-4 hidden px-3 text-xs font-semibold uppercase tracking-widest text-foreground-muted lg:block">
            {user.role.toLowerCase()} workspace
          </p>
          <nav aria-label="Workspace navigation" className="flex gap-1 overflow-x-auto lg:flex-col">
            {tabs
              .filter((t) => t.id !== "audit" || user.role === "ADMIN")
              .map(({ id, label, Icon }) => (
                <button
                  key={id}
                  onClick={() => changeTab(id)}
                  aria-current={tab === id ? "page" : undefined}
                  className={`flex min-h-11 shrink-0 items-center gap-3 rounded-xl px-3 py-3 text-sm font-semibold transition-colors ${tab === id ? "bg-primary-subtle text-primary-on-subtle" : "text-foreground-secondary hover:bg-surface-muted"}`}
                >
                  <Icon size={18} aria-hidden="true" />
                  {label}
                </button>
              ))}
          </nav>
        </aside>
        <main id="main-content" className="min-w-0 px-5 py-8 sm:px-8 lg:p-10">
          <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-sm text-foreground-muted">
                KlipForge / {tabs.find((t) => t.id === tab)?.label}
              </p>
              <h1 className="mt-2 text-2xl font-semibold tracking-tight text-foreground-strong sm:text-3xl">
                {titles[tab]}
              </h1>
              <p className="mt-3 max-w-xl text-sm leading-6 text-foreground-secondary">
                {tab === "guide"
                  ? "Follow the roles, learn the terms, and try the campaign workflow."
                  : tab === "campaigns"
                    ? user.role === "CLIPPER"
                      ? "Explore active briefs, join a campaign, and submit your published content."
                      : "Manage briefs, dates, budgets, and the campaign lifecycle."
                    : tab === "submissions"
                      ? "Keep content, review decisions, and the next action together."
                      : "A clear view of your campaigns and approved content."}
              </p>
            </div>
            {tab === "campaigns" && user.role === "BRAND" && (
              <button
                className="kf-button"
                onClick={() => {
                  setEditing(undefined);
                  setDemo(false);
                  setCreating((v) => !v);
                }}
              >
                <Plus size={18} aria-hidden="true" />
                {creating ? "Close form" : "New campaign"}
              </button>
            )}
          </div>
          {notice && (
            <p
              role="status"
              className="mb-5 rounded-xl border border-primary-border bg-primary-subtle p-4 text-sm text-primary-on-subtle"
            >
              {notice}
            </p>
          )}
          {!!actionError && (
            <div className="mb-5">
              <ErrorNotice error={actionError} />
            </div>
          )}
          {tab === "guide" && (
            <WorkspaceGuide
              role={user.role}
              onStart={startPractice}
              onOverview={() => changeTab("overview")}
              onPayouts={() => changeTab("payouts")}
            />
          )}
          {tab === "overview" &&
            campaigns.isSuccess &&
            (user.role !== "CLIPPER" || guideSubmissions.isSuccess) && (
              <NextStepCard
                role={user.role}
                step={nextStep}
                onAction={openNextStep}
                onGuide={() => changeTab("guide")}
              />
            )}
          {tab === "overview" && user.role === "CLIPPER" && guideSubmissions.error && (
            <div className="mb-6">
              <ErrorNotice error={guideSubmissions.error} retry={() => void guideSubmissions.refetch()} />
            </div>
          )}
          {(tab === "payouts" || tab === "notifications" || (tab === "audit" && user.role === "ADMIN")) && (
            <OperationsPanel user={user} mode={tab as "payouts" | "notifications" | "audit"} />
          )}
          {tab === "analytics" && (
            <div className="mb-6 block max-w-lg text-sm font-medium">
              <label htmlFor="workspace-select-1">Campaign</label>
              <select
                id="workspace-select-1"
                className="kf-input mt-2"
                value={selected}
                onChange={(e) => setSelected(e.target.value)}
              >
                <option value="">All your campaigns</option>
                {list.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.title}
                  </option>
                ))}
              </select>
            </div>
          )}
          {(tab === "overview" || tab === "analytics") && (
            <>
              <div className="kf-notice mb-6">
                <strong>Simulated social metrics.</strong> This MVP uses deterministic providers, not live
                TikTok, Instagram, or YouTube data.{" "}
                {data?.last_updated
                  ? `Last collected ${new Date(data.last_updated).toLocaleString()}.`
                  : "No collection has completed yet."}
              </div>
              {analytics.isPending ? (
                <Loading />
              ) : analytics.error ? (
                <ErrorNotice error={analytics.error} retry={() => void analytics.refetch()} />
              ) : (
                data && (
                  <>
                    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                      {[
                        ["Campaigns", number(data.campaigns)],
                        ["Approved clips", number(data.approved_submissions)],
                        ["Recorded views", number(data.metrics.views)],
                        ["Engagement rate", `${data.engagement_rate.toFixed(2)}%`],
                      ].map(([label, value]) => (
                        <section key={label} className="kf-panel">
                          <h2 className="text-sm font-medium text-foreground-secondary">{label}</h2>
                          <p className="mt-4 text-3xl font-semibold tabular-nums text-foreground-strong">
                            {value}
                          </p>
                        </section>
                      ))}
                    </div>
                    <section className="kf-panel mt-6">
                      <div className="flex flex-wrap items-start justify-between gap-4">
                        <div>
                          <h2 className="text-lg font-semibold">Recorded content views</h2>
                          <p className="mt-2 text-sm text-foreground-secondary">
                            Daily totals from the latest approved-clip snapshots, over the last 30 days.
                          </p>
                        </div>
                        <button
                          className="kf-button kf-button-secondary"
                          onClick={() => void analytics.refetch()}
                        >
                          <RefreshCw size={16} aria-hidden="true" />
                          Refresh
                        </button>
                      </div>
                      {data.timeline.length ? (
                        <>
                          <div
                            className="mt-8 h-60 min-w-0 overflow-hidden"
                            role="img"
                            aria-label="Line chart of recorded daily views. Exact values are available below."
                          >
                            <ResponsiveContainer width="100%" height="100%">
                              <LineChart data={data.timeline}>
                                <XAxis dataKey="date" tick={{ fontSize: 11 }} minTickGap={40} />
                                <YAxis width={55} tick={{ fontSize: 11 }} />
                                <Tooltip />
                                <Line
                                  dataKey="views"
                                  name="Recorded views"
                                  type="monotone"
                                  stroke="var(--kf-color-primary)"
                                  strokeWidth={2}
                                  dot={false}
                                  isAnimationActive={false}
                                />
                              </LineChart>
                            </ResponsiveContainer>
                          </div>
                          <details className="mt-5 text-sm">
                            <summary className="cursor-pointer py-2 font-medium">
                              View exact daily values
                            </summary>
                            <div className="mt-3 overflow-x-auto">
                              <table className="w-full text-left">
                                <thead>
                                  <tr>
                                    <th className="p-2">Date</th>
                                    <th className="p-2">Views</th>
                                    <th className="p-2">Engagements</th>
                                  </tr>
                                </thead>
                                <tbody>
                                  {data.timeline.map((p) => (
                                    <tr key={p.date} className="border-t border-border-subtle">
                                      <td className="p-2">{p.date}</td>
                                      <td className="p-2 tabular-nums">{number(p.views)}</td>
                                      <td className="p-2 tabular-nums">{number(p.engagements)}</td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </details>
                        </>
                      ) : (
                        <div className="mt-8 rounded-xl bg-surface-muted p-8 text-center text-sm leading-6 text-foreground-secondary">
                          No metric snapshots yet. Metrics are collected for approved submissions.
                        </div>
                      )}
                    </section>
                  </>
                )
              )}
            </>
          )}
          {tab === "analytics" && selected && <Leaderboard campaignID={selected} />}
          {tab === "analytics" && (
            <YouTubeMetrics key={selected || "all"} user={user} campaignID={selected} />
          )}
          {tab === "campaigns" && (
            <>
              {creating && (
                <CampaignForm
                  key={editing?.id || (demo ? "demo" : "new")}
                  campaign={editing}
                  demo={demo && !editing}
                  onCreated={() => {
                    setCreating(false);
                    setNotice(editing ? "Campaign updated." : "Draft campaign created.");
                    void campaigns.refetch();
                  }}
                />
              )}
              {campaigns.isPending ? (
                <Loading />
              ) : campaigns.error ? (
                <ErrorNotice error={campaigns.error} />
              ) : !list.length ? (
                <section className="kf-panel text-sm text-foreground-secondary">
                  {user.role === "CLIPPER"
                    ? "No active campaigns are available right now."
                    : "No campaigns yet. Create a draft to get started."}
                </section>
              ) : (
                <div className="grid gap-5 xl:grid-cols-2">
                  {list.map((c) => (
                    <article key={c.id} className="kf-panel min-w-0">
                      <div className="flex items-start justify-between gap-3">
                        <h2 className="text-lg font-semibold">{c.title}</h2>
                        <Status value={c.status} />
                      </div>
                      <p className="mt-3 text-xs leading-5 text-foreground-muted">
                        {statusHelp.campaign[c.status]}
                      </p>
                      <p className="mt-4 text-sm leading-6 text-foreground-secondary">{c.description}</p>
                      <p className="mt-4 text-xs text-foreground-muted">
                        {c.start_date.slice(0, 10)} / {c.end_date.slice(0, 10)} /{" "}
                        {(c.platforms ?? []).join(", ")}
                      </p>
                      {c.total_budget && (
                        <p className="mt-3 text-sm tabular-nums">
                          Budget: {c.currency_code} {c.total_budget} ? Remaining: {c.remaining_budget}
                        </p>
                      )}
                      <details className="mt-4 text-sm">
                        <summary className="cursor-pointer py-2 font-semibold">Campaign brief</summary>
                        <p className="mt-2 whitespace-pre-wrap leading-6 text-foreground-secondary">
                          {c.brief}
                        </p>
                        <ul className="mt-3 list-disc space-y-2 pl-5">
                          {(c.requirements ?? []).map((v, i) => (
                            <li key={i}>{v.description}</li>
                          ))}
                        </ul>
                      </details>
                      <div className="mt-5 flex flex-wrap gap-2">
                        {user.role === "CLIPPER" ? (
                          <>
                            <button
                              disabled={busy}
                              className="kf-button"
                              onClick={() => {
                                setDecision({
                                  title: "Join campaign",
                                  description: `Join ${c.title}? Read the brief and requirements before creating content. Joining does not guarantee approval or payment.`,
                                  confirmLabel: "Confirm participation",
                                  run: () => mutate(`campaigns/${c.id}/join`),
                                });
                              }}
                            >
                              Join campaign
                            </button>
                            <button
                              className="kf-button kf-button-secondary"
                              onClick={() => {
                                setSelected(c.id);
                                changeTab("submissions");
                              }}
                            >
                              Submit a clip
                            </button>
                          </>
                        ) : (
                          <>
                            {user.role === "BRAND" && ["DRAFT", "PAUSED"].includes(c.status) && (
                              <button
                                className="kf-button kf-button-secondary"
                                onClick={() => {
                                  setEditing(c);
                                  setDemo(false);
                                  setCreating(true);
                                }}
                              >
                                Edit campaign
                              </button>
                            )}
                            {user.role === "BRAND" &&
                              [
                                ...(c.status === "DRAFT"
                                  ? ["publish"]
                                  : c.status === "ACTIVE"
                                    ? ["pause", "complete"]
                                    : c.status === "PAUSED"
                                      ? ["resume", "complete"]
                                      : []),
                                ...(["DRAFT", "ACTIVE", "PAUSED"].includes(c.status) ? ["cancel"] : []),
                              ].map((action) => (
                                <button
                                  key={action}
                                  disabled={busy}
                                  className="kf-button kf-button-secondary capitalize"
                                  onClick={() => {
                                    setDecision({
                                      title: `${action[0].toUpperCase() + action.slice(1)} campaign`,
                                      description: `${c.title}. ${action === "publish" ? "Publishing makes the brief visible to creators; joining and submissions follow its dates." : action === "complete" ? "Completing closes new submissions and makes approved clips eligible for payout review. This cannot be undone." : action === "cancel" ? "Cancellation closes this campaign. It cannot be resumed." : action === "pause" ? "New participation and submissions will pause. Existing records remain available." : "Participation and submissions can resume within the campaign dates."}`,
                                      confirmLabel: `Confirm ${action}`,
                                      run: () => mutate(`brand/campaigns/${c.id}/${action}`),
                                    });
                                  }}
                                >
                                  {action}
                                </button>
                              ))}
                            <button
                              className="kf-button kf-button-secondary"
                              onClick={() => {
                                setSelected(c.id);
                                changeTab("submissions");
                              }}
                            >
                              Review submissions
                            </button>
                          </>
                        )}
                      </div>
                    </article>
                  ))}
                </div>
              )}
            </>
          )}
          {tab === "submissions" && (
            <>
              {editingSubmission && (
                <SubmissionEditor
                  clip={editingSubmission}
                  platforms={
                    list.find((c) => c.id === editingSubmission.campaign_id)?.platforms || [
                      editingSubmission.platform,
                    ]
                  }
                  onCancel={() => setEditingSubmission(undefined)}
                  onSaved={() => {
                    setEditingSubmission(undefined);
                    setNotice("Submission updated.");
                    void submissions.refetch();
                  }}
                />
              )}
              {user.role !== "ADMIN" && (
                <div className="mb-6 block max-w-lg text-sm font-medium">
                  <label htmlFor="workspace-select-2">Campaign</label>
                  <select
                    id="workspace-select-2"
                    value={selected}
                    onChange={(e) => setSelected(e.target.value)}
                    className="kf-input mt-2"
                  >
                    <option value="">
                      {user.role === "BRAND" ? "Choose a campaign" : "All your submissions"}
                    </option>
                    {list.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.title}
                      </option>
                    ))}
                  </select>
                </div>
              )}
              {user.role !== "CLIPPER" && (user.role === "ADMIN" || selected) && (
                <div className="mb-6 max-w-lg">
                  <label htmlFor="review-status" className="text-sm font-medium">
                    Review status
                  </label>
                  <select
                    id="review-status"
                    className="kf-input mt-2"
                    value={reviewStatus}
                    onChange={(event) => setReviewStatus(event.target.value)}
                  >
                    <option value="">Needs review (Pending and Flagged)</option>
                    <option value="PENDING">Pending</option>
                    <option value="FLAGGED">Flagged</option>
                    <option value="APPROVED">Approved</option>
                    <option value="REJECTED">Rejected</option>
                  </select>
                  <p className="mt-2 text-xs leading-5 text-foreground-muted">
                    Approved and rejected clips leave the review queue. Use this filter to inspect their
                    decisions.
                  </p>
                </div>
              )}
              {user.role === "CLIPPER" && selected && (
                <SubmitForm
                  campaign={list.find((c) => c.id === selected)}
                  onSubmit={(body) => mutate(`campaigns/${selected}/submissions`, body)}
                  busy={busy}
                />
              )}
              {user.role === "BRAND" && !selected ? (
                <section className="kf-panel text-sm text-foreground-secondary">
                  Choose one of your campaigns to open its review queue.
                </section>
              ) : submissions.isPending ? (
                <Loading />
              ) : submissions.error ? (
                <ErrorNotice error={submissions.error} />
              ) : !submissions.data?.submissions.length ? (
                <section className="kf-panel text-sm text-foreground-secondary">
                  No submissions match this workspace.
                </section>
              ) : (
                <div className="space-y-4">
                  {submissions.data.submissions.map((s) => (
                    <article key={s.id} className="kf-panel">
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <h2 className="font-semibold">
                          {s.campaign_title || s.platform}{" "}
                          {s.clipper_display_name && `/ ${s.clipper_display_name}`}
                        </h2>
                        <Status value={s.status} />
                      </div>
                      <p className="mt-3 text-xs leading-5 text-foreground-muted">
                        {statusHelp.submission[s.status]}
                      </p>
                      <p className="mt-4 whitespace-pre-wrap text-sm leading-6 text-foreground-secondary">
                        {s.caption}
                      </p>
                      <a
                        href={s.content_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-4 inline-flex min-h-11 items-center gap-2 text-sm font-semibold text-primary"
                      >
                        View published clip
                        <ArrowUpRight size={16} aria-hidden="true" />
                      </a>
                      {(s.reason || s.rejection_reason) && (
                        <p className="mt-3 rounded-xl bg-surface-muted p-3 text-sm">
                          Review note: {s.reason || s.rejection_reason}
                        </p>
                      )}
                      {user.role === "CLIPPER" && s.status === "PENDING" && (
                        <button
                          className="kf-button kf-button-secondary mt-4"
                          onClick={() => {
                            setEditingSubmission(s);
                            requestAnimationFrame(() =>
                              document
                                .getElementById("submission-editor")
                                ?.scrollIntoView({ block: "center", behavior: "instant" }),
                            );
                          }}
                        >
                          Edit pending clip
                        </button>
                      )}
                      {user.role !== "CLIPPER" && ["PENDING", "FLAGGED"].includes(s.status) && (
                        <div className="mt-4 flex flex-wrap gap-2">
                          {["approve", "reject", ...(s.status === "PENDING" ? ["flag"] : [])].map(
                            (action) => (
                              <button
                                disabled={busy}
                                key={action}
                                className="kf-button kf-button-secondary capitalize"
                                onClick={() => {
                                  setDecision({
                                    title: `${action[0].toUpperCase() + action.slice(1)} submission`,
                                    description: `${s.campaign_title || s.platform}. ${action === "approve" ? "Approval is final and allows metrics collection. Payout also requires a completed campaign." : action === "reject" ? "Rejection is final. The creator can read your reason but cannot edit this clip afterward." : "Flagging requests further review and locks creator editing. A reviewer can later approve or reject it."}`,
                                    confirmLabel: `Confirm ${action}`,
                                    field:
                                      action === "approve"
                                        ? undefined
                                        : { label: "Review reason", kind: "reason" },
                                    run: (reason) =>
                                      mutate(
                                        user.role === "ADMIN"
                                          ? `admin/submissions/${s.id}/${action}`
                                          : `brand/campaigns/${s.campaign_id}/submissions/${s.id}/${action}`,
                                        reason ? { reason } : {},
                                      ),
                                  });
                                }}
                              >
                                {action}
                              </button>
                            ),
                          )}
                        </div>
                      )}
                    </article>
                  ))}
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}

function SubmitForm({
  campaign,
  onSubmit,
  busy,
}: {
  campaign?: Campaign;
  onSubmit: (body: unknown) => Promise<unknown>;
  busy: boolean;
}) {
  return (
    <form
      className="kf-panel mb-6 space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        const f = new FormData(e.currentTarget);
        void onSubmit({ platform: f.get("platform"), content_url: f.get("url"), caption: f.get("caption") });
      }}
    >
      <h2 className="font-semibold">Submit published content</h2>
      <p className="text-sm leading-6 text-foreground-secondary">
        Join the campaign before submitting. Use the direct link to your TikTok video, Instagram reel, or
        YouTube video.
      </p>
      <div className="block text-sm font-medium">
        <label htmlFor="workspace-select-3">Platform</label>
        <select id="workspace-select-3" name="platform" className="kf-input mt-2">
          {(campaign?.platforms || []).map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
      </div>
      <label className="block text-sm font-medium">
        Published clip URL
        <input
          name="url"
          type="url"
          required
          aria-describedby="published-link-help"
          className="kf-input mt-2"
          onChange={(event) => {
            const platform = platformForVideo(event.target.value);
            const select = event.currentTarget.form?.elements.namedItem(
              "platform",
            ) as HTMLSelectElement | null;
            if (platform && select && [...select.options].some((o) => o.value === platform))
              select.value = platform;
          }}
        />
      </label>
      <p id="published-link-help" className="mt-2 text-xs leading-5 text-foreground-muted">
        Paste the direct video link. A recognized host selects its platform automatically; the server checks
        eligibility and the full URL.
      </p>
      <div>
        <label htmlFor="new-clip-text-1" className="block text-sm font-medium">
          Caption
        </label>
        <textarea
          id="new-clip-text-1"
          name="caption"
          required
          maxLength={2200}
          className="kf-input mt-2"
          rows={3}
        />
      </div>
      <button className="kf-button" disabled={busy}>
        {busy ? "Submitting..." : "Submit for review"}
      </button>
    </form>
  );
}
