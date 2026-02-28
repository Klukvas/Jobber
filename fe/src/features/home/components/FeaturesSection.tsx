import { useTranslation } from 'react-i18next';
import { Clock, CheckCircle, TrendingUp, Layers } from 'lucide-react';

const FEATURES = [
  { key: 'track', icon: Clock, color: 'bg-blue-500/10 text-blue-500' },
  { key: 'organize', icon: CheckCircle, color: 'bg-green-500/10 text-green-500' },
  { key: 'analytics', icon: TrendingUp, color: 'bg-purple-500/10 text-purple-500' },
  { key: 'stages', icon: Layers, color: 'bg-orange-500/10 text-orange-500' },
] as const;

export function FeaturesSection() {
  const { t } = useTranslation();

  return (
    <section id="features" className="bg-muted/30 py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4">
        <div className="mb-16 text-center">
          <h2 className="mb-4 text-3xl font-bold sm:text-4xl">
            {t('home.features.title')}
          </h2>
          <p className="mx-auto max-w-2xl text-muted-foreground">
            {t('home.features.subtitle')}
          </p>
        </div>

        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {FEATURES.map(({ key, icon: Icon, color }) => (
            <div
              key={key}
              className="rounded-xl border bg-card p-6 shadow-sm transition-shadow hover:shadow-md"
            >
              <div className={`mb-4 inline-flex rounded-lg p-3 ${color}`}>
                <Icon className="h-6 w-6" />
              </div>
              <h3 className="mb-2 text-lg font-semibold">
                {t(`home.features.${key}.title`)}
              </h3>
              <p className="text-sm text-muted-foreground">
                {t(`home.features.${key}.description`)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
