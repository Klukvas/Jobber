import { useTranslation } from "react-i18next";

const STEPS = ["step1", "step2", "step3", "step4"] as const;

export function HowItWorksSection() {
  const { t } = useTranslation();

  return (
    <section
      id="how-it-works"
      className="border-y border-white/[0.07] bg-card px-6 py-24"
    >
      <div className="mx-auto max-w-[1080px]">
        <div className="mb-14 text-center">
          <div className="mb-3.5 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
            {t("home.howItWorks.label")}
          </div>
          <h2 className="text-[clamp(28px,4vw,44px)] font-extrabold leading-[1.1] tracking-[-0.035em] text-slate-100">
            {t("home.howItWorks.title")}
          </h2>
        </div>

        <div className="relative grid gap-8 md:gap-0 md:grid-cols-4">
          {/* Connecting line */}
          <div className="absolute top-5 left-[10%] right-[10%] hidden h-px bg-gradient-to-r from-transparent via-white/[0.07] to-transparent md:block" />

          {STEPS.map((step, i) => (
            <div key={step} className="px-0 md:px-6 text-center">
              <div className="relative z-10 mx-auto mb-5 flex h-10 w-10 items-center justify-center rounded-full border border-white/[0.07] bg-card font-mono text-[13px] font-medium text-lime-400">
                {String(i + 1).padStart(2, "0")}
              </div>
              <h3 className="mb-2 text-[15px] font-bold text-slate-100">
                {t(`home.howItWorks.${step}.title`)}
              </h3>
              <p className="text-[13px] leading-snug text-slate-400">
                {t(`home.howItWorks.${step}.description`)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
