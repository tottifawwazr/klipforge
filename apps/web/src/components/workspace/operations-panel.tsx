"use client";
import { useRef, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ActionDialog, type Decision } from "@/components/workspace/action-dialog";
import { statusHelp } from "@/lib/workflow";
import { api, type User } from "@/lib/api";
type Payout = {
  id: string;
  submission_id: string;
  campaign_title: string;
  display_name: string;
  status: string;
  amount: string;
  currency_code: string;
  basis_views: number;
  basis_snapshot_at?: string | null;
  processor_reference?: string;
  created_at: string;
};
type Notification = { id: string; title: string; body: string; read_at?: string; created_at: string };
type Audit = { id: string; action: string; actor_role: string; entity_type: string; created_at: string };
type Clip = {
  id: string;
  campaign_title: string;
  clipper_display_name: string;
  platform: string;
  caption: string;
};
export function OperationsPanel({ user, mode }: { user: User; mode: "payouts" | "notifications" | "audit" }) {
  const client = useQueryClient();
  const [decision, setDecision] = useState<Decision>();
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState("");
  const [quote, setQuote] = useState<Payout>();
  const key = useRef("");
  const query = useQuery({
    queryKey: ["operations", mode, user.id, page],
    queryFn: () =>
      api<{ payouts?: Payout[]; notifications?: Notification[]; events?: Audit[] }>(
        mode === "audit" ? "admin/audit" : `account/${mode}?page=${page}`,
      ),
  });
  const clips = useQuery({
    queryKey: ["payable-clips", user.id],
    queryFn: () => api<{ submissions: Clip[] }>("admin/moderation/submissions?status=APPROVED&limit=100"),
    enabled: mode === "payouts" && user.role === "ADMIN",
  });
  async function mutate(path: string, body: unknown = {}, headers: Record<string, string> = {}) {
    if (busy) return "Another operation is still saving. Please wait.";
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await api(path, { method: "POST", headers, body: JSON.stringify(body) });
      setNotice("Operation recorded.");
      setQuote(undefined);
      await client.invalidateQueries({ predicate: (q) => q.queryKey[0] !== "session" });
    } catch (e) {
      const message = e instanceof Error ? e.message : "The operation failed.";
      setError(message);
      return message;
    } finally {
      setBusy(false);
    }
  }
  async function preview(e: FormEvent) {
    e.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    setQuote(undefined);
    try {
      const d = await api<{ quote: Payout }>("admin/payout-quotes", {
        method: "POST",
        body: JSON.stringify({ submission_id: selected }),
      });
      key.current = crypto.randomUUID();
      setQuote(d.quote);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not preview payout.");
    } finally {
      setBusy(false);
    }
  }
  if (query.isPending)
    return (
      <p role="status" className="py-10 text-sm text-foreground-secondary">
        Loading records...
      </p>
    );
  return (
    <div className="space-y-5">
      {decision && <ActionDialog decision={decision} onClose={() => setDecision(undefined)} />}
      {mode === "payouts" && (
        <div className="kf-notice">
          <strong>External payment records.</strong> KlipForge calculates and reserves campaign budget.
          Recording a payment does not transfer money. Social metrics used for calculation are simulated in
          this MVP.
        </div>
      )}
      {(error || query.error) && (
        <p
          role="alert"
          className="rounded-xl border border-danger-border bg-danger-subtle p-4 text-sm text-danger-on-subtle"
        >
          {error || (query.error instanceof Error ? query.error.message : "Records could not be loaded.")}
        </p>
      )}
      {notice && (
        <p role="status" className="rounded-xl bg-primary-subtle p-4 text-sm text-primary-on-subtle">
          {notice}
        </p>
      )}
      {mode === "payouts" && user.role === "ADMIN" && (
        <section className="kf-panel">
          <h2 className="text-lg font-semibold">Calculate a payout</h2>
          <p className="mt-2 text-sm leading-6 text-foreground-secondary">
            Choose an approved clip from a completed campaign. Preview the amount before reserving budget for
            approval.
          </p>
          <form onSubmit={(e) => void preview(e)} className="mt-5 flex flex-wrap items-end gap-3">
            <div className="min-w-0 flex-1 text-sm font-medium">
              <label htmlFor="operations-panel-select-1">Approved clip</label>
              <select
                id="operations-panel-select-1"
                value={selected}
                required
                onChange={(e) => {
                  setSelected(e.target.value);
                  setQuote(undefined);
                }}
                className="kf-input mt-2"
              >
                <option value="">Choose a clip</option>
                {clips.data?.submissions.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.campaign_title} / {s.clipper_display_name} / {s.platform}
                  </option>
                ))}
              </select>
            </div>
            <button className="kf-button kf-button-secondary" disabled={busy || !selected}>
              Preview amount
            </button>
          </form>
          {!!clips.error && (
            <p role="alert" className="mt-3 text-sm text-danger">
              Approved clips could not be loaded.
            </p>
          )}
          {quote && (
            <div className="mt-5 rounded-xl border border-border bg-surface-subtle p-5">
              <p className="font-semibold">
                {quote.currency_code} {quote.amount} for {quote.display_name}
              </p>
              <p className="mt-2 text-sm leading-6 text-foreground-secondary">
                {quote.campaign_title} / {quote.basis_views.toLocaleString()} recorded views. CPM calculation
                is rounded down to cents and capped at the campaign maximum.
              </p>
              <button
                disabled={busy}
                className="kf-button mt-4"
                onClick={() => {
                  setDecision({
                    title: "Reserve payout budget",
                    description: `${quote.currency_code} ${quote.amount} for ${quote.display_name} / ${quote.campaign_title}. This reduces the campaign's available budget and creates a Pending payout. It does not send money.`,
                    confirmLabel: "Confirm reservation",
                    run: () =>
                      mutate(
                        "admin/payouts",
                        { submission_id: selected, expected_amount: quote.amount },
                        { "Idempotency-Key": key.current },
                      ),
                  });
                }}
              >
                Reserve {quote.currency_code} {quote.amount}
              </button>
            </div>
          )}
        </section>
      )}
      {mode === "payouts" && (
        <>
          <div className="space-y-4">
            {query.data?.payouts?.map((p) => (
              <article key={p.id} className="kf-panel">
                <div className="flex flex-wrap justify-between gap-3">
                  <div>
                    <h2 className="font-semibold">{p.campaign_title}</h2>
                    <p className="mt-2 text-sm text-foreground-secondary">{p.display_name}</p>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold tabular-nums">
                      {p.currency_code} {p.amount}
                    </p>
                    <p className="mt-2 text-xs font-semibold text-foreground-secondary">{p.status}</p>
                  </div>
                </div>
                <p className="mt-3 text-xs leading-5 text-foreground-muted">{statusHelp.payout[p.status]}</p>
                <p className="mt-4 text-xs text-foreground-muted">
                  Basis:{" "}
                  {p.basis_snapshot_at
                    ? `${p.basis_views.toLocaleString()} recorded views`
                    : "Legacy fixture; calculation basis unavailable"}{" "}
                  / {new Date(p.created_at).toLocaleDateString()}
                </p>
                {p.processor_reference && (
                  <p className="mt-2 break-all text-xs text-foreground-secondary">
                    External payment reference: {p.processor_reference}
                  </p>
                )}
                {user.role === "ADMIN" && p.status !== "PROCESSED" && (
                  <div className="mt-5 flex flex-wrap gap-2">
                    {(p.status === "PENDING"
                      ? ["approve", "fail"]
                      : p.status === "APPROVED"
                        ? ["process", "fail"]
                        : p.status === "FAILED"
                          ? ["retry"]
                          : []
                    ).map((action) => (
                      <button
                        key={action}
                        disabled={busy}
                        className="kf-button kf-button-secondary"
                        onClick={() => {
                          const label =
                            action === "process"
                              ? "Record external payment"
                              : action === "approve"
                                ? "Approve payout"
                                : action === "retry"
                                  ? "Reserve again"
                                  : "Mark failed";
                          setDecision({
                            title: label,
                            description: `${p.currency_code} ${p.amount} / ${p.display_name} / ${p.campaign_title}. ${action === "process" ? "Continue only after this payment was completed outside KlipForge. This records a receipt; it does not transfer funds. The processed record is final." : action === "fail" ? "The reserved budget will be released once. Provide the reason for this failure." : action === "retry" ? "This reserves the amount from the campaign budget again and returns the payout to Pending." : "This approves the reserved amount for an external payment. No transfer takes place here."}`,
                            confirmLabel:
                              action === "process"
                                ? "Confirm payment record"
                                : action === "approve"
                                  ? "Confirm approval"
                                  : action === "retry"
                                    ? "Confirm new reservation"
                                    : "Confirm failure",
                            field:
                              action === "process"
                                ? { label: "External payment reference", kind: "reference" }
                                : action === "fail"
                                  ? { label: "Failure reason", kind: "reason" }
                                  : undefined,
                            run: (value) =>
                              mutate(
                                `admin/payouts/${p.id}/${action}`,
                                action === "process"
                                  ? {
                                      reference: value.startsWith("manual:") ? value : `manual:${value}`,
                                      acknowledge_manual_record: true,
                                    }
                                  : action === "fail"
                                    ? { reason: value }
                                    : {},
                              ),
                          });
                        }}
                      >
                        {action === "process"
                          ? "Record external payment"
                          : action === "approve"
                            ? "Approve payout"
                            : action === "retry"
                              ? "Reserve again"
                              : "Mark failed"}
                      </button>
                    ))}
                  </div>
                )}
              </article>
            ))}
          </div>
          {!query.data?.payouts?.length && (
            <section className="kf-panel text-sm text-foreground-secondary">No payout records yet.</section>
          )}
          <div className="flex items-center justify-between gap-4">
            <button
              className="kf-button kf-button-secondary"
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
            >
              Previous
            </button>
            <span className="text-sm text-foreground-secondary">Page {page}</span>
            <button
              className="kf-button kf-button-secondary"
              disabled={(query.data?.payouts?.length || 0) < 50}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </button>
          </div>
        </>
      )}
      {mode === "notifications" && (
        <div className="space-y-4">
          {query.data?.notifications?.map((n) => (
            <article key={n.id} className="kf-panel">
              <div className="flex flex-wrap justify-between gap-3">
                <h2 className="font-semibold">{n.title}</h2>
                <span className="text-xs text-foreground-muted">{n.read_at ? "Read" : "Unread"}</span>
              </div>
              <p className="mt-3 text-sm leading-6 text-foreground-secondary">{n.body}</p>
              <p className="mt-3 text-xs text-foreground-muted">{new Date(n.created_at).toLocaleString()}</p>
              {!n.read_at && (
                <button
                  className="kf-button kf-button-secondary mt-4"
                  disabled={busy}
                  onClick={() => void mutate(`account/notifications/${n.id}/read`)}
                >
                  Mark as read
                </button>
              )}
            </article>
          ))}
          {!query.data?.notifications?.length && (
            <section className="kf-panel text-sm text-foreground-secondary">
              You are all caught up. New activity will appear here.
            </section>
          )}
        </div>
      )}
      {mode === "audit" && (
        <section className="kf-panel">
          <h2 className="font-semibold">Recent administrative activity</h2>
          <p className="mt-2 text-sm text-foreground-secondary">
            Latest 100 events. Sensitive request fields and credentials are excluded.
          </p>
          <div className="mt-5 overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr>
                  {["Action", "Role", "Resource", "Time"].map((t) => (
                    <th key={t} className="p-3 font-semibold">
                      {t}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {query.data?.events?.map((e) => (
                  <tr key={e.id} className="border-t border-border-subtle">
                    <td className="p-3">{e.action}</td>
                    <td className="p-3">{e.actor_role}</td>
                    <td className="p-3">{e.entity_type}</td>
                    <td className="whitespace-nowrap p-3">{new Date(e.created_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}
