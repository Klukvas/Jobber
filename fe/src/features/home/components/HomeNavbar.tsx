import { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Link, useLocation, useNavigate } from "react-router-dom";
import {
  Briefcase,
  Sun,
  Moon,
  Menu,
  X,
  ChevronDown,
  LayoutList,
  FileText,
  Mail,
} from "lucide-react";
import { Button } from "@/shared/ui/Button";
import { LanguageSwitcher } from "@/shared/ui/LanguageSwitcher";
import { useThemeStore } from "@/stores/themeStore";

interface HomeNavbarProps {
  readonly isAuthenticated: boolean;
  readonly onLogin: () => void;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
  readonly darkHero?: boolean;
}

const FEATURE_LINKS = [
  { key: "applications", to: "/features/applications", Icon: LayoutList },
  { key: "resumeBuilder", to: "/features/resume-builder", Icon: FileText },
  { key: "coverLetters", to: "/features/cover-letters", Icon: Mail },
] as const;

export function HomeNavbar({
  isAuthenticated,
  onLogin,
  onRegister,
  onGoPlatform,
  darkHero = false,
}: HomeNavbarProps) {
  const { t } = useTranslation();
  const { theme, toggleTheme } = useThemeStore();
  const location = useLocation();
  const navigate = useNavigate();
  const [scrolled, setScrolled] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [featuresOpen, setFeaturesOpen] = useState(false);
  const featuresRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (
        featuresRef.current &&
        !featuresRef.current.contains(e.target as Node)
      ) {
        setFeaturesOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
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

  // When the hero is dark and we haven't scrolled yet, use white text
  const onDark = darkHero && !scrolled && !mobileMenuOpen;
  const linkCls = onDark
    ? "text-sm text-white/70 transition-colors hover:text-white"
    : "text-sm text-muted-foreground transition-colors hover:text-foreground";

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
          <Briefcase
            className={`h-6 w-6 ${onDark ? "text-white" : "text-primary"}`}
          />
          <span className={`text-xl font-bold ${onDark ? "text-white" : ""}`}>
            Jobber
          </span>
        </Link>

        <div className="hidden items-center gap-6 md:flex">
          {/* Features dropdown */}
          <div ref={featuresRef} className="relative">
            <button
              type="button"
              onClick={() => setFeaturesOpen((prev) => !prev)}
              onKeyDown={(e) => e.key === "Escape" && setFeaturesOpen(false)}
              aria-expanded={featuresOpen}
              aria-haspopup="menu"
              aria-controls="features-dropdown"
              className={`flex items-center gap-1 ${linkCls}`}
            >
              {t("home.nav.features")}
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform duration-200 ${featuresOpen ? "rotate-180" : ""}`}
              />
            </button>

            {featuresOpen && (
              <div
                id="features-dropdown"
                role="menu"
                className="absolute left-1/2 top-full mt-2 w-56 -translate-x-1/2 overflow-hidden rounded-xl border bg-background/95 shadow-lg backdrop-blur-md"
              >
                {FEATURE_LINKS.map(({ key, to, Icon }) => (
                  <Link
                    key={key}
                    to={to}
                    role="menuitem"
                    onClick={() => setFeaturesOpen(false)}
                    className="flex items-center gap-3 px-4 py-3 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                  >
                    <Icon className="h-4 w-4 shrink-0 text-primary" />
                    {t(`home.features.${key}.title`)}
                  </Link>
                ))}
              </div>
            )}
          </div>

          <button
            type="button"
            onClick={() => scrollTo("how-it-works")}
            className={linkCls}
          >
            {t("home.nav.howItWorks")}
          </button>
          <button
            type="button"
            onClick={() => scrollTo("pricing")}
            className={linkCls}
          >
            {t("home.nav.pricing")}
          </button>
          <Link to="/blog" className={linkCls}>
            {t("blog.title")}
          </Link>
        </div>

        <div className="flex items-center gap-2">
          <LanguageSwitcher
            iconSize="sm"
            className={
              onDark
                ? "text-white/70 hover:text-white hover:bg-white/10"
                : undefined
            }
          />
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleTheme}
            aria-label={
              theme === "light"
                ? t("settings.switchToDark")
                : t("settings.switchToLight")
            }
            className={
              onDark
                ? "h-9 w-9 text-white/70 hover:text-white hover:bg-white/10"
                : "h-9 w-9"
            }
          >
            {theme === "light" ? (
              <Sun className="h-4 w-4" />
            ) : (
              <Moon className="h-4 w-4" />
            )}
          </Button>
          {isAuthenticated ? (
            <Button
              size="sm"
              onClick={onGoPlatform}
              className="hidden sm:inline-flex"
            >
              {t("home.hero.ctaGoPlatform")}
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={onLogin}
                className={
                  onDark
                    ? "hidden md:inline-flex text-white/70 hover:text-white hover:bg-white/10"
                    : "hidden md:inline-flex"
                }
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
            aria-label={
              mobileMenuOpen ? t("common.close") : t("common.openMenu")
            }
            className={
              onDark
                ? "h-9 w-9 md:hidden text-white/70 hover:text-white hover:bg-white/10"
                : "h-9 w-9 md:hidden"
            }
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
            {/* Features group in mobile */}
            <div className="rounded-md px-3 py-2">
              <p className="mb-1.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground/60">
                {t("home.nav.features")}
              </p>
              <div className="flex flex-col gap-0.5">
                {FEATURE_LINKS.map(({ key, to, Icon }) => (
                  <Link
                    key={key}
                    to={to}
                    onClick={handleLinkClick}
                    className="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                  >
                    <Icon className="h-4 w-4 shrink-0 text-primary" />
                    {t(`home.features.${key}.title`)}
                  </Link>
                ))}
              </div>
            </div>

            <button
              type="button"
              onClick={() => scrollTo("how-it-works")}
              className="rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {t("home.nav.howItWorks")}
            </button>
            <button
              type="button"
              onClick={() => scrollTo("pricing")}
              className="rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {t("home.nav.pricing")}
            </button>
            <Link
              to="/blog"
              onClick={handleLinkClick}
              className="rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              {t("blog.title")}
            </Link>
            <div className="mt-2 flex flex-col gap-2 border-t pt-3">
              {isAuthenticated ? (
                <Button
                  size="sm"
                  onClick={() => {
                    setMobileMenuOpen(false);
                    onGoPlatform();
                  }}
                  className="w-full justify-center"
                >
                  {t("home.hero.ctaGoPlatform")}
                </Button>
              ) : (
                <>
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
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </nav>
  );
}
