import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Briefcase, Sun, Moon, Menu, X } from "lucide-react";
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
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const scrollTo = (id: string) => {
    setMobileMenuOpen(false);
    if (location.pathname === "/") {
      document.getElementById(id)?.scrollIntoView({ behavior: "smooth" });
    } else {
      navigate(`/#${id}`);
    }
  };

  const handleLinkClick = () => {
    setMobileMenuOpen(false);
  };

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-40 transition-all duration-300 ${
        scrolled || mobileMenuOpen
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
              theme === "light"
                ? t("settings.switchToDark")
                : t("settings.switchToLight")
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
              <Button
                variant="ghost"
                size="sm"
                onClick={onLogin}
                className="hidden md:inline-flex"
              >
                {t("auth.login")}
              </Button>
              <Button
                size="sm"
                onClick={onRegister}
                className="hidden md:inline-flex"
              >
                {t("auth.register")}
              </Button>
            </>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setMobileMenuOpen((prev) => !prev)}
            aria-label="Toggle mobile menu"
            className="h-9 w-9 md:hidden"
          >
            {mobileMenuOpen ? (
              <X className="h-5 w-5" />
            ) : (
              <Menu className="h-5 w-5" />
            )}
          </Button>
        </div>
      </div>

      {mobileMenuOpen && (
        <div className="border-t bg-background/95 backdrop-blur-md px-4 pb-4 md:hidden">
          <div className="flex flex-col gap-1 pt-2">
            <button
              onClick={() => scrollTo("features")}
              className="rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {t("home.nav.features")}
            </button>
            <button
              onClick={() => scrollTo("how-it-works")}
              className="rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {t("home.nav.howItWorks")}
            </button>
            <Link
              to="/blog"
              onClick={handleLinkClick}
              className="rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {t("blog.title")}
            </Link>
            {!isAuthenticated && (
              <div className="mt-2 flex flex-col gap-2 border-t pt-3">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setMobileMenuOpen(false);
                    onLogin();
                  }}
                  className="w-full justify-center"
                >
                  {t("auth.login")}
                </Button>
                <Button
                  size="sm"
                  onClick={() => {
                    setMobileMenuOpen(false);
                    onRegister();
                  }}
                  className="w-full justify-center"
                >
                  {t("auth.register")}
                </Button>
              </div>
            )}
          </div>
        </div>
      )}
    </nav>
  );
}
