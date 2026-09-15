import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { resetConsent } from "@/shared/lib/consent";
import { PoweredByFluxLab } from "@/shared/ui/PoweredByFluxLab";
import { TAP_TARGET_INLINE } from "@/shared/ui/tapTarget";

/**
 * Footer links were 13px text with no padding — an 18px tap target, well under
 * the 44px WCAG 2.5.5 and the Apple HIG both ask for, and sitting right above
 * the consent banner where a mis-tap is easy. The row gets a 44x44 minimum on
 * phones and drops back to plain inline text from `sm` up, so pointer-driven
 * layouts keep their density. `gap-x` shrinks on the narrowest screens so five
 * links still wrap inside 375px rather than overflowing.
 */
const FOOTER_LINK =
  `${TAP_TARGET_INLINE} text-[13px] text-slate-600 transition-colors ` +
  "hover:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-lime-400 " +
  "focus-visible:ring-offset-2 focus-visible:ring-offset-transparent";

export function FooterSection() {
  const { t } = useTranslation();

  return (
    <footer className="border-t border-white/[0.07] px-6 py-8 md:px-10">
      <div className="mx-auto flex max-w-[1080px] flex-col items-center gap-4 sm:flex-row sm:justify-between">
        <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
          <div className="flex items-center gap-2 text-[15px] font-bold tracking-[-0.03em] text-slate-400">
            <span className="h-2 w-2 rounded-full bg-lime-400 shadow-[0_0_8px_rgba(163,230,53,0.6)]" />
            Jobber
          </div>
          <span className="font-mono text-xs text-slate-600">
            &copy; {new Date().getFullYear()} {t("home.footer.copyright")}
          </span>
          <PoweredByFluxLab className="text-[13px] text-slate-600 hover:text-slate-400" />
        </div>
        <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-0 sm:gap-x-5 sm:gap-y-1">
          <Link to="/#faq" className={FOOTER_LINK}>
            {t("home.nav.faq")}
          </Link>
          <Link to="/privacy" className={FOOTER_LINK}>
            {t("home.footer.privacy")}
          </Link>
          <Link to="/terms" className={FOOTER_LINK}>
            {t("home.footer.terms")}
          </Link>
          <Link to="/refund" className={FOOTER_LINK}>
            {t("home.footer.refund")}
          </Link>
          <button type="button" onClick={resetConsent} className={FOOTER_LINK}>
            {t("cookieConsent.settings")}
          </button>
        </div>
      </div>
    </footer>
  );
}
