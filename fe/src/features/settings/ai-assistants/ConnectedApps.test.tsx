import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createInstance } from "i18next";
import { I18nextProvider } from "react-i18next";
import en from "@/shared/locales/en.json";
import { apiClient } from "@/services/api";
const service = vi.hoisted(() => ({ grants: vi.fn(), revoke: vi.fn() }));
vi.mock("@/services/oauthService", () => ({ oauthService: service }));
import { ConnectedApps } from "./ConnectedApps";
import { ConnectorUrl } from "./ConnectorUrl";
const i18n = createInstance();
await i18n.init({ lng: "en", resources: { en: { translation: en } }, interpolation: { escapeValue: false } });
const grant = { id: "11111111-1111-4111-8111-111111111111", client_name: "Assistant", client_id: "test", redirect_host: "assistant.example", created_at: "2026-01-01T00:00:00Z", last_used_at: null };
function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(<I18nextProvider i18n={i18n}><QueryClientProvider client={client}><ConnectorUrl /><ConnectedApps userId="user-1" /></QueryClientProvider></I18nextProvider>);
  return client;
}
beforeEach(() => { vi.clearAllMocks(); service.grants.mockResolvedValue([]); service.revoke.mockResolvedValue(undefined); });
afterEach(cleanup);

describe("connected apps", () => {
  it("explains tokenless connection, copies the origin MCP URL and shows empty state", async () => {
    const user = userEvent.setup();
    const copy = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
    mount();
    expect(await screen.findByText(en.oauth.grantsEmpty)).toBeInTheDocument();
    expect(screen.getByText(en.oauth.connectorDescription)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: en.oauth.copyUrl }));
    expect(copy).toHaveBeenCalledWith(`${window.location.origin}/api/mcp`);
  });
  it("shows loading and retries a failed list", async () => {
    let fail: ((error: Error) => void) | undefined;
    service.grants.mockImplementationOnce(() => new Promise((_, reject) => { fail = reject; }));
    mount();
    expect(screen.getByRole("status", { name: "Loading..." })).toBeInTheDocument();
    await act(async () => { fail?.(new Error("offline")); });
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.grantsError);
    await userEvent.click(screen.getByRole("button", { name: "Try Again" }));
    expect(await screen.findByText(en.oauth.grantsEmpty)).toBeInTheDocument();
  });
  it("keeps valid apps revocable when the actual service receives a malformed row", async () => {
    const actual = await vi.importActual<typeof import("@/services/oauthService")>("@/services/oauthService");
    const get = vi.spyOn(apiClient, "get").mockResolvedValue([
      { ...grant, id: "bad-id", client_name: "Malformed" }, grant,
    ]);
    service.grants.mockImplementation(actual.oauthService.grants);
    try {
      mount();
      await userEvent.click(await screen.findByRole("button", { name: "Disconnect Assistant" }));
      expect(screen.queryByText("Malformed")).not.toBeInTheDocument();
      await userEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Disconnect" }));
      await waitFor(() => expect(service.revoke).toHaveBeenCalledWith(grant.id, expect.anything()));
    } finally {
      get.mockRestore();
    }
  });
  it("shows dates, confirms revoke, restores cancel focus, guards double-click and focuses heading after removal", async () => {
    const user = userEvent.setup();
    service.grants.mockResolvedValueOnce([grant]).mockResolvedValue([]);
    let finish: (() => void) | undefined;
    service.revoke.mockImplementation(() => new Promise<void>((resolve) => { finish = resolve; }));
    mount();
    const trigger = await screen.findByRole("button", { name: "Disconnect Assistant" });
    expect(screen.getByText("assistant.example")).toBeInTheDocument();
    expect(screen.getByText("Jan 1, 2026")).toBeInTheDocument();
    expect(screen.getByText("Never")).toBeInTheDocument();
    await user.click(trigger);
    expect(service.revoke).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(trigger).toHaveFocus();
    await user.click(trigger);
    const dialog = screen.getByRole("dialog", { name: en.oauth.revokeTitle });
    expect(dialog).toHaveFocus();
    const confirm = within(dialog).getByRole("button", { name: "Disconnect" });
    fireEvent.click(confirm); fireEvent.click(confirm);
    await waitFor(() => expect(service.revoke).toHaveBeenCalledTimes(1));
    expect(screen.getByRole("button", { name: "Revoking…" })).toBeDisabled();
    fireEvent.keyDown(document, { key: "Escape" });
    expect(dialog).toBeInTheDocument();
    await act(async () => { finish?.(); });
    expect(await screen.findByText(en.oauth.grantsEmpty)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("heading", { name: en.oauth.connectedApps })).toHaveFocus());
  });
  it("preserves cached rows on failed refresh and keeps the selected app on revoke failure", async () => {
    const user = userEvent.setup();
    service.grants.mockResolvedValue([grant]);
    service.revoke.mockRejectedValue(new Error("offline"));
    const client = mount();
    await user.click(await screen.findByRole("button", { name: "Disconnect Assistant" }));
    await user.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Disconnect" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.revokeError);
    expect(screen.getByRole("dialog")).toHaveTextContent("Assistant");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    service.grants.mockRejectedValue(new Error("offline"));
    await act(async () => { await client.invalidateQueries({ queryKey: ["oauth-grants"] }); });
    expect(await screen.findByRole("alert")).toHaveTextContent(en.oauth.grantsError);
    expect(screen.getByText("Assistant")).toBeInTheDocument();
  });
});
