import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PremiumGate } from "./PremiumGate";

const mockNavigate = vi.fn();
let mockIsFree = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (params?.feature) return `${key}:${params.feature}`;
      return key;
    },
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({
    isFree: mockIsFree,
    plan: mockIsFree ? "free" : "pro",
    isPro: !mockIsFree,
  }),
}));

describe("PremiumGate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockIsFree = true;
  });

  it("shows premium gate for free users", () => {
    render(
      <PremiumGate feature="AI Assistant">
        <div>Premium Content</div>
      </PremiumGate>,
    );

    expect(screen.getByText("premium.title")).toBeInTheDocument();
    expect(
      screen.getByText("premium.description:AI Assistant"),
    ).toBeInTheDocument();
    expect(screen.getByText("premium.upgrade")).toBeInTheDocument();
    expect(screen.queryByText("Premium Content")).not.toBeInTheDocument();
  });

  it("shows children for premium users", () => {
    mockIsFree = false;
    render(
      <PremiumGate feature="AI Assistant">
        <div>Premium Content</div>
      </PremiumGate>,
    );

    expect(screen.getByText("Premium Content")).toBeInTheDocument();
    expect(screen.queryByText("premium.title")).not.toBeInTheDocument();
  });

  it("navigates to settings on upgrade click", async () => {
    const user = userEvent.setup();
    render(
      <PremiumGate feature="AI Assistant">
        <div>Premium Content</div>
      </PremiumGate>,
    );

    await user.click(screen.getByText("premium.upgrade"));
    expect(mockNavigate).toHaveBeenCalledWith("/app/settings");
  });

  it("renders the lock icon for free users", () => {
    const { container } = render(
      <PremiumGate feature="test">
        <div>Content</div>
      </PremiumGate>,
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeTruthy();
  });
});
