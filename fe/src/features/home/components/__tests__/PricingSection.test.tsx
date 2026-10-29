import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PricingSection } from "../PricingSection";

const featuresMock = vi.hoisted(() => ({ FEATURES: { PAYMENTS: false } }));
vi.mock("@/shared/lib/features", () => featuresMock);

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

  it("renders only Free and Pro", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.free")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.pro")).toBeInTheDocument();
    expect(screen.queryByText("home.pricing.enterprise")).not.toBeInTheDocument();
  });

  it("renders prices", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("$0")).toBeInTheDocument();
    expect(screen.getByText("24")).toBeInTheDocument();
    expect(screen.queryByText("19")).not.toBeInTheDocument();
  });

  it("renders pro badge", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.proBadge")).toBeInTheDocument();
  });

  it("renders feature lists for each tier", () => {
    render(<PricingSection {...defaultProps} />);
    expect(screen.getByText("home.pricing.freeFeature1")).toBeInTheDocument();
    expect(screen.getByText("home.pricing.proFeature1")).toBeInTheDocument();
    expect(screen.queryByText("home.pricing.enterpriseFeature1")).not.toBeInTheDocument();
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

  // A limitation dressed in the same green tick as every included feature reads
  // as a feature. The Free plan has no cover letters at all.
  it("includes the Free cover letter", () => {
    render(<PricingSection {...defaultProps} />);
    const bullet = screen
      .getByText("home.pricing.freeFeature7")
      .closest("div") as HTMLElement;

    expect(bullet.textContent).toContain("✓");
  });

  it("keeps a tick on the Free features that are actually included", () => {
    render(<PricingSection {...defaultProps} />);
    const bullet = screen
      .getByText("home.pricing.freeFeature1")
      .closest("div") as HTMLElement;

    expect(bullet.textContent).toContain("✓");
  });

  describe("paid-tier CTAs", () => {
    it("reads 'coming soon' and is disabled while payments are off", () => {
      featuresMock.FEATURES.PAYMENTS = false;
      const onRegister = vi.fn();
      render(
        <PricingSection
          isAuthenticated={false}
          onRegister={onRegister}
          onGoPlatform={vi.fn()}
        />,
      );
      const soon = screen.getAllByText("home.pricing.comingSoon");
      expect(soon).toHaveLength(1);
      soon.forEach((el) => {
        expect(el.closest("button")).toBeDisabled();
        fireEvent.click(el);
      });
      expect(onRegister).not.toHaveBeenCalled();
      expect(screen.queryByText(/home\.pricing\.proCta/)).toBeNull();
      expect(screen.queryByText("home.pricing.enterpriseCta")).toBeNull();
    });

    it("shows the plan CTAs when payments are on", () => {
      featuresMock.FEATURES.PAYMENTS = true;
      const onRegister = vi.fn();
      render(
        <PricingSection
          isAuthenticated={false}
          onRegister={onRegister}
          onGoPlatform={vi.fn()}
        />,
      );
      expect(screen.queryByText("home.pricing.comingSoon")).toBeNull();
      fireEvent.click(screen.getByText(/home\.pricing\.proCta/));
      expect(onRegister).toHaveBeenCalledOnce();
      featuresMock.FEATURES.PAYMENTS = false;
    });
  });
});
