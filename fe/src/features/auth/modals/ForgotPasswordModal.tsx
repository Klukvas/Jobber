import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { authService } from "@/services/authService";
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
import { Loader2, CheckCircle2 } from "lucide-react";

interface ForgotPasswordModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onBackToLogin: () => void;
}

type Step = "email" | "code" | "done";

function ModalContent({
  onBackToLogin,
}: Omit<ForgotPasswordModalProps, "open" | "onOpenChange">) {
  const { t } = useTranslation();
  const [step, setStep] = useState<Step>("email");
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [error, setError] = useState("");

  const forgotMutation = useMutation({
    mutationFn: authService.forgotPassword,
    onSuccess: () => setStep("code"),
    onError: () => setStep("code"), // Always advance to prevent enumeration
  });

  const resend = useResendCode({
    mutationFn: () => authService.forgotPassword({ email }),
    storageKey: `reset_${email}`,
  });

  const resetMutation = useMutation({
    mutationFn: authService.resetPassword,
    onSuccess: () => setStep("done"),
    onError: (err: ApiError) => {
      if (err.code === "TOO_MANY_ATTEMPTS") {
        setError(t("auth.tooManyAttempts"));
      } else {
        setError(t("auth.invalidCode"));
      }
    },
  });

  const handleEmailSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!email || !/\S+@\S+\.\S+/.test(email)) {
      setError(t("errors.invalidEmail"));
      return;
    }

    forgotMutation.mutate({ email });
  };

  const handleResetSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (code.length !== 6) {
      setError(t("auth.invalidCode"));
      return;
    }

    if (newPassword.length < 8) {
      setError(t("errors.passwordTooShort"));
      return;
    }

    resetMutation.mutate({ email, code, password: newPassword });
  };

  if (step === "done") {
    return (
      <>
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold">
            {t("auth.passwordResetDone")}
          </DialogTitle>
        </DialogHeader>
        <div className="flex flex-col items-center gap-4 py-6 text-center">
          <CheckCircle2 className="h-12 w-12 text-green-500" />
          <p className="text-muted-foreground">
            {t("auth.passwordResetDoneDescription")}
          </p>
          <Button onClick={onBackToLogin}>{t("auth.login")}</Button>
        </div>
      </>
    );
  }

  if (step === "code") {
    return (
      <>
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold">
            {t("auth.resetPasswordTitle")}
          </DialogTitle>
          <DialogDescription>
            {t("auth.enterResetCode", { email })}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleResetSubmit} className="mt-4 space-y-4">
          <div className="space-y-2">
            <Label>{t("auth.verificationCode")}</Label>
            <Input
              value={code}
              onChange={(e) => {
                const val = e.target.value.replace(/\D/g, "").slice(0, 6);
                setCode(val);
                setError("");
              }}
              placeholder="000000"
              inputMode="numeric"
              maxLength={6}
              autoFocus
              className="text-center text-2xl font-mono tracking-[0.3em]"
              aria-invalid={!!error}
            />
          </div>
          <div className="space-y-2">
            <Label>{t("auth.newPassword")}</Label>
            <PasswordInput
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <Button
            type="submit"
            className="w-full"
            disabled={resetMutation.isPending || code.length !== 6}
          >
            {resetMutation.isPending ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : null}
            {t("auth.resetPassword")}
          </Button>
          {resend.isLimitReached ? (
            <p className="text-center text-sm text-destructive">
              {t("auth.resendLimitReached")}
            </p>
          ) : (
            <>
              <div className="flex justify-center">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={resend.resend}
                  disabled={resend.isDisabled}
                >
                  {resend.isPending ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : null}
                  {resend.cooldown > 0
                    ? t("auth.resendCodeCooldown", {
                        seconds: resend.cooldown,
                      })
                    : t("auth.resendCode")}
                </Button>
              </div>
              {resend.isSuccess && (
                <p className="text-center text-sm text-green-600">
                  {t("auth.codeResent")}
                </p>
              )}
              {resend.resendError && (
                <p className="text-center text-sm text-destructive">
                  {t(resend.resendError)}
                </p>
              )}
            </>
          )}
          <button
            type="button"
            onClick={onBackToLogin}
            className="w-full text-center text-sm font-medium text-primary hover:underline"
          >
            {t("auth.backToLogin")}
          </button>
        </form>
      </>
    );
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-2xl font-bold">
          {t("auth.forgotPasswordTitle")}
        </DialogTitle>
        <DialogDescription>
          {t("auth.forgotPasswordDescription")}
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleEmailSubmit} className="mt-4">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="forgot-email">{t("auth.email")}</Label>
            <Input
              id="forgot-email"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              aria-invalid={!!error}
            />
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
        </div>
        <div className="mt-6 flex flex-col gap-4">
          <Button
            type="submit"
            className="w-full"
            disabled={forgotMutation.isPending}
          >
            {forgotMutation.isPending ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : null}
            {t("auth.sendResetCode")}
          </Button>
          <button
            type="button"
            onClick={onBackToLogin}
            className="text-center text-sm font-medium text-primary hover:underline"
          >
            {t("auth.backToLogin")}
          </button>
        </div>
      </form>
    </>
  );
}

export function ForgotPasswordModal({
  open,
  onOpenChange,
  onBackToLogin,
}: ForgotPasswordModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <ModalContent
          key={open ? "open" : "closed"}
          onBackToLogin={onBackToLogin}
        />
      </DialogContent>
    </Dialog>
  );
}
