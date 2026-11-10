"use client";
import { ArrowRight, BookOpen, ClipboardCheck, FileVideo, Megaphone, Wallet } from "lucide-react";
import type { User } from "@/lib/api";
import type { NextStep } from "@/lib/getting-started";

const roleCopy = {
  BRAND: {
    label: "Brand",
    job: "You set the brief and budget, publish campaigns, and review the clips creators submit.",
    action: "Create a practice draft",
  },
  CLIPPER: {
    label: "Clipper",
    job: "You join a campaign, publish your content on a supported social platform, and send its video link for review.",
    action: "Explore campaigns",
  },
  ADMIN: {
    label: "Admin",
    job: "You oversee content reviews, calculate eligible payouts, and record externally completed payments.",
    action: "Open review queue",
  },
};
const journey = [
  {
    title: "Set the brief",
    owner: "Brand",
    Icon: Megaphone,
    copy: "Create a campaign with content requirements, dates, budget and a rate per 1,000 views. Publish it when ready.",
  },
  {
    title: "Publish and submit",
    owner: "Clipper",
    Icon: FileVideo,
    copy: "Join the campaign, publish a video on TikTok, Instagram or YouTube, then submit the direct video link here.",
  },
  {
    title: "Review the content",
    owner: "Brand / Admin",
    Icon: ClipboardCheck,
    copy: "Approve, reject or flag the submission. Rejected content includes a reason for the creator to read.",
  },
  {
    title: "Track the performance",
    owner: "KlipForge",
    Icon: BookOpen,
    copy: "Approved clips receive simulated views and engagement in this MVP. The brand completes the campaign when it is finished.",
  },
  {
    title: "Record a payout",
    owner: "Admin",
    Icon: Wallet,
    copy: "Preview the amount for an approved clip from a completed campaign, reserve budget, approve it, and record an external payment reference.",
  },
];
export function NextStepCard({
  role,
  step,
  onAction,
  onGuide,
}: {
  role: User["role"];
  step: NextStep;
  onAction: () => void;
  onGuide: () => void;
}) {
  return (
    <section aria-labelledby="next-step-heading" className="kf-panel mb-6 border-primary-border">
      <p className="text-xs font-semibold uppercase tracking-widest text-primary">
        Your next step / {roleCopy[role].label}
      </p>
      <h2 id="next-step-heading" className="mt-3 text-xl font-semibold text-foreground-strong">
        {step.title}
      </h2>
      <p className="mt-3 max-w-2xl break-words text-sm leading-6 text-foreground-secondary">
        {step.description}
      </p>
      <div className="mt-5 flex flex-wrap gap-3">
        <button className="kf-button" onClick={onAction}>
          {step.label}
          <ArrowRight size={16} aria-hidden="true" />
        </button>
        <button className="kf-button kf-button-secondary" onClick={onGuide}>
          How it works
        </button>
      </div>
    </section>
  );
}
export function WorkspaceGuide({
  role,
  onStart,
  onOverview,
  onPayouts,
}: {
  role: User["role"];
  onStart: () => void;
  onOverview: () => void;
  onPayouts: () => void;
}) {
  const current = roleCopy[role];
  return (
    <div className="space-y-6">
      <section className="kf-panel" aria-labelledby="guide-intro-heading">
        <p className="text-xs font-semibold uppercase tracking-widest text-primary">Start here</p>
        <h2 id="guide-intro-heading" className="mt-3 text-xl font-semibold">
          A campaign, from brief to payment record
        </h2>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-foreground-secondary">
          KlipForge connects brands that need promotional content with creators who make it. The campaign
          brief, content review, performance and payout record stay together in one workspace.
        </p>
        <div className="mt-5 rounded-xl border border-primary-border bg-primary-subtle p-4">
          <h3 className="text-sm font-semibold text-primary-on-subtle">Your role: {current.label}</h3>
          <p className="mt-2 text-sm leading-6 text-primary-on-subtle">{current.job}</p>
        </div>
        <div className="mt-5 flex flex-wrap gap-3">
          <button className="kf-button" onClick={onStart}>
            {current.action}
            <ArrowRight size={16} aria-hidden="true" />
          </button>
          <button className="kf-button kf-button-secondary" onClick={onOverview}>
            Back to overview
          </button>
        </div>
        {role === "BRAND" && (
          <p className="mt-3 text-xs leading-5 text-foreground-muted">
            The practice template fills in a fictional coffee campaign. Opening it saves nothing; you review
            and create the private draft yourself.
          </p>
        )}
      </section>
      <section aria-labelledby="journey-heading">
        <h2 id="journey-heading" className="text-lg font-semibold">
          How the five steps connect
        </h2>
        <ol className="mt-4 grid gap-4 sm:grid-cols-2">
          {journey.map(({ title, owner, Icon, copy }, index) => (
            <li key={title} className="kf-panel min-w-0">
              <div className="flex items-center justify-between gap-3">
                <span className="text-xs font-semibold uppercase tracking-widest text-foreground-muted">
                  Step {index + 1} / {owner}
                </span>
                <Icon size={20} className="shrink-0 text-primary" aria-hidden="true" />
              </div>
              <h3 className="mt-4 font-semibold">{title}</h3>
              <p className="mt-2 text-sm leading-6 text-foreground-secondary">{copy}</p>
            </li>
          ))}
        </ol>
      </section>
      <section className="kf-panel" aria-labelledby="demo-limits-heading">
        <h2 id="demo-limits-heading" className="text-lg font-semibold">
          What this MVP demonstrates
        </h2>
        <ul className="mt-4 list-disc space-y-3 pl-5 text-sm leading-6 text-foreground-secondary">
          <li>Campaigns, submissions and review decisions are saved to your workspace.</li>
          <li>Views and engagement are simulated. They are not live social-platform statistics.</li>
          <li>Payouts are manual records. Recording a payment does not transfer money.</li>
        </ul>
        <details className="mt-5 border-t border-border-subtle pt-4 text-sm">
          <summary className="min-h-11 cursor-pointer py-3 font-semibold">See a payout example</summary>
          <p className="mt-2 leading-6 text-foreground-secondary">
            Example only: a USD 10 rate per 1,000 views gives USD 10 for 1,000 views. With a USD 20 per-clip
            cap, 3,000 views gives USD 20. The campaign must be completed, the clip approved, and enough
            campaign budget available before an admin can create the payout.
          </p>
          <button onClick={onPayouts} className="kf-button kf-button-secondary mt-4">
            Open payout records
          </button>
        </details>
        <details className="mt-4 border-t border-border-subtle pt-4 text-sm">
          <summary className="min-h-11 cursor-pointer py-3 font-semibold">
            Understand the product terms
          </summary>
          <dl className="mt-2 grid gap-4 sm:grid-cols-2">
            {[
              ["Campaign", "The brand's brief, requirements, schedule and budget."],
              ["Submission", "A published video link a creator sends for review."],
              ["CPM", "The campaign's payment rate for every 1,000 recorded views."],
              ["Payout", "The calculated payment record, with approval and processing status."],
            ].map(([term, meaning]) => (
              <div key={term}>
                <dt className="font-semibold">{term}</dt>
                <dd className="mt-1 leading-6 text-foreground-secondary">{meaning}</dd>
              </div>
            ))}
          </dl>
        </details>
      </section>
    </div>
  );
}
