import { useTranslation } from "react-i18next";
import { useThemeStore } from "@/stores/themeStore";
import { useSidebarStore } from "@/stores/sidebarStore";
import { Button } from "@/shared/ui/Button";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { APP_SIDEBAR_ID } from "@/widgets/Sidebar";
import { Sun, Moon, Menu, X } from "lucide-react";

export function Header() {
  const { t } = useTranslation();
  const { theme, toggleTheme } = useThemeStore();
  const toggleMobile = useSidebarStore((state) => state.toggleMobile);
  const isMobileOpen = useSidebarStore((state) => state.isMobileOpen);

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between gap-2 border-b bg-background px-4">
      {/* The navigation disclosure. It said "open menu" whether the drawer was
          open or shut and named neither its state nor what it controls, so the
          only cue that anything had happened was a visual one. */}
      <Button
        variant="ghost"
        size="icon"
        onClick={toggleMobile}
        className="md:hidden"
        aria-label={isMobileOpen ? t("common.close") : t("common.openMenu")}
        aria-expanded={isMobileOpen}
        aria-controls={APP_SIDEBAR_ID}
      >
        {isMobileOpen ? (
          <X className="h-5 w-5" />
        ) : (
          <Menu className="h-5 w-5" />
        )}
      </Button>
      <div className="hidden md:block" />

      <div className="flex items-center gap-2">
        {/* Theme Toggle */}
        <Button
          variant="ghost"
          size="icon"
          onClick={toggleTheme}
          aria-label={
            theme === "light"
              ? t("settings.switchToDark")
              : t("settings.switchToLight")
          }
        >
          {theme === "light" ? (
            <Sun className="h-5 w-5" />
          ) : (
            <Moon className="h-5 w-5" />
          )}
        </Button>

        {/* Language Switcher */}
        <LanguageSwitcher />
      </div>
    </header>
  );
}
