import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Sidebar } from "../Sidebar";
import { useSidebarStore } from "@/stores/sidebarStore";
import { installBrowserFocusRules } from "@/test/browserFocusability";
import {
  installManualAnimationFrames,
  type ManualAnimationFrames,
} from "@/test/animationFrames";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}));

vi.mock("react-router-dom", () => ({
  NavLink: ({
    children,
    to,
    className,
  }: {
    children: React.ReactNode;
    to: string;
    className: unknown;
  }) => (
    <a
      href={to}
      className={
        typeof className === "function"
          ? className({ isActive: false })
          : className
      }
    >
      {children}
    </a>
  ),
  Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  useNavigate: () => vi.fn(),
  useLocation: () => ({ pathname: "/app/jobs" }),
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/stores/authStore", () => ({
  useAuthStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ user: { email: "test@example.com" }, clearAuth: vi.fn() }),
}));

vi.mock("@/shared/hooks/useSubscription", () => ({
  useSubscription: () => ({ plan: "free" }),
}));

vi.mock("@/services/authService", () => ({
  authService: { logout: vi.fn() },
}));

vi.mock("@/features/onboarding/useOnboarding", () => ({
  useOnboardingHighlight: () => null,
}));

/** The real store, so the drawer is opened the way the header opens it. */
function openDrawer(): void {
  act(() => useSidebarStore.setState({ isMobileOpen: true }));
}

function closeDrawer(): void {
  act(() => useSidebarStore.setState({ isMobileOpen: false }));
}

/**
 * The phone navigation drawer covers the page it is navigating, and behaved
 * like nothing of the sort: Escape reached the document and was ignored, the
 * page behind it went on scrolling under the panel, and the hamburger that
 * opened it never got the keyboard back. Its close button worked, which was
 * the whole of the way out.
 *
 * `matchMedia` is what decides whether the sidebar is a drawer at all, so
 * every test here says which viewport it is on.
 */
describe("the phone navigation drawer", () => {
  let originalMatchMedia: typeof window.matchMedia;

  function setViewport(isPhone: boolean): void {
    window.matchMedia = ((query: string) => ({
      matches: isPhone,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })) as unknown as typeof window.matchMedia;
  }

  beforeEach(() => {
    originalMatchMedia = window.matchMedia;
    setViewport(true);
    useSidebarStore.setState({ isExpanded: true, isMobileOpen: false });
    document.body.style.overflow = "";
  });

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
    // Inside `act`: a test that ends with the drawer open still has a mounted
    // Sidebar subscribed to this store, and closing it from here re-renders.
    act(() => useSidebarStore.setState({ isMobileOpen: false }));
    document.body.style.overflow = "";
  });

  it("closes on Escape", async () => {
    const user = userEvent.setup();
    render(<Sidebar />);
    openDrawer();

    await user.keyboard("{Escape}");

    expect(useSidebarStore.getState().isMobileOpen).toBe(false);
  });

  it("ignores Escape while it is shut", async () => {
    const user = userEvent.setup();
    render(<Sidebar />);

    await user.keyboard("{Escape}");

    expect(useSidebarStore.getState().isMobileOpen).toBe(false);
  });

  it("locks the page behind it", () => {
    render(<Sidebar />);
    expect(document.body.style.overflow).toBe("");

    openDrawer();

    expect(document.body.style.overflow).toBe("hidden");
  });

  it("gives the page back when it closes", () => {
    render(<Sidebar />);
    openDrawer();

    closeDrawer();

    expect(document.body.style.overflow).toBe("");
  });

  it("gives the page back when it unmounts while open", () => {
    const { unmount } = render(<Sidebar />);
    openDrawer();

    unmount();

    expect(document.body.style.overflow).toBe("");
  });

  it("restores an overflow the page already had", () => {
    document.body.style.overflow = "clip";
    render(<Sidebar />);

    openDrawer();
    expect(document.body.style.overflow).toBe("hidden");

    closeDrawer();
    expect(document.body.style.overflow).toBe("clip");
  });

  it("takes the keyboard when it opens", () => {
    render(<Sidebar />);

    openDrawer();

    const panel = document.getElementById("app-sidebar");
    expect(panel?.contains(document.activeElement)).toBe(true);
  });

  /**
   * Closed, the panel is `invisible`, and `transition-all` covers visibility —
   * so for the first frames after the hamburger is tapped the drawer is on
   * screen to look at and still `visibility: hidden` to the browser, which
   * refuses focus to it and to every row inside it. On a phone that left the
   * keyboard on the hamburger, underneath the panel it had just opened, with no
   * `focusin` fired at all.
   *
   * jsdom focuses hidden elements happily, so both browser rules — focus needs
   * a box, and frames are frames — are installed here.
   */
  describe("while the panel is still becoming visible", () => {
    let frames: ManualAnimationFrames;
    let restoreFocusRules: () => void;
    let hamburger: HTMLButtonElement;

    beforeEach(() => {
      restoreFocusRules = installBrowserFocusRules();
      frames = installManualAnimationFrames();
      hamburger = document.createElement("button");
      document.body.appendChild(hamburger);
    });

    afterEach(() => {
      frames.restore();
      restoreFocusRules();
      hamburger.remove();
    });

    it("still takes the keyboard, a frame or two later", () => {
      render(<Sidebar />);
      const panel = document.getElementById("app-sidebar") as HTMLElement;
      panel.style.visibility = "hidden";
      hamburger.focus();

      openDrawer();
      expect(document.activeElement).toBe(hamburger);

      act(() => frames.runFrame());
      panel.style.visibility = "visible";
      act(() => frames.runFrame());

      expect(panel.contains(document.activeElement)).toBe(true);
    });
  });

  it("hands the keyboard back to the control that opened it", async () => {
    const user = userEvent.setup();
    const opener = document.createElement("button");
    document.body.appendChild(opener);
    render(<Sidebar />);

    opener.focus();
    openDrawer();
    expect(document.activeElement).not.toBe(opener);

    await user.keyboard("{Escape}");
    closeDrawer();

    expect(document.activeElement).toBe(opener);
    opener.remove();
  });

  it("keeps Tab inside the panel", async () => {
    const user = userEvent.setup();
    const outside = document.createElement("button");
    document.body.appendChild(outside);
    render(<Sidebar />);
    openDrawer();

    await user.tab();
    await user.tab();

    const panel = document.getElementById("app-sidebar");
    expect(panel?.contains(document.activeElement)).toBe(true);
    expect(document.activeElement).not.toBe(outside);
    outside.remove();
  });

  // From `md` up the panel is docked beside the page rather than drawn over
  // it. Freezing the page or trapping the keyboard there would be a bug, and
  // `isMobileOpen` survives a visitor widening the window with the drawer up.
  describe("once the sidebar is docked", () => {
    beforeEach(() => setViewport(false));

    it("does not lock the page", () => {
      render(<Sidebar />);

      openDrawer();

      expect(document.body.style.overflow).toBe("");
    });

    it("leaves Escape to whatever else is listening", async () => {
      const user = userEvent.setup();
      render(<Sidebar />);
      openDrawer();

      await user.keyboard("{Escape}");

      expect(useSidebarStore.getState().isMobileOpen).toBe(true);
    });
  });
});
