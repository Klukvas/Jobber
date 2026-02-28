import { useTranslation } from 'react-i18next';
import { Briefcase } from 'lucide-react';

export function FooterSection() {
  const { t } = useTranslation();
  const year = new Date().getFullYear();

  return (
    <footer className="border-t py-8">
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-4 px-4 text-center sm:flex-row sm:justify-between sm:text-left">
        <div className="flex items-center gap-2">
          <Briefcase className="h-5 w-5 text-primary" />
          <span className="font-semibold">Jobber</span>
          <span className="text-sm text-muted-foreground">
            — {t('home.footer.tagline')}
          </span>
        </div>
        <p className="text-sm text-muted-foreground">
          &copy; {year} {t('home.footer.copyright')}
        </p>
      </div>
    </footer>
  );
}
