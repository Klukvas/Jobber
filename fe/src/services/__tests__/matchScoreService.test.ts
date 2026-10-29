import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { matchScoreService } from "../matchScoreService";
import { AICreditConfirmation } from "@/features/subscription/components/AICreditConfirmation";
import { aiConsentStore } from "@/shared/lib/aiCredits";
import "@/shared/lib/i18n";

const score = { overall_score: 85, categories: [], missing_keywords: [], strengths: [], summary: "Saved result", from_cache: true };
let requests: Request[];
function mount(remaining = 0, cached: Response = Response.json(score)) {
  vi.stubGlobal("fetch", vi.fn(async (request: Request) => {
    requests.push(request);
    if (request.url.includes("subscription")) return Response.json({ credits: { remaining }, credit_costs: { match: 5 } });
    return request.method === "GET" ? cached.clone() : Response.json({ ...score, from_cache: false });
  }));
  return render(createElement(QueryClientProvider, { client: new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } }) }, createElement(AICreditConfirmation)));
}
const miss = () => Response.json({ error_code: "MATCH_SCORE_NOT_READY", error_message: "No saved analysis" }, { status: 404 });
const paid = () => requests.filter((request) => request.method === "POST");
beforeEach(() => { requests = []; });
afterEach(() => { aiConsentStore.getSnapshot()?.finish(false); cleanup(); vi.unstubAllGlobals(); });

describe("cache-first match score through the real API and credit confirmation", () => {
  it.each([0, 1, 2, 3, 4])("opens a saved result at balance %s without reading subscription or confirming", async (remaining) => {
    mount(remaining);
    const request = matchScoreService.checkMatch("job", "resume").catch((error: unknown) => error);
    await waitFor(() => expect(requests).toHaveLength(1));
    expect(requests[0].method).toBe("GET");
    expect(requests[0].url).toContain("match-score?job_id=job&resume_id=resume");
    expect(await request).toEqual(score);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(paid()).toHaveLength(0);
  });
  it.each([0, 1, 2, 3, 4])("an exact miss with balance %s disables paid dispatch", async (remaining) => {
    mount(remaining, miss());
    const request = matchScoreService.checkMatch("job", "resume").catch((error: unknown) => error);
    await screen.findByText(`Cost: 5 credits · Available: ${remaining}`);
    expect(screen.getByRole("button", { name: "Use 5 credits" })).toBeDisabled();
    expect(paid()).toHaveLength(0);
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(await request).toMatchObject({ name: "AbortError" });
  });
  it.each([false, true])("a miss at sufficient balance respects accept=%s", async (accept) => {
    mount(5, miss());
    const request = matchScoreService.checkMatch("job", "resume").catch((error: unknown) => error);
    await screen.findByText("Cost: 5 credits · Available: 5");
    expect(paid()).toHaveLength(0);
    await userEvent.click(screen.getByRole("button", { name: accept ? "Use 5 credits" : "Cancel" }));
    const result = await request;
    expect(paid()).toHaveLength(accept ? 1 : 0);
    if (accept) expect(result).toEqual({ ...score, from_cache: false });
    else expect(result).toMatchObject({ name: "AbortError" });
  });
  it.each([
    [401, { error_code: "UNAUTHORIZED" }], [403, { error_code: "FORBIDDEN" }],
    [404, { error_code: "JOB_NOT_FOUND" }], [404, {}], [500, { error_code: "INTERNAL_ERROR" }],
    [200, {}], [200, { ...score, from_cache: false }],
  ])("fails closed on response %s %j", async (status, body) => {
    mount(30, Response.json(body, { status }));
    const request = matchScoreService.checkMatch("job", "resume").catch((error: unknown) => error);
    await waitFor(() => expect(requests.length).toBeGreaterThan(0));
    expect(await request).toBeInstanceOf(Error);
    expect(aiConsentStore.getSnapshot()).toBeNull();
    expect(paid().filter((r) => !r.url.includes("auth/refresh"))).toHaveLength(0);
  });
  it("does not open consent after the originating form aborts during lookup", async () => {
    let resolve: (response: Response) => void = () => {};
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>((done) => { resolve = done; })));
    const controller = new AbortController();
    const request = matchScoreService.checkMatch("job", "resume", { consentSignal: controller.signal }).catch((error: unknown) => error);
    await waitFor(() => expect(fetch).toHaveBeenCalled());
    controller.abort();
    await act(async () => resolve(miss()));
    expect(await request).toMatchObject({ name: "AbortError" });
    expect(aiConsentStore.getSnapshot()).toBeNull();
  });
});
