import { useLayoutEffect } from "react";
import { MemoryRouter, Routes, Route, useLocation, useNavigate } from "react-router-dom";
import { useAIConsentSignal } from "@/shared/hooks/useAIConsentSignal";
import { apiClient } from "@/services/api";
import { afterEach, expect, it, vi } from "vitest";
import { render, screen, cleanup, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AICreditConfirmation } from "./AICreditConfirmation";
import { confirmAIAction, aiConsentStore, creditOperation } from "@/shared/lib/aiCredits";
import { subscriptionService } from "@/services/subscriptionService";
import "@/shared/lib/i18n";
vi.mock("@/services/subscriptionService", () => ({ subscriptionService: { getSubscription: vi.fn() } }));
afterEach(() => { aiConsentStore.getSnapshot()?.finish(false); cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
function mount(remaining = 30, cost = 7) {
  vi.mocked(subscriptionService.getSubscription).mockResolvedValue({ plan: "free", status: "free", credits: { granted: 30, used: 30 - remaining, remaining, kind: "free_once", spend_limit_nanodollars: 3_000_000_000, spend_allocated_nanodollars: 0 }, credit_costs: { resume_import: cost }, limits: { max_jobs:25,max_resumes:1,max_resume_builders:2,max_cover_letters:1,max_job_parses:0,max_ai_requests:30 }, usage: {jobs:0,resumes:0,resume_builders:0,cover_letters:0,job_parses:0,ai_requests:0} });
  return render(<QueryClientProvider client={new QueryClient()}><AICreditConfirmation /></QueryClientProvider>);
}
it("waits for acceptance and shows the authoritative cost and failure policy", async () => {
  mount(30, 9);
  let request: Promise<string | undefined> = Promise.resolve(undefined);
  act(() => { request = confirmAIAction("resume-builder/import/pdf"); });
  expect(await screen.findByText("Cost: 9 credits · Available: 30")).toBeVisible();
  expect(screen.getByText(/Errors after AI starts still use credits/)).toBeVisible();
  expect(screen.getByText(/Retrying an existing interview operation is included.*3 attempts/)).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "Use 9 credits" }));
  expect(await request).toMatch(/^[0-9a-f-]{36}$/);
});
it.each([30, 2])("cancel preserves the request and never starts AI with balance %s", async (remaining) => {
  mount(remaining);
  let request: Promise<unknown> = Promise.resolve();
  act(() => { request = confirmAIAction("resume-builder/import/text").catch((error: unknown) => error); });
  await screen.findByText(`Cost: 7 credits · Available: ${remaining}`);
  if (remaining < 7) expect(screen.getByRole("button", { name: "Use 7 credits" })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});
it.each([
  ["match-score", "match"], ["cover-letters/ai/generate", "cover_letter"],
  ["resume-builder/import/text", "resume_import"], ["resume-builder/import/pdf", "resume_import"],
  ["jobs/j/resume-tailoring/import", "resume_import"], ["jobs/j/resume-tailoring", "tailor"],
  ["resume-builder/r/ats-check", "ats"], ["resume-builder/ai/suggest-bullets", "bullet"],
  ["resume-builder/ai/suggest-summary", "summary"], ["resume-builder/ai/improve-text", "improve"],
  ["jobs/j/interview-prep/generate", "interview_plan"],
  ["jobs/j/interview-prep/sessions/s/answers/a/feedback", "interview_feedback"],
  ["jobs/j/interview-prep/sessions/s/summary", "interview_summary"],
  ["resumes/r/autofill-profile", "autofill_import"], ["jobs/parse", "job_parse"],
  ["jobs/j/resume-tailoring/recognition", null], ["jobs/j/resume-tailoring/p/apply", null],
  ["jobs/j/interview-prep/operations/o/retry", null],
])("classifies %s and leaves non-provider steps free", (path, expected) => {
  expect(creditOperation(path)).toBe(expected);
});

it("unmount cancels without network activity and a later action can open again", async () => {
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  const first = mount();
  let request: Promise<unknown> = Promise.resolve();
  act(() => { request = apiClient.post("resume-builder/import/text", {}).catch((error: unknown) => error); });
  await screen.findByRole("dialog");
  first.unmount();
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(fetch).not.toHaveBeenCalled();
  mount();
  act(() => { request = apiClient.post("resume-builder/import/text", {}).catch((error: unknown) => error); });
  await screen.findByRole("dialog");
  await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(fetch).not.toHaveBeenCalled();
});

it("blocks dispatch when the authoritative balance cannot be verified", async () => {
  mount();
  vi.mocked(subscriptionService.getSubscription).mockRejectedValue(new Error("offline"));
  let request: Promise<unknown> = Promise.resolve();
  act(() => { request = confirmAIAction("resume-builder/import/pdf").catch((error: unknown) => error); });
  expect(await screen.findByText(/Could not verify the credit cost and balance/)).toBeVisible();
  expect(screen.getByRole("button", { name: /Use.*credits/ })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(await request).toMatchObject({ name: "AbortError" });
});

it("caller abort clears pending consent before dispatch", async () => {
  mount();
  const controller = new AbortController();
  let request: Promise<unknown> = Promise.resolve();
  act(() => { request = confirmAIAction("resume-builder/import/text", controller.signal).catch((error: unknown) => error); });
  await screen.findByRole("dialog");
  act(() => controller.abort());
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(aiConsentStore.getSnapshot()).toBeNull();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});

it("Back cancels pending consent while the global confirmation remains mounted", async () => {
  mount(); cleanup();
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  let goBack = () => {};
  let request: Promise<unknown> = Promise.resolve();
  function Boundary() {
    const location = useLocation();
    const navigate = useNavigate();
    useLayoutEffect(() => { goBack = () => navigate(-1); }, [navigate]);
    return <><Routes>
      <Route path="/origin" element={<button onClick={() => { request = apiClient.post("resume-builder/import/text", {}).catch((error: unknown) => error); }}>Start import</button>} />
      <Route path="/other" element={<h1>Other page</h1>} />
    </Routes><AICreditConfirmation originKey={location.key} /></>;
  }
  render(<QueryClientProvider client={new QueryClient()}><MemoryRouter initialEntries={["/other", "/origin"]} initialIndex={1}><Boundary /></MemoryRouter></QueryClientProvider>);
  await userEvent.click(screen.getByRole("button", { name: "Start import" }));
  await screen.findByText("Cost: 7 credits · Available: 30");
  act(() => goBack());
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(screen.getByRole("heading", { name: "Other page" })).toBeVisible();
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(fetch).not.toHaveBeenCalled();
});

it("unmounting only the originating form cancels consent without dispatch", async () => {
  mount(); cleanup();
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  let request: Promise<unknown> = Promise.resolve();
  function Form() {
    const getSignal = useAIConsentSignal();
    return <button onClick={() => { request = apiClient.post("resume-builder/import/text", {}, { consentSignal: getSignal() }).catch((error: unknown) => error); }}>Start scoped import</button>;
  }
  const client = new QueryClient();
  const tree = (show: boolean) => <QueryClientProvider client={client}>{show && <Form />}<AICreditConfirmation originKey="same-route" /></QueryClientProvider>;
  const view = render(tree(true));
  await userEvent.click(screen.getByRole("button", { name: "Start scoped import" }));
  await screen.findByText("Cost: 7 credits · Available: 30");
  view.rerender(tree(false));
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(fetch).not.toHaveBeenCalled();
});
