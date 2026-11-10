import { describe, expect, it } from "vitest";
import { demoCampaignDefaults, nextStepFor, type GuideCampaign } from "./getting-started";

const now = new Date("2026-11-10T12:00:00Z");
const campaign: GuideCampaign = {
  id: "campaign-a",
  title: "Example brief",
  status: "ACTIVE",
  start_date: "2026-11-01T00:00:00Z",
  end_date: "2026-11-20T00:00:00Z",
};
describe("contextual workspace guidance", () => {
  it("takes a brand to the exact draft instead of another active campaign", () => {
    const step = nextStepFor("BRAND", [campaign, { ...campaign, id: "draft-b", status: "DRAFT" }], [], now);
    expect(step).toMatchObject({ view: "campaigns", campaignID: "draft-b", editor: "existing" });
  });
  it("explains an expired ACTIVE campaign without saying it still accepts submissions", () => {
    const step = nextStepFor("BRAND", [{ ...campaign, end_date: now.toISOString() }], [], now);
    expect(step.title).toContain("end date");
    expect(step).toMatchObject({ view: "submissions", campaignID: campaign.id });
  });
  it("prioritizes pending creator reviews over approved historical clips", () => {
    expect(
      nextStepFor(
        "CLIPPER",
        [],
        [
          { status: "APPROVED", campaign_id: "old" },
          { status: "PENDING", campaign_id: "pending" },
        ],
        now,
      ),
    ).toMatchObject({ view: "submissions", campaignID: "pending" });
  });
  it("sends rejected creators to feedback without offering to edit a terminal submission", () => {
    const step = nextStepFor("CLIPPER", [], [{ status: "REJECTED", campaign_id: "rejected" }], now);
    expect(step).toMatchObject({ view: "submissions", campaignID: "rejected" });
    expect(step.description).toContain("cannot be edited");
  });
  it.each(["2026-11-10T12:00:00Z", "2026-11-09T00:00:00Z", "invalid"])(
    "does not call a campaign open after its end boundary (%s)",
    (end_date) => {
      expect(nextStepFor("CLIPPER", [{ ...campaign, end_date }], [], now).title).toBe(
        "Check for an open campaign",
      );
    },
  );
  it("offers the appropriate entry points for new users and admins", () => {
    expect(nextStepFor("BRAND", [], [], now)).toMatchObject({ view: "campaigns", editor: "new" });
    expect(nextStepFor("CLIPPER", [campaign], [], now).title).toContain("join");
    expect(nextStepFor("ADMIN", [campaign], [], now)).toMatchObject({
      view: "submissions",
      label: "Open review queue",
    });
  });
  it("provides clearly fictional demo defaults that straddle today, including year boundaries", () => {
    const example = demoCampaignDefaults(new Date("2026-12-31T12:00:00Z"));
    expect(example).toMatchObject({
      start: "2026-12-30",
      end: "2027-01-14",
      budget: "100",
      cpm: "10",
      maximum: "20",
    });
    expect(example.description).toContain("fictional");
    expect(example.description).toContain("no real payment transfer");
    expect(example.platforms).toEqual(["TIKTOK", "INSTAGRAM", "YOUTUBE"]);
  });
});
