import { useState } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { Button } from "@/shared/ui/Button";
import { PasswordInput } from "@/shared/ui/PasswordInput";
import { Label } from "@/shared/ui/Label";
import { ApiError } from "@/services/api";
import {
  MAX_PASSWORD_BYTES,
  MIN_PASSWORD_CHARS,
  passwordByteLength,
  passwordCharLength,
} from "@/shared/lib/validation";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";

/**
 * What the API will accept as a reset code: exactly six characters, which is
 * what `POST /auth/reset-password` binds with `len=6` and what the email sends.
 * Checked here so a truncated or mangled URL is named as a bad link rather than
 * costing the customer a rejected round-trip against one of their few attempts.
 */
const RESET_CODE_LENGTH = 6;

export default function ResetPassword() {
  const { t } = useTranslation();
  usePageMeta({ noindex: true });
  const [searchParams] = useSearchParams();
  const email = searchParams.get("email") ?? "";
  const code = searchParams.get("code") ?? "";
  // The request carries both, and the server requires both. A URL with only
  // one of them cannot reset anything, so it is a broken link, not a form.
  const hasUsableLink = email !== "" && code.length === RESET_CODE_LENGTH;

  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [errors, setErrors] = useState<{
    password?: string;
    confirmPassword?: string;
  }>({});

  // Both password rejections belong on the password field, not in the banner
  // above the form — and both get localized copy, because the backend can only
  // answer in English. `error.message` used to be preferred for
  // INVALID_PASSWORD, which put "password must be at least 8 characters" into
  // the middle of an otherwise Russian or Ukrainian page. It is the same rule
  // the form already states; it is said in the customer's language.
  const resetMutation = useMutation({
    mutationFn: authService.resetPassword,
    onError: (error: ApiError) => {
      if (error.code === "PASSWORD_TOO_LONG") {
        setErrors({ password: t("errors.passwordTooLong") });
      } else if (error.code === "INVALID_PASSWORD") {
        setErrors({ password: t("errors.passwordTooShort") });
      }
    },
  });

  const isPasswordFieldError = (error: unknown): boolean =>
    error instanceof ApiError &&
    (error.code === "INVALID_PASSWORD" || error.code === "PASSWORD_TOO_LONG");

  if (!hasUsableLink) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="flex w-full max-w-md flex-col items-center gap-4 text-center">
          <XCircle className="h-12 w-12 text-destructive" />
          <h1 className="text-2xl font-bold">{t("auth.invalidResetLink")}</h1>
          {/* The way out, not just the way home: this is the same flow that
              issues the code, so a customer with a broken link can get a
              working one without hunting for it. */}
          <Button asChild>
            <Link to="/forgot-password">{t("auth.sendResetCode")}</Link>
          </Button>
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
    } else if (passwordCharLength(password) < MIN_PASSWORD_CHARS) {
      // Code points, like the server: `.length` counts UTF-16 units and reads
      // four emoji as eight characters.
      newErrors.password = t("errors.passwordTooShort");
    } else if (passwordByteLength(password) > MAX_PASSWORD_BYTES) {
      // bcrypt's limit is in bytes: 40 Cyrillic characters are 80 bytes and
      // would be refused by the server after a pointless round-trip.
      newErrors.password = t("errors.passwordTooLong");
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
      resetMutation.mutate({ email, code, password });
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md">
        <h1 className="mb-2 text-2xl font-bold">
          {t("auth.resetPasswordTitle")}
        </h1>
        <p className="mb-6 text-muted-foreground">
          {t("auth.resetPasswordDescription")}
        </p>

        {resetMutation.isError &&
          !isPasswordFieldError(resetMutation.error) && (
            <div
              role="alert"
              className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
            >
              {(resetMutation.error as ApiError).message ||
                t("errors.somethingWentWrong")}
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
                autoComplete="new-password"
                aria-invalid={!!errors.password}
                aria-describedby={
                  errors.password ? "new-password-error" : undefined
                }
              />
              {errors.password && (
                <p
                  id="new-password-error"
                  role="alert"
                  className="text-sm text-destructive"
                >
                  {errors.password}
                </p>
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
                autoComplete="new-password"
                aria-invalid={!!errors.confirmPassword}
                aria-describedby={
                  errors.confirmPassword
                    ? "confirm-new-password-error"
                    : undefined
                }
              />
              {errors.confirmPassword && (
                <p
                  id="confirm-new-password-error"
                  role="alert"
                  className="text-sm text-destructive"
                >
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
