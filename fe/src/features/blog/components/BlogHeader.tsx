import { useTranslation } from 'react-i18next';
import { BookOpen } from 'lucide-react';

export function BlogHeader() {
  const { t } = useTranslation();

  return (
    <div className="mb-12 text-center">
      <div className="mb-4 inline-flex items-center justify-center rounded-full bg-primary/10 p-3">
        <BookOpen className="h-6 w-6 text-primary" />
      </div>
      <h1 className="text-4xl font-bold tracking-tight">
        {t('blog.title')}
      </h1>
      <p className="mt-3 text-lg text-muted-foreground">
        {t('blog.subtitle')}
      </p>
      <div className="mx-auto mt-6 h-px w-24 bg-gradient-to-r from-transparent via-primary/50 to-transparent" />
    </div>
  );
}
