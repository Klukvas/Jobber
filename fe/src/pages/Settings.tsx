import { useTranslation } from "react-i18next";
import { useThemeStore } from "@/stores/themeStore";
import { useAuthStore } from "@/stores/authStore";
import { useMutation } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { useNavigate } from "react-router-dom";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { usePageTitle } from "@/shared/lib/usePageTitle";

export default function Settings() {
  const { t, i18n } = useTranslation();
  usePageTitle("settings.title");
  const { theme, setTheme } = useThemeStore();
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const navigate = useNavigate();

  const logoutMutation = useMutation({
    mutationFn: authService.logout,
    onSettled: () => {
      clearAuth();
      navigate("/");
    },
  });

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">{t("settings.title")}</h1>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.theme")}</CardTitle>
          <CardDescription>{t("settings.themeDescription")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-4">
            <Button
              variant={theme === "light" ? "default" : "outline"}
              onClick={() => setTheme("light")}
            >
              {t("settings.light")}
            </Button>
            <Button
              variant={theme === "dark" ? "default" : "outline"}
              onClick={() => setTheme("dark")}
            >
              {t("settings.dark")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.language")}</CardTitle>
          <CardDescription>{t("settings.languageDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex gap-4">
            <Button
              variant={i18n.language === "en" ? "default" : "outline"}
              onClick={() => i18n.changeLanguage("en")}
            >
              {t("settings.english")}
            </Button>
            <Button
              variant={i18n.language === "ua" ? "default" : "outline"}
              onClick={() => i18n.changeLanguage("ua")}
            >
              {t("settings.ukrainian")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.account")}</CardTitle>
          <CardDescription>{t("settings.accountDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            variant="destructive"
            onClick={() => logoutMutation.mutate()}
            disabled={logoutMutation.isPending}
          >
            {logoutMutation.isPending ? t("common.loading") : t("auth.logout")}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
