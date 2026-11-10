"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { campaignFormIssues } from "@/lib/workflow";
import { demoCampaignDefaults } from "@/lib/getting-started";
import { api } from "@/lib/api";
type Draft = {
  id: string;
  title: string;
  description: string;
  brief: string;
  start_date: string;
  end_date: string;
  platforms: string[];
  requirements: { type: string; description: string }[];
};
export function CampaignForm({
  campaign,
  demo = false,
  onCreated,
}: {
  campaign?: Draft;
  demo?: boolean;
  onCreated: () => void;
}) {
  const [example] = useState(() => (demo && !campaign ? demoCampaignDefaults() : undefined));
  const [issues, setIssues] = useState<Record<string, string>>({});
  const [money, setMoney] = useState({
    budget: example?.budget ?? "",
    cpm: example?.cpm ?? "",
    maximum: example?.maximum ?? "",
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const form = useRef<HTMLFormElement>(null);
  useEffect(() => {
    form.current?.scrollIntoView({ block: "start", behavior: "instant" });
    form.current?.querySelector("input")?.focus({ preventScroll: true });
  }, []);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy) return;
    const f = new FormData(e.currentTarget);
    setError("");
    const errors = campaignFormIssues(f);
    setIssues(errors);
    if (Object.keys(errors).length) {
      const field = e.currentTarget.elements.namedItem(Object.keys(errors)[0]);
      if (field instanceof HTMLElement) field.focus();
      else if (field instanceof RadioNodeList && field[0] instanceof HTMLElement) field[0].focus();
      return;
    }
    setBusy(true);
    setError("");
    const common = {
      title: f.get("title"),
      description: f.get("description"),
      brief: f.get("brief"),
      start_date: f.get("start"),
      end_date: f.get("end"),
      platforms: f.getAll("platforms"),
      requirements: campaign?.requirements?.length
        ? campaign.requirements.map((v, i) => ({
            type: v.type,
            description: f.get(`requirement-${i}`),
            position: i + 1,
          }))
        : [{ type: "CONTENT", description: f.get("requirement"), position: 1 }],
    };
    try {
      await api(campaign ? `brand/campaigns/${campaign.id}` : "campaigns", {
        method: campaign ? "PATCH" : "POST",
        body: JSON.stringify({
          ...common,
          ...(!campaign
            ? {
                total_budget: f.get("budget"),
                cpm_rate: f.get("cpm"),
                maximum_payout_per_clip: f.get("maximum"),
              }
            : {}),
        }),
      });
      onCreated();
    } catch (e) {
      setError(e instanceof Error ? e.message : "The campaign could not be saved.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <form id="campaign-editor" ref={form} onSubmit={submit} className="kf-panel mb-6 space-y-5 scroll-mt-6">
      <h2 className="text-lg font-semibold">{campaign ? "Edit campaign" : "New campaign draft"}</h2>
      <p className="text-sm leading-6 text-foreground-secondary">
        {campaign
          ? "Only draft or paused campaigns can be edited. Budget and payout rates remain unchanged."
          : "Drafts are private until published. Campaign currency is USD."}
      </p>
      {example && (
        <div className="rounded-xl border border-primary-border bg-primary-subtle p-4 text-sm leading-6 text-primary-on-subtle">
          <strong>Demo template.</strong> Review the fictional coffee brief and example budget below. It stays
          private after you create a draft; publishing is a separate action.
        </div>
      )}
      <label className="block text-sm font-medium">
        Title
        <input
          className="kf-input mt-2"
          name="title"
          defaultValue={campaign?.title ?? example?.title}
          required
          maxLength={160}
        />
      </label>
      <div>
        <label htmlFor="campaign-text-1" className="block text-sm font-medium">
          Description
        </label>
        <textarea
          id="campaign-text-1"
          className="kf-input mt-2"
          name="description"
          defaultValue={campaign?.description ?? example?.description}
          required
          rows={2}
        />
      </div>
      <div>
        <label htmlFor="campaign-text-2" className="block text-sm font-medium">
          Content brief
        </label>
        <textarea
          id="campaign-text-2"
          className="kf-input mt-2"
          name="brief"
          defaultValue={campaign?.brief ?? example?.brief}
          required
          rows={4}
        />
      </div>
      {campaign?.requirements?.length ? (
        campaign.requirements.map((v, i) => (
          <div key={i}>
            <label htmlFor={`campaign-requirement-${i}`} className="block text-sm font-medium">
              {v.type} guideline
            </label>
            <textarea
              id={`campaign-requirement-${i}`}
              className="kf-input mt-2"
              name={`requirement-${i}`}
              defaultValue={v.description}
              required
              rows={2}
            />
          </div>
        ))
      ) : (
        <div>
          <label htmlFor="campaign-text-4" className="block text-sm font-medium">
            Required content guideline
          </label>
          <textarea
            id="campaign-text-4"
            className="kf-input mt-2"
            name="requirement"
            defaultValue={example?.requirement}
            required
            rows={2}
          />
        </div>
      )}
      {!campaign && (
        <div className="grid gap-4 sm:grid-cols-3">
          {[
            ["budget", "Total budget (USD)"],
            ["cpm", "Rate per 1,000 views (USD)"],
            ["maximum", "Maximum per clip (USD)"],
          ].map(([name, label]) => (
            <label key={name} className="block text-sm font-medium">
              {label}
              <input
                name={name}
                defaultValue={example?.[name as "budget" | "cpm" | "maximum"]}
                onChange={(event) => setMoney((previous) => ({ ...previous, [name]: event.target.value }))}
                type="number"
                min="0.01"
                step={name === "cpm" ? "0.0001" : "0.01"}
                required
                className="kf-input mt-2"
              />
            </label>
          ))}
        </div>
      )}
      {!campaign &&
        Number.isFinite(Number(money.cpm)) &&
        Number.isFinite(Number(money.maximum)) &&
        Number(money.cpm) > 0 &&
        Number(money.maximum) > 0 && (
          <p className="rounded-xl border border-border bg-surface-subtle p-4 text-sm leading-6 text-foreground-secondary">
            Example at 1,000 views: USD {Math.min(Number(money.cpm), Number(money.maximum)).toFixed(2)},
            capped at USD {Number(money.maximum).toFixed(2)} per clip. The final payout is calculated by the
            API after approval, campaign completion and budget checks.
          </p>
        )}
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="block text-sm font-medium">
          Start date
          <input
            type="date"
            name="start"
            defaultValue={campaign?.start_date.slice(0, 10) ?? example?.start}
            required
            className="kf-input mt-2"
          />
        </label>
        <label className="block text-sm font-medium">
          End date
          <input
            type="date"
            name="end"
            aria-invalid={!!issues.end}
            aria-describedby={issues.end ? "campaign-end-error" : undefined}
            defaultValue={campaign?.end_date.slice(0, 10) ?? example?.end}
            required
            className="kf-input mt-2"
          />
        </label>
      </div>
      {issues.end && (
        <p id="campaign-end-error" role="alert" className="text-sm text-danger">
          {issues.end}
        </p>
      )}
      <fieldset
        aria-invalid={!!issues.platforms}
        aria-describedby={issues.platforms ? "campaign-platforms-error" : undefined}
      >
        <legend className="text-sm font-medium">Platforms</legend>
        <div className="mt-3 flex flex-wrap gap-5">
          {["TIKTOK", "INSTAGRAM", "YOUTUBE"].map((v) => (
            <label key={v} className="flex min-h-11 items-center gap-2 text-sm">
              <input
                type="checkbox"
                name="platforms"
                value={v}
                defaultChecked={
                  campaign
                    ? (campaign.platforms ?? []).includes(v)
                    : example
                      ? example.platforms.includes(v)
                      : v === "TIKTOK"
                }
              />
              {v}
            </label>
          ))}
        </div>
      </fieldset>
      {issues.platforms && (
        <p id="campaign-platforms-error" role="alert" className="text-sm text-danger">
          {issues.platforms}
        </p>
      )}
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-danger-border bg-danger-subtle p-4 text-sm text-danger-on-subtle"
        >
          {error}
        </p>
      )}
      <button disabled={busy} className="kf-button">
        {busy ? "Saving..." : campaign ? "Save campaign" : "Create draft"}
      </button>
    </form>
  );
}
