import { useState } from "react";
import { useTranslation } from "react-i18next";
import { LoginModal } from "@/features/auth/modals/LoginModal";
import { RegisterModal } from "@/features/auth/modals/RegisterModal";
import { ForgotPasswordModal } from "@/features/auth/modals/ForgotPasswordModal";
import { Button } from "@/shared/ui/Button";

export function ConsentAuth({ onAuthenticated }: { onAuthenticated: () => void }) {
  const { t } = useTranslation();
  const [modal, setModal] = useState<"login" | "register" | "forgot" | null>("login");
  const close = (open: boolean) => { if (!open) setModal(null); };
  // Stay on this URL, including every OAuth parameter, after either auth flow.
  const authenticated = () => {
    setModal(null);
    onAuthenticated();
  };
  return (
    <div className="space-y-4">
      <p>{t("oauth.signInDescription")}</p>
      <Button onClick={() => setModal("login")}>{t("auth.login")}</Button>
      <LoginModal open={modal === "login"} onOpenChange={close} onAuthenticated={authenticated}
        onSwitchToRegister={() => setModal("register")} onForgotPassword={() => setModal("forgot")} />
      <RegisterModal open={modal === "register"} onOpenChange={close} onAuthenticated={authenticated}
        onSwitchToLogin={() => setModal("login")} />
      <ForgotPasswordModal open={modal === "forgot"} onOpenChange={close} onBackToLogin={() => setModal("login")} />
    </div>
  );
}
