import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import Home from "../Home";
import Blog from "../Blog";
import BlogPost from "../BlogPost";
import FeatureApplications from "../FeatureApplications";
import FeatureResumeBuilder from "../FeatureResumeBuilder";
import FeatureCoverLetters from "../FeatureCoverLetters";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en", getFixedT: () => (key: string) => key },
  }),
}));

vi.mock("react-router-dom", () => ({
  useNavigate: () => vi.fn(),
  useParams: () => ({}),
  useLocation: () => ({ pathname: "/", search: "", hash: "" }),
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  Navigate: () => null,
}));

vi.mock("@/shared/lib/usePageMeta", () => ({
  usePageMeta: vi.fn(),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({
      user: null,
      isAuthenticated: false,
      setAuth: vi.fn(),
      clearAuth: vi.fn(),
    }),
}));

vi.mock("@/features/auth/modals/LoginModal", () => ({
  LoginModal: () => null,
}));

vi.mock("@/features/auth/modals/RegisterModal", () => ({
  RegisterModal: () => null,
}));

vi.mock("@/features/auth/modals/ForgotPasswordModal", () => ({
  ForgotPasswordModal: () => null,
}));

vi.mock("@/features/home/components/HomeNavbar", () => ({
  HomeNavbar: () => <nav data-testid="home-navbar" />,
}));

vi.mock("@/features/home/components/JsonLd", () => ({
  JsonLd: () => null,
}));

vi.mock("@/features/home/components/HeroSection", () => ({
  HeroSection: () => <div data-testid="hero-section" />,
}));

vi.mock("@/features/home/components/SocialProofBar", () => ({
  SocialProofBar: () => null,
}));

vi.mock("@/features/home/components/FeaturesSection", () => ({
  FeaturesSection: () => null,
}));

vi.mock("@/features/home/components/HowItWorksSection", () => ({
  HowItWorksSection: () => null,
}));

vi.mock("@/features/home/components/AiHighlightSection", () => ({
  AiHighlightSection: () => null,
}));

vi.mock("@/features/home/components/PricingSection", () => ({
  PricingSection: () => null,
}));

vi.mock("@/features/home/components/FooterCtaSection", () => ({
  FooterCtaSection: () => null,
}));

vi.mock("@/features/home/components/FooterSection", () => ({
  FooterSection: () => <footer data-testid="footer-section" />,
}));

vi.mock("@/features/blog/lib/blogLoader", () => ({
  getAllPosts: () => [],
  getPostBySlug: () => undefined,
}));

vi.mock("@/features/blog/components/BlogHeader", () => ({
  BlogHeader: () => <div data-testid="blog-header" />,
}));

vi.mock("@/features/blog/components/BlogPostCard", () => ({
  BlogPostCard: () => null,
}));

vi.mock("@/features/blog/components/BlogArticle", () => ({
  BlogArticle: () => null,
}));

vi.mock("@/shared/ui/BrowserMockup", () => ({
  BrowserMockup: () => <div data-testid="browser-mockup" />,
}));

describe("Home", () => {
  it("renders without crash", () => {
    render(<Home />);
    expect(screen.getByTestId("home-navbar")).toBeInTheDocument();
  });

  it("renders hero section", () => {
    render(<Home />);
    expect(screen.getByTestId("hero-section")).toBeInTheDocument();
  });

  it("renders footer", () => {
    render(<Home />);
    expect(screen.getByTestId("footer-section")).toBeInTheDocument();
  });
});

describe("Blog", () => {
  it("renders without crash", () => {
    render(<Blog />);
    expect(screen.getByTestId("home-navbar")).toBeInTheDocument();
  });

  it("shows no posts message when empty", () => {
    render(<Blog />);
    expect(screen.getByText("blog.noPosts")).toBeInTheDocument();
  });
});

describe("BlogPost", () => {
  it("renders without crash", () => {
    render(<BlogPost />);
    expect(screen.getByTestId("home-navbar")).toBeInTheDocument();
  });

  it("shows not found when no slug matches", () => {
    render(<BlogPost />);
    expect(screen.getByText("blog.notFound")).toBeInTheDocument();
  });

  it("shows back to blog link", () => {
    render(<BlogPost />);
    expect(screen.getAllByText("blog.backToBlog").length).toBeGreaterThan(0);
  });
});

describe("FeatureApplications", () => {
  it("renders without crash", () => {
    render(<FeatureApplications />);
    expect(screen.getByTestId("home-navbar")).toBeInTheDocument();
  });

  it("shows hero title", () => {
    render(<FeatureApplications />);
    expect(
      screen.getByText("featurePages.applications.hero.title"),
    ).toBeInTheDocument();
  });

  it("shows CTA section", () => {
    render(<FeatureApplications />);
    expect(
      screen.getByText("featurePages.applications.cta.title"),
    ).toBeInTheDocument();
  });
});

describe("FeatureResumeBuilder", () => {
  it("renders without crash", () => {
    render(<FeatureResumeBuilder />);
    expect(screen.getByTestId("home-navbar")).toBeInTheDocument();
  });

  it("shows hero title", () => {
    render(<FeatureResumeBuilder />);
    expect(
      screen.getByText("featurePages.resumeBuilder.hero.title"),
    ).toBeInTheDocument();
  });

  it("shows CTA section", () => {
    render(<FeatureResumeBuilder />);
    expect(
      screen.getByText("featurePages.resumeBuilder.cta.title"),
    ).toBeInTheDocument();
  });
});

describe("FeatureCoverLetters", () => {
  it("renders without crash", () => {
    render(<FeatureCoverLetters />);
    expect(screen.getByTestId("home-navbar")).toBeInTheDocument();
  });

  it("shows hero title", () => {
    render(<FeatureCoverLetters />);
    expect(
      screen.getByText("featurePages.coverLetters.hero.title"),
    ).toBeInTheDocument();
  });

  it("shows CTA section", () => {
    render(<FeatureCoverLetters />);
    expect(
      screen.getByText("featurePages.coverLetters.cta.title"),
    ).toBeInTheDocument();
  });
});
