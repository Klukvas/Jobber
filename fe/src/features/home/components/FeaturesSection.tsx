import { useTranslation } from "react-i18next";
import {
  Kanban,
  Sparkles,
  FileText,
  Link2,
  BarChart3,
  CalendarDays,
  type LucideIcon,
} from "lucide-react";
import { Reveal } from "./Reveal";

const FEATURES: readonly { readonly key: string; readonly Icon: LucideIcon }[] =
  [
    { key: "kanban", Icon: Kanban },
    { key: "aiMatch", Icon: Sparkles },
    { key: "resume", Icon: FileText },
    { key: "jobImport", Icon: Link2 },
    { key: "analyticsCard", Icon: BarChart3 },
    { key: "calendar", Icon: CalendarDays },
  ];

export function FeaturesSection() {
  const { t } = useTranslation();

  return (
    <section id="features" className="px-6 py-24">
      <div className="mx-auto max-w-[1080px]">
        <Reveal>
          <div className="mb-3.5 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
            {t("home.features.label")}
          </div>
          <h2 className="mb-4 text-[clamp(28px,4vw,44px)] font-extrabold leading-[1.1] tracking-[-0.035em] text-slate-100">
            {t("home.features.title")}
          </h2>
          <p className="max-w-[480px] text-[17px] leading-relaxed text-slate-400">
            {t("home.features.subtitle")}
          </p>
        </Reveal>

        <div className="mt-14 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map(({ key, Icon }, index) => (
            <Reveal key={key} delayMs={index * 80} className="h-full">
              <div className="group h-full rounded-xl border border-white/[0.07] bg-card p-7 transition-all hover:-translate-y-1 hover:border-white/[0.14] hover:shadow-[0_0_0_1px_rgba(163,230,53,0.06),0_8px_32px_rgba(0,0,0,0.3)]">
                <div
                  className="mb-4 flex h-9 w-9 items-center justify-center rounded-lg border border-lime-400/20 bg-lime-400/[0.08] transition-transform duration-300 group-hover:-rotate-6 group-hover:scale-110"
                  aria-hidden="true"
                >
                  <Icon className="h-[18px] w-[18px] text-lime-400" />
                </div>
                <h3 className="mb-1.5 text-[15px] font-bold tracking-[-0.02em] text-slate-100">
                  {t(`home.features.${key}.title`)}
                </h3>
                <p className="text-[13px] leading-relaxed text-slate-400">
                  {t(`home.features.${key}.description`)}
                </p>
              </div>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
