import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { Briefcase } from "lucide-react";

export function FooterSection() {
  const { t } = useTranslation();
  const year = new Date().getFullYear();

  return (
    <footer className="border-t py-8">
      <div className="mx-auto max-w-6xl px-4">
        <div className="flex flex-col gap-8 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex items-center gap-2">
            <Briefcase className="h-5 w-5 text-primary" />
            <span className="font-semibold">Jobber</span>
            <span className="text-sm text-muted-foreground">
              — {t("home.footer.tagline")}
            </span>
          </div>

          <div className="flex flex-wrap gap-8">
            <div>
              <p className="mb-2 text-sm font-semibold">
                {t("home.footer.featuresTitle")}
              </p>
              <div className="flex flex-col gap-1">
                <Link
                  to="/features/applications"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("home.footer.featureApplications")}
                </Link>
                <Link
                  to="/features/resume-builder"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("home.footer.featureResumeBuilder")}
                </Link>
                <Link
                  to="/features/cover-letters"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("home.footer.featureCoverLetters")}
                </Link>
              </div>
            </div>

            <div>
              <p className="mb-2 text-sm font-semibold">
                {t("home.footer.resourcesTitle")}
              </p>
              <div className="flex flex-col gap-1">
                <Link
                  to="/blog"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("blog.title")}
                </Link>
                <Link
                  to="/terms"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("home.footer.terms")}
                </Link>
                <Link
                  to="/privacy"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("home.footer.privacy")}
                </Link>
                <Link
                  to="/refund"
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {t("home.footer.refund")}
                </Link>
              </div>
            </div>
          </div>
        </div>

        <div className="mt-8 border-t pt-4 text-center text-sm text-muted-foreground sm:text-left">
          &copy; {year} {t("home.footer.copyright")}
        </div>
      </div>
    </footer>
  );
}
