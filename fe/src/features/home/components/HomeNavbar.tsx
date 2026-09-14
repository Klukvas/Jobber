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
import { TOP_INSET_VAR, scrollToSection } from "@/shared/lib/scrollToSection";
import { TAP_TARGET_ICON } from "@/shared/ui/tapTarget";

interface HomeNavbarProps {
  readonly isAuthenticated: boolean;
  readonly onLogin: () => void;
  readonly onRegister: () => void;
  readonly onGoPlatform: () => void;
  readonly darkHero?: boolean;
}

/**
 * Icon-only controls measured 36x36 from `sm` up — including at 768px, where
 * this navbar is still being driven by a finger. The shared constant grows the
 * hit box to 44x44 up to `lg` and absorbs the extra into a negative margin, so
 * the cluster occupies exactly the space it did before.
 */
const TOUCH_ICON_SIZE = TAP_TARGET_ICON;

/** Same floor for the stacked rows of the mobile menu, which were 32-36px. */
const TOUCH_ROW_HEIGHT = "min-h-11";

/**
 * The 44px floor for the navbar's own text buttons — Login, Register, "Go to
 * platform". They are already 44 below `sm` through the button scale, and 36
 * above it, which left 768px short. Absorbed into `py-3` like everything else
 * in this row, so only the hit area changes.
 */
const TOUCH_ROW_BUTTON = "sm:-my-1 sm:min-h-11 lg:my-0 lg:min-h-0";

/**
 * A 44px-tall target for a control that renders as plain text in the navbar
 * row: the brand link, and the nav links that appear from `md` up.
 *
 * At exactly 768px the layout swaps the 44x44 hamburger for this row of links,
 * and they measured 20px high and 26-27px wide — "FAQ" and "Blog" are simply
 * short words — under the 24px WCAG 2.5.8 requires, let alone the 44 in each
 * direction a finger wants on a screen that is still touch-sized.
 *
 * Both axes are bought the same way. A bare `min-h-11`/`min-w-11` would push
 * the navbar 8px taller and shove every neighbour sideways, because the row's
 * height comes from the 36px buttons beside these links and its width is
 * already nearly full at 768px in the longer languages; the equal negative
 * margins let the grown box overflow into the row's own `py-3` padding and
 * `gap-6` instead. The hit area a finger gets is 44x44, the layout is the one
 * that was drawn.
 */
const TOUCH_INLINE_TARGET =
  "flex min-h-11 min-w-11 items-center justify-center -mx-2 -my-1 px-2";

/** The mobile menu the hamburger opens, named so it can point `aria-controls` at it. */
const MOBILE_MENU_ID = "home-mobile-menu";

/** The features dropdown, named for its own button's `aria-controls`. */
const FEATURES_MENU_ID = "features-dropdown";

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
  const navBarRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleScroll = () => {
      setScrolled(window.scrollY > 20);
    };
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  // The navbar is fixed, so it hides the top stripe of the viewport the same
  // way the consent banner hides the bottom one. Publishing its measured height
  // lets anchor scrolling land a section *below* it instead of under it, and
  // gives the browser's own scrolling (native #hash jumps, focus moves) the
  // same clearance through scroll-padding-top. Measured rather than hard-coded:
  // the row grows when a language wraps onto a second line.
  //
  // What is measured is the *row*, not the whole `<nav>`. The expanded mobile
  // menu is a panel inside the same element, and every navigation that uses
  // this inset collapses that panel first: publishing the open height meant
  // `#faq` was aligned against a 300-400px navbar that was already gone by the
  // time the scroll ran, and the section landed that far down the viewport.
  // The panel is also irrelevant to the browser's own scroll-padding, which
  // only ever applies with the menu shut.
  useEffect(() => {
    const bar = navBarRef.current;
    if (!bar) return;

    const root = document.documentElement;
    const previous = root.style.getPropertyValue(TOP_INSET_VAR);

    const publishHeight = () => {
      root.style.setProperty(TOP_INSET_VAR, `${bar.offsetHeight}px`);
    };
    publishHeight();

    const observer =
      typeof ResizeObserver === "undefined"
        ? null
        : new ResizeObserver(publishHeight);
    observer?.observe(bar);
    window.addEventListener("resize", publishHeight);

    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", publishHeight);
      if (previous) {
        root.style.setProperty(TOP_INSET_VAR, previous);
      } else {
        root.style.removeProperty(TOP_INSET_VAR);
      }
    };
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
      scrollToSection(id);
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
    ? `${TOUCH_INLINE_TARGET} text-sm text-white/70 transition-colors hover:text-white`
    : `${TOUCH_INLINE_TARGET} text-sm text-muted-foreground transition-colors hover:text-foreground`;

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-40 transition-all duration-300 ${
        scrolled || mobileMenuOpen
          ? "bg-background/95 backdrop-blur-md border-b shadow-sm"
          : "bg-transparent"
      }`}
    >
      <div
        ref={navBarRef}
        className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3"
      >
        {/* 28px of glyph and text on a phone. The same 44px floor as the
            links, absorbed the same way so the navbar keeps its height. */}
        <Link to="/" className={`${TOUCH_INLINE_TARGET} gap-2`}>
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
              aria-controls={FEATURES_MENU_ID}
              className={`gap-1 ${linkCls}`}
            >
              {t("home.nav.features")}
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform duration-200 ${featuresOpen ? "rotate-180" : ""}`}
              />
            </button>

            {/* Present in both states for the same reason as the mobile menu
                below: the button names this element, and a name that resolves
                to nothing is no name at all. */}
            <div
              id={FEATURES_MENU_ID}
              role="menu"
              hidden={!featuresOpen}
              className="absolute left-1/2 top-full mt-2 w-56 -translate-x-1/2 overflow-hidden rounded-xl border bg-background/95 shadow-lg backdrop-blur-md"
            >
              {featuresOpen &&
                FEATURE_LINKS.map(({ key, to, Icon }) => (
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
          <button
            type="button"
            onClick={() => scrollTo("faq")}
            className={linkCls}
          >
            {t("home.nav.faq")}
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
                ? `${TOUCH_ICON_SIZE} text-white/70 hover:text-white hover:bg-white/10`
                : TOUCH_ICON_SIZE
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
              className={`hidden sm:inline-flex ${TOUCH_ROW_BUTTON}`}
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
                    ? `hidden md:inline-flex ${TOUCH_ROW_BUTTON} text-white/70 hover:text-white hover:bg-white/10`
                    : `hidden md:inline-flex ${TOUCH_ROW_BUTTON}`
                }
              >
                {t("auth.login")}
              </Button>
              <Button
                size="sm"
                onClick={onRegister}
                className={`hidden md:inline-flex ${TOUCH_ROW_BUTTON}`}
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
            // A disclosure button, so it has to say what it controls and
            // whether that thing is open — the label alone left assistive tech
            // to guess from an icon. The panel below stays in the DOM in both
            // states so this IDREF always resolves; a name that points at
            // nothing is the same to a screen reader as no name at all.
            aria-expanded={mobileMenuOpen}
            aria-controls={MOBILE_MENU_ID}
            className={
              onDark
                ? `${TOUCH_ICON_SIZE} md:hidden text-white/70 hover:text-white hover:bg-white/10`
                : `${TOUCH_ICON_SIZE} md:hidden`
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

      {/* Rendered in both states, hidden rather than unmounted, so the
          hamburger's `aria-controls` resolves while the menu is collapsed. The
          rows themselves are still mounted only when open — `hidden` already
          takes the panel out of the layout, the tab order and the
          accessibility tree. */}
      <div
        id={MOBILE_MENU_ID}
        hidden={!mobileMenuOpen}
        className="border-t bg-background/95 backdrop-blur-md px-4 pb-4 md:hidden"
      >
        {mobileMenuOpen && (
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
                    className={`flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground ${TOUCH_ROW_HEIGHT}`}
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
              className={`flex items-center rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground ${TOUCH_ROW_HEIGHT}`}
            >
              {t("home.nav.howItWorks")}
            </button>
            <button
              type="button"
              onClick={() => scrollTo("pricing")}
              className={`flex items-center rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground ${TOUCH_ROW_HEIGHT}`}
            >
              {t("home.nav.pricing")}
            </button>
            <button
              type="button"
              onClick={() => scrollTo("faq")}
              className={`flex items-center rounded-md px-3 py-2 text-left text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground ${TOUCH_ROW_HEIGHT}`}
            >
              {t("home.nav.faq")}
            </button>
            <Link
              to="/blog"
              onClick={handleLinkClick}
              className={`flex items-center rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground ${TOUCH_ROW_HEIGHT}`}
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
        )}
      </div>
    </nav>
  );
}
