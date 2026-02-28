import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Briefcase, Sun, Moon } from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { useThemeStore } from "@/stores/themeStore";

interface HomeNavbarProps {
  isAuthenticated: boolean;
  onLogin: () => void;
  onRegister: () => void;
  onGoPlatform: () => void;
}

export function HomeNavbar({
  isAuthenticated,
  onLogin,
  onRegister,
  onGoPlatform,
}: HomeNavbarProps) {
  const { t } = useTranslation();
  const { theme, toggleTheme } = useThemeStore();
  const location = useLocation();
  const navigate = useNavigate();
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const scrollTo = (id: string) => {
    if (location.pathname === "/") {
      document.getElementById(id)?.scrollIntoView({ behavior: "smooth" });
    } else {
      navigate(`/#${id}`);
    }
  };

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-40 transition-all duration-300 ${
        scrolled
          ? "bg-background/95 backdrop-blur-md border-b shadow-sm"
          : "bg-transparent"
      }`}
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <Link to="/" className="flex items-center gap-2">
          <Briefcase className="h-6 w-6 text-primary" />
          <span className="text-xl font-bold">Jobber</span>
        </Link>

        <div className="hidden items-center gap-6 md:flex">
          <button
            onClick={() => scrollTo("features")}
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("home.nav.features")}
          </button>
          <button
            onClick={() => scrollTo("how-it-works")}
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("home.nav.howItWorks")}
          </button>
          <Link
            to="/blog"
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {t("blog.title")}
          </Link>
        </div>

        <div className="flex items-center gap-2">
          <LanguageSwitcher iconSize="sm" />
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label={
              theme === "light" ? "Switch to dark mode" : "Switch to light mode"
            }
            className="h-9 w-9"
          >
            {theme === "light" ? (
              <Sun className="h-4 w-4" />
            ) : (
              <Moon className="h-4 w-4" />
            )}
          </Button>
          {isAuthenticated ? (
            <Button size="sm" onClick={onGoPlatform}>
              {t("home.hero.ctaGoPlatform")}
            </Button>
          ) : (
            <>
              <Button variant="ghost" size="sm" onClick={onLogin}>
                {t("auth.login")}
              </Button>
              <Button size="sm" onClick={onRegister}>
                {t("auth.register")}
              </Button>
            </>
          )}
        </div>
      </div>
    </nav>
  );
}
