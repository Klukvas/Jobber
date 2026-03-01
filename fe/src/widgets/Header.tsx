import { useTranslation } from "react-i18next";
import { useThemeStore } from "@/stores/themeStore";
import { useSidebarStore } from "@/stores/sidebarStore";
import { Button } from "@/shared/ui/Button";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { Sun, Moon, Menu } from "lucide-react";

export function Header() {
  const { t } = useTranslation();
  const { theme, toggleTheme } = useThemeStore();
  const toggleMobile = useSidebarStore((state) => state.toggleMobile);

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between gap-2 border-b bg-background px-4">
      {/* Mobile Menu Button */}
      <Button
        variant="ghost"
        size="icon"
        onClick={toggleMobile}
        className="md:hidden"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5" />
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
