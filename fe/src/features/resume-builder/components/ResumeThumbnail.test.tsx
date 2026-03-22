import { describe, it, expect, vi, beforeEach, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { ResumeThumbnail } from "./ResumeThumbnail";

beforeAll(() => {
  global.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver;
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

const mockDataRef = {
  current: null as Record<string, unknown> | null,
};

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({
    data: mockDataRef.current,
    isLoading: !mockDataRef.current,
  }),
}));

vi.mock("@/services/resumeBuilderService", () => ({
  resumeBuilderService: {
    getById: vi.fn(),
  },
}));

vi.mock("../lib/templateRegistry", () => ({
  TEMPLATE_MAP: {
    "00000000-0000-0000-0000-000000000001": () => (
      <div data-testid="template">Template</div>
    ),
  },
}));

vi.mock("./preview/ProfessionalTemplate", () => ({
  ProfessionalTemplate: () => (
    <div data-testid="professional-template">Professional</div>
  ),
}));

vi.mock("./preview/ResumePreviewContext", () => ({
  ResumePreviewProvider: ({
    children,
  }: {
    value: unknown;
    children: React.ReactNode;
  }) => <div data-testid="preview-provider">{children}</div>,
}));

describe("ResumeThumbnail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockDataRef.current = null;
  });

  it("renders loading skeleton when data is not available", () => {
    const { container } = render(
      <ResumeThumbnail
        resumeId="resume-1"
        templateId="00000000-0000-0000-0000-000000000001"
      />,
    );
    const skeleton = container.querySelector(".animate-pulse");
    expect(skeleton).toBeInTheDocument();
  });

  it("renders the template when data is available", () => {
    mockDataRef.current = {
      id: "resume-1",
      template_id: "00000000-0000-0000-0000-000000000001",
      font_family: "Georgia",
      margin_top: 40,
      margin_right: 40,
      margin_bottom: 40,
      margin_left: 40,
      spacing: 100,
    };
    render(
      <ResumeThumbnail
        resumeId="resume-1"
        templateId="00000000-0000-0000-0000-000000000001"
      />,
    );
    expect(screen.getByTestId("template")).toBeInTheDocument();
    expect(screen.getByTestId("preview-provider")).toBeInTheDocument();
  });

  it("falls back to ProfessionalTemplate for unknown templateId", () => {
    mockDataRef.current = {
      id: "resume-1",
      template_id: "unknown-template",
      font_family: "Georgia",
      margin_top: 40,
      margin_right: 40,
      margin_bottom: 40,
      margin_left: 40,
      spacing: 100,
    };
    render(
      <ResumeThumbnail
        resumeId="resume-1"
        templateId="unknown-template"
      />,
    );
    expect(
      screen.getByTestId("professional-template"),
    ).toBeInTheDocument();
  });
});
