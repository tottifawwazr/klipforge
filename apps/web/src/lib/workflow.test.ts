import { describe, expect, it } from "vitest";
import { campaignFormIssues, platformForVideo } from "./workflow";
describe("campaign form checks", () => {
  it("rejects equal/reversed dates and missing platforms together", () => {
    const data = new FormData();
    data.set("start", "2026-11-10");
    data.set("end", "2026-11-10");
    expect(campaignFormIssues(data)).toEqual({
      end: "Choose an end date after the start date.",
      platforms: "Choose at least one platform for this campaign.",
    });
  });
  it("accepts a future end date and selected platform without calculating financial eligibility", () => {
    const data = new FormData();
    data.set("start", "2026-11-10");
    data.set("end", "2026-11-11");
    data.append("platforms", "YOUTUBE");
    expect(campaignFormIssues(data)).toEqual({});
  });
  it.each([
    ["https://www.youtube.com/shorts/abcdefghijk", "YOUTUBE"],
    ["https://youtu.be/abcdefghijk", "YOUTUBE"],
    ["https://m.tiktok.com/@creator/video/1234", "TIKTOK"],
    ["https://www.instagram.com/reel/example", "INSTAGRAM"],
    ["https://youtube.com.attacker.example/watch?v=abcdefghijk", undefined],
    ["javascript:alert(1)", undefined],
  ])("detects only recognized platform hosts for %s", (url, platform) => {
    expect(platformForVideo(url)).toBe(platform);
  });
});
