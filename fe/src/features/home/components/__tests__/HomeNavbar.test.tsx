import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { HomeNavbar } from "../HomeNavbar";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  // className is forwarded: the navbar puts its tap-target sizing on links,
  // and a stub that swallowed it would hide exactly what those tests check.
  Link: ({
    children,
    to,
    className,
  }: {
    children: React.ReactNode;
    to: string;
    className?: string;
  }) => (
    <a href={to} className={className}>
      {children}
    </a>
  ),
  useLocation: () => ({ pathname: "/", search: "", hash: "" }),
  useNavigate: () => vi.fn(),
}));

vi.mock("@/stores/themeStore", () => ({
  useThemeStore: () => ({
    theme: "dark",
    toggleTheme: vi.fn(),
  }),
}));

vi.mock("@/shared/ui/LanguageSwitcher", () => ({
  LanguageSwitcher: () => <div data-testid="lang-switcher" />,
}));

describe("HomeNavbar", () => {
  const defaultProps = {
    isAuthenticated: false,
    onLogin: vi.fn(),
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("renders brand name", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByText("Jobber")).toBeInTheDocument();
  });

  it("renders nav links", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByText("home.nav.features")).toBeInTheDocument();
    expect(screen.getByText("home.nav.howItWorks")).toBeInTheDocument();
    expect(screen.getByText("home.nav.pricing")).toBeInTheDocument();
    expect(screen.getByText("blog.title")).toBeInTheDocument();
  });

  // Without this the FAQ was only reachable by scrolling the whole landing page.
  it("exposes the FAQ as a keyboard-reachable nav control", () => {
    render(<HomeNavbar {...defaultProps} />);
    const faq = screen.getAllByText("home.nav.faq")[0];

    expect(faq).toBeInTheDocument();
    expect(faq.tagName).toBe("BUTTON");
    expect(faq).not.toBeDisabled();
  });

  it("offers the FAQ in the mobile menu too", () => {
    render(<HomeNavbar {...defaultProps} />);
    fireEvent.click(screen.getByLabelText("common.openMenu"));

    expect(screen.getAllByText("home.nav.faq").length).toBeGreaterThan(1);
  });

  it("renders login and register buttons when not authenticated", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByText("auth.login")).toBeInTheDocument();
    expect(screen.getByText("auth.register")).toBeInTheDocument();
  });

  it("renders go-to-platform button when authenticated", () => {
    render(<HomeNavbar {...defaultProps} isAuthenticated={true} />);
    expect(screen.getByText("home.hero.ctaGoPlatform")).toBeInTheDocument();
    expect(screen.queryByText("auth.login")).not.toBeInTheDocument();
  });

  it("renders theme toggle button", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByLabelText("settings.switchToLight")).toBeInTheDocument();
  });

  it("renders language switcher", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByTestId("lang-switcher")).toBeInTheDocument();
  });

  it("opens features dropdown when clicked", () => {
    render(<HomeNavbar {...defaultProps} />);
    const btn = screen.getByText("home.nav.features");
    fireEvent.click(btn);
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(
      screen.getByText("home.features.applications.title"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("home.features.resumeBuilder.title"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("home.features.coverLetters.title"),
    ).toBeInTheDocument();
  });

  it("renders mobile menu toggle", () => {
    render(<HomeNavbar {...defaultProps} />);
    expect(screen.getByLabelText("common.openMenu")).toBeInTheDocument();
  });

  it("opens mobile menu when toggle is clicked", () => {
    render(<HomeNavbar {...defaultProps} />);
    fireEvent.click(screen.getByLabelText("common.openMenu"));
    // Mobile menu should now show duplicate nav items
    const allHowItWorks = screen.getAllByText("home.nav.howItWorks");
    expect(allHowItWorks.length).toBeGreaterThan(1);
  });
});

/**
 * Icon-only controls measured 36x36 on a 390px viewport and the stacked menu
 * rows 32-36px, under the 44px comfortable target WCAG 2.5.5 asks for. Asserted
 * on classes rather than geometry: jsdom has no layout, and the classes are
 * what actually decide the size.
 */
describe("HomeNavbar touch targets", () => {
  const defaultProps = {
    isAuthenticated: false,
    onLogin: vi.fn(),
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };
  const TOUCH_SIZE = /(^|\s)h-11(\s|$)/;
  const SHRINKS_ON_POINTER = /lg:h-9/;

  it("gives the theme toggle a 44px target on phones", () => {
    render(<HomeNavbar {...defaultProps} />);
    const toggle = screen.getByLabelText("settings.switchToLight");

    expect(toggle.className).toMatch(TOUCH_SIZE);
    expect(toggle.className).toMatch(/(^|\s)w-11(\s|$)/);
  });

  // 768px is the interesting width: the hamburger is gone, the pointer row of
  // links is up, and the screen is still a tablet. The compact size therefore
  // waits for `lg` rather than `sm`.
  it("keeps the compact size from the lg breakpoint up, not sm", () => {
    render(<HomeNavbar {...defaultProps} />);
    const toggle = screen.getByLabelText("settings.switchToLight");

    expect(toggle.className).toMatch(SHRINKS_ON_POINTER);
    expect(toggle.className).not.toMatch(/sm:h-9/);
    expect(toggle.className).not.toMatch(/sm:w-9/);
  });

  // The full box has to fit somewhere: it overflows into the row's own padding
  // and gaps instead of pushing the navbar around at 768px, where the longer
  // languages already fill the row.
  it("absorbs the icon target into the row instead of widening it", () => {
    render(<HomeNavbar {...defaultProps} />);

    expect(screen.getByLabelText("settings.switchToLight").className).toMatch(
      /sm:-m-1/,
    );
  });

  it("gives the mobile menu toggle a 44px target", () => {
    render(<HomeNavbar {...defaultProps} />);
    const hamburger = screen.getByLabelText("common.openMenu");

    expect(hamburger.className).toMatch(TOUCH_SIZE);
    expect(hamburger.className).toMatch(/(^|\s)w-11(\s|$)/);
  });

  it("gives every mobile menu row a 44px floor", () => {
    render(<HomeNavbar {...defaultProps} />);
    fireEvent.click(screen.getByLabelText("common.openMenu"));

    // Each anchor control is rendered twice — desktop first, mobile second.
    for (const label of [
      "home.nav.howItWorks",
      "home.nav.pricing",
      "home.nav.faq",
    ]) {
      const copies = screen.getAllByText(label);
      expect(copies.length).toBeGreaterThan(1);
      expect(copies[copies.length - 1].className).toMatch(/min-h-11/);
    }
  });
});

/**
 * At exactly 768px the layout drops the 44x44 hamburger and shows this row of
 * links instead — 20px-high text, under the 24px WCAG 2.5.8 requires and far
 * under what a finger needs on a screen that is still touch-sized. The height
 * is bought with a negative margin so the navbar itself does not grow.
 */
describe("HomeNavbar nav link targets", () => {
  const defaultProps = {
    isAuthenticated: false,
    onLogin: vi.fn(),
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it.each([
    "home.nav.features",
    "home.nav.howItWorks",
    "home.nav.pricing",
    "home.nav.faq",
    "blog.title",
  ])("gives the %s link a 44px height", (label) => {
    render(<HomeNavbar {...defaultProps} />);
    // The desktop copy is the first one rendered; the mobile menu is closed.
    const link = screen.getAllByText(label)[0];

    expect(link.className).toMatch(/min-h-11/);
    // Absorbed into the row's own padding, so the navbar keeps its height.
    expect(link.className).toMatch(/-my-1/);
  });

  // "FAQ" measured 26px wide and "Blog" 27 at 768px: tall enough after the
  // height pass, still a quarter of a fingertip across.
  it.each([
    "home.nav.features",
    "home.nav.howItWorks",
    "home.nav.pricing",
    "home.nav.faq",
    "blog.title",
  ])("gives the %s link a 44px width", (label) => {
    render(<HomeNavbar {...defaultProps} />);
    const link = screen.getAllByText(label)[0];

    expect(link.className).toMatch(/min-w-11/);
    // Same absorption sideways: the padding that widens the box is pulled back
    // out again, so the row is laid out exactly as it was.
    expect(link.className).toMatch(/-mx-2/);
    expect(link.className).toMatch(/(^|\s)px-2(\s|$)/);
  });

  it.each(["auth.login", "auth.register"])(
    "gives the %s button a 44px height at tablet widths",
    (label) => {
      render(<HomeNavbar {...defaultProps} />);
      const button = screen.getByText(label);

      expect(button.className).toMatch(/sm:min-h-11/);
      expect(button.className).toMatch(/lg:min-h-0/);
    },
  );

  // 28px of glyph and text, and the first control on the page.
  it("gives the brand link a 44px height", () => {
    render(<HomeNavbar {...defaultProps} />);
    const brand = screen.getByText("Jobber").closest("a");

    expect(brand?.className).toMatch(/min-h-11/);
  });
});

/**
 * The hamburger is a disclosure button. It said nothing about what it controls
 * or whether that thing was open, so assistive tech had only an icon to go on.
 */
describe("HomeNavbar mobile menu semantics", () => {
  const defaultProps = {
    isAuthenticated: false,
    onLogin: vi.fn(),
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  it("reports the menu as collapsed to start with", () => {
    render(<HomeNavbar {...defaultProps} />);

    expect(screen.getByLabelText("common.openMenu")).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("reports it as expanded once it is open", () => {
    render(<HomeNavbar {...defaultProps} />);

    fireEvent.click(screen.getByLabelText("common.openMenu"));

    expect(screen.getByLabelText("common.close")).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("names the menu it controls", () => {
    render(<HomeNavbar {...defaultProps} />);
    const controls = screen
      .getByLabelText("common.openMenu")
      .getAttribute("aria-controls");

    expect(controls).toBeTruthy();

    fireEvent.click(screen.getByLabelText("common.openMenu"));

    expect(document.getElementById(controls ?? "")).toBeInTheDocument();
  });

  // A dangling IDREF is the same to a screen reader as no relationship at all,
  // and collapsed is the state the button spends most of its life in.
  it("still names an element that exists while collapsed", () => {
    render(<HomeNavbar {...defaultProps} />);
    const controls = screen
      .getByLabelText("common.openMenu")
      .getAttribute("aria-controls");

    const panel = document.getElementById(controls ?? "");

    expect(panel).toBeInTheDocument();
    expect(panel).not.toBeVisible();
  });

  // Same disclosure contract on the features button beside it: the dropdown
  // used to exist only while open, so its `aria-controls` dangled the rest of
  // the time.
  it("keeps the features dropdown's IDREF valid in both states", () => {
    render(<HomeNavbar {...defaultProps} />);
    const features = screen.getByText("home.nav.features");
    const controls = features.getAttribute("aria-controls");

    expect(controls).toBeTruthy();
    expect(document.getElementById(controls ?? "")).toBeInTheDocument();
    expect(document.getElementById(controls ?? "")).not.toBeVisible();

    fireEvent.click(features);

    expect(document.getElementById(controls ?? "")).toBeVisible();
  });

  it("shows that same element once the menu opens", () => {
    render(<HomeNavbar {...defaultProps} />);
    const controls = screen
      .getByLabelText("common.openMenu")
      .getAttribute("aria-controls");

    fireEvent.click(screen.getByLabelText("common.openMenu"));

    expect(document.getElementById(controls ?? "")).toBeVisible();
  });
});

/**
 * Where a mobile-menu anchor actually lands.
 *
 * The navbar publishes its height so anchor scrolling can clear it, and the
 * open mobile menu is a panel inside the same `<nav>`. Publishing the whole
 * element's height meant `#faq` was aligned against a navbar three hundred
 * pixels taller than the one on screen by the time the scroll ran — the menu is
 * closed by the very click that starts it — so the section arrived that far
 * down the viewport, with the content above it filling the screen.
 */
describe("HomeNavbar — anchor scrolling from the mobile menu", () => {
  /** Realistic phone measurements: a 64px row, a 336px menu panel under it. */
  const ROW_HEIGHT = 64;
  const OPEN_MENU_HEIGHT = 336;

  const FAQ_TOP = 3200;
  /** Shorter than the usable band, so its top edge is the whole alignment. */
  const FAQ_HEIGHT = 400;
  const VIEWPORT = 667;

  const heights = new Map<Element, number>();
  let scrollTo: ReturnType<typeof vi.fn>;
  let originalOffsetHeight: PropertyDescriptor | undefined;

  const defaultProps = {
    isAuthenticated: false,
    onLogin: vi.fn(),
    onRegister: vi.fn(),
    onGoPlatform: vi.fn(),
  };

  beforeEach(() => {
    heights.clear();
    // Captured, not assumed absent. `delete` on this property used to be the
    // teardown, which does not restore jsdom's own descriptor — it removes it,
    // so every later test in the process read `undefined` for `offsetHeight`
    // instead of jsdom's 0. Nothing in this file noticed; the next file to
    // measure an element would have.
    originalOffsetHeight = Object.getOwnPropertyDescriptor(
      HTMLElement.prototype,
      "offsetHeight",
    );
    Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
      configurable: true,
      get(this: HTMLElement) {
        return heights.get(this) ?? 0;
      },
    });

    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: VIEWPORT,
    });
    Object.defineProperty(document.documentElement, "scrollHeight", {
      configurable: true,
      value: 9000,
    });

    const faq = document.createElement("section");
    faq.id = "faq";
    faq.getBoundingClientRect = () =>
      ({ top: FAQ_TOP - window.scrollY, height: FAQ_HEIGHT }) as DOMRect;
    const heading = document.createElement("h2");
    heading.getBoundingClientRect = () =>
      ({ top: FAQ_TOP + 96 - window.scrollY, height: 40 }) as DOMRect;
    faq.appendChild(heading);
    document.body.appendChild(faq);

    scrollTo = vi.fn();
    window.scrollTo = scrollTo as unknown as typeof window.scrollTo;
  });

  afterEach(() => {
    document.getElementById("faq")?.remove();
    document.documentElement.style.removeProperty("--app-top-inset");
    if (originalOffsetHeight) {
      Object.defineProperty(
        HTMLElement.prototype,
        "offsetHeight",
        originalOffsetHeight,
      );
    } else {
      delete (HTMLElement.prototype as { offsetHeight?: number }).offsetHeight;
    }
    originalOffsetHeight = undefined;
  });

  it("hands jsdom's own offsetHeight back when a test is done with it", () => {
    // The stub is installed by beforeEach and taken off by afterEach; what this
    // asserts is that there was a descriptor to hand back at all, so the
    // restore above is doing something rather than papering over a delete.
    expect(originalOffsetHeight).toBeDefined();
    expect(typeof originalOffsetHeight?.get).toBe("function");
  });

  it("measures the collapsed row rather than the expanded menu", () => {
    const { container } = render(<HomeNavbar {...defaultProps} />);
    const nav = container.querySelector("nav") as HTMLElement;
    const row = nav.firstElementChild as HTMLElement;

    heights.set(row, ROW_HEIGHT);
    heights.set(nav, ROW_HEIGHT + OPEN_MENU_HEIGHT);

    fireEvent.click(screen.getByLabelText("common.openMenu"));
    // What the ResizeObserver would do once the panel is on the page.
    fireEvent(window, new Event("resize"));

    expect(
      document.documentElement.style.getPropertyValue("--app-top-inset"),
    ).toBe(`${ROW_HEIGHT}px`);
  });

  it("lands the FAQ just under the navbar the buyer will actually see", () => {
    const { container } = render(<HomeNavbar {...defaultProps} />);
    const nav = container.querySelector("nav") as HTMLElement;
    const row = nav.firstElementChild as HTMLElement;

    heights.set(row, ROW_HEIGHT);
    heights.set(nav, ROW_HEIGHT + OPEN_MENU_HEIGHT);

    fireEvent.click(screen.getByLabelText("common.openMenu"));
    fireEvent(window, new Event("resize"));

    // The mobile menu's own FAQ row — the last of the two the navbar renders.
    const faqControls = screen.getAllByText("home.nav.faq");
    fireEvent.click(faqControls[faqControls.length - 1]);

    // The section's top edge on the first pixel below the 64px row. Aligned
    // against the open menu instead, this stopped hundreds of pixels short and
    // the FAQ arrived halfway down the screen.
    expect(scrollTo).toHaveBeenCalledWith(
      expect.objectContaining({ top: FAQ_TOP - ROW_HEIGHT }),
    );
  });
});
