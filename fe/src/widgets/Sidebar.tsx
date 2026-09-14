import { useCallback, useRef } from "react";
import { NavLink, Link, useNavigate, useLocation } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useMutation } from "@tanstack/react-query";
import { useSidebarStore } from "@/stores/sidebarStore";
import { useAuthStore } from "@/stores/authStore";
import { endSession } from "@/shared/lib/session";
import { useThemeStore } from "@/stores/themeStore";
import { useBodyScrollLock } from "@/shared/hooks/useBodyScrollLock";
import { useDialogFocus } from "@/shared/hooks/useDialogFocus";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { useTopmostEscape } from "@/shared/hooks/useTopmostEscape";
import { useSubscription } from "@/shared/hooks/useSubscription";
import { authService } from "@/services/authService";
import { useOnboardingHighlight } from "@/features/onboarding/useOnboarding";
import { cn } from "@/shared/lib/utils";
import {
  Briefcase,
  FileText,
  Building2,
  ListOrdered,
  ChevronLeft,
  ChevronRight,
  X,
  BarChart3,
  LogOut,
  Mail,
} from "lucide-react";

/**
 * A 44px floor for the sidebar's full-width rows while it is the phone
 * slide-over, released from `md` up where the sidebar is docked and driven by a
 * pointer. Height only — these rows already span the panel.
 */
const MOBILE_ROW = "min-h-11 md:min-h-0";

/**
 * The panel the header's hamburger opens, named so `aria-controls` has an
 * IDREF that resolves in both states. The panel is always in the DOM — closed
 * on a phone it is pushed off-screen and made `invisible` — so the name is
 * never dangling.
 */
export const APP_SIDEBAR_ID = "app-sidebar";

/**
 * Below this width the sidebar is a slide-over drawn over the page; from `md`
 * up it is docked beside it. Kept in step with the `md:` classes on the panel
 * itself — the drawer only behaves like a modal while it actually is one, so a
 * visitor who widens the window with it open is not left with a trapped
 * keyboard and a frozen page next to a permanently docked sidebar.
 */
const MOBILE_VIEWPORT = "(max-width: 767px)";

const navItems = [
  { path: "/app/jobs", icon: Briefcase, labelKey: "nav.applications" },
  { path: "/app/resumes", icon: FileText, labelKey: "nav.resumes" },
  { path: "/app/companies", icon: Building2, labelKey: "nav.companies" },
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
  const panelRef = useRef<HTMLElement>(null);
  const isSlideOver = useMediaQuery(MOBILE_VIEWPORT);
  // The drawer covers the page it is navigating, so while it is up it is a
  // modal in everything but name: nothing behind it scrolls, Tab cannot walk
  // out into the page underneath, Escape dismisses it, and the hamburger that
  // opened it gets the keyboard back.
  const isModal = isSlideOver && isMobileOpen;
  const location = useLocation();
  const highlightedPath = useOnboardingHighlight();
  const user = useAuthStore((state) => state.user);
  const { plan } = useSubscription();
  const badge = getPlanBadge(plan, t);
  const theme = useThemeStore((s) => s.theme);
  const logoSrc = theme === "dark" ? "/favicon.svg" : "/favicon-light.svg";

  useBodyScrollLock(isModal);
  useDialogFocus(panelRef, { open: isModal });
  // Escape closes this drawer only while it is the topmost overlay — the same
  // stack the dialogs and sheets register in, so a dialog opened over the
  // drawer answers the keypress alone and the drawer stays put behind it.
  const closeDrawer = useCallback(() => closeMobile(), [closeMobile]);
  useTopmostEscape(isModal, panelRef, closeDrawer);

  const logoutMutation = useMutation({
    mutationFn: authService.logout,
    onSettled: () => {
      // Not just the auth store: every cached query belongs to the account
      // that is leaving, and the next sign-in on this tab would otherwise be
      // served the previous one's rows.
      endSession();
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
        id={APP_SIDEBAR_ID}
        ref={panelRef}
        // Focusable only on purpose: the drawer takes the keyboard when it
        // opens so a screen reader announces the panel rather than one row out
        // of it. `-1` keeps it out of the tab order either way.
        tabIndex={-1}
        className={cn(
          "fixed left-0 top-0 z-50 h-screen border-r bg-card transition-all duration-300",
          "md:sticky md:top-0 md:z-0",
          {
            "w-64": isExpanded,
            "w-16": !isExpanded,
            // Closed on a phone this panel is only pushed off-screen, and a
            // transform hides nothing from the keyboard: the first nine Tab
            // stops on every app page were these rows, sitting at x=-248 where
            // nobody could see the focus ring. `invisible` takes it out of the
            // tab order and the accessibility tree in one step, and — unlike
            // `inert` or `aria-hidden`, which would need the breakpoint
            // duplicated in JavaScript — it lifts again from `md` up, where
            // the sidebar is docked and always in use. `transition-all` covers
            // visibility, so the slide-out still plays before it disappears.
            "-translate-x-full invisible md:visible md:translate-x-0":
              !isMobileOpen,
            "translate-x-0": isMobileOpen,
          },
        )}
      >
        <div className="flex h-full flex-col">
          {/* Logo / Brand */}
          <div className="flex h-16 items-center justify-between border-b px-4">
            <Link
              to="/"
              className={cn(
                "flex items-center gap-2 hover:opacity-80 transition-opacity",
                // 32px of logo on a phone. The header row is a fixed h-16, so
                // the taller target costs the layout nothing.
                MOBILE_ROW,
              )}
            >
              <img src={logoSrc} alt="Jobber" className="h-8 w-8 rounded-lg" />
              {/* The FluxLab attribution deliberately lives only in the app
                  footer: here it would have to sit inside this brand <Link>,
                  and an anchor cannot contain another anchor. */}
              {isExpanded && <span className="text-xl font-bold">Jobber</span>}
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
            {/* Mobile-only, so the 44x44 box costs the desktop header nothing:
                p-2 around a 20px glyph was a 36x36 target. */}
            <button
              onClick={closeMobile}
              className="flex h-11 w-11 items-center justify-center rounded-md hover:bg-accent md:hidden"
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
                      // 36px tall as drawn. This sidebar is a slide-over on
                      // phones, where these rows are the primary navigation and
                      // sit 4px apart; the minimum lifts from `md` up, where the
                      // sidebar is a pointer target again.
                      MOBILE_ROW,
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
                  MOBILE_ROW,
                  "hover:bg-accent hover:text-accent-foreground",
                  {
                    "bg-accent text-accent-foreground":
                      location.pathname === "/app/settings",
                    "text-muted-foreground":
                      location.pathname !== "/app/settings",
                  },
                )}
              >
                {/* The name people chose for themselves leads; the email stays
                    on the second line because it is the account they signed in
                    with and the one support will ask for. Falls back to the
                    email alone when no name is set. */}
                <span className="min-w-0 flex-1">
                  <span
                    className="block truncate text-sm font-medium"
                    title={user?.name || user?.email}
                  >
                    {user?.name || user?.email}
                  </span>
                  {user?.name && user.email && (
                    <span
                      className="block truncate text-xs text-muted-foreground"
                      title={user.email}
                    >
                      {user.email}
                    </span>
                  )}
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
                  "flex items-center justify-center rounded-md px-3 py-2 transition-colors",
                  MOBILE_ROW,
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
                MOBILE_ROW,
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
