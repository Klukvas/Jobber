import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { StageTemplateDTO } from "@/shared/types/api";

const mockListStageTemplates = vi.hoisted(() => vi.fn());
const mockCreate = vi.hoisted(() => vi.fn());

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: mockListStageTemplates,
    create: mockCreate,
    update: vi.fn(),
    delete: vi.fn(),
    reorder: vi.fn(),
  },
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: vi.fn(),
  showErrorNotification: vi.fn(),
}));
vi.mock("@/features/stages/modals/CreateStageTemplateModal", () => ({
  CreateStageTemplateModal: () => null,
}));
vi.mock("@/features/stages/modals/EditStageTemplateModal", () => ({
  EditStageTemplateModal: () => null,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", getFixedT: () => (key: string) => key },
  }),
}));

import StageTemplates from "../StageTemplates";

function template(over: Partial<StageTemplateDTO>): StageTemplateDTO {
  return {
    id: "t1",
    name: "Wishlist",
    order: 1,
    created_at: "2026-01-01T00:00:00Z",
    ...over,
  };
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <StageTemplates />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockCreate.mockResolvedValue(template({}));
});

/**
 * Analytics reads a column's kind, so the kinds have to be right from the
 * moment a pipeline is built — and visible afterwards, because a column that
 * counts towards nothing ("Other") is otherwise indistinguishable from one
 * that counts.
 */
describe("StageTemplates — what each column means", () => {
  it("creates a recommended stage with the kind it obviously is", async () => {
    const user = userEvent.setup();
    mockListStageTemplates.mockResolvedValue({
      items: [],
      pagination: { total: 0, limit: 100, offset: 0 },
    });
    renderPage();

    await user.click(
      await screen.findByRole("button", { name: /stages\.phoneScreen/ }),
    );

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toEqual({
      name: "stages.phoneScreen",
      order: 3,
      kind: "response",
    });
  });

  it("classifies the whole recommended pipeline when it is added at once", async () => {
    const user = userEvent.setup();
    mockListStageTemplates.mockResolvedValue({
      items: [],
      pagination: { total: 0, limit: 100, offset: 0 },
    });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "stages.addAll" }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(7));
    expect(mockCreate.mock.calls.map((call) => call[0])).toEqual([
      { name: "stages.wishlist", order: 1, kind: "saved" },
      { name: "stages.applied", order: 2, kind: "applied" },
      { name: "stages.phoneScreen", order: 3, kind: "response" },
      { name: "stages.technicalInterview", order: 4, kind: "interview" },
      { name: "stages.onsiteInterview", order: 5, kind: "interview" },
      { name: "stages.offer", order: 6, kind: "offer" },
      { name: "stages.rejected", order: 7, kind: "rejected" },
    ]);
  });

  it("shows each column's kind, and Other for one that has none", async () => {
    mockListStageTemplates.mockResolvedValue({
      items: [
        template({ id: "t1", name: "Wishlist", order: 1, kind: "saved" }),
        // Written before kinds existed: no kind on the wire at all.
        template({ id: "t2", name: "Coffee chat", order: 2 }),
      ],
      pagination: { total: 2, limit: 100, offset: 0 },
    });
    renderPage();

    expect(await screen.findByText("Coffee chat")).toBeInTheDocument();
    expect(screen.getByText("stages.kinds.saved")).toBeInTheDocument();
    expect(screen.getByText("stages.kinds.other")).toBeInTheDocument();
  });
});
