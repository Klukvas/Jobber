import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";

interface FooterCtaSectionProps {
  readonly isAuthenticated: boolean;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
}

export function FooterCtaSection({
  isAuthenticated,
  onRegister,
  onGoPlatform,
}: FooterCtaSectionProps) {
  const { t } = useTranslation();

  return (
    <section className="landing-grid landing-cta-glow relative overflow-hidden px-6 py-20 text-center">
      <h2 className="relative z-10 mx-auto mb-4 max-w-[680px] text-[clamp(28px,4.5vw,52px)] font-extrabold leading-[1.1] tracking-[-0.04em] text-slate-100">
        {t("home.cta.title")}
      </h2>
      <p className="relative z-10 mb-9 text-[17px] text-slate-400">
        {t("home.cta.subtitle")}
      </p>
      <Button
        size="lg"
        className="relative z-10 px-6 text-[15px]"
        onClick={isAuthenticated ? onGoPlatform : onRegister}
      >
        {t("home.cta.button")} &rarr;
      </Button>
    </section>
  );
}
