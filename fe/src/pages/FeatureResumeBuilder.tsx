import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowRight, CheckCircle } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { HomeNavbar } from "@/features/home/components/HomeNavbar";
import { FooterSection } from "@/features/home/components/FooterSection";
import { BrowserMockup } from "@/shared/ui/BrowserMockup";
import { useAuthStore } from "@/stores/authStore";
import { usePageMeta } from "@/shared/lib/usePageMeta";

const NS = "featurePages.resumeBuilder";

const FEATURE1_POINTS = ["point1", "point2", "point3"] as const;
const FEATURE2_POINTS = ["point1", "point2", "point3"] as const;

export default function FeatureResumeBuilder() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  usePageMeta({
    titleKey: `${NS}.meta.title`,
    descriptionKey: `${NS}.meta.description`,
  });

  const handleCta = () => {
    if (isAuthenticated) {
      navigate("/app/resumes");
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
        onGoPlatform={() => navigate("/app/resumes")}
      />

      <main className="flex-1">
        {/* Hero */}
        <section className="relative overflow-hidden bg-gradient-to-b from-slate-950 via-slate-900 to-background pb-0 pt-32">
          <div className="pointer-events-none absolute inset-0">
            <div className="absolute -top-24 left-1/4 h-96 w-96 rounded-full bg-lime-500/10 blur-3xl" />
            <div className="absolute right-1/4 top-32 h-64 w-64 rounded-full bg-lime-600/10 blur-3xl" />
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
                url="jobber-app.com/app/resumes"
                src="/screenshots/04_resumes.png"
                alt="Jobber resume builder"
              />
            </div>
          </div>
        </section>

        {/* Stats bar */}
        <section className="border-y bg-muted/30 py-12">
          <div className="mx-auto max-w-3xl px-4">
            <div className="grid grid-cols-3 divide-x text-center">
              <div className="px-6">
                <div className="text-3xl font-bold text-primary">3+</div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {t(`${NS}.stats.templates`)}
                </div>
              </div>
              <div className="px-6">
                <div className="text-3xl font-bold text-primary">ATS</div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {t(`${NS}.stats.atsScore`)}
                </div>
              </div>
              <div className="px-6">
                <div className="text-3xl font-bold text-primary">&lt;5m</div>
                <div className="mt-1 text-sm text-muted-foreground">
                  {t(`${NS}.stats.timeToResume`)}
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Feature 1 — screenshot left, text right */}
        <section className="py-24">
          <div className="mx-auto max-w-6xl px-4">
            <div className="grid items-center gap-16 lg:grid-cols-2">
              <BrowserMockup
                url="jobber-app.com/app/resumes"
                src="/screenshots/04_resumes.png"
                alt="Resume templates"
              />
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
            </div>
          </div>
        </section>

        {/* Feature 2 — text left, visual right */}
        <section className="bg-muted/30 py-24">
          <div className="mx-auto max-w-6xl px-4">
            <div className="grid items-center gap-16 lg:grid-cols-2">
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
              {/* Stacked mini resume thumbnails */}
              <div className="relative flex items-center justify-center">
                <div className="relative h-80 w-full">
                  <div className="absolute left-0 top-4 w-48 overflow-hidden rounded-lg border bg-card shadow-lg">
                    <img
                      src="/screenshots/04_resumes.png"
                      alt="Resume version 1"
                      className="block w-full"
                      loading="lazy"
                      style={{
                        objectFit: "cover",
                        objectPosition: "top left",
                        height: "120px",
                      }}
                    />
                    <div className="px-3 py-2">
                      <div className="h-2 w-24 rounded bg-muted" />
                      <div className="mt-1.5 h-2 w-16 rounded bg-muted/60" />
                    </div>
                  </div>
                  <div className="absolute left-16 top-16 w-48 overflow-hidden rounded-lg border bg-card shadow-lg ring-2 ring-primary/30">
                    <img
                      src="/screenshots/04_resumes.png"
                      alt="Resume version 2"
                      className="block w-full"
                      loading="lazy"
                      style={{
                        objectFit: "cover",
                        objectPosition: "top center",
                        height: "120px",
                      }}
                    />
                    <div className="px-3 py-2">
                      <div className="h-2 w-28 rounded bg-muted" />
                      <div className="mt-1.5 h-2 w-20 rounded bg-muted/60" />
                    </div>
                  </div>
                  <div className="absolute right-0 top-8 w-48 overflow-hidden rounded-lg border bg-card shadow-lg">
                    <img
                      src="/screenshots/04_resumes.png"
                      alt="Resume version 3"
                      className="block w-full"
                      loading="lazy"
                      style={{
                        objectFit: "cover",
                        objectPosition: "top right",
                        height: "120px",
                      }}
                    />
                    <div className="px-3 py-2">
                      <div className="h-2 w-20 rounded bg-muted" />
                      <div className="mt-1.5 h-2 w-14 rounded bg-muted/60" />
                    </div>
                  </div>
                </div>
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
