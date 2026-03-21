import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export function FooterSection() {
  const { t } = useTranslation();

  return (
    <footer className="border-t border-white/[0.07] px-6 py-8 md:px-10">
      <div className="mx-auto flex max-w-[1080px] flex-col items-center gap-4 sm:flex-row sm:justify-between">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-[15px] font-bold tracking-[-0.03em] text-slate-400">
            <span className="h-2 w-2 rounded-full bg-lime-400 shadow-[0_0_8px_rgba(163,230,53,0.6)]" />
            Jobber
          </div>
          <span className="font-mono text-xs text-slate-600">
            &copy; {new Date().getFullYear()} {t("home.footer.copyright")}
          </span>
        </div>
        <div className="flex gap-5">
          <Link
            to="/privacy"
            className="text-[13px] text-slate-600 transition-colors hover:text-slate-400"
          >
            {t("home.footer.privacy")}
          </Link>
          <Link
            to="/terms"
            className="text-[13px] text-slate-600 transition-colors hover:text-slate-400"
          >
            {t("home.footer.terms")}
          </Link>
          <Link
            to="/refund"
            className="text-[13px] text-slate-600 transition-colors hover:text-slate-400"
          >
            {t("home.footer.refund")}
          </Link>
        </div>
      </div>
    </footer>
  );
}
