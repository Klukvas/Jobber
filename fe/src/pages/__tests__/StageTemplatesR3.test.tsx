import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, expect, it, vi } from "vitest";
import StageTemplates from "../StageTemplates";

const remove = vi.hoisted(() => vi.fn());
const notify = vi.hoisted(() => vi.fn());
vi.mock("@/services/stageTemplatesService", () => ({ stageTemplatesService: {
  list: async () => ({ items: [{ id: "t1", name: "Custom", order: 0, created_at: "2026-01-01" }], total: 1 }),
  delete: remove, create: vi.fn(), reorder: vi.fn(),
} }));
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en", getFixedT: () => (key: string) => key } }) }));
vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/notifications", () => ({ showSuccessNotification: notify, showErrorNotification: vi.fn() }));
vi.mock("@/features/stages/modals/CreateStageTemplateModal", () => ({ CreateStageTemplateModal: () => null }));
vi.mock("@/features/stages/modals/EditStageTemplateModal", () => ({ EditStageTemplateModal: () => null }));
beforeEach(() => vi.clearAllMocks());
it("stage deletion makes one request for same-tick clicks, preserves the dialog on failure, and retries", async () => {
  let fail: (error: Error) => void = () => {};
  remove.mockReturnValueOnce(new Promise((_resolve, reject) => { fail = reject; })).mockResolvedValueOnce(undefined);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  render(<QueryClientProvider client={client}><StageTemplates /></QueryClientProvider>);
  await screen.findByText("Custom");
  fireEvent.click(screen.getByRole("button", { name: "common.delete" }));
  const confirm = screen.getByRole("dialog").querySelector<HTMLButtonElement>('button[class*="bg-destructive"]');
  expect(confirm).not.toBeNull();
  if (!confirm) return;
  act(() => { fireEvent.click(confirm); fireEvent.click(confirm); });
  await waitFor(() => expect(remove).toHaveBeenCalledTimes(1));
  await act(async () => fail(new Error("try again")));
  await waitFor(() => expect(confirm).toBeEnabled());
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  fireEvent.click(confirm);
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  expect(remove).toHaveBeenCalledTimes(2);
  expect(notify).toHaveBeenCalledExactlyOnceWith("stages.deleteSuccess");
});
