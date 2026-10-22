import { test, expect } from "@playwright/test";

test("saved plan, answerable followups, resume, summary, failure retry and stale privacy", async ({
  page,
  request,
}, testInfo) => {
  const api = "http://127.0.0.1:58081";
  const fixture = (await (await request.get(`${api}/fixture`)).json()) as {
    owner: string;
    job: string;
    resume: string;
  };
  await request.post(`${api}/fixture/mode`, { data: { mode: "" } });
  await request.post(`${api}/fixture/quota`, { data: { limit: -1 } });
  await page.context().route("**/*", async (route) => {
    const host = new URL(route.request().url()).hostname;
    if (host === "127.0.0.1" || host === "localhost") await route.continue();
    else await route.abort();
  });
  await page.addInitScript(({ owner }) => {
    localStorage.setItem(
      "jobber-auth",
      JSON.stringify({
        state: {
          user: {
            id: owner,
            name: "Synthetic Candidate",
            email: "synthetic@example.invalid",
            locale: "en",
            created_at: new Date().toISOString(),
          },
          isAuthenticated: true,
        },
        version: 0,
      }),
    );
    localStorage.setItem("language", "en");
    localStorage.setItem("jobber-onboarding-completed", "true");
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "essential", version: 1 }),
    );
  }, fixture);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(`/app/jobs/${fixture.job}`);
  await page
    .getByRole("button", { name: "Open interview preparation", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Interview preparation",
    exact: true,
  });
  await expect(
    dialog.getByLabel("Builder Resume", { exact: true }),
  ).toHaveValue(fixture.resume);
  await dialog
    .getByRole("button", { name: /Generate (preparation|a new version)/ })
    .click();
  await expect(
    dialog.getByText("Completed and saved.", { exact: true }),
  ).toBeVisible({ timeout: 30000 });
  await expect(
    dialog.getByRole("heading", { name: "Requirements and your experience" }),
  ).toBeVisible();
  await expect(dialog.locator(".ph-no-capture")).toHaveCount(1);
  await dialog
    .getByRole("button", { name: "Mark studied", exact: true })
    .first()
    .click();
  await expect(
    dialog.getByRole("button", { name: "Studied · undo", exact: true }),
  ).toBeVisible();
  await dialog.screenshot({
    path: `../docs/qa/evidence/interview-preparation-2026-10-22/${testInfo.project.name}-plan-details.png`,
  });
  await dialog.locator(".min-h-0").evaluate((node) => {
    node.scrollTop = 0;
  });
  await dialog.screenshot({
    path: `../docs/qa/evidence/interview-preparation-2026-10-22/${testInfo.project.name}-plan.png`,
  });
  await dialog.getByRole("button", { name: "Practice", exact: true }).click();
  await dialog
    .getByRole("button", { name: "Start a new practice session", exact: true })
    .click();
  await expect(dialog.getByLabel("Your answer", { exact: true })).toBeVisible();
  await expect(dialog.getByLabel("Your answer", { exact: true })).toBeVisible();
  const sessionId = await dialog
    .getByLabel("Saved sessions", { exact: true })
    .inputValue();
  await dialog
    .getByLabel("Your answer", { exact: true })
    .fill(
      "I investigated latency, compared traces and explained the change to my team.",
    );
  await dialog
    .getByRole("button", { name: "Save answer", exact: true })
    .click();
  await expect(
    dialog.getByText(
      "I investigated latency, compared traces and explained the change to my team.",
      { exact: true },
    ),
  ).toBeVisible();
  await dialog
    .getByRole("button", {
      name: "Get answer feedback · 1 AI request",
      exact: true,
    })
    .click();
  await expect(dialog.getByText("Content · 2/4", { exact: true })).toBeVisible({
    timeout: 30000,
  });
  await dialog
    .getByRole("button", { name: "Pause session", exact: true })
    .click();
  await expect(
    dialog.getByRole("button", { name: "Resume session", exact: true }),
  ).toBeVisible();
  await page.reload();
  await page
    .getByRole("button", { name: "Open interview preparation", exact: true })
    .click();
  await dialog.getByRole("button", { name: "Practice", exact: true }).click();
  await dialog
    .getByLabel("Saved sessions", { exact: true })
    .selectOption(sessionId);
  await dialog
    .getByRole("button", { name: "Resume session", exact: true })
    .click();
  await dialog
    .getByRole("button", { name: "Skip question", exact: true })
    .click();
  await dialog
    .getByLabel("Your answer", { exact: true })
    .fill("I would clarify the requirements and discuss a measured tradeoff.");
  await dialog
    .getByRole("button", { name: "Save answer", exact: true })
    .click();
  await expect(
    dialog.getByRole("heading", {
      name: "What did you personally change?",
      exact: true,
    }),
  ).toBeVisible();
  await dialog
    .getByLabel("Your answer", { exact: true })
    .fill(
      "I personally changed the trace sampling and documented what I observed.",
    );
  await dialog
    .getByRole("button", { name: "Save answer", exact: true })
    .click();
  await dialog
    .getByRole("button", { name: "Finish session", exact: true })
    .click();
  await dialog
    .getByRole("button", {
      name: "Get final review · 1 AI request",
      exact: true,
    })
    .click();
  await expect(
    dialog.getByRole("heading", { name: "Content · 2/4", exact: true }),
  ).toHaveCount(2, { timeout: 30000 });
  await dialog
    .getByRole("heading", { name: "Final review", exact: true })
    .scrollIntoViewIfNeeded();
  await dialog.screenshot({
    path: `../docs/qa/evidence/interview-preparation-2026-10-22/${testInfo.project.name}-practice.png`,
  });
  await request.post(`${api}/fixture/mode`, { data: { mode: "malformed" } });
  await dialog
    .getByRole("button", { name: "Generate a new version", exact: true })
    .click();
  await expect(
    dialog.getByText(/AI returned an unsupported result/),
  ).toBeVisible({ timeout: 30000 });
  await request.post(`${api}/fixture/mode`, { data: { mode: "" } });
  await dialog
    .getByRole("button", {
      name: "Retry same operation · no new charge",
      exact: true,
    })
    .first()
    .click();
  await expect(
    dialog.getByText("Completed and saved.", { exact: true }),
  ).toBeVisible({ timeout: 30000 });
  await request.post(`${api}/fixture/mode`, { data: { mode: "unavailable" } });
  await page.reload();
  await page
    .getByRole("button", { name: "Open interview preparation", exact: true })
    .click();
  await expect(
    dialog.getByText(
      "AI is unavailable. Saved preparation, sessions and answers remain available.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(
    dialog.getByRole("heading", { name: "Requirements and your experience" }),
  ).toBeVisible();
  await request.post(`${api}/fixture/mode`, { data: { mode: "" } });
  await request.post(`${api}/fixture/stale`);
  await page.reload();
  await page
    .getByRole("button", { name: "Open interview preparation", exact: true })
    .click();
  await expect(
    dialog.getByText(/Source data changed or the Builder Resume was deleted/),
  ).toBeVisible();
  await dialog.screenshot({
    path: `../docs/qa/evidence/interview-preparation-2026-10-22/${testInfo.project.name}-stale.png`,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", {
      name: "Open interview preparation",
      exact: true,
    }),
  ).toBeFocused();
  expect(errors).toEqual([]);
});

test("recovers pending generation and feedback after reload, interrupted retry and quota", async ({
  page,
  request,
}, testInfo) => {
  const api = "http://127.0.0.1:58081";
  const fixture = (await (await request.get(`${api}/fixture`)).json()) as {
    owner: string;
    job: string;
    resume: string;
  };
  await request.post(`${api}/fixture/quota`, { data: { limit: -1 } });
  await request.post(`${api}/fixture/mode`, { data: { mode: "slow" } });
  await page.context().route("**/*", async (route) => {
    if (
      ["127.0.0.1", "localhost"].includes(
        new URL(route.request().url()).hostname,
      )
    )
      await route.continue();
    else await route.abort();
  });
  await page.addInitScript(({ owner }) => {
    localStorage.setItem(
      "jobber-auth",
      JSON.stringify({
        state: {
          user: {
            id: owner,
            name: "Synthetic Candidate",
            email: "synthetic@example.invalid",
            locale: "en",
            created_at: new Date().toISOString(),
          },
          isAuthenticated: true,
        },
        version: 0,
      }),
    );
    localStorage.setItem("language", "en");
    localStorage.setItem("jobber-onboarding-completed", "true");
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "essential", version: 1 }),
    );
  }, fixture);
  await page.goto(`/app/jobs/${fixture.job}`);
  const open = async () =>
    page
      .getByRole("button", { name: "Open interview preparation", exact: true })
      .click();
  await open();
  const dialog = page.getByRole("dialog", {
    name: "Interview preparation",
    exact: true,
  });
  const prefix = `${api}/api/v1/jobs/${fixture.job}/interview-prep`;
  const before = (await (await request.get(prefix)).json()) as {
    versions: { id: string }[];
  };
  await dialog
    .getByRole("button", { name: "Generate a new version", exact: true })
    .click();
  await expect(
    dialog.getByText(/Generating — safe to close this page/).first(),
  ).toBeVisible({ timeout: 10000 });
  await page.keyboard.press("Escape");
  await page.reload();
  await open();
  await expect(
    dialog.getByRole("heading", {
      name: "Requirements and your experience",
      exact: true,
    }),
  ).toBeVisible();
  await expect
    .poll(
      async () =>
        (
          (await (await request.get(prefix)).json()) as {
            versions: { id: string }[];
          }
        ).versions.length,
      { timeout: 30000 },
    )
    .toBe(before.versions.length + 1);
  const generatedHistory = (await (await request.get(prefix)).json()) as {
    versions: { id: string }[];
  };
  const generatedVersion = generatedHistory.versions.find(
    (v) => !before.versions.some((previous) => previous.id === v.id),
  );
  expect(generatedVersion).toBeDefined();
  if (!generatedVersion) throw new Error("Expected one durable new version");
  await expect(
    dialog.getByLabel("Preparation versions", { exact: true }),
  ).toHaveValue(generatedVersion.id, { timeout: 10000 });
  await dialog.getByRole("button", { name: "Practice", exact: true }).click();
  await dialog
    .getByRole("button", { name: "Start a new practice session", exact: true })
    .click();
  await expect(dialog.getByLabel("Your answer", { exact: true })).toBeVisible();
  const sessionId = await dialog
    .getByLabel("Saved sessions", { exact: true })
    .inputValue();
  const startedSession = (await (
    await request.get(`${prefix}/sessions/${sessionId}`)
  ).json()) as { version_id: string };
  expect(startedSession.version_id).toBe(generatedVersion.id);
  await dialog
    .getByLabel("Your answer", { exact: true })
    .fill("This answer is durable before slow feedback completes.");
  await dialog
    .getByRole("button", { name: "Save answer", exact: true })
    .click();
  await dialog
    .getByRole("button", {
      name: "Get answer feedback · 1 AI request",
      exact: true,
    })
    .click();
  await expect(
    dialog.getByText(/Generating — safe to close this page/).first(),
  ).toBeVisible({ timeout: 10000 });
  await page.reload();
  await open();
  await dialog.getByRole("button", { name: "Practice", exact: true }).click();
  await dialog
    .getByLabel("Saved sessions", { exact: true })
    .selectOption(sessionId);
  await expect(
    dialog.locator("article > p").filter({
      hasText: "This answer is durable before slow feedback completes.",
    }),
  ).toBeVisible();
  await expect(
    dialog.getByText(/Generating — safe to close this page/).first(),
  ).toBeVisible({ timeout: 10000 });
  await expect(
    dialog.getByRole("heading", { name: "Content · 2/4", exact: true }),
  ).toBeVisible({ timeout: 30000 });
  await dialog
    .getByRole("heading", { name: "Content · 2/4", exact: true })
    .scrollIntoViewIfNeeded();
  await dialog.screenshot({
    path: `../docs/qa/evidence/interview-preparation-2026-10-22/${testInfo.project.name}-feedback-viewport.png`,
  });
  await request.post(`${api}/fixture/quota`, { data: { limit: 0 } });
  await dialog
    .getByRole("button", { name: "Generate a new version", exact: true })
    .click();
  await expect(
    dialog.getByText(
      "Your AI allowance is exhausted. Saved answers and history remain available.",
      { exact: true },
    ),
  ).toBeVisible();
  await expect(
    dialog.locator("article > p").filter({
      hasText: "This answer is durable before slow feedback completes.",
    }),
  ).toBeVisible();
  await request.post(`${api}/fixture/quota`, { data: { limit: -1 } });
  await dialog
    .getByRole("button", { name: "Generate a new version", exact: true })
    .click();
  await expect(
    dialog.getByText(/Generating — safe to close this page/).first(),
  ).toBeVisible({ timeout: 10000 });
  await request.post(`${api}/fixture/interrupt`);
  await expect(
    dialog.getByText(/The server interrupted this operation/),
  ).toBeVisible({ timeout: 10000 });
  await request.post(`${api}/fixture/mode`, { data: { mode: "" } });
  await dialog
    .getByRole("button", {
      name: "Retry same operation · no new charge",
      exact: true,
    })
    .first()
    .click();
  await expect(
    dialog.getByText("Completed and saved.", { exact: true }),
  ).toBeVisible({ timeout: 30000 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
});

test("explicitly recovers a failed summary outside the latest thirty operations", async ({
  page,
  request,
}, testInfo) => {
  const api = "http://127.0.0.1:58081";
  const fixture = (await (await request.get(`${api}/fixture`)).json()) as {
    owner: string;
    job: string;
    resume: string;
  };
  const prefix = `${api}/api/v1/jobs/${fixture.job}/interview-prep`;
  await request.post(`${api}/fixture/quota`, { data: { limit: -1 } });
  await request.post(`${api}/fixture/mode`, { data: { mode: "" } });
  const generated = (await (
    await request.post(`${prefix}/generate`, {
      data: {
        resume_builder_id: fixture.resume,
        locale: "en",
        key: crypto.randomUUID(),
      },
    })
  ).json()) as { id: string };
  await expect
    .poll(
      async () =>
        (
          (await (
            await request.get(`${prefix}/operations/${generated.id}`)
          ).json()) as { status: string }
        ).status,
      { timeout: 30000 },
    )
    .toBe("succeeded");
  const operation = (await (
    await request.get(`${prefix}/operations/${generated.id}`)
  ).json()) as { version_id: string };
  const started = (await (
    await request.post(`${prefix}/sessions`, {
      data: { version_id: operation.version_id, key: crypto.randomUUID() },
    })
  ).json()) as { id: string; revision: number; questions: { id: string }[] };
  const answered = (await (
    await request.post(`${prefix}/sessions/${started.id}/answers`, {
      data: {
        key: crypto.randomUUID(),
        question_id: started.questions[0].id,
        text: "My saved history recovery answer",
        skip: false,
        revision: started.revision,
      },
    })
  ).json()) as { revision: number };
  expect(
    (
      await request.post(`${prefix}/sessions/${started.id}/action`, {
        data: {
          key: crypto.randomUUID(),
          action: "finish",
          revision: answered.revision,
        },
      })
    ).ok(),
  ).toBe(true);
  await request.post(`${api}/fixture/mode`, { data: { mode: "fail" } });
  const failed = (await (
    await request.post(`${prefix}/sessions/${started.id}/summary`, {
      data: { key: crypto.randomUUID() },
    })
  ).json()) as { id: string };
  await expect
    .poll(
      async () =>
        (
          (await (
            await request.get(`${prefix}/operations/${failed.id}`)
          ).json()) as { status: string }
        ).status,
      { timeout: 30000 },
    )
    .toBe("failed");
  expect(
    (
      await request.post(`${api}/fixture/history-noise`, {
        data: { operation_id: failed.id },
      })
    ).ok(),
  ).toBe(true);
  const history = (await (await request.get(prefix)).json()) as {
    operations: { id: string }[];
  };
  expect(history.operations).toHaveLength(30);
  expect(history.operations.some((op) => op.id === failed.id)).toBe(false);
  await page.context().route("**/*", async (route) => {
    if (
      ["127.0.0.1", "localhost"].includes(
        new URL(route.request().url()).hostname,
      )
    )
      await route.continue();
    else await route.abort();
  });
  await page.addInitScript(({ owner }) => {
    localStorage.setItem(
      "jobber-auth",
      JSON.stringify({
        state: {
          user: {
            id: owner,
            name: "Synthetic Candidate",
            email: "synthetic@example.invalid",
            locale: "en",
            created_at: new Date().toISOString(),
          },
          isAuthenticated: true,
        },
        version: 0,
      }),
    );
    localStorage.setItem("language", "en");
    localStorage.setItem("jobber-onboarding-completed", "true");
    localStorage.setItem(
      "cookie-consent",
      JSON.stringify({ choice: "essential", version: 1 }),
    );
  }, fixture);
  let retries = 0;
  page.on("request", (request) => {
    if (
      request.method() === "POST" &&
      request.url().endsWith(`/operations/${failed.id}/retry`)
    )
      retries++;
  });
  await page.goto(`/app/jobs/${fixture.job}`);
  await page
    .getByRole("button", { name: "Open interview preparation", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Interview preparation",
    exact: true,
  });
  await dialog.getByRole("button", { name: "Practice", exact: true }).click();
  await dialog
    .getByLabel("Saved sessions", { exact: true })
    .selectOption(started.id);
  await dialog
    .getByRole("button", {
      name: "Get final review · 1 AI request",
      exact: true,
    })
    .click();
  const retry = dialog.getByRole("button", {
    name: "Retry same operation · no new charge",
    exact: true,
  });
  await expect(retry).toBeVisible();
  expect(retries).toBe(0);
  await request.post(`${api}/fixture/mode`, { data: { mode: "" } });
  await retry.click();
  await expect.poll(() => retries).toBe(1);
  await expect(
    dialog.getByText("Practice a concrete example", { exact: true }),
  ).toBeVisible({ timeout: 30000 });
  await dialog
    .getByRole("heading", { name: "Final review", exact: true })
    .scrollIntoViewIfNeeded();
  await dialog.screenshot({
    path: `../docs/qa/evidence/interview-preparation-2026-10-22/${testInfo.project.name}-history-recovery.png`,
  });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("button", {
      name: "Open interview preparation",
      exact: true,
    }),
  ).toBeFocused();
});
