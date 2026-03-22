import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { render } from "@testing-library/react";
import { PreviewPanel } from "./PreviewPanel";
import {
  createMockResume,
  createMockStoreState,
} from "../__tests__/testHelpers";

beforeAll(() => {
  global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
});

const mockState = createMockStoreState();

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/resumeBuilderStore", () => ({
  useResumeBuilderStore: (selector: (state: typeof mockState) => unknown) =>
    selector(mockState),
}));

// Mock template registry and components to avoid deep rendering
vi.mock("../../lib/templateRegistry", () => ({
  TEMPLATE_MAP: {
    "00000000-0000-0000-0000-000000000001": () => (
      <div data-testid="template-preview">Template</div>
    ),
  },
}));

vi.mock("./ProfessionalTemplate", () => ({
  ProfessionalTemplate: () => (
    <div data-testid="professional-template">Professional</div>
  ),
}));

describe("PreviewPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
  });

  it("renders nothing when resume is null", () => {
    Object.assign(mockState, { resume: null });
    const { container } = render(<PreviewPanel />);
    expect(container.innerHTML).toBe("");
  });

  it("renders the preview container when resume is provided", () => {
    const { container } = render(<PreviewPanel />);
    expect(container.innerHTML).not.toBe("");
  });

  it("renders the template component", () => {
    render(<PreviewPanel />);
    // The template is rendered based on the template_id in the resume
    const templateEl = document.querySelector("[data-testid='template-preview']");
    expect(templateEl).toBeInTheDocument();
  });

  it("renders with editable=false by default", () => {
    const { container } = render(<PreviewPanel />);
    // Just verify it renders without crash
    expect(container.querySelector("div")).toBeInTheDocument();
  });

  it("renders with editable=true", () => {
    const { container } = render(<PreviewPanel editable />);
    expect(container.querySelector("div")).toBeInTheDocument();
  });
});
