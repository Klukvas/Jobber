import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { useAuthStore } from "@/stores/authStore";
import { useResendCode } from "@/features/auth/hooks/useResendCode";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { PasswordInput } from "@/shared/ui/PasswordInput";
import { Label } from "@/shared/ui/Label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { ApiError } from "@/services/api";
import { Loader2, Mail } from "lucide-react";

interface LoginModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSwitchToRegister: () => void;
  onForgotPassword?: () => void;
}

function ModalContent({
  onOpenChange,
  onSwitchToRegister,
  onForgotPassword,
}: Omit<LoginModalProps, "open">) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setAuth = useAuthStore((state) => state.setAuth);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [emailNotVerified, setEmailNotVerified] = useState(false);
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>(
    {},
  );

  const loginMutation = useMutation({
    mutationFn: authService.login,
    onSuccess: (data) => {
      setAuth(data.user);
      onOpenChange(false);
      navigate("/app");
    },
    onError: (error: ApiError) => {
      if (error.code === "EMAIL_NOT_VERIFIED") {
        setEmailNotVerified(true);
      } else {
        setErrors({ password: error.message });
      }
    },
  });

  const resend = useResendCode({
    mutationFn: () => authService.resendVerification({ email }),
    storageKey: `verify_${email}`,
  });

  const verifyMutation = useMutation({
    mutationFn: authService.verifyEmail,
    onSuccess: () => {
      setEmailNotVerified(false);
      setCode("");
      setCodeError("");
      // Auto-login after verification
      loginMutation.mutate({ email, password });
    },
    onError: (error: ApiError) => {
      if (error.code === "TOO_MANY_ATTEMPTS") {
        setCodeError(t("auth.tooManyAttempts"));
      } else {
        setCodeError(t("auth.invalidCode"));
      }
    },
  });

  const validate = () => {
    const newErrors: { email?: string; password?: string } = {};

    if (!email) {
      newErrors.email = t("errors.required");
    } else if (!/\S+@\S+\.\S+/.test(email)) {
      newErrors.email = t("errors.invalidEmail");
    }

    if (!password) {
      newErrors.password = t("errors.required");
    } else if (password.length < 8) {
      newErrors.password = t("errors.passwordTooShort");
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setEmailNotVerified(false);
    if (validate()) {
      loginMutation.mutate({ email, password });
    }
  };

  const handleVerify = (e: React.FormEvent) => {
    e.preventDefault();
    setCodeError("");
    if (code.length !== 6) {
      setCodeError(t("auth.invalidCode"));
      return;
    }
    verifyMutation.mutate({ email, code });
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-2xl font-bold">
          {t("auth.login")}
        </DialogTitle>
        <DialogDescription>{t("auth.loginDescription")}</DialogDescription>
      </DialogHeader>

      {emailNotVerified && (
        <div className="rounded-md border border-yellow-300 bg-yellow-50 p-3 dark:border-yellow-700 dark:bg-yellow-950">
          <div className="flex items-start gap-2">
            <Mail className="mt-0.5 h-4 w-4 text-yellow-600 dark:text-yellow-400" />
            <div className="w-full space-y-3 text-sm">
              <p className="font-medium text-yellow-800 dark:text-yellow-200">
                {t("auth.emailNotVerified")}
              </p>
              <form onSubmit={handleVerify} className="space-y-2">
                <Input
                  value={code}
                  onChange={(e) => {
                    const val = e.target.value.replace(/\D/g, "").slice(0, 6);
                    setCode(val);
                    setCodeError("");
                  }}
                  placeholder="000000"
                  inputMode="numeric"
                  maxLength={6}
                  autoFocus
                  className="text-center text-lg font-mono tracking-[0.3em]"
                  aria-invalid={!!codeError}
                />
                {codeError && (
                  <p className="text-sm text-destructive">{codeError}</p>
                )}
                <Button
                  type="submit"
                  size="sm"
                  className="w-full"
                  disabled={verifyMutation.isPending || code.length !== 6}
                >
                  {verifyMutation.isPending ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : null}
                  {t("auth.verifyCode")}
                </Button>
              </form>
              {resend.isLimitReached ? (
                <p className="text-sm text-destructive">
                  {t("auth.resendLimitReached")}
                </p>
              ) : (
                <>
                  <Button
                    variant="link"
                    size="sm"
                    className="h-auto p-0 text-yellow-700 dark:text-yellow-300"
                    onClick={resend.resend}
                    disabled={resend.isDisabled}
                  >
                    {resend.cooldown > 0
                      ? t("auth.resendCodeCooldown", {
                          seconds: resend.cooldown,
                        })
                      : t("auth.resendCode")}
                  </Button>
                  {resend.isSuccess && (
                    <p className="text-green-600 dark:text-green-400">
                      {t("auth.codeResent")}
                    </p>
                  )}
                  {resend.resendError && (
                    <p className="text-sm text-destructive">
                      {t(resend.resendError)}
                    </p>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
      )}

      <form onSubmit={handleSubmit} className="mt-4">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="login-email">{t("auth.email")}</Label>
            <Input
              id="login-email"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              aria-invalid={!!errors.email}
              aria-describedby={errors.email ? "login-email-error" : undefined}
            />
            {errors.email && (
              <p id="login-email-error" className="text-sm text-destructive">
                {errors.email}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="login-password">{t("auth.password")}</Label>
              {onForgotPassword && (
                <button
                  type="button"
                  onClick={onForgotPassword}
                  className="text-xs font-medium text-primary hover:underline"
                >
                  {t("auth.forgotPassword")}
                </button>
              )}
            </div>
            <PasswordInput
              id="login-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-invalid={!!errors.password}
              aria-describedby={
                errors.password ? "login-password-error" : undefined
              }
            />
            {errors.password && (
              <p id="login-password-error" className="text-sm text-destructive">
                {errors.password}
              </p>
            )}
          </div>
        </div>
        <div className="mt-6 flex flex-col gap-4">
          <Button
            type="submit"
            className="w-full"
            disabled={loginMutation.isPending}
          >
            {loginMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t("common.loading")}
              </>
            ) : (
              t("auth.login")
            )}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            {t("auth.dontHaveAccount")}{" "}
            <button
              type="button"
              onClick={onSwitchToRegister}
              className="font-medium text-primary hover:underline"
            >
              {t("auth.register")}
            </button>
          </p>
        </div>
      </form>
    </>
  );
}

export function LoginModal({
  open,
  onOpenChange,
  onSwitchToRegister,
  onForgotPassword,
}: LoginModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <ModalContent
          key={open ? "open" : "closed"}
          onOpenChange={onOpenChange}
          onSwitchToRegister={onSwitchToRegister}
          onForgotPassword={onForgotPassword}
        />
      </DialogContent>
    </Dialog>
  );
}
