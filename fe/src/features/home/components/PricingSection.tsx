import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";

interface PricingSectionProps {
  readonly isAuthenticated: boolean;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
}

const FREE_FEATURES = [
  "freeFeature1",
  "freeFeature2",
  "freeFeature3",
  "freeFeature4",
  "freeFeature5",
  "freeFeature6",
  "freeFeature7",
] as const;

const PRO_FEATURES = [
  "proFeature1",
  "proFeature2",
  "proFeature3",
  "proFeature4",
  "proFeature5",
  "proFeature6",
  "proFeature7",
] as const;

const ENTERPRISE_FEATURES = [
  "enterpriseFeature1",
  "enterpriseFeature2",
  "enterpriseFeature3",
  "enterpriseFeature4",
  "enterpriseFeature5",
  "enterpriseFeature6",
  "enterpriseFeature7",
] as const;

export function PricingSection({
  isAuthenticated,
  onRegister,
  onGoPlatform,
}: PricingSectionProps) {
  const { t } = useTranslation();
  const handleCta = isAuthenticated ? onGoPlatform : onRegister;

  return (
    <section id="pricing" className="px-6 py-24 text-center">
      <div className="mx-auto max-w-[1080px]">
        <div className="mb-3.5 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
          {t("home.pricing.label")}
        </div>
        <h2 className="mb-4 text-[clamp(28px,4vw,44px)] font-extrabold leading-[1.1] tracking-[-0.035em] text-slate-100">
          {t("home.pricing.title")}
        </h2>
        <p className="mx-auto max-w-[480px] text-[17px] leading-relaxed text-slate-400">
          {t("home.pricing.subtitle")}
        </p>

        <div className="mx-auto mt-14 grid max-w-[1020px] gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {/* Free */}
          <div className="rounded-2xl border border-white/[0.07] bg-card p-8 text-left">
            <div className="mb-3 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
              {t("home.pricing.free")}
            </div>
            <div className="mb-1 text-[40px] font-extrabold tracking-[-0.04em] text-slate-100">
              $0
            </div>
            <div className="mb-6 text-[13px] text-slate-500">
              {t("home.pricing.freePeriod")}
            </div>
            <hr className="mb-5 border-white/[0.07]" />
            <div className="space-y-2.5">
              {FREE_FEATURES.map((key) => (
                <div key={key} className="flex items-start gap-2.5 text-[13px] text-slate-400">
                  <span className="mt-0.5 shrink-0 text-xs text-lime-400">&#10003;</span>
                  <span>{t(`home.pricing.${key}`)}</span>
                </div>
              ))}
            </div>
            <Button
              variant="outline"
              className="mt-6 w-full justify-center border-white/[0.07] bg-transparent text-slate-300 hover:border-white/[0.14] hover:text-white"
              onClick={handleCta}
            >
              {t("home.pricing.freeCta")}
            </Button>
          </div>

          {/* Pro */}
          <div className="relative rounded-2xl border border-lime-400/25 bg-card p-8 text-left shadow-[0_0_0_1px_rgba(163,230,53,0.08),0_16px_48px_rgba(0,0,0,0.4)]">
            <div className="absolute -top-2.5 left-1/2 -translate-x-1/2 rounded-full bg-lime-400 px-3 py-0.5 font-mono text-[10px] font-bold uppercase tracking-wider text-[#0B0F17]">
              {t("home.pricing.proBadge")}
            </div>
            <div className="mb-3 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
              {t("home.pricing.pro")}
            </div>
            <div className="mb-1 text-[40px] font-extrabold tracking-[-0.04em] text-slate-100">
              <span className="align-super text-base font-medium text-slate-400">$</span>
              7
            </div>
            <div className="mb-6 text-[13px] text-slate-500">
              {t("home.pricing.proPeriod")}
            </div>
            <hr className="mb-5 border-white/[0.07]" />
            <div className="space-y-2.5">
              {PRO_FEATURES.map((key) => (
                <div key={key} className="flex items-start gap-2.5 text-[13px] text-slate-400">
                  <span className="mt-0.5 shrink-0 text-xs text-lime-400">&#10003;</span>
                  <span>{t(`home.pricing.${key}`)}</span>
                </div>
              ))}
            </div>
            <Button className="mt-6 w-full justify-center" onClick={handleCta}>
              {t("home.pricing.proCta")} &rarr;
            </Button>
          </div>

          {/* Enterprise */}
          <div className="rounded-2xl border border-white/[0.07] bg-card p-8 text-left">
            <div className="mb-3 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
              {t("home.pricing.enterprise")}
            </div>
            <div className="mb-1 text-[40px] font-extrabold tracking-[-0.04em] text-slate-100">
              <span className="align-super text-base font-medium text-slate-400">$</span>
              19
            </div>
            <div className="mb-6 text-[13px] text-slate-500">
              {t("home.pricing.enterprisePeriod")}
            </div>
            <hr className="mb-5 border-white/[0.07]" />
            <div className="space-y-2.5">
              {ENTERPRISE_FEATURES.map((key) => (
                <div key={key} className="flex items-start gap-2.5 text-[13px] text-slate-400">
                  <span className="mt-0.5 shrink-0 text-xs text-lime-400">&#10003;</span>
                  <span>{t(`home.pricing.${key}`)}</span>
                </div>
              ))}
            </div>
            <Button
              variant="outline"
              className="mt-6 w-full justify-center border-white/[0.07] bg-transparent text-slate-300 hover:border-white/[0.14] hover:text-white"
              onClick={handleCta}
            >
              {t("home.pricing.enterpriseCta")}
            </Button>
          </div>
        </div>

        <p className="mt-5 font-mono text-xs text-slate-500">
          {t("home.pricing.comparison")}
        </p>
      </div>
    </section>
  );
}
