import { NavLink, Link, useNavigate, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { useSidebarStore } from "@/stores/sidebarStore";
import { useAuthStore } from "@/stores/authStore";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { authService } from "@/services/authService";
import { useOnboardingHighlight } from "@/features/onboarding/useOnboarding";
import { cn } from "@/shared/lib/utils";
import {
  Briefcase,
  FileText,
  Building2,
  Search,
  ListOrdered,
  ChevronLeft,
  ChevronRight,
  X,
  BarChart3,
  LogOut,
  Mail,
} from "lucide-react";

const navItems = [
  { path: "/app/applications", icon: Briefcase, labelKey: "nav.applications" },
  { path: "/app/resumes", icon: FileText, labelKey: "nav.resumes" },
  { path: "/app/companies", icon: Building2, labelKey: "nav.companies" },
  { path: "/app/jobs", icon: Search, labelKey: "nav.jobs" },
  { path: "/app/cover-letters", icon: Mail, labelKey: "nav.coverLetters" },
  { path: "/app/stages", icon: ListOrdered, labelKey: "nav.stages" },
  { path: "/app/analytics", icon: BarChart3, labelKey: "nav.analytics" },
];

function getPlanBadge(
  plan: string,
  t: (key: string) => string,
): { label: string; className: string } {
  switch (plan) {
    case "enterprise":
      return {
        label: t("settings.subscription.enterprisePlan"),
        className:
          "bg-lime-100 text-lime-700 dark:bg-lime-900/30 dark:text-lime-300",
      };
    case "pro":
      return {
        label: t("settings.subscription.proPlan"),
        className:
          "bg-lime-100 text-lime-700 dark:bg-lime-900/30 dark:text-lime-300",
      };
    default:
      return {
        label: t("settings.subscription.freePlan"),
        className:
          "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
      };
  }
}

export function Sidebar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { isExpanded, isMobileOpen, toggleExpanded, closeMobile } =
    useSidebarStore();
  const location = useLocation();
  const highlightedPath = useOnboardingHighlight();
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const { plan } = useSubscription();
  const badge = getPlanBadge(plan, t);

  const logoutMutation = useMutation({
    mutationFn: authService.logout,
    onSettled: () => {
      clearAuth();
      navigate("/");
    },
  });

  return (
    <>
      {/* Mobile Overlay */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-background/80 backdrop-blur-sm md:hidden"
          onClick={closeMobile}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed left-0 top-0 z-50 h-screen border-r bg-card transition-all duration-300",
          "md:sticky md:top-0 md:z-0",
          {
            "w-64": isExpanded,
            "w-16": !isExpanded,
            "-translate-x-full md:translate-x-0": !isMobileOpen,
            "translate-x-0": isMobileOpen,
          },
        )}
      >
        <div className="flex h-full flex-col">
          {/* Logo / Brand */}
          <div className="flex h-16 items-center justify-between border-b px-4">
            <Link
              to="/"
              className="flex items-center gap-2 hover:opacity-80 transition-opacity"
            >
              <img
                src="/favicon.svg"
                alt="Jobber"
                className="h-8 w-8 rounded-lg"
              />
              {isExpanded && (
                <div>
                  <span className="text-xl font-bold">Jobber</span>
                  <span className="block text-[10px] text-muted-foreground leading-tight">
                    Powered By FluxLab
                  </span>
                </div>
              )}
            </Link>
            <button
              onClick={toggleExpanded}
              className="hidden rounded-md p-2 hover:bg-accent md:block"
              aria-label={
                isExpanded
                  ? t("common.collapseSidebar")
                  : t("common.expandSidebar")
              }
            >
              {isExpanded ? (
                <ChevronLeft className="h-5 w-5" />
              ) : (
                <ChevronRight className="h-5 w-5" />
              )}
            </button>
            <button
              onClick={closeMobile}
              className="rounded-md p-2 hover:bg-accent md:hidden"
              aria-label={t("common.closeSidebar")}
            >
              <X className="h-5 w-5" />
            </button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 space-y-1 p-2">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isHighlighted = highlightedPath === item.path;
              return (
                <NavLink
                  key={item.path}
                  to={item.path}
                  onClick={() => {
                    if (window.innerWidth < 768) {
                      closeMobile();
                    }
                  }}
                  className={({ isActive }) =>
                    cn(
                      "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                      "hover:bg-accent hover:text-accent-foreground",
                      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                      {
                        "bg-accent text-accent-foreground": isActive,
                        "text-muted-foreground": !isActive,
                        "justify-center": !isExpanded,
                        "bg-primary/10 text-primary ring-2 ring-primary/30 animate-onboarding-pulse":
                          isHighlighted,
                      },
                    )
                  }
                  title={!isExpanded ? t(item.labelKey) : undefined}
                >
                  <Icon className="h-5 w-5 flex-shrink-0" />
                  {isExpanded && <span>{t(item.labelKey)}</span>}
                </NavLink>
              );
            })}
          </nav>

          {/* User Footer */}
          <div className="border-t p-2 space-y-1">
            {/* Email + Plan Badge */}
            {isExpanded ? (
              <NavLink
                to="/app/settings"
                className={cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 transition-colors",
                  "hover:bg-accent hover:text-accent-foreground",
                  {
                    "bg-accent text-accent-foreground":
                      location.pathname === "/app/settings",
                    "text-muted-foreground":
                      location.pathname !== "/app/settings",
                  },
                )}
              >
                <span
                  className="truncate text-sm font-medium"
                  title={user?.email}
                >
                  {user?.email}
                </span>
                <span
                  className={cn(
                    "flex-shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold leading-none",
                    badge.className,
                  )}
                >
                  {badge.label}
                </span>
              </NavLink>
            ) : (
              <NavLink
                to="/app/settings"
                className={cn(
                  "flex justify-center rounded-md px-3 py-2 transition-colors",
                  "hover:bg-accent hover:text-accent-foreground",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  {
                    "bg-accent text-accent-foreground":
                      location.pathname === "/app/settings",
                    "text-muted-foreground":
                      location.pathname !== "/app/settings",
                  },
                )}
                title={user?.email}
              >
                <span
                  className={cn(
                    "rounded-full px-1.5 py-0.5 text-[10px] font-semibold leading-none",
                    badge.className,
                  )}
                >
                  {badge.label.charAt(0)}
                </span>
              </NavLink>
            )}

            {/* Logout */}
            <button
              onClick={() => logoutMutation.mutate()}
              disabled={logoutMutation.isPending}
              className={cn(
                "flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                "text-muted-foreground hover:bg-destructive/10 hover:text-destructive",
                { "justify-center": !isExpanded },
              )}
              title={!isExpanded ? t("auth.logout") : undefined}
            >
              <LogOut className="h-5 w-5 flex-shrink-0" />
              {isExpanded && <span>{t("auth.logout")}</span>}
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}
