import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import StageTemplates from "../StageTemplates";
import type { StageTemplateDTO } from "@/shared/types/api";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: {
      language: "en",
      getFixedT: () => (key: string) => key,
    },
  }),
}));

vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: vi.fn() }));
vi.mock("@/shared/lib/dateFnsLocale", () => ({ useDateLocale: () => undefined }));
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

const reorderMock = vi.fn();
vi.mock("@/services/stageTemplatesService", () => ({
  stageTemplatesService: {
    list: vi.fn(),
    create: vi.fn(),
    delete: vi.fn(),
    reorder: (...args: unknown[]) => reorderMock(...args),
  },
}));

const templates: StageTemplateDTO[] = [
  { id: "t1", name: "Wishlist", order: 1, created_at: "2026-01-01T00:00:00Z" },
  { id: "t2", name: "Applied", order: 2, created_at: "2026-01-01T00:00:00Z" },
  { id: "t3", name: "Offer", order: 3, created_at: "2026-01-01T00:00:00Z" },
];

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: { items: templates },
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useMutation: (opts: { mutationFn: (v: unknown) => unknown }) => ({
    mutate: (vars: unknown) => opts.mutationFn(vars),
    isPending: false,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

describe("StageTemplates — reorder", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the user's stages in order with reorder controls", () => {
    render(<StageTemplates />);
    expect(screen.getByText("Wishlist")).toBeInTheDocument();
    expect(screen.getByText("Applied")).toBeInTheDocument();
    expect(screen.getByText("Offer")).toBeInTheDocument();
    expect(screen.getAllByLabelText("stages.moveDown").length).toBe(3);
    expect(screen.getAllByLabelText("stages.moveUp").length).toBe(3);
  });

  it("disables move-up on the first row and move-down on the last row", () => {
    render(<StageTemplates />);
    const ups = screen.getAllByLabelText("stages.moveUp");
    const downs = screen.getAllByLabelText("stages.moveDown");
    expect(ups[0]).toBeDisabled();
    expect(downs[downs.length - 1]).toBeDisabled();
  });

  it("reorders with the full ordered id list when moving a stage down", () => {
    render(<StageTemplates />);
    // Move the first stage (Wishlist) down → swap with Applied.
    fireEvent.click(screen.getAllByLabelText("stages.moveDown")[0]);
    expect(reorderMock).toHaveBeenCalledWith(["t2", "t1", "t3"]);
  });

  it("reorders when moving a stage up", () => {
    render(<StageTemplates />);
    // Move the last stage (Offer) up → swap with Applied.
    fireEvent.click(screen.getAllByLabelText("stages.moveUp")[2]);
    expect(reorderMock).toHaveBeenCalledWith(["t1", "t3", "t2"]);
  });
});
