import { useTranslation } from "react-i18next";

const COMPANIES = [
  "Google",
  "Meta",
  "Stripe",
  "Vercel",
  "GitLab",
  "Figma",
  "Notion",
] as const;

function LogoRow({ ariaHidden = false }: { ariaHidden?: boolean }) {
  return (
    <div
      aria-hidden={ariaHidden || undefined}
      className="flex shrink-0 items-center gap-x-12 pr-12"
    >
      {COMPANIES.map((name) => (
        <span
          key={name}
          className="text-sm font-bold tracking-tight text-slate-600"
        >
          {name}
        </span>
      ))}
    </div>
  );
}

export function SocialProofBar() {
  const { t } = useTranslation();

  return (
    <div className="border-y border-white/[0.07] px-6 py-7">
      <div className="mx-auto flex max-w-[1080px] flex-col items-center gap-4 sm:flex-row sm:gap-8">
        <span className="shrink-0 whitespace-nowrap font-mono text-[11px] uppercase tracking-wider text-slate-600">
          {t("home.socialProof.label")} &rarr;
        </span>
        <div className="w-full min-w-0 flex-1 overflow-hidden [mask-image:linear-gradient(to_right,transparent,black_12%,black_88%,transparent)]">
          {/* The row is rendered twice; the loop translates by exactly one copy */}
          <div className="landing-marquee flex w-max">
            <LogoRow />
            <LogoRow ariaHidden />
          </div>
        </div>
      </div>
    </div>
  );
}
