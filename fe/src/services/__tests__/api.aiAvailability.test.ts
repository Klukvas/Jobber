import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient } from "@tanstack/react-query";

beforeEach(() => { vi.resetModules(); vi.stubEnv("VITE_API_BASE_URL", "http://api.test/api/v1"); });
afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
describe("HTTP error capability classification", () => {
  it.each([
    [404, "RESUME_BUILDER_NOT_FOUND", false],
    [404, "COVER_LETTER_NOT_FOUND", false],
    [404, "NOT_FOUND", false],
    [403, "PLAN_LIMIT_REACHED", false],
    [503, "AI_UNAVAILABLE", true],
    [503, "AI_NOT_CONFIGURED", true],
    [500, "INTERNAL_ERROR", false],
    [404, "UNKNOWN_ERROR", false],
  ])("structured %s %s never masquerades as an absent route", async (status, code, unavailable) => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({ error_code: code, error_message: "Synthetic error" }), { status, headers: { "content-type": "application/json" } })));
    const { apiClient, ApiError } = await import("../api");
    const { markAIUnavailable, aiErrorKey } = await import("@/shared/hooks/useAIAvailability");
    const client = new QueryClient(); client.setQueryData(["features", "ai"], { available: true });
    const error = await apiClient.post("resume-builder/synthetic/ats-check", {}).catch((failure: unknown) => failure);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as InstanceType<typeof ApiError>).unstructuredResponse).toBe(false);
    markAIUnavailable(client, error);
    expect(client.getQueryData(["features", "ai"])).toEqual({ available: !unavailable });
    expect(aiErrorKey(error)).toBe(unavailable ? "ai.unavailable" : "ai.requestFailed");
  });
  it("recognizes a legacy non-JSON 404 at the actual API boundary", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("404 page not found", { status: 404, headers: { "content-type": "text/plain" } })));
    const { apiClient, ApiError } = await import("../api");
    const { markAIUnavailable } = await import("@/shared/hooks/useAIAvailability");
    const client = new QueryClient(); client.setQueryData(["features", "ai"], { available: true });
    const error = await apiClient.post("resume-builder/ai/improve-text", {}).catch((failure: unknown) => failure);
    expect(error).toBeInstanceOf(ApiError); expect(error).toMatchObject({ code: "UNKNOWN_ERROR", status: 404, unstructuredResponse: true });
    markAIUnavailable(client, error); expect(client.getQueryData(["features", "ai"])).toEqual({ available: false });
  });
  it("valid JSON without a code is structured and does not disable AI", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("{}", { status: 404, headers: { "content-type": "application/json" } })));
    const { apiClient } = await import("../api");
    const { isAIUnavailable } = await import("@/shared/hooks/useAIAvailability");
    const error = await apiClient.post("resume-builder/synthetic/ats-check", {}).catch((failure: unknown) => failure);
    expect(isAIUnavailable(error)).toBe(false);
  });
});
