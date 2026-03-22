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

vi.mock("react-router-dom", () => ({
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
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

  it("renders company names", () => {
    render(<SocialProofBar />);
    expect(screen.getByText("Google")).toBeInTheDocument();
    expect(screen.getByText("Meta")).toBeInTheDocument();
    expect(screen.getByText("Stripe")).toBeInTheDocument();
    expect(screen.getByText("Vercel")).toBeInTheDocument();
    expect(screen.getByText("Notion")).toBeInTheDocument();
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
