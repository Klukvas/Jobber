import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  applyConsent,
  getStoredConsent,
  isAnalyticsConfigured,
  CONSENT_RESET_EVENT,
  type CookieConsent as Consent,
} from "@/shared/lib/consent";
import { BOTTOM_INSET_VAR } from "@/shared/lib/scrollToSection";

// Gap between the card and the viewport edge — matches the strip's p-4.
const BANNER_GUTTER_PX = 16;

/**
 * Shared shape of the two consent buttons: a 44px tap target on phones, the
 * compact 36px pointer density from `sm` up. They share everything but their
 * colours, so the sizing cannot drift between "accept" and "essential only".
 */
const CONSENT_BUTTON_BASE =
  "inline-flex min-h-11 flex-1 items-center justify-center rounded-md px-3 py-2 text-sm transition-colors " +
  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 " +
  "sm:min-h-9 sm:flex-none";

// Mounted outside the router (next to Toaster), so the privacy link is a plain
// <a>, not a router Link.
export function CookieConsent() {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(() => getStoredConsent() === null);
  const cardRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Footer "Cookie settings" (resetConsent) re-opens the banner.
    const reopen = () => setVisible(true);
    window.addEventListener(CONSENT_RESET_EVENT, reopen);
    return () => window.removeEventListener(CONSENT_RESET_EVENT, reopen);
  }, []);

  // The banner is fixed to the bottom, so it covers the last stripe of the
  // viewport. Two separate problems come out of that, and both are handled
  // here:
  //
  //  1. The end of the document is unreachable — fixed by padding the body, so
  //     the footer can still be scrolled above the banner.
  //  2. A jump into the *middle* of the page lands content in the covered
  //     stripe. Padding cannot help there, so the height is also published as
  //     a CSS variable and `scrollToSection` scrolls clear of it. The same
  //     variable drives `scroll-padding-bottom`, which covers the scrolls the
  //     browser performs on its own (native `#hash`, focus, scroll-into-view).
  //
  // Measured rather than hard-coded: the card grows to two or three lines in
  // Russian and Ukrainian and on narrow screens.
  useEffect(() => {
    if (!visible) return;
    const card = cardRef.current;
    if (!card) return;

    // Captured, not assumed empty: another feature may already be padding the
    // body, and clearing it outright would silently undo that.
    const root = document.documentElement;
    const previousPaddingBottom = document.body.style.paddingBottom;
    const previousInset = root.style.getPropertyValue(BOTTOM_INSET_VAR);

    const reserveSpace = () => {
      const inset = card.offsetHeight + BANNER_GUTTER_PX * 2;
      document.body.style.paddingBottom = `${inset}px`;
      root.style.setProperty(BOTTOM_INSET_VAR, `${inset}px`);
    };
    reserveSpace();

    const observer =
      typeof ResizeObserver === "undefined"
        ? null
        : new ResizeObserver(reserveSpace);
    observer?.observe(card);
    window.addEventListener("resize", reserveSpace);

    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", reserveSpace);
      document.body.style.paddingBottom = previousPaddingBottom;
      if (previousInset) {
        root.style.setProperty(BOTTOM_INSET_VAR, previousInset);
      } else {
        root.style.removeProperty(BOTTOM_INSET_VAR);
      }
    };
  }, [visible]);

  if (!visible) return null;

  const choose = (consent: Consent) => {
    applyConsent(consent);
    setVisible(false);
  };

  return (
    // pointer-events-none on the full-width strip: only the card itself may
    // swallow clicks, or the transparent gutters block the support button
    // and footer links underneath.
    <div className="pointer-events-none fixed inset-x-0 bottom-0 z-50 p-4">
      <div
        ref={cardRef}
        role="region"
        aria-label={t("cookieConsent.ariaLabel")}
        className="pointer-events-auto mx-auto flex max-w-3xl flex-col gap-3 rounded-lg border bg-card p-4 text-card-foreground shadow-lg sm:flex-row sm:items-center"
      >
        <p className="flex-1 text-sm text-muted-foreground">
          {/* Only promise the trackers this build actually ships with — the
              analytics keys are environment-specific and are frequently absent. */}
          {isAnalyticsConfigured()
            ? t("cookieConsent.message")
            : t("cookieConsent.messageEssentialOnly")}{" "}
          <a href="/privacy" className="text-primary underline">
            {t("cookieConsent.privacyLink")}
          </a>
        </p>
        <div className="flex flex-wrap gap-2">
          {/* 44px tall on phones, the compact 36px from `sm` up. These two are
              the only way to answer a banner that covers the bottom of the
              screen, and at 36px they were under the minimum a thumb can hit.
              `flex-1` on the narrowest layout keeps the pair on one row inside
              375px instead of overflowing the card. */}
          <button
            type="button"
            onClick={() => choose("essential")}
            className={CONSENT_BUTTON_BASE + " border hover:bg-accent"}
          >
            {t("cookieConsent.essentialOnly")}
          </button>
          <button
            type="button"
            onClick={() => choose("accepted")}
            className={
              CONSENT_BUTTON_BASE +
              " bg-primary font-medium text-primary-foreground hover:bg-primary/90"
            }
          >
            {t("cookieConsent.acceptAll")}
          </button>
        </div>
      </div>
    </div>
  );
}
