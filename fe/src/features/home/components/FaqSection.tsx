import { useTranslation } from "react-i18next";
import { FaqJsonLd } from "./FaqJsonLd";
import { Reveal } from "./Reveal";

const FAQ_KEYS = [
  "free",
  "what",
  "vsCompetitors",
  "aiMatch",
  "import",
  "privacy",
  "install",
] as const;

export function FaqSection() {
  const { t } = useTranslation();

  const items = FAQ_KEYS.map((key) => ({
    key,
    question: t(`home.faq.items.${key}.q`),
    answer: t(`home.faq.items.${key}.a`),
  }));

  return (
    <section id="faq" className="px-6 py-24">
      <FaqJsonLd items={items} />
      <div className="mx-auto max-w-[860px]">
        <Reveal className="mb-12 text-center">
          <div className="mb-3 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
            {t("home.faq.label")}
          </div>
          <h2 className="mb-4 text-[clamp(28px,4vw,44px)] font-extrabold leading-[1.15] tracking-[-0.035em] text-slate-100">
            {t("home.faq.title")}
          </h2>
          <p className="text-[15px] leading-[1.7] text-slate-400">
            {t("home.faq.subtitle")}
          </p>
        </Reveal>

        <div className="space-y-3">
          {items.map(({ key, question, answer }, index) => (
            <Reveal key={key} delayMs={index * 60}>
              <details className="group rounded-xl border border-white/[0.07] bg-card p-5 transition-colors hover:border-white/[0.14]">
                <summary className="flex cursor-pointer list-none items-center justify-between gap-4 text-[15px] font-semibold text-slate-100">
                  <span>{question}</span>
                  <span
                    aria-hidden="true"
                    className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-white/[0.1] text-lime-400 transition-transform group-open:rotate-45"
                  >
                    +
                  </span>
                </summary>
                <p className="mt-4 text-[14px] leading-[1.7] text-slate-400">
                  {answer}
                </p>
              </details>
            </Reveal>
          ))}
        </div>
      </div>
    </section>
  );
}
