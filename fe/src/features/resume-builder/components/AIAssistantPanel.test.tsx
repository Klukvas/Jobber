import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { AIAssistantPanel } from "./AIAssistantPanel";
import {
  createMockResume,
  createMockStoreState,
} from "./__tests__/testHelpers";

const mockSuggestSummary = {
  mutate: vi.fn(),
  isPending: false,
  isError: false,
  data: null as null | { summary: string },
  reset: vi.fn(),
};

const mockSuggestBullets = {
  mutate: vi.fn(),
  isPending: false,
  isError: false,
  data: null as null | { bullets: string[] },
  reset: vi.fn(),
};

const mockImproveText = {
  mutate: vi.fn(),
  isPending: false,
  isError: false,
  data: null as null | { improved_text: string },
  reset: vi.fn(),
};

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

vi.mock("../hooks/useAISuggestions", () => ({
  useAISuggestions: () => ({
    suggestSummary: mockSuggestSummary,
    suggestBullets: mockSuggestBullets,
    improveText: mockImproveText,
  }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    canCreate: () => true,
  }),
}));

vi.mock("@/features/subscription/components/UpgradeBanner", () => ({
  UpgradeBanner: () => <div data-testid="upgrade-banner">Upgrade</div>,
}));

describe("AIAssistantPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const freshState = createMockStoreState();
    Object.assign(mockState, freshState);
    Object.assign(mockSuggestSummary, {
      mutate: vi.fn(),
      isPending: false,
      isError: false,
      data: null,
      reset: vi.fn(),
    });
    Object.assign(mockSuggestBullets, {
      mutate: vi.fn(),
      isPending: false,
      isError: false,
      data: null,
      reset: vi.fn(),
    });
    Object.assign(mockImproveText, {
      mutate: vi.fn(),
      isPending: false,
      isError: false,
      data: null,
      reset: vi.fn(),
    });
  });

  it("renders the AI title heading", () => {
    render(<AIAssistantPanel />);
    expect(screen.getByText("resumeBuilder.ai.title")).toBeInTheDocument();
  });

  it("renders the suggest summary button", () => {
    render(<AIAssistantPanel />);
    expect(
      screen.getByText("resumeBuilder.ai.suggestSummary"),
    ).toBeInTheDocument();
  });

  it("renders the improve text section", () => {
    render(<AIAssistantPanel />);
    // The label and the button both contain this text
    const elements = screen.getAllByText("resumeBuilder.ai.improveText");
    expect(elements.length).toBeGreaterThanOrEqual(1);
  });

  it("renders instruction buttons", () => {
    render(<AIAssistantPanel />);
    expect(
      screen.getByText("resumeBuilder.ai.instructions.concise"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.ai.instructions.metrics"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.ai.instructions.professional"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("resumeBuilder.ai.instructions.action_verbs"),
    ).toBeInTheDocument();
  });

  it("does not render upgrade banner when AI limit is not reached", () => {
    render(<AIAssistantPanel />);
    expect(screen.queryByTestId("upgrade-banner")).not.toBeInTheDocument();
  });

  it("does not show suggest bullets section when there are no experiences", () => {
    render(<AIAssistantPanel />);
    expect(
      screen.queryByText("resumeBuilder.ai.selectExperience"),
    ).not.toBeInTheDocument();
  });

  it("shows suggest bullets section when there are experiences", () => {
    Object.assign(mockState, {
      resume: createMockResume({
        experiences: [
          {
            id: "exp-1",
            position: "Developer",
            company: "Acme",
            location: "",
            description: "",
            start_date: "2023-01",
            end_date: "",
            is_current: false,
            sort_order: 0,
          },
        ],
      }),
    });
    render(<AIAssistantPanel />);
    expect(
      screen.getByText("resumeBuilder.ai.selectExperience"),
    ).toBeInTheDocument();
  });
});
