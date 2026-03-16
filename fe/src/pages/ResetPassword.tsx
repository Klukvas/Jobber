import { useState } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { Button } from "@/shared/ui/Button";
import { PasswordInput } from "@/shared/ui/PasswordInput";
import { Label } from "@/shared/ui/Label";
import { ApiError } from "@/services/api";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";

export default function ResetPassword() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const token = searchParams.get("token");

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [errors, setErrors] = useState<{
    password?: string;
    confirmPassword?: string;
  }>({});

  const resetMutation = useMutation({
    mutationFn: authService.resetPassword,
    onError: (error: ApiError) => {
      if (error.code === "INVALID_PASSWORD") {
        setErrors({ password: error.message });
      }
    },
  });

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="flex w-full max-w-md flex-col items-center gap-4 text-center">
          <XCircle className="h-12 w-12 text-destructive" />
          <h1 className="text-2xl font-bold">{t("auth.invalidResetLink")}</h1>
          <Button variant="outline" asChild>
            <Link to="/">{t("common.backToHome")}</Link>
          </Button>
        </div>
      </div>
    );
  }

  if (resetMutation.isSuccess) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="flex w-full max-w-md flex-col items-center gap-4 text-center">
          <CheckCircle2 className="h-12 w-12 text-green-600" />
          <h1 className="text-2xl font-bold">{t("auth.passwordResetDone")}</h1>
          <p className="text-muted-foreground">
            {t("auth.passwordResetDoneDescription")}
          </p>
          <Button asChild>
            <Link to="/login">{t("auth.login")}</Link>
          </Button>
        </div>
      </div>
    );
  }

  const validate = () => {
    const newErrors: { password?: string; confirmPassword?: string } = {};

    if (!password) {
      newErrors.password = t("errors.required");
    } else if (password.length < 8) {
      newErrors.password = t("errors.passwordTooShort");
    }

    if (password !== confirmPassword) {
      newErrors.confirmPassword = t("errors.passwordsDontMatch");
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      resetMutation.mutate({ token, password });
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md">
        <h1 className="mb-2 text-2xl font-bold">{t("auth.resetPasswordTitle")}</h1>
        <p className="mb-6 text-muted-foreground">
          {t("auth.resetPasswordDescription")}
        </p>

        {resetMutation.isError &&
          (resetMutation.error as ApiError).code !== "INVALID_PASSWORD" && (
            <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
              {(resetMutation.error as ApiError).message}
            </div>
          )}

        <form onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="new-password">{t("auth.newPassword")}</Label>
              <PasswordInput
                id="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                aria-invalid={!!errors.password}
              />
              {errors.password && (
                <p className="text-sm text-destructive">{errors.password}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm-new-password">
                {t("auth.confirmPassword")}
              </Label>
              <PasswordInput
                id="confirm-new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                aria-invalid={!!errors.confirmPassword}
              />
              {errors.confirmPassword && (
                <p className="text-sm text-destructive">
                  {errors.confirmPassword}
                </p>
              )}
            </div>
          </div>
          <Button
            type="submit"
            className="mt-6 w-full"
            disabled={resetMutation.isPending}
          >
            {resetMutation.isPending ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : null}
            {t("auth.resetPassword")}
          </Button>
        </form>
      </div>
    </div>
  );
}
