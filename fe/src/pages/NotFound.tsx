import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ArrowLeft, FileQuestion } from "lucide-react";
import { LogoMark } from "@/shared/ui/Logo";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { buttonVariants } from "@/shared/ui/buttonVariants";

export default function NotFound() {
  const { t } = useTranslation();
  usePageMeta({ title: `404 — Cursopus`, noindex: true });

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
          <LogoMark className="h-6 w-6" />
          <span className="font-bold">Cursopus</span>
        </Link>
      </nav>

      <div className="flex flex-1 flex-col items-center justify-center text-center">
        <div className="mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-muted">
          <FileQuestion className="h-10 w-10 text-muted-foreground" />
        </div>

        <h1 className="mb-2 text-6xl font-bold tracking-tight text-primary-ink">
          404
        </h1>

        <h2 className="mb-3 text-2xl font-semibold">{t("notFound.title")}</h2>

        <p className="mb-8 max-w-md text-muted-foreground">
          {t("notFound.description")}
        </p>

        <div className="flex gap-3">
          <Link to="/" className={buttonVariants()}>
            {t("common.backToHome")}
          </Link>
          <Link to="/app/jobs" className={buttonVariants({ variant: "outline" })}>
            {t("notFound.goToApp")}
          </Link>
        </div>
      </div>
    </div>
  );
}
