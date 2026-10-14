import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { StageTemplateDTO } from "@/shared/types/api";

const mockListStageTemplates = vi.hoisted(() => vi.fn());
const mockCreate = vi.hoisted(() => vi.fn());
const mockDelete = vi.hoisted(() => vi.fn());
const mockShowSuccess = vi.hoisted(() => vi.fn());

vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: mockListStageTemplates,
    create: mockCreate,
    update: vi.fn(),
    delete: mockDelete,
    reorder: vi.fn(),
  },
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/dateFnsLocale", () => ({
  useDateLocale: () => undefined,
}));
vi.mock("@/shared/lib/notifications", () => ({
  showSuccessNotification: mockShowSuccess,
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
      order: 0,
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
      { name: "stages.wishlist", order: 0, kind: "saved" },
      { name: "stages.applied", order: 1, kind: "applied" },
      { name: "stages.phoneScreen", order: 2, kind: "response" },
      { name: "stages.technicalInterview", order: 3, kind: "interview" },
      { name: "stages.onsiteInterview", order: 4, kind: "interview" },
      { name: "stages.offer", order: 5, kind: "offer" },
      { name: "stages.rejected", order: 6, kind: "rejected" },
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

/**
 * A new account is seeded with 7 stage templates at orders 0..6. Anything the
 * Stages page creates must continue after them rather than reuse 1..7.
 */
describe("StageTemplates — order of created stages", () => {
  const seeded = Array.from({ length: 7 }, (_, i) =>
    template({ id: `seed-${i}`, name: `Seeded ${i}`, order: i }),
  );

  it("appends a recommended stage after the highest existing order", async () => {
    const user = userEvent.setup();
    mockListStageTemplates.mockResolvedValue({
      items: seeded,
      pagination: { total: 7, limit: 100, offset: 0 },
    });
    renderPage();

    await user.click(
      await screen.findByRole("button", { name: /stages\.phoneScreen/ }),
    );

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(1));
    expect(mockCreate.mock.calls[0][0]).toMatchObject({
      name: "stages.phoneScreen",
      order: 7,
    });
  });

  it("Add All numbers the missing stages sequentially, skipping existing ones", async () => {
    const user = userEvent.setup();
    mockListStageTemplates.mockResolvedValue({
      items: [
        template({ id: "a", name: "stages.wishlist", order: 0 }),
        template({ id: "b", name: "stages.offer", order: 4 }),
      ],
      pagination: { total: 2, limit: 100, offset: 0 },
    });
    renderPage();

    await user.click(await screen.findByRole("button", { name: "stages.addAll" }));

    await waitFor(() => expect(mockCreate).toHaveBeenCalledTimes(5));
    expect(
      mockCreate.mock.calls.map((call) => [call[0].name, call[0].order]),
    ).toEqual([
      ["stages.applied", 5],
      ["stages.phoneScreen", 6],
      ["stages.technicalInterview", 7],
      ["stages.onsiteInterview", 8],
      ["stages.rejected", 9],
    ]);
  });
});

describe("StageTemplates — deleting", () => {
  it("shows a success toast once the stage is deleted", async () => {
    const user = userEvent.setup();
    mockListStageTemplates.mockResolvedValue({
      items: [template({ id: "t1", name: "Wishlist", order: 0 })],
      pagination: { total: 1, limit: 100, offset: 0 },
    });
    mockDelete.mockResolvedValue(undefined);
    renderPage();

    await user.click(await screen.findByRole("button", { name: "common.delete" }));
    // The confirm dialog's button shares the row button's accessible name.
    const buttons = await screen.findAllByRole("button", { name: "common.delete" });
    await user.click(buttons[buttons.length - 1]);

    await waitFor(() => expect(mockDelete).toHaveBeenCalled());
    await waitFor(() =>
      expect(mockShowSuccess).toHaveBeenCalledWith("stages.deleteSuccess"),
    );
  });
});
