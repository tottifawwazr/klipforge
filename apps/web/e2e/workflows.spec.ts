import { test, expect, type Page } from "@playwright/test";
const base = process.env.E2E_BASE_URL || "http://127.0.0.1:3002";
async function login(page: Page, role: "brand" | "clipper" | "admin") {
  await page.goto("/login");
  await page.getByLabel("Email address").fill(`${role}@klipforge.local`);
  await page
    .getByLabel("Password", { exact: true })
    .fill(role === "brand" ? "Brand123!" : role === "clipper" ? "Clipper123!" : "Admin123!");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page.getByRole("navigation", { name: "Workspace navigation" })).toBeVisible();
  await expect(page.getByText(role.toUpperCase(), { exact: true })).toBeVisible();
}
async function tab(page: Page, name: string) {
  await page
    .getByRole("navigation", { name: "Workspace navigation" })
    .getByRole("button", { name, exact: true })
    .click();
}
async function logout(page: Page) {
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
}
test("brand, clipper and admin complete the campaign-to-payout workflow", async ({ page }) => {
  const suffix = Date.now().toString();
  const initial = `QA creator campaign ${suffix}`;
  const title = `QA reviewed campaign ${suffix}`;
  const caption = `QA clip ${suffix}`;
  const editedCaption = `QA edited clip ${suffix}`;

  expect((await page.request.get(`${base}/api/backend/integrations/youtube`)).status()).toBe(401);
  await login(page, "brand");
  await tab(page, "How it works");
  await page.getByRole("button", { name: "Create a practice draft", exact: true }).click();
  const form = page.locator("#campaign-editor");
  await expect(form.getByLabel("Title", { exact: true })).toHaveValue("Demo: Coffee launch");
  await form.getByLabel("Title", { exact: true }).fill(initial);
  await form
    .getByLabel("Description", { exact: true })
    .fill("A local automated test campaign with a clear creator brief.");
  await form
    .getByLabel("Content brief", { exact: true })
    .fill("Show the product clearly and follow the provided disclosure requirements.");
  await form
    .getByLabel("Required content guideline", { exact: true })
    .fill("Show the product within the first five seconds.");
  await form.getByLabel("Total budget (USD)", { exact: true }).fill("100");
  await form.getByLabel("Rate per 1,000 views (USD)", { exact: true }).fill("10");
  await form.getByLabel("Maximum per clip (USD)", { exact: true }).fill("10");
  const start = new Date(Date.now() - 86_400_000).toISOString().slice(0, 10);
  const end = new Date(Date.now() + 7 * 86_400_000).toISOString().slice(0, 10);
  await form.getByLabel("Start date", { exact: true }).fill(start);
  await form.getByLabel("End date", { exact: true }).fill(end);
  const created = page.waitForResponse(
    (r) => r.url().endsWith("/api/backend/campaigns") && r.request().method() === "POST",
  );
  await form.getByRole("button", { name: "Create draft", exact: true }).click();
  const createdResponse = await created;
  expect(createdResponse.status()).toBe(201);
  const campaignID = (await createdResponse.json()).campaign.id as string;
  let card = page
    .locator("article")
    .filter({ has: page.getByRole("heading", { name: initial, exact: true }) });
  await expect(card).toBeVisible();
  await card.getByRole("button", { name: "Edit campaign", exact: true }).click();
  await form.getByLabel("Title", { exact: true }).fill(title);
  await form.getByRole("button", { name: "Save campaign", exact: true }).click();
  card = page.locator("article").filter({ has: page.getByRole("heading", { name: title, exact: true }) });
  await expect(card).toBeVisible();
  await card.getByRole("button", { name: /^publish$/i }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Confirm publish", exact: true }).click();
  await expect(card.getByText("ACTIVE", { exact: true })).toBeVisible();
  await logout(page);
  await login(page, "clipper");
  await tab(page, "Campaigns");
  card = page.locator("article").filter({ has: page.getByRole("heading", { name: title, exact: true }) });
  await card.getByRole("button", { name: "Join campaign", exact: true }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Confirm participation", exact: true }).click();
  await expect(page.getByRole("status")).toContainText("Changes saved.");
  await card.getByRole("button", { name: "Submit a clip", exact: true }).click();
  const submitForm = page
    .locator("form")
    .filter({ has: page.getByRole("heading", { name: "Submit published content" }) });
  await submitForm.getByLabel("Platform", { exact: true }).selectOption("TIKTOK");
  await submitForm
    .getByLabel("Published clip URL", { exact: true })
    .fill(`https://www.tiktok.com/@klipforgeqa/video/${suffix}0000`);
  await submitForm.getByLabel("Caption", { exact: true }).fill(caption);
  const submitted = page.waitForResponse(
    (r) => r.url().includes(`/campaigns/${campaignID}/submissions`) && r.request().method() === "POST",
  );
  await submitForm.getByRole("button", { name: "Submit for review", exact: true }).click();
  const submittedResponse = await submitted;
  expect(submittedResponse.status()).toBe(201);
  const submissionID = (await submittedResponse.json()).submission.id as string;
  card = page.locator("article").filter({ has: page.getByText(caption, { exact: true }) });
  await card.getByRole("button", { name: "Edit pending clip", exact: true }).click();
  await page.locator("#submission-editor").getByLabel("Caption", { exact: true }).fill(editedCaption);
  await page.getByRole("button", { name: "Save submission", exact: true }).click();
  await expect(
    page.locator("article").filter({ has: page.getByText(editedCaption, { exact: true }) }),
  ).toBeVisible();
  await logout(page);
  await login(page, "brand");
  await tab(page, "Submissions");
  await page.getByLabel("Campaign", { exact: true }).selectOption(campaignID);
  card = page.locator("article").filter({ has: page.getByText(editedCaption, { exact: true }) });
  await card.getByRole("button", { name: /^approve$/i }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Confirm approve", exact: true }).click();
  await expect(card).toHaveCount(0);
  await page.getByLabel("Review status", { exact: true }).selectOption("APPROVED");
  await expect(card.getByText("APPROVED", { exact: true })).toBeVisible();
  await tab(page, "Campaigns");
  card = page.locator("article").filter({ has: page.getByRole("heading", { name: title, exact: true }) });
  await card.getByRole("button", { name: /^complete$/i }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Confirm complete", exact: true }).click();
  await expect(card.getByText("COMPLETED", { exact: true })).toBeVisible();
  await logout(page);
  await login(page, "admin");
  await expect
    .poll(
      async () => {
        await page.request.post(`${base}/api/backend/admin/metrics/collect`, {
          headers: { Origin: base },
          data: {},
        });
        return (
          await page.request.post(`${base}/api/backend/admin/payout-quotes`, {
            headers: { Origin: base },
            data: { submission_id: submissionID },
          })
        ).status();
      },
      { timeout: 75_000, intervals: [1000, 3000, 5000] },
    )
    .toBe(200);
  await tab(page, "Payouts");
  await page.getByLabel("Approved clip", { exact: true }).selectOption(submissionID);
  await page.getByRole("button", { name: "Preview amount", exact: true }).click();
  await page.getByRole("button", { name: /^Reserve USD / }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Confirm reservation", exact: true }).click();
  card = page.locator("article").filter({ has: page.getByRole("heading", { name: title, exact: true }) });
  await expect(card.getByText("PENDING", { exact: true })).toBeVisible();
  await card.getByRole("button", { name: "Approve payout", exact: true }).click();
  await page.getByRole("dialog").getByRole("button", { name: "Confirm approval", exact: true }).click();
  await expect(card.getByText("APPROVED", { exact: true })).toBeVisible();
  await card.getByRole("button", { name: "Record external payment", exact: true }).click();
  await page
    .getByRole("dialog")
    .getByLabel("External payment reference (required)", { exact: true })
    .fill(`qa-receipt-${suffix}`);
  await page.getByRole("dialog").getByRole("button", { name: "Confirm payment record", exact: true }).click();
  await expect(card.getByText("PROCESSED", { exact: true })).toBeVisible();
  await expect(card).toContainText(`manual:qa-receipt-${suffix}`);
  await tab(page, "Audit");
  await expect(page.getByRole("cell", { name: "payout.process", exact: true }).first()).toBeVisible();
  await logout(page);
  await login(page, "clipper");
  await tab(page, "Notifications");
  card = page
    .locator("article")
    .filter({ has: page.getByRole("heading", { name: "Payout status: PROCESSED", exact: true }) })
    .filter({ hasText: title });
  await expect(card).toBeVisible();
  await card.getByRole("button", { name: "Mark as read", exact: true }).click();
  await expect(card.getByText("Read", { exact: true })).toBeVisible();
  expect((await page.request.get(`${base}/api/backend/admin/audit`)).status()).toBe(403);
  expect(
    (
      await page.request.post(`${base}/api/backend/admin/youtube/submissions/${submissionID}/refresh`, {
        headers: { Origin: base },
        data: {},
      })
    ).status(),
  ).toBe(403);
  const integration = await page.request.get(`${base}/api/backend/integrations/youtube`);
  expect(integration.status()).toBe(200);
  expect(await integration.json()).not.toHaveProperty("api_key");
  expect(
    (
      await page.request.post(`${base}/api/backend/admin/payouts`, {
        headers: { Origin: base },
        data: { submission_id: submissionID, expected_amount: "1.00" },
      })
    ).status(),
  ).toBe(403);
  expect(
    (
      await page.request.post(`${base}/api/backend/auth/logout`, {
        headers: { Origin: "https://attacker.example" },
        data: {},
      })
    ).status(),
  ).toBe(403);
  expect(await page.evaluate(() => document.cookie)).not.toContain("klipforge_access");
  await page.context().clearCookies({ name: "klipforge_access" });
  await page.reload();
  await expect(page.getByRole("navigation", { name: "Workspace navigation" })).toBeVisible();
  await expect(page.getByText("CLIPPER", { exact: true })).toBeVisible();
});
