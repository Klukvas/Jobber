import { useTranslation } from 'react-i18next';
import { Button } from '@/shared/ui/Button';

interface HeroSectionProps {
  isAuthenticated: boolean;
  onRegister: () => void;
  onLogin: () => void;
  onGoPlatform: () => void;
}

export function HeroSection({ isAuthenticated, onRegister, onLogin, onGoPlatform }: HeroSectionProps) {
  const { t } = useTranslation();

  return (
    <section className="relative flex min-h-screen items-center justify-center overflow-hidden px-4">
      {/* Background gradient */}
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-background to-primary/10" />

      {/* Decorative shapes */}
      <div className="absolute -top-40 -right-40 h-80 w-80 rounded-full bg-primary/5 blur-3xl" />
      <div className="absolute -bottom-40 -left-40 h-80 w-80 rounded-full bg-primary/10 blur-3xl" />

      <div className="relative z-10 mx-auto max-w-4xl text-center">
        <h1 className="mb-6 animate-fade-in-up text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl lg:text-7xl">
          {t('home.hero.title')}
        </h1>
        <p className="mx-auto mb-10 max-w-2xl animate-fade-in-up text-lg text-muted-foreground [animation-delay:200ms] sm:text-xl">
          {t('home.hero.subtitle')}
        </p>
        <div className="flex animate-fade-in-up flex-col gap-4 [animation-delay:400ms] sm:flex-row sm:justify-center">
          {isAuthenticated ? (
            <Button size="lg" className="text-base px-8" onClick={onGoPlatform}>
              {t('home.hero.ctaGoPlatform')}
            </Button>
          ) : (
            <>
              <Button size="lg" className="text-base px-8" onClick={onRegister}>
                {t('home.hero.cta')}
              </Button>
              <Button variant="outline" size="lg" className="text-base px-8" onClick={onLogin}>
                {t('home.hero.ctaSecondary')}
              </Button>
            </>
          )}
        </div>
      </div>
    </section>
  );
}
