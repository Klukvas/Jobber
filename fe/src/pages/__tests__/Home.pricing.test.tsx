import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";

import Home from "../Home";
import { CHOOSE_PLAN_PATH } from "@/features/subscription/choosePlanLink";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: unknown) => unknown) =>
    selector({ isAuthenticated: true }),
}));

vi.mock("@/shared/lib/features", () => ({ FEATURES: { PAYMENTS: true } }));
vi.mock("@/shared/lib/usePageMeta", () => ({ usePageMeta: () => undefined }));
vi.mock("@/features/auth/modals/LoginModal", () => ({ LoginModal: () => null }));
vi.mock("@/features/auth/modals/RegisterModal", () => ({
  RegisterModal: () => null,
}));
vi.mock("@/features/auth/modals/ForgotPasswordModal", () => ({
  ForgotPasswordModal: () => null,
}));
vi.mock("@/features/home/components/HomeNavbar", () => ({
  HomeNavbar: () => null,
}));
vi.mock("@/features/home/components/JsonLd", () => ({ JsonLd: () => null }));
vi.mock("@/features/home/components/HeroSection", () => ({
  HeroSection: () => null,
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
vi.mock("@/features/home/components/FaqSection", () => ({
  FaqSection: () => null,
}));
vi.mock("@/features/home/components/FooterCtaSection", () => ({
  FooterCtaSection: () => null,
}));
vi.mock("@/features/home/components/FooterSection", () => ({
  FooterSection: () => null,
}));

function CurrentUrl() {
  const { pathname, search } = useLocation();
  return <div data-testid="url">{`${pathname}${search}`}</div>;
}

describe("Home — pricing for a signed-in visitor", () => {
  it("sends Choose Pro to the plan choice", () => {
    render(
      <MemoryRouter>
        <Home />
        <CurrentUrl />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByText(/home\.pricing\.proCta/));

    expect(screen.getByTestId("url")).toHaveTextContent(CHOOSE_PLAN_PATH);
  });

  it("still sends the Free CTA to the dashboard", () => {
    render(
      <MemoryRouter>
        <Home />
        <CurrentUrl />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByText("home.pricing.freeCta"));

    expect(screen.getByTestId("url")).toHaveTextContent("/app/jobs");
  });
});
