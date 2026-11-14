import { beforeEach, describe, expect, it, vi } from "vitest";
const client = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), delete: vi.fn() }));
vi.mock("./api", async (original) => ({ ...await original<typeof import("./api")>(), apiClient: client }));
import { oauthService, readAuthorizationParams } from "./oauthService";

const callback = "https://assistant.example/callback?existing=1";
const search = `?client_id=test&redirect_uri=${encodeURIComponent(callback)}&state=test-state&code_challenge=test-challenge&code_challenge_method=S256&response_type=code&scope=mcp`;
const params = readAuthorizationParams(search);
const preview = { client: { client_id: "test", name: "Assistant", redirect_host: "assistant.example", is_metadata_document: false }, scope: "mcp", resource: "https://cursopus.com/api/mcp", plan_allowed: true };
const grant = { id: "11111111-1111-4111-8111-111111111111", client_name: "Assistant", client_id: "test", redirect_host: "assistant.example", created_at: "2026-11-14T00:00:00Z", last_used_at: null };
beforeEach(() => vi.clearAllMocks());

describe("oauthService", () => {
  it("previews the unchanged query with cancellation and validates/strips the response", async () => {
    client.get.mockResolvedValue({ ...preview, secret: "discard-me" });
    const options = { signal: new AbortController().signal };
    expect(await oauthService.preview(search, options)).toEqual(preview);
    expect(client.get).toHaveBeenCalledWith(`oauth/authorize/preview${search}`, options);
  });
  it.each(["allow", "deny"] as const)("submits %s with all OAuth parameters and returns the callback", async (decision) => {
    const response = { redirect_to: `${callback}&code=test-only-code` };
    client.post.mockResolvedValue(response);
    expect(await oauthService.authorize(params, decision)).toEqual(response);
    expect(client.post).toHaveBeenCalledWith("oauth/authorize", { ...params, decision }, undefined);
    expect(params).not.toHaveProperty("decision");
  });
  it("accepts a trusted OAuth error redirect from preview", async () => {
    const response = { redirect_to: `${callback}&error=invalid_request&state=test-state` };
    client.get.mockResolvedValue(response);
    expect(await oauthService.preview(search)).toEqual(response);
  });
  it.each(["preview", "authorize"] as const)("accepts a maximum callback and percent-encoded state during %s", async (stage) => {
    const prefix = "https://assistant.example/callback?existing=";
    const redirect_uri = prefix + "a".repeat(2048 - prefix.length);
    const state = "&".repeat(2048);
    const url = new URL(redirect_uri);
    url.searchParams.set("code", "test-only-code");
    url.searchParams.set("state", state);
    url.searchParams.set("iss", "https://cursopus.com");
    const response = { redirect_to: url.href };
    expect(response.redirect_to.length).toBeGreaterThan(8192);
    client.get.mockResolvedValue(response);
    client.post.mockResolvedValue(response);
    const result = stage === "preview"
      ? await oauthService.preview(`?${new URLSearchParams({ ...params, redirect_uri, state })}`)
      : await oauthService.authorize({ ...params, redirect_uri, state }, "allow");
    expect(result).toEqual(response);
  });
  it.each(["javascript:alert(1)", "data:text/html,test", "http://evil.example/callback", "https://evil.example/callback", "https://assistant.example/other", "https://user:pass@assistant.example/callback"])("rejects unsafe or unrelated callback %s without disclosing its contents", async (redirect_to) => {
    client.post.mockResolvedValue({ redirect_to });
    await expect(oauthService.authorize(params, "allow")).rejects.toMatchObject({ code: "INVALID_RESPONSE", message: "" });
  });
  it.each(["http://127.0.0.1:5000/cb", "http://localhost:5000/cb", "http://[::1]:5000/cb"])("accepts loopback callback %s", async (redirect_uri) => {
    client.post.mockResolvedValue({ redirect_to: `${redirect_uri}?code=test` });
    expect(await oauthService.authorize({ redirect_uri }, "allow")).toEqual({ redirect_to: `${redirect_uri}?code=test` });
  });
  it.each(["?client_id=a&client_id=b", "?state=a&state=b", `?state=${"a".repeat(16_384)}`])("rejects ambiguous/oversized input before fetching", async (query) => {
    await expect(oauthService.preview(query)).rejects.toMatchObject({ code: "INVALID_REQUEST" });
    expect(client.get).not.toHaveBeenCalled();
  });
  it("leaves protocol validation to the server so it can return trusted errors", () => {
    expect(readAuthorizationParams("?response_type=unsupported&scope=other&resource=wrong&unknown=ignored")).toEqual({ response_type: "unsupported", scope: "other", resource: "wrong" });
  });
  it.each([{}, { ...preview, plan_allowed: "yes" }, { ...preview, scope: "admin" }])("rejects malformed preview responses", async (value) => {
    client.get.mockResolvedValue(value);
    await expect(oauthService.preview(search)).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
  it("lists validated grants and revokes a UUID", async () => {
    const options = { signal: new AbortController().signal };
    client.get.mockResolvedValue([{ ...grant, access_token: "discard-me" }]);
    expect(await oauthService.grants(options)).toEqual([grant]);
    expect(client.get).toHaveBeenCalledWith("oauth/grants", options);
    await oauthService.revoke(grant.id);
    expect(client.delete).toHaveBeenCalledWith(`oauth/grants/${grant.id}`);
  });
  it.each([100, 101])("counts %i supplementary Unicode characters in preview names", async (count) => {
    client.get.mockResolvedValue({ ...preview, client: { ...preview.client, name: "😀".repeat(count) } });
    if (count === 100) expect(await oauthService.preview(search)).toMatchObject({ client: { name: "😀".repeat(count) } });
    else await expect(oauthService.preview(search)).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
  });
  it.each([100, 101])("counts %i supplementary Unicode characters in grant names", async (count) => {
    client.get.mockResolvedValue([{ ...grant, client_name: "😀".repeat(count) }]);
    expect(await oauthService.grants()).toHaveLength(count === 100 ? 1 : 0);
  });
  it("keeps valid grants available when another row is malformed", async () => {
    client.get.mockResolvedValue([{ ...grant, created_at: "invalid" }, grant, null]);
    expect(await oauthService.grants()).toEqual([grant]);
  });
  it("rejects malformed list envelopes and revoke IDs", async () => {
    client.get.mockResolvedValue({ grants: [grant] });
    await expect(oauthService.grants()).rejects.toMatchObject({ code: "INVALID_RESPONSE" });
    await expect(oauthService.revoke("../other")).rejects.toMatchObject({ code: "INVALID_REQUEST" });
    expect(client.delete).not.toHaveBeenCalled();
  });
});
