import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";

interface HeroSectionProps {
  readonly isAuthenticated: boolean;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
}

type BadgeVariant = "lime" | "sky" | "gray" | "teal" | "amber";

const BADGE_STYLES: Record<BadgeVariant, string> = {
  lime: "bg-lime-400/[0.08] text-lime-400 border-lime-400/[0.18]",
  sky: "bg-sky-400/[0.08] text-sky-400 border-sky-400/[0.18]",
  gray: "bg-white/[0.04] text-slate-400 border-white/[0.07]",
  teal: "bg-teal-400/[0.08] text-teal-400 border-teal-400/[0.18]",
  amber: "bg-amber-400/[0.08] text-amber-400 border-amber-400/[0.18]",
};

const KANBAN_COLUMNS = [
  {
    title: "Applied",
    count: 3,
    cards: [
      {
        company: "Stripe",
        role: "Senior Frontend Engineer",
        badges: [
          { text: "Remote", variant: "sky" },
          { text: "$180k", variant: "gray" },
        ],
      },
      {
        company: "Vercel",
        role: "Staff Engineer",
        badges: [{ text: "87% match", variant: "lime" }],
      },
    ],
  },
  {
    title: "Phone Screen",
    count: 2,
    cards: [
      {
        company: "Linear",
        role: "Product Engineer",
        badges: [{ text: "Tomorrow 14:00", variant: "amber" }],
      },
      {
        company: "Notion",
        role: "Full-Stack Engineer",
        badges: [{ text: "Hybrid", variant: "sky" }],
      },
    ],
  },
  {
    title: "Technical",
    count: 1,
    cards: [
      {
        company: "PlanetScale",
        role: "Backend Engineer",
        badges: [
          { text: "92% match", variant: "lime" },
          { text: "$200k", variant: "gray" },
        ],
      },
    ],
  },
  {
    title: "Offer",
    count: 1,
    cards: [
      {
        company: "Railway",
        role: "Infrastructure Engineer",
        badges: [{ text: "\u{1F389} Offer", variant: "teal" }],
      },
    ],
  },
] as const;

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

  return (
    <section className="landing-grid landing-hero-glow relative flex min-h-[88vh] flex-col items-center justify-center overflow-hidden px-6 pt-20 pb-16 text-center">
      {/* Badge */}
      <div className="landing-fade-up mb-7 inline-flex items-center gap-1.5 rounded-full border border-lime-400/20 bg-lime-400/[0.07] px-2.5 py-1 font-mono text-[11px] font-medium uppercase tracking-wider text-lime-400">
        <span className="landing-pulse-dot h-[5px] w-[5px] rounded-full bg-lime-400" />
        {t("home.hero.badge")}
      </div>

      {/* Headline */}
      <h1 className="landing-fade-up landing-fade-up-1 mb-6 max-w-[820px] text-[clamp(32px,7vw,80px)] font-extrabold leading-[1.05] tracking-[-0.04em] text-slate-100">
        {t("home.hero.titleStart")}
        <em className="not-italic text-lime-400">
          {t("home.hero.titleAccent")}
        </em>
      </h1>

      {/* Subtitle */}
      <p className="landing-fade-up landing-fade-up-2 mb-9 max-w-[520px] text-[clamp(16px,2vw,19px)] leading-relaxed text-slate-400">
        {t("home.hero.subtitle")}
      </p>

      {/* CTA */}
      <div className="landing-fade-up landing-fade-up-3 flex flex-wrap items-center justify-center gap-3">
        {isAuthenticated ? (
          <Button size="lg" className="px-6 text-[15px]" onClick={onGoPlatform}>
            {t("home.hero.ctaGoPlatform")}
          </Button>
        ) : (
          <>
            <Button size="lg" className="px-6 text-[15px]" onClick={onRegister}>
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

      {/* Kanban Preview */}
      <div
        className="landing-fade-up landing-fade-up-4 mt-14 hidden w-full max-w-[860px] sm:block"
        aria-hidden="true"
      >
        <div className="grid grid-cols-4 gap-3 rounded-[14px] border border-white/[0.07] bg-card p-5">
          {KANBAN_COLUMNS.map((col) => (
            <div key={col.title}>
              <div className="mb-2.5 flex items-center justify-between font-mono text-[11px] font-medium uppercase tracking-wider text-slate-500">
                {col.title}
                <span className="rounded bg-muted px-1.5 py-px text-[10px] text-slate-400">
                  {col.count}
                </span>
              </div>
              <div className="flex flex-col gap-2">
                {col.cards.map((card) => (
                  <div
                    key={card.role}
                    className="rounded-lg border border-white/[0.07] bg-muted p-3"
                  >
                    <div className="mb-0.5 font-mono text-[10px] text-slate-500">
                      {card.company}
                    </div>
                    <div className="mb-2 text-[13px] font-semibold leading-snug text-slate-100">
                      {card.role}
                    </div>
                    <div className="flex flex-wrap items-center gap-1.5">
                      {card.badges.map((badge) => (
                        <span
                          key={badge.text}
                          className={`inline-flex items-center gap-0.5 rounded border px-1.5 py-px font-mono text-[10px] font-medium ${BADGE_STYLES[badge.variant]}`}
                        >
                          {badge.text}
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
