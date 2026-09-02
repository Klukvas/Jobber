import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  applyConsent,
  getStoredConsent,
  CONSENT_RESET_EVENT,
  type CookieConsent as Consent,
} from "@/shared/lib/consent";

// Mounted outside the router (next to Toaster), so the privacy link is a plain
// <a>, not a router Link.
export function CookieConsent() {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(() => getStoredConsent() === null);

  useEffect(() => {
    // Footer "Cookie settings" (resetConsent) re-opens the banner.
    const reopen = () => setVisible(true);
    window.addEventListener(CONSENT_RESET_EVENT, reopen);
    return () => window.removeEventListener(CONSENT_RESET_EVENT, reopen);
  }, []);

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
        role="region"
        aria-label={t("cookieConsent.ariaLabel")}
        className="pointer-events-auto mx-auto flex max-w-3xl flex-col gap-3 rounded-lg border bg-card p-4 text-card-foreground shadow-lg sm:flex-row sm:items-center"
      >
        <p className="flex-1 text-sm text-muted-foreground">
          {t("cookieConsent.message")}{" "}
          <a href="/privacy" className="text-primary underline">
            {t("cookieConsent.privacyLink")}
          </a>
        </p>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={() => choose("essential")}
            className="rounded-md border px-3 py-2 text-sm transition-colors hover:bg-accent"
          >
            {t("cookieConsent.essentialOnly")}
          </button>
          <button
            type="button"
            onClick={() => choose("accepted")}
            className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            {t("cookieConsent.acceptAll")}
          </button>
        </div>
      </div>
    </div>
  );
}
