import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AppLayout } from "../AppLayout";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ plan: "free" }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: unknown) => unknown) =>
    selector({ isAuthenticated: true }),
}));

vi.mock("@/features/onboarding/useOnboarding", () => ({
  useOnboarding: () => ({ shouldShow: false, complete: vi.fn() }),
}));
vi.mock("@/features/onboarding/WelcomeWizard", () => ({
  WelcomeWizard: () => null,
}));
vi.mock("@/widgets/Sidebar", () => ({ Sidebar: () => null }));
vi.mock("@/widgets/Header", () => ({ Header: () => null }));
vi.mock("@/features/support/SupportButton", () => ({
  SupportButton: () => null,
}));
vi.mock("@/features/subscription/components/SubscriptionSuccessModal", () => ({
  SubscriptionSuccessModal: () => null,
}));

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={["/app/jobs"]}>
      <AppLayout />
    </MemoryRouter>,
  );
}

/**
 * The in-app footer carries the same five links as the landing footer, and had
 * none of the tap sizing the landing one was given: 12px text with no padding
 * of its own. The two now share a single class constant, so they cannot drift
 * apart again.
 */
describe("AppLayout footer tap targets", () => {
  it.each([
    "home.nav.faq",
    "home.footer.terms",
    "home.footer.privacy",
    "home.footer.refund",
    "cookieConsent.settings",
  ])("gives %s a 44x44 minimum on phones", (label) => {
    renderLayout();
    const control = screen.getByText(label);

    expect(control.className).toContain("min-h-11");
    expect(control.className).toContain("min-w-11");
  });

  it("drops the minimum from sm up so the pointer footer keeps its density", () => {
    renderLayout();
    const control = screen.getByText("home.footer.terms");

    expect(control.className).toContain("sm:min-h-0");
    expect(control.className).toContain("sm:min-w-0");
  });

  it("gives the FluxLab attribution the same height without moving its href", () => {
    renderLayout();
    const link = screen.getByRole("link", { name: "Powered by FluxLab" });

    expect(link.className).toContain("min-h-11");
    expect(link).toHaveAttribute("href", "https://flux-lab.dev/en");
  });

  // Every one of these still has to go where it went before.
  it.each([
    ["home.nav.faq", "/#faq"],
    ["home.footer.terms", "/terms"],
    ["home.footer.privacy", "/privacy"],
    ["home.footer.refund", "/refund"],
  ])("keeps %s pointing at %s", (label, href) => {
    renderLayout();

    expect(screen.getByText(label).closest("a")).toHaveAttribute("href", href);
  });
});
