import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";

interface AiHighlightSectionProps {
  readonly isAuthenticated: boolean;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
}

const SCORE_ROWS = [
  { key: "technicalSkills", value: 95, accent: true },
  { key: "experience", value: 88, accent: true },
  { key: "keywords", value: 91, accent: false },
  { key: "cultureFit", value: 78, accent: false },
  { key: "seniority", value: 97, accent: true },
] as const;

const MISSING_KEYWORDS = ["MySQL", "Vitess", "distributed SQL"] as const;

export function AiHighlightSection({
  isAuthenticated,
  onRegister,
  onGoPlatform,
}: AiHighlightSectionProps) {
  const { t } = useTranslation();

  return (
    <section className="px-6 py-24">
      <div className="landing-ai-glow relative mx-auto max-w-[1080px] overflow-hidden rounded-2xl border border-lime-400/25 bg-card p-5 shadow-[0_0_60px_rgba(163,230,53,0.04)] sm:rounded-[20px] sm:p-10 md:grid md:grid-cols-2 md:gap-14 md:p-14">
        {/* Left - Text */}
        <div>
          <div className="mb-5 inline-flex items-center gap-1.5 font-mono text-[11px] font-medium uppercase tracking-[0.1em] text-lime-400">
            <span>&#10022;</span>
            {t("home.ai.label")}
          </div>
          <h2 className="mb-4 text-[clamp(24px,3vw,36px)] font-extrabold leading-[1.15] tracking-[-0.035em] text-slate-100">
            {t("home.ai.title")}
          </h2>
          <p className="mb-7 text-[15px] leading-[1.7] text-slate-400">
            {t("home.ai.description")}
          </p>
          <Button
            className="px-5"
            onClick={isAuthenticated ? onGoPlatform : onRegister}
          >
            {t("home.ai.cta")} &rarr;
          </Button>
        </div>

        {/* Right - Score Card */}
        <div className="mt-10 md:mt-0">
          <div className="rounded-xl border border-white/[0.07] bg-muted p-6">
            {/* Header */}
            <div className="mb-5 flex items-center justify-between">
              <span className="text-sm font-bold text-slate-100">
                {t("home.ai.scoreTitle")} &mdash; {t("home.ai.company")}
              </span>
              <span className="font-mono text-[28px] font-medium text-lime-400">
                92%
              </span>
            </div>

            {/* Score Bars */}
            <div className="space-y-3">
              {SCORE_ROWS.map((row) => (
                <div key={row.key}>
                  <div className="mb-1 flex justify-between text-xs">
                    <span className="text-slate-400">
                      {t(`home.ai.${row.key}`)}
                    </span>
                    <span className="font-mono text-slate-100">
                      {row.value}%
                    </span>
                  </div>
                  <div className="h-1 overflow-hidden rounded-sm bg-white/[0.06]">
                    <div
                      role="progressbar"
                      aria-valuenow={row.value}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      aria-label={t(`home.ai.${row.key}`)}
                      className={`h-full rounded-sm transition-all duration-1000 ${row.accent ? "bg-lime-400" : "bg-sky-400"}`}
                      style={{ width: `${row.value}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>

            {/* Missing Keywords */}
            <div className="mt-4 border-t border-white/[0.07] pt-3.5">
              <div className="mb-2 font-mono text-[10px] font-medium uppercase tracking-[0.1em] text-lime-400">
                {t("home.ai.missingKeywords")}
              </div>
              <div className="flex flex-wrap gap-1.5">
                {MISSING_KEYWORDS.map((kw) => (
                  <span
                    key={kw}
                    className="rounded border border-white/[0.07] bg-white/[0.04] px-2 py-0.5 font-mono text-[10px] font-medium text-slate-400"
                  >
                    {kw}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
