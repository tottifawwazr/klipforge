export const statusHelp: Record<string, Record<string, string>> = {
  campaign: {
    DRAFT: "Private draft. Review the brief, dates and budget, then publish when ready.",
    ACTIVE: "Published. Joining and new submissions are allowed only within the campaign dates.",
    PAUSED: "New participation and submissions are paused. Review the campaign before resuming.",
    COMPLETED: "Closed to new submissions. Approved clips can move to payout review.",
    CANCELLED: "This campaign is closed and cannot accept new content.",
  },
  submission: {
    PENDING: "Waiting for review. The creator can edit this submission while it remains Pending.",
    FLAGGED: "Further review is needed. Creator editing is locked until a reviewer decides.",
    APPROVED: "Content approved. Payout eligibility also requires a completed campaign and recorded metrics.",
    REJECTED:
      "Read the feedback below. A rejected clip cannot be edited; another submission must follow the campaign rules.",
  },
  payout: {
    PENDING: "Budget reserved. An admin must approve this record before external payment can be recorded.",
    APPROVED:
      "Ready for payment outside KlipForge. Record it only after the transfer has actually completed.",
    FAILED: "Reserved budget has been released. Retrying reserves the amount again.",
    PROCESSED: "An externally completed payment has been recorded. KlipForge did not transfer the money.",
  },
};
export function campaignFormIssues(data: FormData): Record<string, string> {
  const issues: Record<string, string> = {};
  const start = Date.parse(String(data.get("start")));
  const end = Date.parse(String(data.get("end")));
  if (Number.isFinite(start) && Number.isFinite(end) && end <= start)
    issues.end = "Choose an end date after the start date.";
  if (!data.getAll("platforms").length) issues.platforms = "Choose at least one platform for this campaign.";
  return issues;
}
export function platformForVideo(value: string): string | undefined {
  try {
    const url = new URL(value);
    if (url.protocol !== "https:" && url.protocol !== "http:") return;
    const host = url.hostname.toLowerCase();
    if (host === "youtube.com" || host.endsWith(".youtube.com") || host === "youtu.be") return "YOUTUBE";
    if (host === "tiktok.com" || host.endsWith(".tiktok.com")) return "TIKTOK";
    if (host === "instagram.com" || host.endsWith(".instagram.com")) return "INSTAGRAM";
  } catch {}
  return undefined;
}
