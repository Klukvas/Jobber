import { useTranslation } from 'react-i18next';

const MOCK_COLUMNS = [
  {
    title: 'Applied',
    color: 'bg-blue-500',
    cards: ['Frontend Dev @ Stripe', 'Backend Dev @ Vercel'],
  },
  {
    title: 'Interview',
    color: 'bg-yellow-500',
    cards: ['Full Stack @ GitHub'],
  },
  {
    title: 'Offer',
    color: 'bg-green-500',
    cards: ['SRE @ Cloudflare'],
  },
  {
    title: 'Rejected',
    color: 'bg-red-500',
    cards: ['DevOps @ Netflix'],
  },
];

export function DemoPreviewSection() {
  const { t } = useTranslation();

  return (
    <section className="bg-muted/30 py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4">
        <div className="mb-12 text-center">
          <h2 className="mb-4 text-3xl font-bold sm:text-4xl">
            {t('home.demo.title')}
          </h2>
          <p className="mx-auto max-w-2xl text-muted-foreground">
            {t('home.demo.subtitle')}
          </p>
        </div>

        {/* Browser-window mockup */}
        <div className="mx-auto max-w-4xl overflow-hidden rounded-xl border bg-card shadow-2xl">
          {/* Title bar */}
          <div className="flex items-center gap-2 border-b bg-muted/50 px-4 py-3">
            <div className="flex gap-1.5">
              <div className="h-3 w-3 rounded-full bg-red-400" />
              <div className="h-3 w-3 rounded-full bg-yellow-400" />
              <div className="h-3 w-3 rounded-full bg-green-400" />
            </div>
            <div className="mx-auto rounded-md bg-background px-4 py-1 text-xs text-muted-foreground">
              jobber.app/applications
            </div>
          </div>

          {/* Kanban board mockup */}
          <div className="grid grid-cols-2 gap-3 p-4 sm:grid-cols-4">
            {MOCK_COLUMNS.map((col) => (
              <div key={col.title} className="space-y-2">
                <div className="flex items-center gap-2 px-1">
                  <div className={`h-2 w-2 rounded-full ${col.color}`} />
                  <span className="text-xs font-medium">{col.title}</span>
                  <span className="text-xs text-muted-foreground">
                    {col.cards.length}
                  </span>
                </div>
                {col.cards.map((card) => (
                  <div
                    key={card}
                    className="rounded-lg border bg-background p-2 text-xs shadow-sm sm:p-3"
                  >
                    {card}
                  </div>
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
