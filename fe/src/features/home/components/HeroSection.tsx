import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";
import { HeroShowcase } from "./HeroShowcase";

interface HeroSectionProps {
  readonly isAuthenticated: boolean;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
}

export function HeroSection({
  isAuthenticated,
  onRegister,
  onGoPlatform,
}: HeroSectionProps) {
  const { t } = useTranslation();

  const scrollToHow = () => {
    document
      .getElementById("how-it-works")
      ?.scrollIntoView({ behavior: "smooth" });
  };

  // The accent ends with a decorative "_" in every locale — replace it
  // with a blinking terminal caret.
  const accentText = t("home.hero.titleAccent").replace(/_+$/, "");

  return (
    <section className="landing-grid landing-hero-glow relative flex min-h-[88vh] items-center overflow-hidden px-6 pb-24 pt-28 lg:pt-24">
      <div className="relative mx-auto grid w-full max-w-[1180px] items-center gap-y-20 lg:grid-cols-[1.02fr_0.98fr] lg:gap-x-12">
        {/* Copy */}
        <div className="flex flex-col items-start text-left">
          <div className="landing-fade-up mb-7 inline-flex items-center gap-1.5 rounded-full border border-lime-400/20 bg-lime-400/[0.07] px-2.5 py-1 font-mono text-[11px] font-medium uppercase tracking-wider text-lime-400">
            <span className="landing-pulse-dot h-[5px] w-[5px] rounded-full bg-lime-400" />
            {t("home.hero.badge")}
          </div>

          <h1 className="landing-fade-up landing-fade-up-1 mb-6 max-w-[560px] text-[clamp(34px,4.8vw,64px)] font-extrabold leading-[1.06] tracking-[-0.035em] text-slate-100">
            {t("home.hero.titleStart")}
            <em className="not-italic text-lime-400">
              {accentText}
              <span
                className="hero-caret ml-1 inline-block h-[0.72em] w-[0.44em] translate-y-[0.06em] bg-lime-400"
                aria-hidden="true"
              />
            </em>
          </h1>

          <p className="landing-fade-up landing-fade-up-2 mb-9 max-w-[480px] text-[clamp(16px,2vw,18px)] leading-relaxed text-slate-400">
            {t("home.hero.subtitle")}
          </p>

          <div className="landing-fade-up landing-fade-up-3 flex flex-wrap items-center gap-3">
            {isAuthenticated ? (
              <Button
                size="lg"
                className="px-6 text-[15px]"
                onClick={onGoPlatform}
              >
                {t("home.hero.ctaGoPlatform")}
              </Button>
            ) : (
              <>
                <Button
                  size="lg"
                  className="px-6 text-[15px]"
                  onClick={onRegister}
                >
                  {t("home.hero.cta")} &rarr;
                </Button>
                <Button
                  variant="outline"
                  size="lg"
                  className="border-white/[0.07] bg-transparent px-6 text-[15px] text-slate-300 hover:border-white/[0.14] hover:bg-white/[0.03] hover:text-white"
                  onClick={scrollToHow}
                >
                  {t("home.hero.ctaSecondary")}
                </Button>
              </>
            )}
          </div>

          <p className="landing-fade-up landing-fade-up-4 mt-5 font-mono text-[11px] uppercase tracking-wider text-slate-600">
            {t("home.hero.microcopy")}
          </p>
        </div>

        {/* Live product collage */}
        <HeroShowcase />
      </div>
    </section>
  );
}
