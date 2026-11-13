import { beforeEach, describe, expect, it, vi } from "vitest";
const client = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), delete: vi.fn() }));
vi.mock("./api", async (original) => ({ ...await original<typeof import("./api")>(), apiClient: client }));
import { apiTokensService } from "./apiTokensService";

const token = { id: "1", name: "Laptop", token_prefix: "jbr_pat_ab12", scopes: ["mcp"], created_at: "2026-01-01T00:00:00Z", last_used_at: null, expires_at: null, revoked_at: null };
beforeEach(() => vi.clearAllMocks());

describe("apiTokensService", () => {
  it("lists validated DTOs without retaining unexpected secret fields", async () => {
    client.get.mockResolvedValue([{ ...token, token: "unexpected-secret" }]);
    expect(await apiTokensService.list()).toEqual([token]);
    expect(client.get).toHaveBeenCalledWith("api-tokens", undefined);
  });
  it.each([undefined, null, 30, 90, 365] as const)("creates with expiry %s and reads the flat response", async (days) => {
    client.post.mockResolvedValue({ ...token, token: "test-only-token" });
    const input = { name: "Laptop", ...(days === undefined ? {} : { expires_in_days: days }) };
    const options = { signal: new AbortController().signal };
    expect(await apiTokensService.create(input, options)).toEqual({ ...token, token: "test-only-token" });
    expect(client.post).toHaveBeenCalledWith("api-tokens", input, options);
  });
  it("revokes via DELETE", async () => {
    await apiTokensService.revoke("token/1");
    expect(client.delete).toHaveBeenCalledWith("api-tokens/token%2F1");
  });
  it("fetches the last 20 calls with the caller's cancellation signal", async () => {
    const activity = { id: "a1", tool: "list_jobs", ok: true, error_code: null, duration_ms: 1, created_at: token.created_at, api_token_id: null, token_name: null };
    client.get.mockResolvedValue([activity]);
    const options = { signal: new AbortController().signal };
    expect(await apiTokensService.activity(options)).toEqual([activity]);
    expect(client.get).toHaveBeenCalledWith("api-tokens/activity?limit=20", options);
  });
  it("rejects malformed responses without putting plaintext into the error", async () => {
    client.post.mockResolvedValue({ token: "test-only-secret" });
    await expect(apiTokensService.create({ name: "Laptop" })).rejects.toMatchObject({ code: "INVALID_RESPONSE", message: "" });
  });
});
