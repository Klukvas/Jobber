import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { HeroSection } from "../HeroSection";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

describe("HeroSection", () => {
  const defaultProps = {
    isAuthenticated: false,
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("renders headline translation keys", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.titleStart")).toBeInTheDocument();
    expect(screen.getByText("home.hero.titleAccent")).toBeInTheDocument();
  });

  it("renders subtitle", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.subtitle")).toBeInTheDocument();
  });

  it("renders badge", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.badge")).toBeInTheDocument();
  });

  it("shows register CTA when not authenticated", () => {
    const onRegister = vi.fn();
    render(
      <HeroSection
        isAuthenticated={false}
        onRegister={onRegister}
        onGoPlatform={vi.fn()}
      />,
    );
    // The primary button text is "home.hero.cta" + " →"
    const buttons = screen.getAllByRole("button");
    const primaryBtn = buttons.find(
      (btn) =>
        btn.textContent?.includes("home.hero.cta") &&
        !btn.textContent?.includes("Secondary"),
    );
    expect(primaryBtn).toBeTruthy();
    fireEvent.click(primaryBtn!);
    expect(onRegister).toHaveBeenCalledOnce();
  });

  it("shows secondary CTA when not authenticated", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("home.hero.ctaSecondary")).toBeInTheDocument();
  });

  it("shows go-to-platform CTA when authenticated", () => {
    const onGoPlatform = vi.fn();
    render(
      <HeroSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={onGoPlatform}
      />,
    );
    const btn = screen.getByText("home.hero.ctaGoPlatform");
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(onGoPlatform).toHaveBeenCalledOnce();
  });

  it("does not show register CTA when authenticated", () => {
    render(
      <HeroSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={vi.fn()}
      />,
    );
    expect(
      screen.queryByText("home.hero.ctaSecondary"),
    ).not.toBeInTheDocument();
  });

  it("renders the resume review preview instead of the board", () => {
    render(<HeroSection {...defaultProps} />);
    expect(screen.getByText("jobs.tailoring.review")).toBeInTheDocument();
    expect(screen.getByText("Staff Engineer — Vercel")).toBeInTheDocument();
    expect(screen.getAllByText("jobs.tailoring.before")).toHaveLength(2);
    expect(screen.getAllByText("jobs.tailoring.after")).toHaveLength(2);
    expect(screen.getAllByText("jobs.tailoring.reason")).toHaveLength(2);
    expect(screen.getByText("home.showcase.summaryBefore")).toBeInTheDocument();
    expect(screen.getByText("home.showcase.summaryAfter")).toBeInTheDocument();
    expect(screen.getByText("home.showcase.summaryReason")).toBeInTheDocument();
    expect(screen.queryByText("Stripe")).not.toBeInTheDocument();
    expect(screen.queryByText("home.showcase.board")).not.toBeInTheDocument();
  });

  it("shows the static review until a recording is configured", () => {
    const { container } = render(<HeroSection {...defaultProps} />);
    expect(
      container.querySelector("[data-hero-media-slot='resume-review']"),
    ).toBeTruthy();
    expect(container.querySelector("[data-hero-review='static']")).toBeTruthy();
    expect(container.querySelector("video")).toBeNull();
  });
});
