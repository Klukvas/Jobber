import { useTranslation } from "react-i18next";
import { useThemeStore } from "@/stores/themeStore";
import { useAuthStore } from "@/stores/authStore";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authService } from "@/services/authService";
import { calendarService } from "@/services/calendarService";
import { subscriptionService } from "@/services/subscriptionService";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useEffect, useState } from "react";
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
import { Info } from "lucide-react";
import { usePageMeta } from "@/shared/lib/usePageMeta";
import { useOnboarding } from "@/features/onboarding/useOnboarding";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { PricingModal } from "@/features/subscription/components/PricingModal";

export default function Settings() {
  const { t, i18n } = useTranslation();
  usePageMeta({ titleKey: "settings.title", noindex: true });
  const { theme, setTheme } = useThemeStore();
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const { restart } = useOnboarding();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const { subscription, isPro, isEnterprise, isFree, nextPlan, usage, limits } =
    useSubscription();
  const [pricingOpen, setPricingOpen] = useState(false);

  const portalMutation = useMutation({
    mutationFn: subscriptionService.createPortalSession,
    onSuccess: (data) => {
      window.open(data.url, "_blank");
    },
    onError: (error: Error) => {
      showErrorNotification(error.message);
    },
  });

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
  const subscriptionParam = searchParams.get("subscription");
  useEffect(() => {
    if (calendarParam === "connected") {
      showSuccessNotification(t("settings.calendar.connectSuccess"));
      queryClient.invalidateQueries({ queryKey: ["calendar-status"] });
      setSearchParams({});
    } else if (calendarParam === "error") {
      showErrorNotification(t("settings.calendar.connectError"));
      setSearchParams({});
    }
    if (subscriptionParam === "success") {
      queryClient.invalidateQueries({ queryKey: ["subscription"] });
      setSearchParams({});
    }
  }, [calendarParam, subscriptionParam, t, queryClient, setSearchParams]);

  const planLabel = isEnterprise
    ? t("settings.subscription.enterprisePlan")
    : isPro
      ? t("settings.subscription.proPlan")
      : t("settings.subscription.freePlan");

  const hasLimits = !isEnterprise;

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
          <CardTitle>{t("settings.subscription.title")}</CardTitle>
          <CardDescription>
            {t("settings.subscription.description")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">
                {t("settings.subscription.currentPlan")}
              </p>
              <p className="font-semibold text-lg">{planLabel}</p>
              {subscription?.status === "past_due" && (
                <p className="text-sm text-amber-600 dark:text-amber-400">
                  {t("settings.subscription.pastDue")}
                </p>
              )}
              {subscription?.cancel_at && (
                <p className="text-sm text-muted-foreground">
                  {t("settings.subscription.cancelledOn", {
                    date: new Date(subscription.cancel_at).toLocaleDateString(),
                  })}
                </p>
              )}
              {isPro &&
                subscription?.current_period_end &&
                !subscription?.cancel_at && (
                  <p className="text-sm text-muted-foreground">
                    {t("settings.subscription.renewsOn", {
                      date: new Date(
                        subscription.current_period_end,
                      ).toLocaleDateString(),
                    })}
                  </p>
                )}
            </div>
            <div className="flex gap-2">
              {isPro && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => portalMutation.mutate()}
                  disabled={portalMutation.isPending}
                >
                  {portalMutation.isPending
                    ? t("common.loading")
                    : t("settings.subscription.managePlan")}
                </Button>
              )}
              {nextPlan && (
                <button
                  className="inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-semibold text-white shadow-sm transition-all bg-gradient-to-r from-blue-600 to-violet-600 hover:from-blue-700 hover:to-violet-700 hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  onClick={() => setPricingOpen(true)}
                  disabled
                >
                  {t("settings.subscription.upgrade")}
                </button>
              )}
            </div>
          </div>

          {/* Payments disabled notice */}
          <div className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
            <Info className="mt-0.5 h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" />
            <p className="text-sm text-amber-800 dark:text-amber-200">
              {t("settings.subscription.paymentsDisabled")}
            </p>
          </div>

          {hasLimits && (
            <div className="space-y-2 pt-2 border-t">
              <p className="text-sm font-medium">
                {t("settings.subscription.usage")}
              </p>
              <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
                <div>
                  <p className="text-muted-foreground">
                    {t("settings.subscription.jobs")}
                  </p>
                  <p className="font-medium">
                    {usage.jobs} {t("settings.subscription.of")}{" "}
                    {limits.max_jobs < 0
                      ? t("settings.subscription.unlimited")
                      : limits.max_jobs}
                  </p>
                </div>
                <div>
                  <p className="text-muted-foreground">
                    {t("settings.subscription.resumes")}
                  </p>
                  <p className="font-medium">
                    {usage.resumes} {t("settings.subscription.of")}{" "}
                    {limits.max_resumes < 0
                      ? t("settings.subscription.unlimited")
                      : limits.max_resumes}
                  </p>
                </div>
                <div>
                  <p className="text-muted-foreground">
                    {t("settings.subscription.applications")}
                  </p>
                  <p className="font-medium">
                    {usage.applications} {t("settings.subscription.of")}{" "}
                    {limits.max_applications < 0
                      ? t("settings.subscription.unlimited")
                      : limits.max_applications}
                  </p>
                </div>
                <div>
                  <p className="text-muted-foreground">
                    {t("settings.subscription.aiRequests")}
                  </p>
                  <p className="font-medium">
                    {usage.ai_requests} {t("settings.subscription.of")}{" "}
                    {limits.max_ai_requests < 0
                      ? t("settings.subscription.unlimited")
                      : limits.max_ai_requests}
                  </p>
                </div>
              </div>
            </div>
          )}
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

      <PricingModal open={pricingOpen} onOpenChange={setPricingOpen} />
    </div>
  );
}
