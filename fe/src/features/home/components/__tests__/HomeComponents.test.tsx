import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { FeaturesSection } from "../FeaturesSection";
import { HowItWorksSection } from "../HowItWorksSection";
import { FooterSection } from "../FooterSection";
import { FooterCtaSection } from "../FooterCtaSection";
import { SocialProofBar } from "../SocialProofBar";
import { AiHighlightSection } from "../AiHighlightSection";
import { JsonLd } from "../JsonLd";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

// className is forwarded: the footer's tap-target sizing lives on it, and a
// mock that swallowed it would make those assertions vacuous.
vi.mock("react-router-dom", () => ({
  Link: ({
    children,
    to,
    className,
  }: {
    children: React.ReactNode;
    to: string;
    className?: string;
  }) => (
    <a href={to} className={className}>
      {children}
    </a>
  ),
}));

// ---------- FeaturesSection ----------
describe("FeaturesSection", () => {
  it("renders section title and subtitle", () => {
    render(<FeaturesSection />);
    expect(screen.getByText("home.features.title")).toBeInTheDocument();
    expect(screen.getByText("home.features.subtitle")).toBeInTheDocument();
  });

  it("renders label", () => {
    render(<FeaturesSection />);
    expect(screen.getByText("home.features.label")).toBeInTheDocument();
  });

  it("renders all 6 feature cards", () => {
    render(<FeaturesSection />);
    expect(screen.getByText("home.features.kanban.title")).toBeInTheDocument();
    expect(screen.getByText("home.features.aiMatch.title")).toBeInTheDocument();
    expect(screen.getByText("home.features.resume.title")).toBeInTheDocument();
    expect(
      screen.getByText("home.features.jobImport.title"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("home.features.analyticsCard.title"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("home.features.calendar.title"),
    ).toBeInTheDocument();
  });

  it("renders feature descriptions", () => {
    render(<FeaturesSection />);
    expect(
      screen.getByText("home.features.kanban.description"),
    ).toBeInTheDocument();
  });
});

// ---------- HowItWorksSection ----------
describe("HowItWorksSection", () => {
  it("renders section title", () => {
    render(<HowItWorksSection />);
    expect(screen.getByText("home.howItWorks.title")).toBeInTheDocument();
  });

  it("renders label", () => {
    render(<HowItWorksSection />);
    expect(screen.getByText("home.howItWorks.label")).toBeInTheDocument();
  });

  it("renders 4 steps", () => {
    render(<HowItWorksSection />);
    expect(screen.getByText("home.howItWorks.step1.title")).toBeInTheDocument();
    expect(screen.getByText("home.howItWorks.step2.title")).toBeInTheDocument();
    expect(screen.getByText("home.howItWorks.step3.title")).toBeInTheDocument();
    expect(screen.getByText("home.howItWorks.step4.title")).toBeInTheDocument();
  });

  it("renders step numbers", () => {
    render(<HowItWorksSection />);
    expect(screen.getByText("01")).toBeInTheDocument();
    expect(screen.getByText("04")).toBeInTheDocument();
  });
});

// ---------- FooterSection ----------
describe("FooterSection", () => {
  it("renders brand name", () => {
    render(<FooterSection />);
    expect(screen.getByText("Jobber")).toBeInTheDocument();
  });

  it("renders footer links", () => {
    render(<FooterSection />);
    expect(screen.getByText("home.footer.privacy")).toBeInTheDocument();
    expect(screen.getByText("home.footer.terms")).toBeInTheDocument();
    expect(screen.getByText("home.footer.refund")).toBeInTheDocument();
  });

  it("renders copyright", () => {
    render(<FooterSection />);
    expect(screen.getByText(/home\.footer\.copyright/)).toBeInTheDocument();
  });

  it("links to correct pages", () => {
    render(<FooterSection />);
    expect(
      screen.getByText("home.footer.privacy").closest("a"),
    ).toHaveAttribute("href", "/privacy");
    expect(screen.getByText("home.footer.terms").closest("a")).toHaveAttribute(
      "href",
      "/terms",
    );
    expect(screen.getByText("home.footer.refund").closest("a")).toHaveAttribute(
      "href",
      "/refund",
    );
  });

  it("carries the FluxLab attribution as a safe external link", () => {
    render(<FooterSection />);
    const link = screen.getByText("Powered by FluxLab").closest("a");

    expect(link).toHaveAttribute("href", "https://flux-lab.dev/en");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link?.getAttribute("rel")).toContain("noopener");
    expect(link?.getAttribute("rel")).toContain("noreferrer");
  });

  // The FAQ lives at /#faq on the landing page; the footer is one of the two
  // places that makes it findable without scrolling the whole page.
  it("links to the FAQ anchor", () => {
    render(<FooterSection />);
    expect(screen.getByText("home.nav.faq").closest("a")).toHaveAttribute(
      "href",
      "/#faq",
    );
  });

  /**
   * 13px text with no padding is an 18px tap target, well under the 44px WCAG
   * 2.5.5 asks for — and these sit directly above the consent banner, where a
   * mis-tap is easiest.
   */
  it.each([
    "home.nav.faq",
    "home.footer.privacy",
    "home.footer.terms",
    "home.footer.refund",
    "cookieConsent.settings",
  ])("gives %s a 44px minimum on phones", (label) => {
    render(<FooterSection />);

    expect(screen.getByText(label).className).toContain("min-h-11");
  });

  it("drops back to plain inline text from sm up", () => {
    render(<FooterSection />);

    expect(screen.getByText("home.footer.privacy").className).toContain(
      "sm:min-h-0",
    );
  });

  /**
   * Height alone was not enough. The earlier pass gave these links `min-h-11`
   * and left the width at whatever the word happened to be: "FAQ" measured
   * 24x44 and "Terms" 36x44, both short in the axis nobody checked.
   */
  it.each([
    "home.nav.faq",
    "home.footer.privacy",
    "home.footer.terms",
    "home.footer.refund",
    "cookieConsent.settings",
  ])("gives %s 44px of width as well as height on phones", (label) => {
    render(<FooterSection />);
    const link = screen.getByText(label);

    expect(link.className).toContain("min-w-11");
    expect(link.className).toContain("justify-center");
    expect(link.className).toContain("sm:min-w-0");
  });

  // 119x20 as measured: wide enough already, half the height a thumb needs.
  it("gives the FluxLab attribution a 44px minimum on phones", () => {
    render(<FooterSection />);
    const link = screen.getByText("Powered by FluxLab");

    expect(link.className).toContain("min-h-11");
    expect(link.className).toContain("sm:min-h-0");
  });

  // A tighter horizontal gap below `sm` is what keeps five 44px rows wrapping
  // inside a 375px viewport instead of overflowing it.
  it("tightens the row gap on the narrowest screens", () => {
    render(<FooterSection />);
    const row = screen.getByText("home.footer.privacy").parentElement;

    expect(row?.className).toContain("gap-x-4");
    expect(row?.className).toContain("sm:gap-x-5");
  });
});

// ---------- FooterCtaSection ----------
describe("FooterCtaSection", () => {
  it("renders title and subtitle", () => {
    render(
      <FooterCtaSection
        isAuthenticated={false}
        onRegister={vi.fn()}
        onGoPlatform={vi.fn()}
      />,
    );
    expect(screen.getByText("home.cta.title")).toBeInTheDocument();
    expect(screen.getByText("home.cta.subtitle")).toBeInTheDocument();
  });

  it("calls onRegister when not authenticated", () => {
    const onRegister = vi.fn();
    render(
      <FooterCtaSection
        isAuthenticated={false}
        onRegister={onRegister}
        onGoPlatform={vi.fn()}
      />,
    );
    const btn = screen.getByRole("button");
    fireEvent.click(btn);
    expect(onRegister).toHaveBeenCalledOnce();
  });

  it("calls onGoPlatform when authenticated", () => {
    const onGoPlatform = vi.fn();
    render(
      <FooterCtaSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={onGoPlatform}
      />,
    );
    const btn = screen.getByRole("button");
    fireEvent.click(btn);
    expect(onGoPlatform).toHaveBeenCalledOnce();
  });
});

// ---------- SocialProofBar ----------
describe("SocialProofBar", () => {
  it("renders label", () => {
    render(<SocialProofBar />);
    expect(screen.getByText(/home.socialProof.label/)).toBeInTheDocument();
  });

  it("renders company names twice for the seamless marquee loop", () => {
    render(<SocialProofBar />);
    for (const name of ["Google", "Meta", "Stripe", "Vercel", "Notion"]) {
      expect(screen.getAllByText(name)).toHaveLength(2);
    }
  });
});

// ---------- AiHighlightSection ----------
describe("AiHighlightSection", () => {
  const defaultProps = {
    isAuthenticated: false,
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("renders title and description", () => {
    render(<AiHighlightSection {...defaultProps} />);
    expect(screen.getByText("home.ai.title")).toBeInTheDocument();
    expect(screen.getByText("home.ai.description")).toBeInTheDocument();
  });

  it("renders label", () => {
    render(<AiHighlightSection {...defaultProps} />);
    expect(screen.getByText("home.ai.label")).toBeInTheDocument();
  });

  it("renders score card with 92%", () => {
    render(<AiHighlightSection {...defaultProps} />);
    expect(screen.getByText("92%")).toBeInTheDocument();
  });

  it("renders score bars with progress", () => {
    render(<AiHighlightSection {...defaultProps} />);
    const progressBars = screen.getAllByRole("progressbar");
    expect(progressBars.length).toBe(5);
    expect(progressBars[0]).toHaveAttribute("aria-valuenow", "95");
  });

  it("renders missing keywords", () => {
    render(<AiHighlightSection {...defaultProps} />);
    expect(screen.getByText("MySQL")).toBeInTheDocument();
    expect(screen.getByText("Vitess")).toBeInTheDocument();
    expect(screen.getByText("distributed SQL")).toBeInTheDocument();
  });

  it("calls onRegister when not authenticated", () => {
    const onRegister = vi.fn();
    render(
      <AiHighlightSection
        isAuthenticated={false}
        onRegister={onRegister}
        onGoPlatform={vi.fn()}
      />,
    );
    const btn = screen.getByRole("button");
    fireEvent.click(btn);
    expect(onRegister).toHaveBeenCalledOnce();
  });

  it("calls onGoPlatform when authenticated", () => {
    const onGoPlatform = vi.fn();
    render(
      <AiHighlightSection
        isAuthenticated={true}
        onRegister={vi.fn()}
        onGoPlatform={onGoPlatform}
      />,
    );
    const btn = screen.getByRole("button");
    fireEvent.click(btn);
    expect(onGoPlatform).toHaveBeenCalledOnce();
  });
});

// ---------- JsonLd ----------
describe("JsonLd", () => {
  it("renders null (no visible UI)", () => {
    const { container } = render(<JsonLd />);
    expect(container.innerHTML).toBe("");
  });

  it("injects script tag in document head", () => {
    render(<JsonLd />);
    const script = document.getElementById("jobber-jsonld");
    expect(script).toBeTruthy();
    expect(script?.getAttribute("type")).toBe("application/ld+json");
  });
});
