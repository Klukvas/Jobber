import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "@/services/api";
import { AIAssistantPanel } from "./AIAssistantPanel";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { createMockResume } from "./__tests__/testHelpers";
const api = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
vi.mock("@/services/api", async (importOriginal) => ({
  ...await importOriginal<typeof import("@/services/api")>(), apiClient: api,
}));
vi.mock("@/shared/hooks/useSubscription", () => ({ useSubscription: () => ({ canCreate: () => true }) }));
vi.mock("../lib/resumeDocumentSync", () => ({ flushResumeDocument: () => Promise.resolve() }));
function showPanel() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><AIAssistantPanel /></QueryClientProvider>);
  return client;
}
beforeEach(() => {
  api.get.mockReset(); api.post.mockReset();
  useResumeBuilderStore.getState().setResume(createMockResume());
});
describe("runtime AI availability", () => {
  it("known unavailable explains before submit and preserves editable input", async () => {
    api.get.mockResolvedValue({ available: false });
    showPanel();
    expect(await screen.findByText("ai.unavailable")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "resumeBuilder.ai.suggestSummary" })).toBeDisabled();
    await userEvent.type(screen.getByRole("textbox"), "Synthetic text retained");
    const improve = screen.getByRole("button", { name: "resumeBuilder.ai.improveText" });
    expect(improve).toBeDisabled();
    await userEvent.click(improve);
    expect(api.post).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox")).toHaveValue("Synthetic text retained");
  });
  it.each([
    [503, "AI_UNAVAILABLE", "ai.unavailable"],
    [500, "INTERNAL_ERROR", "ai.requestFailed"],
  ])("an available action receiving %s retains input with safe localized feedback", async (status, code, expected) => {
    api.get.mockResolvedValue({ available: true });
    api.post.mockRejectedValue(new ApiError("provider-private-details", code, status));
    showPanel();
    await userEvent.type(screen.getByRole("textbox"), "Keep synthetic draft");
    const improve = screen.getByRole("button", { name: "resumeBuilder.ai.improveText" });
    await waitFor(() => expect(improve).toBeEnabled());
    await userEvent.click(improve);
    expect((await screen.findAllByText(expected)).length).toBeGreaterThan(0);
    if (status === 503) expect(improve).toBeDisabled();
    else expect(improve).toBeEnabled();
    expect(screen.queryByText("provider-private-details")).not.toBeInTheDocument();
    expect(screen.getByRole("textbox")).toHaveValue("Keep synthetic draft");
  });
  it("a failed availability probe remains unknown and can recover on refetch", async () => {
    api.get.mockRejectedValueOnce(new Error("offline"));
    const client = showPanel();
    const summary = screen.getByRole("button", { name: "resumeBuilder.ai.suggestSummary" });
    await waitFor(() => expect(summary).toBeEnabled());
    expect(screen.queryByText("ai.unavailable")).not.toBeInTheDocument();
    api.get.mockResolvedValue({ available: false });
    await client.invalidateQueries({ queryKey: ["features", "ai"] });
    expect(await screen.findByText("ai.unavailable")).toBeInTheDocument();
    expect(summary).toBeDisabled();
    api.get.mockResolvedValue({ available: true });
    await client.invalidateQueries({ queryKey: ["features", "ai"] });
    await waitFor(() => expect(summary).toBeEnabled());
  });
});
