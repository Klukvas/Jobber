import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowRight, CheckCircle } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { FooterSection } from "@/features/home/components/FooterSection";
import { BrowserMockup } from "@/shared/ui/BrowserMockup";
import { useAuthStore } from "@/stores/authStore";
import { usePageMeta } from "@/shared/lib/usePageMeta";

const NS = "featurePages.coverLetters";

const FEATURE1_POINTS = ["point1", "point2", "point3"] as const;
const FEATURE2_POINTS = ["point1", "point2", "point3"] as const;

export default function FeatureCoverLetters() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  usePageMeta({
    titleKey: `${NS}.meta.title`,
    descriptionKey: `${NS}.meta.description`,
  });

  const handleCta = () => {
    if (isAuthenticated) {
      navigate("/app/cover-letters");
    } else {
      navigate("/register");
    }
  };

  return (
    <div className="flex min-h-screen flex-col">
      <HomeNavbar
        darkHero
        isAuthenticated={isAuthenticated}
        onLogin={() => navigate("/login")}
        onRegister={() => navigate("/register")}
        onGoPlatform={() => navigate("/app/cover-letters")}
      />

      <main className="flex-1">
        {/* Hero */}
        <section className="relative overflow-hidden bg-gradient-to-b from-slate-950 via-slate-900 to-background pb-0 pt-32">
          <div className="pointer-events-none absolute inset-0">
            <div className="absolute -top-24 right-1/3 h-96 w-96 rounded-full bg-lime-500/10 blur-3xl" />
            <div className="absolute left-1/4 top-32 h-64 w-64 rounded-full bg-lime-600/10 blur-3xl" />
          </div>

          <div className="relative mx-auto max-w-6xl px-4">
            <div className="mx-auto mb-14 max-w-3xl text-center text-white">
              <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs text-slate-400">
                <span className="h-1.5 w-1.5 rounded-full bg-lime-400" />
                {t(`${NS}.hero.badge`)}
              </div>
              <h1 className="mb-5 text-4xl font-bold tracking-tight sm:text-5xl lg:text-6xl">
                {t(`${NS}.hero.title`)}
              </h1>
              <p className="mb-8 text-lg leading-relaxed text-slate-400">
                {t(`${NS}.hero.subtitle`)}
              </p>
              <Button size="lg" onClick={handleCta}>
                {t(`${NS}.hero.cta`)}
                <ArrowRight className="ml-2 h-4 w-4" />
              </Button>
            </div>

            <div className="relative mx-auto max-w-5xl">
              <div className="absolute inset-x-0 -bottom-8 h-40 bg-gradient-to-t from-background to-transparent" />
              <BrowserMockup
                dark
                url="jobber-app.com/app/cover-letters"
                src="/screenshots/07_cover_letters.png"
                alt="Jobber cover letters"
              />
            </div>
          </div>
        </section>

        {/* Stats bar */}
        <section className="border-y bg-muted/30 py-12">
          <div className="mx-auto max-w-3xl px-4">
            <div className="grid grid-cols-3 divide-x text-center">
              <div className="px-6">
                <div className="text-3xl font-bold text-primary">AI</div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {t(`${NS}.stats.generated`)}
                </div>
              </div>
              <div className="px-6">
                <div className="text-3xl font-bold text-primary">&lt;60s</div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {t(`${NS}.stats.timeToWrite`)}
                </div>
              </div>
              <div className="px-6">
                <div className="text-3xl font-bold text-primary">2</div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {t(`${NS}.stats.formats`)}
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Feature 1 — text left, mock letter right */}
        <section className="py-24">
          <div className="mx-auto max-w-6xl px-4">
            <div className="grid items-center gap-16 lg:grid-cols-2">
              <div>
                <p className="mb-3 text-sm font-semibold uppercase tracking-wider text-primary">
                  {t(`${NS}.feature1.label`)}
                </p>
                <h2 className="mb-4 text-3xl font-bold leading-tight">
                  {t(`${NS}.feature1.title`)}
                </h2>
                <p className="mb-8 leading-relaxed text-muted-foreground">
                  {t(`${NS}.feature1.desc`)}
                </p>
                <ul className="space-y-3">
                  {FEATURE1_POINTS.map((p) => (
                    <li key={p} className="flex items-start gap-3">
                      <CheckCircle className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
                      <span className="text-sm text-muted-foreground">
                        {t(`${NS}.feature1.${p}`)}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
              {/* Mock letter card */}
              <div className="rounded-xl border bg-card p-6 shadow-lg">
                <div className="mb-4 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="h-8 w-8 rounded-full bg-primary/10 text-primary flex items-center justify-center text-sm font-bold">
                      AI
                    </div>
                    <div>
                      <div className="text-sm font-semibold">
                        {t(`${NS}.mockLetter.title`)}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {t(`${NS}.mockLetter.company`)}
                      </div>
                    </div>
                  </div>
                  <div className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-600">
                    {t(`${NS}.mockLetter.badge`)}
                  </div>
                </div>
                <div className="space-y-2">
                  <div className="h-2.5 w-full rounded bg-muted" />
                  <div className="h-2.5 w-5/6 rounded bg-muted" />
                  <div className="h-2.5 w-full rounded bg-muted" />
                  <div className="h-2.5 w-4/5 rounded bg-muted" />
                  <div className="mt-4 h-2.5 w-full rounded bg-muted" />
                  <div className="h-2.5 w-full rounded bg-muted" />
                  <div className="h-2.5 w-3/4 rounded bg-muted" />
                  <div className="mt-4 h-2.5 w-full rounded bg-muted" />
                  <div className="h-2.5 w-5/6 rounded bg-muted" />
                  <div className="h-2.5 w-2/3 rounded bg-muted" />
                </div>
                <div className="mt-4 flex gap-2">
                  <div className="rounded bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary">
                    {t(`${NS}.mockLetter.editBtn`)}
                  </div>
                  <div className="rounded bg-muted px-3 py-1.5 text-xs font-medium text-muted-foreground">
                    PDF
                  </div>
                  <div className="rounded bg-muted px-3 py-1.5 text-xs font-medium text-muted-foreground">
                    DOCX
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Feature 2 — screenshot left, text right */}
        <section className="bg-muted/30 py-24">
          <div className="mx-auto max-w-6xl px-4">
            <div className="grid items-center gap-16 lg:grid-cols-2">
              <BrowserMockup
                url="jobber-app.com/app/cover-letters"
                src="/screenshots/07_cover_letters.png"
                alt="Cover letters library"
              />
              <div>
                <p className="mb-3 text-sm font-semibold uppercase tracking-wider text-primary">
                  {t(`${NS}.feature2.label`)}
                </p>
                <h2 className="mb-4 text-3xl font-bold leading-tight">
                  {t(`${NS}.feature2.title`)}
                </h2>
                <p className="mb-8 leading-relaxed text-muted-foreground">
                  {t(`${NS}.feature2.desc`)}
                </p>
                <ul className="space-y-3">
                  {FEATURE2_POINTS.map((p) => (
                    <li key={p} className="flex items-start gap-3">
                      <CheckCircle className="mt-0.5 h-5 w-5 shrink-0 text-primary" />
                      <span className="text-sm text-muted-foreground">
                        {t(`${NS}.feature2.${p}`)}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* CTA */}
        <section className="py-24 text-center">
          <div className="mx-auto max-w-2xl px-4">
            <h2 className="mb-4 text-4xl font-bold">{t(`${NS}.cta.title`)}</h2>
            <p className="mb-8 text-lg text-muted-foreground">
              {t(`${NS}.cta.subtitle`)}
            </p>
            <Button size="lg" onClick={handleCta}>
              {t(`${NS}.cta.button`)}
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </section>
      </main>

      <FooterSection />
    </div>
  );
}
