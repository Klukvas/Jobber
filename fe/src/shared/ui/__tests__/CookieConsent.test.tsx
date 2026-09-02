import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const consentMock = vi.hoisted(() => ({
  applyConsent: vi.fn(),
  getStoredConsent: vi.fn<() => "accepted" | "essential" | null>(() => null),
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
    const link = screen.getByRole("link", { name: "cookieConsent.privacyLink" });
    expect(link).toHaveAttribute("href", "/privacy");
  });
});
