import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { registerSchema, type RegisterFormData } from "@/shared/lib/validation";
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

interface RegisterModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSwitchToLogin: () => void;
}

function ModalContent({
  onOpenChange,
  onSwitchToLogin,
}: Omit<RegisterModalProps, "open">) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const setAuth = useAuthStore((state) => state.setAuth);
  const [registeredEmail, setRegisteredEmail] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [codeError, setCodeError] = useState("");

  const {
    register,
    handleSubmit,
    getValues,
    setError,
    formState: { errors },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
  });

  const registerMutation = useMutation({
    mutationFn: authService.register,
    onSuccess: () => {
      setRegisteredEmail(getValues("email"));
    },
    onError: (error: ApiError) => {
      setError("email", { message: error.message });
    },
  });

  const resend = useResendCode({
    mutationFn: () =>
      authService.resendVerification({ email: registeredEmail! }),
    storageKey: `verify_${getValues("email")}`,
  });

  const loginMutation = useMutation({
    mutationFn: authService.login,
    onSuccess: (data) => {
      setAuth(data.user);
      onOpenChange(false);
      navigate("/app");
    },
  });

  const verifyMutation = useMutation({
    mutationFn: authService.verifyEmail,
    onSuccess: () => {
      const { email, password } = getValues();
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

  const onSubmit = (data: RegisterFormData) => {
    registerMutation.mutate({ email: data.email, password: data.password });
  };

  const handleVerify = (e: React.FormEvent) => {
    e.preventDefault();
    setCodeError("");

    if (code.length !== 6) {
      setCodeError(t("auth.invalidCode"));
      return;
    }

    verifyMutation.mutate({ email: registeredEmail!, code });
  };

  if (registeredEmail) {
    return (
      <>
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold">
            {t("auth.enterCode")}
          </DialogTitle>
        </DialogHeader>
        <div className="flex flex-col items-center gap-4 py-4 text-center">
          <Mail className="h-12 w-12 text-primary" />
          <p className="text-muted-foreground">
            {t("auth.verificationCodeSent", { email: registeredEmail })}
          </p>
          <form onSubmit={handleVerify} className="w-full space-y-4">
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
              className="text-center text-2xl font-mono tracking-[0.3em]"
              aria-invalid={!!codeError}
            />
            {codeError && (
              <p className="text-sm text-destructive">{codeError}</p>
            )}
            <Button
              type="submit"
              className="w-full"
              disabled={
                verifyMutation.isPending ||
                loginMutation.isPending ||
                code.length !== 6
              }
            >
              {verifyMutation.isPending || loginMutation.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : null}
              {loginMutation.isPending
                ? t("common.loading")
                : t("auth.verifyCode")}
            </Button>
          </form>
          {resend.isLimitReached ? (
            <p className="text-sm text-destructive">
              {t("auth.resendLimitReached")}
            </p>
          ) : (
            <>
              <Button
                variant="outline"
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
              {resend.isSuccess && (
                <p className="text-sm text-green-600">{t("auth.codeResent")}</p>
              )}
              {resend.resendError && (
                <p className="text-sm text-destructive">
                  {t(resend.resendError)}
                </p>
              )}
            </>
          )}
          <button
            type="button"
            onClick={onSwitchToLogin}
            className="text-sm font-medium text-primary hover:underline"
          >
            {t("auth.backToLogin")}
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle className="text-2xl font-bold">
          {t("auth.register")}
        </DialogTitle>
        <DialogDescription>{t("auth.registerDescription")}</DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit(onSubmit)} className="mt-4">
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="register-email">{t("auth.email")}</Label>
            <Input
              id="register-email"
              type="email"
              placeholder="you@example.com"
              {...register("email")}
              aria-invalid={!!errors.email}
              aria-describedby={
                errors.email ? "register-email-error" : undefined
              }
            />
            {errors.email && (
              <p id="register-email-error" className="text-sm text-destructive">
                {t(errors.email.message ?? "")}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="register-password">{t("auth.password")}</Label>
            <PasswordInput
              id="register-password"
              {...register("password")}
              aria-invalid={!!errors.password}
              aria-describedby={
                errors.password ? "register-password-error" : undefined
              }
            />
            {errors.password && (
              <p
                id="register-password-error"
                className="text-sm text-destructive"
              >
                {t(errors.password.message ?? "")}
              </p>
            )}
          </div>
          <div className="space-y-2">
            <Label htmlFor="register-confirmPassword">
              {t("auth.confirmPassword")}
            </Label>
            <PasswordInput
              id="register-confirmPassword"
              {...register("confirmPassword")}
              aria-invalid={!!errors.confirmPassword}
              aria-describedby={
                errors.confirmPassword
                  ? "register-confirmPassword-error"
                  : undefined
              }
            />
            {errors.confirmPassword && (
              <p
                id="register-confirmPassword-error"
                className="text-sm text-destructive"
              >
                {t(errors.confirmPassword.message ?? "")}
              </p>
            )}
          </div>
        </div>
        <div className="mt-6 flex flex-col gap-4">
          <Button
            type="submit"
            className="w-full"
            disabled={registerMutation.isPending}
          >
            {registerMutation.isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                {t("common.loading")}
              </>
            ) : (
              t("auth.register")
            )}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            {t("auth.alreadyHaveAccount")}{" "}
            <button
              type="button"
              onClick={onSwitchToLogin}
              className="font-medium text-primary hover:underline"
            >
              {t("auth.login")}
            </button>
          </p>
        </div>
      </form>
    </>
  );
}

export function RegisterModal({
  open,
  onOpenChange,
  onSwitchToLogin,
}: RegisterModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent onClose={() => onOpenChange(false)}>
        <ModalContent
          key={open ? "open" : "closed"}
          onOpenChange={onOpenChange}
          onSwitchToLogin={onSwitchToLogin}
        />
      </DialogContent>
    </Dialog>
  );
}
