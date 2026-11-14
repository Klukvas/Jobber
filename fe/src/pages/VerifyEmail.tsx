import { useEffect, useState } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { authService } from "@/services/authService";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";
import { buttonVariants } from "@/shared/ui/buttonVariants";

type Status = "loading" | "success" | "error";

export default function VerifyEmail() {
  const { t } = useTranslation();
  usePageMeta({ noindex: true });
  const [searchParams] = useSearchParams();
  const email = searchParams.get("email") ?? "";
  const code = searchParams.get("code") ?? "";
  const [status, setStatus] = useState<Status>(() =>
    code ? "loading" : "error",
  );

  useEffect(() => {
    if (!code) return;

    authService
      .verifyEmail({ email, code })
      .then(() => setStatus("success"))
      .catch(() => setStatus("error"));
  }, [email, code]);

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md text-center">
        {status === "loading" && (
          <div className="flex flex-col items-center gap-4">
            <Loader2 className="h-12 w-12 animate-spin text-primary-ink" />
            <p className="text-lg text-muted-foreground">
              {t("auth.verifyingEmail")}
            </p>
          </div>
        )}

        {status === "success" && (
          <div className="flex flex-col items-center gap-4">
            <CheckCircle2 className="h-12 w-12 text-green-600" />
            <h1 className="text-2xl font-bold">{t("auth.emailVerified")}</h1>
            <p className="text-muted-foreground">
              {t("auth.emailVerifiedDescription")}
            </p>
            <Link to="/login" className={buttonVariants()}>
              {t("auth.login")}
            </Link>
          </div>
        )}

        {status === "error" && (
          <div className="flex flex-col items-center gap-4">
            <XCircle className="h-12 w-12 text-destructive" />
            <h1 className="text-2xl font-bold">
              {t("auth.verificationFailed")}
            </h1>
            <p className="text-muted-foreground">
              {t("auth.verificationFailedDescription")}
            </p>
            <Link to="/" className={buttonVariants({ variant: "outline" })}>
              {t("common.backToHome")}
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
