import { useTranslation } from 'react-i18next';

const STEPS = ['step1', 'step2', 'step3', 'step4'] as const;

export function HowItWorksSection() {
  const { t } = useTranslation();

  return (
    <section id="how-it-works" className="py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4">
        <div className="mb-16 text-center">
          <h2 className="mb-4 text-3xl font-bold sm:text-4xl">
            {t('home.howItWorks.title')}
          </h2>
          <p className="mx-auto max-w-2xl text-muted-foreground">
            {t('home.howItWorks.subtitle')}
          </p>
        </div>

        <div className="relative grid gap-8 md:grid-cols-4">
          {/* Connecting line */}
          <div className="absolute left-8 top-8 hidden h-0.5 w-[calc(100%-4rem)] bg-border md:block" />

          {STEPS.map((step, index) => (
            <div key={step} className="relative flex flex-col items-center text-center">
              <div className="relative z-10 mb-4 flex h-16 w-16 items-center justify-center rounded-full border-2 border-primary bg-background text-2xl font-bold text-primary">
                {index + 1}
              </div>
              <h3 className="mb-2 text-lg font-semibold">
                {t(`home.howItWorks.${step}.title`)}
              </h3>
              <p className="text-sm text-muted-foreground">
                {t(`home.howItWorks.${step}.description`)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
