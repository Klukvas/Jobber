import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const consentMock = vi.hoisted(() => ({
  applyConsent: vi.fn(),
  getStoredConsent: vi.fn<() => "accepted" | "essential" | null>(() => null),
  isAnalyticsConfigured: vi.fn<() => boolean>(() => true),
  CONSENT_RESET_EVENT: "jobber:cookie-consent-reset",
}));

vi.mock("@/shared/lib/consent", () => consentMock);
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

import { CookieConsent } from "../CookieConsent";

beforeEach(() => {
  vi.clearAllMocks();
  consentMock.getStoredConsent.mockReturnValue(null);
  consentMock.isAnalyticsConfigured.mockReturnValue(true);
  document.body.style.paddingBottom = "";
  document.documentElement.style.removeProperty("--app-bottom-inset");
});

describe("CookieConsent", () => {
  it("shows the banner when no choice is stored", () => {
    render(<CookieConsent />);
    expect(screen.getByRole("region")).toBeInTheDocument();
    expect(screen.getByText("cookieConsent.acceptAll")).toBeInTheDocument();
  });

  it("stays hidden when a choice already exists", () => {
    consentMock.getStoredConsent.mockReturnValue("essential");
    render(<CookieConsent />);
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
  });

  it("accept-all applies consent and hides the banner", async () => {
    const user = userEvent.setup();
    render(<CookieConsent />);

    await user.click(screen.getByText("cookieConsent.acceptAll"));

    expect(consentMock.applyConsent).toHaveBeenCalledWith("accepted");
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
  });

  it("essential-only applies consent and hides the banner", async () => {
    const user = userEvent.setup();
    render(<CookieConsent />);

    await user.click(screen.getByText("cookieConsent.essentialOnly"));

    expect(consentMock.applyConsent).toHaveBeenCalledWith("essential");
    expect(screen.queryByRole("region")).not.toBeInTheDocument();
  });

  it("reopens on the consent-reset event (footer Cookie settings)", async () => {
    const user = userEvent.setup();
    render(<CookieConsent />);
    await user.click(screen.getByText("cookieConsent.essentialOnly"));
    expect(screen.queryByRole("region")).not.toBeInTheDocument();

    window.dispatchEvent(new Event(consentMock.CONSENT_RESET_EVENT));

    expect(await screen.findByRole("region")).toBeInTheDocument();
  });

  it("links to the privacy policy", () => {
    render(<CookieConsent />);
    const link = screen.getByRole("link", {
      name: "cookieConsent.privacyLink",
    });
    expect(link).toHaveAttribute("href", "/privacy");
  });

  it("promises analytics cookies only when analytics are configured", () => {
    render(<CookieConsent />);
    expect(screen.getByText(/cookieConsent\.message/)).toBeInTheDocument();
    expect(
      screen.queryByText(/cookieConsent\.messageEssentialOnly/),
    ).not.toBeInTheDocument();
  });

  it("says only essential cookies are used when no tracker is configured", () => {
    consentMock.isAnalyticsConfigured.mockReturnValue(false);
    render(<CookieConsent />);
    expect(
      screen.getByText(/cookieConsent\.messageEssentialOnly/),
    ).toBeInTheDocument();
  });

  // The banner is fixed to the bottom of the viewport, so page content — the
  // pricing CTAs above all — has to be pushed clear of it while it is up.
  describe("layout compensation", () => {
    it("reserves space under the page while visible", () => {
      render(<CookieConsent />);
      expect(document.body.style.paddingBottom).not.toBe("");
    });

    it("gives the space back when dismissed", async () => {
      const user = userEvent.setup();
      render(<CookieConsent />);
      await user.click(screen.getByText("cookieConsent.acceptAll"));

      expect(document.body.style.paddingBottom).toBe("");
    });

    it("restores padding another feature had already set", async () => {
      const user = userEvent.setup();
      document.body.style.paddingBottom = "24px";

      render(<CookieConsent />);
      expect(document.body.style.paddingBottom).not.toBe("24px");

      await user.click(screen.getByText("cookieConsent.essentialOnly"));
      expect(document.body.style.paddingBottom).toBe("24px");
    });

    it("reserves space again when the banner is reopened", async () => {
      const user = userEvent.setup();
      render(<CookieConsent />);
      await user.click(screen.getByText("cookieConsent.acceptAll"));
      expect(document.body.style.paddingBottom).toBe("");

      window.dispatchEvent(new Event(consentMock.CONSENT_RESET_EVENT));
      await screen.findByRole("region");

      expect(document.body.style.paddingBottom).not.toBe("");
    });

    // Padding keeps the footer reachable, but it cannot help a jump into the
    // middle of the page. Publishing the height lets scrollToSection (and CSS
    // scroll-padding-bottom) move anchor targets clear of the banner.
    it("publishes the banner height for anchor scrolling", () => {
      render(<CookieConsent />);
      const card = screen.getByRole("region");
      Object.defineProperty(card, "offsetHeight", {
        value: 94,
        configurable: true,
      });
      window.dispatchEvent(new Event("resize"));

      expect(
        document.documentElement.style.getPropertyValue("--app-bottom-inset"),
      ).toBe("126px");
    });

    it("withdraws the published height when dismissed", async () => {
      const user = userEvent.setup();
      render(<CookieConsent />);
      expect(
        document.documentElement.style.getPropertyValue("--app-bottom-inset"),
      ).not.toBe("");

      await user.click(screen.getByText("cookieConsent.acceptAll"));

      expect(
        document.documentElement.style.getPropertyValue("--app-bottom-inset"),
      ).toBe("");
    });

    it("re-publishes the height when the banner is reopened", async () => {
      const user = userEvent.setup();
      render(<CookieConsent />);
      await user.click(screen.getByText("cookieConsent.acceptAll"));

      window.dispatchEvent(new Event(consentMock.CONSENT_RESET_EVENT));
      await screen.findByRole("region");

      expect(
        document.documentElement.style.getPropertyValue("--app-bottom-inset"),
      ).not.toBe("");
    });

    it("re-measures on viewport resize", () => {
      render(<CookieConsent />);
      const card = screen.getByRole("region");
      // jsdom reports 0 height, so drive the measurement explicitly.
      Object.defineProperty(card, "offsetHeight", {
        value: 120,
        configurable: true,
      });

      window.dispatchEvent(new Event("resize"));

      expect(document.body.style.paddingBottom).toBe("152px");
    });

    /**
     * The reserved strip is body padding, and <body> paints it. The landing
     * page forces its dark palette on a wrapper *inside* <body>, so at full
     * scroll depth that strip showed the app's light background as a pale band
     * across the bottom of a dark page. `body.landing-page` (added by the
     * landing page for exactly as long as it is mounted) repaints <body> with
     * the landing background so the strip matches the page above it.
     */
    it("reserves the space on the element the landing page repaints", () => {
      document.body.classList.add("landing-page");
      try {
        render(<CookieConsent />);
        const card = screen.getByRole("region");
        Object.defineProperty(card, "offsetHeight", {
          value: 120,
          configurable: true,
        });
        window.dispatchEvent(new Event("resize"));

        // Same element carries both the padding and the landing background, so
        // the strip cannot be a different colour from the page.
        expect(document.body.style.paddingBottom).toBe("152px");
        expect(document.body.classList.contains("landing-page")).toBe(true);
      } finally {
        document.body.classList.remove("landing-page");
      }
    });

    it("restores the exact previous padding, not an assumed empty one", () => {
      document.body.style.paddingBottom = "24px";
      try {
        const { unmount } = render(<CookieConsent />);
        expect(document.body.style.paddingBottom).not.toBe("24px");

        unmount();

        expect(document.body.style.paddingBottom).toBe("24px");
      } finally {
        document.body.style.paddingBottom = "";
      }
    });
  });

  /**
   * The two buttons are the only way to answer a banner that covers the bottom
   * of the screen. At `px-3 py-2` they were a ~36px tap target, under the 44px
   * WCAG 2.5.5 asks for — and they sit exactly where a thumb rests.
   */
  describe("touch targets", () => {
    it("gives both buttons a 44px minimum on phones", () => {
      render(<CookieConsent />);

      for (const label of ["cookieConsent.essentialOnly", "cookieConsent.acceptAll"]) {
        const button = screen.getByText(label);
        expect(button.className, label).toContain("min-h-11");
      }
    });

    it("keeps the compact density from sm up", () => {
      render(<CookieConsent />);

      for (const label of ["cookieConsent.essentialOnly", "cookieConsent.acceptAll"]) {
        expect(screen.getByText(label).className, label).toContain("sm:min-h-9");
      }
    });

    // flex-1 on the narrowest layout only: the pair shares the row inside
    // 375px instead of pushing the card wider than the screen.
    it("lets the pair share one row on a narrow screen without overflowing", () => {
      render(<CookieConsent />);

      const button = screen.getByText("cookieConsent.acceptAll");
      expect(button.className).toContain("flex-1");
      expect(button.className).toContain("sm:flex-none");
    });
  });
});
