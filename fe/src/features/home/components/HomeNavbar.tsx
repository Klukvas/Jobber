import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Briefcase } from 'lucide-react';
import { Button } from '@/shared/ui/Button';

interface HomeNavbarProps {
  isAuthenticated: boolean;
  onLogin: () => void;
  onRegister: () => void;
  onGoPlatform: () => void;
}

export function HomeNavbar({ isAuthenticated, onLogin, onRegister, onGoPlatform }: HomeNavbarProps) {
  const { t } = useTranslation();
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener('scroll', handleScroll);
    return () => window.removeEventListener('scroll', handleScroll);
  }, []);

  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' });
  };

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-40 transition-all duration-300 ${
        scrolled
          ? 'bg-background/95 backdrop-blur-md border-b shadow-sm'
          : 'bg-transparent'
      }`}
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <div className="flex items-center gap-2">
          <Briefcase className="h-6 w-6 text-primary" />
          <span className="text-xl font-bold">Jobber</span>
        </div>

        <div className="hidden items-center gap-6 md:flex">
          <button
            onClick={() => scrollTo('features')}
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {t('home.nav.features')}
          </button>
          <button
            onClick={() => scrollTo('how-it-works')}
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {t('home.nav.howItWorks')}
          </button>
        </div>

        <div className="flex items-center gap-2">
          {isAuthenticated ? (
            <Button size="sm" onClick={onGoPlatform}>
              {t('home.hero.ctaGoPlatform')}
            </Button>
          ) : (
            <>
              <Button variant="ghost" size="sm" onClick={onLogin}>
                {t('auth.login')}
              </Button>
              <Button size="sm" onClick={onRegister}>
                {t('auth.register')}
              </Button>
            </>
          )}
        </div>
      </div>
    </nav>
  );
}
