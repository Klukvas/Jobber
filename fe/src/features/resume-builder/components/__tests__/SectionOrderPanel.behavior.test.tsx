import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SectionOrderPanel } from "../SectionOrderPanel";
import { createMockStoreState } from "./testHelpers";
import type { SectionOrderDTO } from "@/shared/types/resume-builder";

const mockSetSectionOrder = vi.hoisted(() => vi.fn());
const mockState = createMockStoreState();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", changeLanguage: vi.fn() },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

vi.mock("../../constants/sectionLabels", () => ({
  SECTION_LABEL_KEYS: {
    contact: "label.contact",
    summary: "label.summary",
    experience: "label.experience",
  },
}));

const order: SectionOrderDTO[] = [
  { section_key: "contact", sort_order: 0, is_visible: true, column: "main" },
  { section_key: "summary", sort_order: 1, is_visible: true, column: "main" },
  {
    section_key: "experience",
    sort_order: 2,
    is_visible: false,
    column: "main",
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  Object.assign(
    mockState,
    createMockStoreState({ section_order: order.map((o) => ({ ...o })) }),
  );
  mockState.setSectionOrder = mockSetSectionOrder;
});

describe("SectionOrderPanel reordering behavior", () => {
  it("moves a section up by swapping sort orders", async () => {
    const user = userEvent.setup();
    render(<SectionOrderPanel />);

    // move "summary" (index 1) up
    await user.click(screen.getAllByLabelText("Move up")[1]);

    expect(mockSetSectionOrder).toHaveBeenCalledTimes(1);
    const updated = mockSetSectionOrder.mock.calls[0][0] as SectionOrderDTO[];
    const summary = updated.find((s) => s.section_key === "summary")!;
    const contact = updated.find((s) => s.section_key === "contact")!;
    expect(summary.sort_order).toBe(0);
    expect(contact.sort_order).toBe(1);
  });

  it("moves a section down by swapping sort orders", async () => {
    const user = userEvent.setup();
    render(<SectionOrderPanel />);

    // move "contact" (index 0) down
    await user.click(screen.getAllByLabelText("Move down")[0]);

    const updated = mockSetSectionOrder.mock.calls[0][0] as SectionOrderDTO[];
    const contact = updated.find((s) => s.section_key === "contact")!;
    const summary = updated.find((s) => s.section_key === "summary")!;
    expect(contact.sort_order).toBe(1);
    expect(summary.sort_order).toBe(0);
  });

  it("does not reorder past the top (first move-up is disabled)", async () => {
    const user = userEvent.setup();
    render(<SectionOrderPanel />);
    const firstUp = screen.getAllByLabelText("Move up")[0];
    expect(firstUp).toBeDisabled();
    await user.click(firstUp).catch(() => {});
    expect(mockSetSectionOrder).not.toHaveBeenCalled();
  });

  it("toggles visibility of a section", async () => {
    const user = userEvent.setup();
    render(<SectionOrderPanel />);

    // hide the first visible section (contact)
    await user.click(screen.getAllByLabelText("Hide section")[0]);

    const updated = mockSetSectionOrder.mock.calls[0][0] as SectionOrderDTO[];
    const contact = updated.find((s) => s.section_key === "contact")!;
    expect(contact.is_visible).toBe(false);
  });

  it("shows a hidden section again when its toggle is clicked", async () => {
    const user = userEvent.setup();
    render(<SectionOrderPanel />);

    // "experience" is hidden -> has a "Show section" toggle
    await user.click(screen.getByLabelText("Show section"));

    const updated = mockSetSectionOrder.mock.calls[0][0] as SectionOrderDTO[];
    const experience = updated.find((s) => s.section_key === "experience")!;
    expect(experience.is_visible).toBe(true);
  });
});
