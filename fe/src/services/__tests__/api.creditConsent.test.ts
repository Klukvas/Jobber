import { afterEach, expect, it, vi } from "vitest";
import { aiConsentStore } from "@/shared/lib/aiCredits";
import { apiClient } from "../api";
afterEach(() => { vi.unstubAllGlobals(); });
it("sends nothing before consent and keeps the operation key through a 401 replay", async () => {
  const requests: Request[] = [];
  vi.stubGlobal("fetch", vi.fn(async (request: Request) => {
    requests.push(request);
    if (request.url.includes("auth/refresh")) return new Response("{}", { headers: { "Content-Type": "application/json" } });
    return new Response("{}", { status: requests.length === 1 ? 401 : 200, headers: { "Content-Type": "application/json" } });
  }));
  const request = apiClient.post("match-score", { job_id: "j", resume_id: "r" });
  expect(requests).toHaveLength(0);
  aiConsentStore.getSnapshot()?.finish(true);
  await request;
  expect(requests).toHaveLength(3);
  const key = requests[0].headers.get("Idempotency-Key");
  expect(key).toMatch(/^[0-9a-f-]{36}$/);
  expect(requests[2].headers.get("Idempotency-Key")).toBe(key);
});
it("cancelled consent never reaches fetch", async () => {
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch);
  const request = apiClient.post("match-score", {}).catch((error: unknown) => error);
  aiConsentStore.getSnapshot()?.finish(false);
  expect(await request).toMatchObject({ name: "AbortError" });
  expect(fetch).not.toHaveBeenCalled();
});
