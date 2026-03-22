import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PricingSection } from "../PricingSection";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

describe("PricingSection", () => {
  const defaultProps = {
    isAuthenticated: false,
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("renders section title and subtitle", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.title")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.subtitle")).toBeInTheDocument();
  });

  it("renders three pricing tiers", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.free")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.pro")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.enterprise")).toBeInTheDocument();
  });

  it("renders prices", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("$0")).toBeInTheDocument();
    // Pro price "7" and Enterprise price "19" are text nodes
    expect(screen.getByText("7")).toBeInTheDocument();
    expect(screen.getByText("19")).toBeInTheDocument();
  });

  it("renders pro badge", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.proBadge")).toBeInTheDocument();
  });

  it("renders feature lists for each tier", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.freeFeature1")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.proFeature1")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.enterpriseFeature1")).toBeInTheDocument();
  });

  it("calls onRegister when not authenticated and CTA clicked", () => {
    const onRegister = vi.fn();
    render(
      <PricingSection
        isAuthenticated={false}
        onRegister={onRegister}
        onGoPlatform={vi.fn()}
      />,
    );
    const freeCta = screen.getByText("home.pricing.freeCta");
    fireEvent.click(freeCta);
    expect(onRegister).toHaveBeenCalledOnce();
  });

  it("calls onGoPlatform when authenticated and CTA clicked", () => {
    const onGoPlatform = vi.fn();
    render(
      <PricingSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={onGoPlatform}
      />,
    );
    const freeCta = screen.getByText("home.pricing.freeCta");
    fireEvent.click(freeCta);
    expect(onGoPlatform).toHaveBeenCalledOnce();
  });

  it("renders comparison note", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.comparison")).toBeInTheDocument();
  });
});
