import { useTranslation } from "react-i18next";
import { useThemeStore } from "@/stores/themeStore";
import { useAuthStore } from "@/stores/authStore";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { calendarService } from "@/services/calendarService";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useEffect } from "react";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useOnboarding } from "@/features/onboarding/useOnboarding";

export default function Settings() {
  const { t, i18n } = useTranslation();
  usePageMeta({ titleKey: "settings.title", noindex: true });
  const { theme, setTheme } = useThemeStore();
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const { restart } = useOnboarding();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();

  const logoutMutation = useMutation({
    mutationFn: authService.logout,
    onSettled: () => {
      clearAuth();
      navigate("/");
    },
  });

  const { data: calendarStatus } = useQuery({
    queryKey: ["calendar-status"],
    queryFn: calendarService.getStatus,
  });

  const connectMutation = useMutation({
    mutationFn: calendarService.getAuthURL,
    onSuccess: (data) => {
      window.location.href = data.url;
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("settings.calendar.connectError"),
      );
    },
  });

  const disconnectMutation = useMutation({
    mutationFn: calendarService.disconnect,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["calendar-status"] });
      showSuccessNotification(t("settings.calendar.disconnectSuccess"));
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("settings.calendar.disconnectError"),
      );
    },
  });

  // Handle OAuth callback result from URL params
  const calendarParam = searchParams.get("calendar");
  useEffect(() => {
    if (calendarParam === "connected") {
      showSuccessNotification(t("settings.calendar.connectSuccess"));
      queryClient.invalidateQueries({ queryKey: ["calendar-status"] });
      setSearchParams({});
    } else if (calendarParam === "error") {
      showErrorNotification(t("settings.calendar.connectError"));
      setSearchParams({});
    }
  }, [calendarParam, t, queryClient, setSearchParams]);

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
            <Button
              variant={i18n.language === "ru" ? "default" : "outline"}
              onClick={() => i18n.changeLanguage("ru")}
            >
              {t("settings.russian")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.calendar.title")}</CardTitle>
          <CardDescription>
            {t("settings.calendar.description")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {calendarStatus?.connected ? (
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="inline-block h-2 w-2 rounded-full bg-green-500" />
                <span className="text-sm">
                  {t("settings.calendar.connected")}
                  {calendarStatus.email && (
                    <span className="text-muted-foreground ml-1">
                      ({calendarStatus.email})
                    </span>
                  )}
                </span>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => disconnectMutation.mutate()}
                disabled={disconnectMutation.isPending}
              >
                {disconnectMutation.isPending
                  ? t("common.loading")
                  : t("settings.calendar.disconnect")}
              </Button>
            </div>
          ) : (
            <Button
              onClick={() => connectMutation.mutate()}
              disabled={connectMutation.isPending}
            >
              {connectMutation.isPending
                ? t("common.loading")
                : t("settings.calendar.connect")}
            </Button>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.onboarding.title")}</CardTitle>
          <CardDescription>
            {t("settings.onboarding.description")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            variant="outline"
            onClick={() => {
              restart();
              navigate("/app/applications");
            }}
          >
            {t("settings.onboarding.restart")}
          </Button>
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
