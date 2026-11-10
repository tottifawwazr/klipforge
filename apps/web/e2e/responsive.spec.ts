import { expect, test } from "@playwright/test";

test("public forms and role workspaces fit mobile, tablet and desktop", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => {
    errors.push(error.message);
    console.error("Browser exception:", error.message);
  });
  for (const path of ["/login", "/register"]) {
    await page.goto(path);
    await expect(page.locator("input[name=email]")).toBeVisible();
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: 900 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    }
  }
  for (const role of ["brand", "clipper", "admin"] as const) {
    await page.goto("/login");
    await page.getByLabel("Email address").fill(`${role}@klipforge.local`);
    await page
      .getByLabel("Password", { exact: true })
      .fill(role === "brand" ? "Brand123!" : role === "clipper" ? "Clipper123!" : "Admin123!");
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    const nav = page.getByRole("navigation", { name: "Workspace navigation" });
    await expect(nav).toBeVisible();
    for (const tab of [
      "Overview",
      "How it works",
      "Campaigns",
      "Submissions",
      "Analytics",
      "Payouts",
      "Notifications",
      ...(role === "admin" ? ["Audit"] : []),
    ]) {
      console.info(`Inspecting ${role}/${tab}`);
      await nav.getByRole("button", { name: tab, exact: true }).click();
      await expect(page.locator("main h1")).toBeVisible();
      await expect(page.getByText("Loading workspace...", { exact: true })).toHaveCount(0);
      await expect(page.getByText("Loading records...", { exact: true })).toHaveCount(0);
      if (tab === "Overview") {
        await expect(
          page
            .getByRole("heading", { level: 2 })
            .filter({ hasText: /campaign|content|clip|review/i })
            .first(),
        ).toBeVisible();
        await expect(page.locator("#next-step-heading")).toBeVisible();
      }
      if (tab === "Analytics") {
        await expect(
          page.getByRole("heading", { name: "YouTube public metrics", exact: true }),
        ).toBeVisible();
        const setupResponse = await page.request.get("/api/backend/integrations/youtube");
        expect(setupResponse.ok()).toBe(true);
        const setup = (await setupResponse.json()) as { configured: boolean };
        await expect(
          page.getByText(setup.configured ? "API key configured" : "YouTube is not connected yet.", {
            exact: true,
          }),
        ).toBeVisible();
        await expect(page.getByText("Set up YouTube access", { exact: true })).toHaveCount(
          !setup.configured && role === "admin" ? 1 : 0,
        );
      }
      if (tab === "How it works") {
        await expect(
          page.getByRole("heading", {
            name: `Your role: ${role === "brand" ? "Brand" : role === "clipper" ? "Clipper" : "Admin"}`,
            exact: true,
          }),
        ).toBeVisible();
        await expect(page.getByRole("heading", { name: "How the five steps connect" })).toBeVisible();
        await expect(page.locator("main ol > li")).toHaveCount(5);
        await page.getByText("See a payout example", { exact: true }).click();
        await expect(page.getByText(/Example only: a USD 10/)).toBeVisible();
        await page.getByText("Understand the product terms", { exact: true }).click();
        await expect(page.getByRole("definition").first()).toBeVisible();
      }
      const chart = page.getByRole("img", { name: /Line chart of recorded daily views/ });
      if (await chart.isVisible()) await chart.hover();
      for (const width of [320, 390, 768, 1440]) {
        await page.setViewportSize({ width, height: 900 });
        await expect
          .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), {
            message: `${role}/${tab}/${width}`,
          })
          .toBe(true);
      }
    }
    await nav.getByRole("button", { name: "How it works", exact: true }).click();
    if (role === "brand") {
      let campaignWrites = 0;
      const trackWrites = (request: import("@playwright/test").Request) => {
        if (request.url().endsWith("/api/backend/campaigns") && request.method() === "POST") campaignWrites++;
      };
      page.on("request", trackWrites);
      await page.getByRole("button", { name: "Create a practice draft", exact: true }).click();
      const form = page.locator("#campaign-editor");
      await expect(form.getByLabel("Title", { exact: true })).toHaveValue("Demo: Coffee launch");
      await expect(form.getByLabel("Total budget (USD)", { exact: true })).toHaveValue("100");
      await expect(form.getByLabel("Rate per 1,000 views (USD)", { exact: true })).toHaveValue("10");
      await expect(form.getByLabel("Maximum per clip (USD)", { exact: true })).toHaveValue("20");
      expect(campaignWrites).toBe(0);
      await form
        .getByLabel("End date", { exact: true })
        .fill(await form.getByLabel("Start date", { exact: true }).inputValue());
      for (const checkbox of await form.getByRole("checkbox").all()) await checkbox.uncheck();
      await form.getByRole("button", { name: "Create draft", exact: true }).click();
      await expect(form.getByText("Choose an end date after the start date.", { exact: true })).toBeVisible();
      await expect(
        form.getByText("Choose at least one platform for this campaign.", { exact: true }),
      ).toBeVisible();
      expect(campaignWrites).toBe(0);
      await page.setViewportSize({ width: 320, height: 900 });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
      await nav.getByRole("button", { name: "Overview", exact: true }).click();
      page.off("request", trackWrites);
    } else {
      await expect(page.getByRole("button", { name: "Create a practice draft", exact: true })).toHaveCount(0);
      await page
        .getByRole("button", {
          name: role === "clipper" ? "Explore campaigns" : "Open review queue",
          exact: true,
        })
        .click();
      await expect(page.locator("main h1")).toHaveText(
        role === "clipper" ? "Campaigns" : "Submission workspace",
      );
    }
    await page.getByRole("button", { name: "Sign out" }).click();
    await expect(page).toHaveURL(/\/login$/);
  }
  expect(errors).toEqual([]);
});
