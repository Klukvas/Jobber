import { useEffect, useState } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { authService } from "@/services/authService";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";
import { Button } from "@/shared/ui/Button";

type Status = "loading" | "success" | "error";

export default function VerifyEmail() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token");
  const [status, setStatus] = useState<Status>(() =>
    token ? "loading" : "error",
  );

  useEffect(() => {
    if (!token) return;

    authService
      .verifyEmail({ token })
      .then(() => setStatus("success"))
      .catch(() => setStatus("error"));
  }, [token]);

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md text-center">
        {status === "loading" && (
          <div className="flex flex-col items-center gap-4">
            <Loader2 className="h-12 w-12 animate-spin text-primary" />
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
            <Button asChild>
              <Link to="/login">{t("auth.login")}</Link>
            </Button>
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
            <Button variant="outline" asChild>
              <Link to="/">{t("common.backToHome")}</Link>
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
