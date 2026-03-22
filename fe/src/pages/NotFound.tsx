import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Briefcase, FileQuestion } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { Button } from "@/shared/ui/Button";

export default function NotFound() {
  const { t } = useTranslation();
  usePageMeta({ title: `404 — Jobber` });

  return (
    <div className="mx-auto flex min-h-[80vh] max-w-3xl flex-col px-4 py-12 text-foreground">
      <nav className="mb-8 flex items-center justify-between">
        <Link
          to="/"
          className="flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("common.backToHome")}
        </Link>
        <Link to="/" className="flex items-center gap-2">
          <Briefcase className="h-5 w-5 text-primary" />
          <span className="font-bold">Jobber</span>
        </Link>
      </nav>

      <div className="flex flex-1 flex-col items-center justify-center text-center">
        <div className="mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-muted">
          <FileQuestion className="h-10 w-10 text-muted-foreground" />
        </div>

        <h1 className="mb-2 text-6xl font-bold tracking-tight text-primary">
          404
        </h1>

        <h2 className="mb-3 text-2xl font-semibold">{t("notFound.title")}</h2>

        <p className="mb-8 max-w-md text-muted-foreground">
          {t("notFound.description")}
        </p>

        <div className="flex gap-3">
          <Button asChild>
            <Link to="/">{t("common.backToHome")}</Link>
          </Button>
          <Button variant="outline" asChild>
            <Link to="/app/applications">{t("notFound.goToApp")}</Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
