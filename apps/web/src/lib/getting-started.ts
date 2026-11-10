import type { User } from "@/lib/api";

export type GuideCampaign = {
  id: string;
  title: string;
  status: string;
  start_date: string;
  end_date: string;
};
export type GuideSubmission = { status: string; campaign_id: string };
export type NextStep = {
  title: string;
  description: string;
  label: string;
  view: "campaigns" | "submissions" | "analytics" | "payouts";
  campaignID?: string;
  editor?: "new" | "existing";
};
export function nextStepFor(
  role: User["role"],
  campaigns: GuideCampaign[],
  submissions: GuideSubmission[] = [],
  now = new Date(),
): NextStep {
  if (role === "ADMIN")
    return {
      title: "Start with the content review queue",
      description:
        "Review pending or flagged submissions. Payouts come after approval and campaign completion.",
      label: "Open review queue",
      view: "submissions",
    };
  if (role === "BRAND") {
    const draft = campaigns.find((c) => c.status === "DRAFT");
    if (draft)
      return {
        title: "Finish your campaign draft",
        description: `Review the brief, budget and dates for ${draft.title}. Publish it when it is ready for creators.`,
        label: "Continue draft",
        view: "campaigns",
        campaignID: draft.id,
        editor: "existing",
      };
    const active = campaigns.find((c) => c.status === "ACTIVE");
    if (active && Date.parse(active.end_date) <= now.getTime())
      return {
        title: "Review your campaign after its end date",
        description: `${active.title} has reached its end date. Review any remaining submissions, then complete the campaign when it is ready for payout review.`,
        label: "Review campaign submissions",
        view: "submissions",
        campaignID: active.id,
      };
    if (active)
      return {
        title: "Check the submissions for your active campaign",
        description: `Open ${active.title} to review submitted clips against the brief. New submissions must arrive within the campaign dates.`,
        label: "Review campaign submissions",
        view: "submissions",
        campaignID: active.id,
      };
    if (campaigns.some((c) => c.status === "PAUSED"))
      return {
        title: "Check your paused campaign",
        description:
          "Review the brief and dates before resuming, or complete the campaign when submissions are finished.",
        label: "Open campaigns",
        view: "campaigns",
      };
    if (campaigns.some((c) => c.status === "COMPLETED"))
      return {
        title: "Review the results of your completed campaigns",
        description:
          "Check approved content performance. An admin can preview eligible payouts after the campaign is completed.",
        label: "View campaign results",
        view: "analytics",
      };
    return {
      title: campaigns.length ? "Start a new campaign" : "Create your first campaign",
      description: "Start with a brief, schedule and budget. Your draft stays private until you publish it.",
      label: "Create a campaign",
      view: "campaigns",
      editor: "new",
    };
  }
  const pending = submissions.find((s) => s.status === "PENDING" || s.status === "FLAGGED");
  if (pending)
    return {
      title: "Follow the review of your submitted content",
      description:
        "A submitted clip is waiting for a review decision. You can edit a clip while its status is Pending.",
      label: "View my submissions",
      view: "submissions",
      campaignID: pending.campaign_id,
    };
  const rejected = submissions.find((s) => s.status === "REJECTED");
  if (rejected)
    return {
      title: "Read the feedback on your clip",
      description:
        "Check the rejection reason before preparing another submission. A rejected clip cannot be edited.",
      label: "Read submission feedback",
      view: "submissions",
      campaignID: rejected.campaign_id,
    };
  if (submissions.some((s) => s.status === "APPROVED"))
    return {
      title: "Check your approved clips and payout records",
      description:
        "Approval is one step. The brand must complete the campaign before an admin can calculate a payout.",
      label: "View payout records",
      view: "payouts",
    };
  const open = campaigns.some(
    (c) =>
      c.status === "ACTIVE" &&
      Date.parse(c.start_date) <= now.getTime() &&
      Date.parse(c.end_date) > now.getTime(),
  );
  return {
    title: open ? "Find a campaign and join the brief" : "Check for an open campaign",
    description: open
      ? "Read the requirements, join a campaign, then publish your content and submit its direct video link."
      : "A campaign must be active and within its submission dates before you can join. The brand can publish a practice campaign for a walkthrough.",
    label: "Explore campaigns",
    view: "campaigns",
  };
}

export const DEMO_CAMPAIGN_TITLE = "Demo: Coffee launch";
export function demoCampaignDefaults(now = new Date()) {
  const start = new Date(now);
  start.setUTCDate(start.getUTCDate() - 1);
  const end = new Date(now);
  end.setUTCDate(end.getUTCDate() + 14);
  return {
    title: DEMO_CAMPAIGN_TITLE,
    description:
      "A practice campaign to learn the Brand, Clipper and Admin workflow. This is a fictional coffee launch with simulated metrics and no real payment transfer.",
    brief:
      "Create a 15-30 second video introducing a fictional coffee product. Show how you would use it in a morning routine, include a clear product shot, and identify the content as a practice demo.",
    requirement:
      "Identify this as a practice demo, show the fictional product clearly, and avoid unsupported product claims.",
    budget: "100",
    cpm: "10",
    maximum: "20",
    start: start.toISOString().slice(0, 10),
    end: end.toISOString().slice(0, 10),
    platforms: ["TIKTOK", "INSTAGRAM", "YOUTUBE"],
  };
}
