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

export function SocialProofBar() {
  const { t } = useTranslation();

  return (
    <div className="border-y border-white/[0.07] px-6 py-7">
      <div className="mx-auto flex max-w-[1080px] flex-wrap items-center justify-center gap-x-8 gap-y-3">
        <span className="whitespace-nowrap font-mono text-[11px] uppercase tracking-wider text-slate-600">
          {t("home.socialProof.label")} &rarr;
        </span>
        {COMPANIES.map((name) => (
          <span
            key={name}
            className="text-sm font-bold tracking-tight text-slate-600"
          >
            {name}
          </span>
        ))}
      </div>
    </div>
  );
}
