import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  BOTTOM_INSET_VAR,
  TOP_INSET_VAR,
  readBottomInset,
  readTopInset,
  scrollToSection,
} from "../scrollToSection";

const DESKTOP_VIEWPORT = 813;
const PHONE_VIEWPORT = 667;

/** A realistic landing navbar, and the banner heights RU/UK wrap to. */
const NAVBAR = 64;
const SHORT_BANNER = 96;
const TALL_BANNER = 204;

interface SectionSpec {
  readonly id?: string;
  readonly top?: number;
  readonly height?: number;
  /** Distance from the section's top edge down to its <h2>. */
  readonly headingOffset?: number;
  readonly documentHeight?: number;
}

/**
 * Places a section in a document tall enough to scroll, mimicking the landing
 * page: `#pricing` sits ~2724px down and is taller than the usable viewport.
 * The heading sits below the section's own padding and its eyebrow label, which
 * is what gives the scroll any room to move at all.
 */
function mountSection({
  id = "pricing",
  top = 2724,
  height = 903,
  headingOffset = 124,
  documentHeight = 8000,
}: SectionSpec = {}) {
  const el = document.createElement("section");
  el.id = id;
  el.scrollIntoView = vi.fn();
  el.getBoundingClientRect = () =>
    ({ top: top - window.scrollY, height }) as DOMRect;

  const heading = document.createElement("h2");
  heading.getBoundingClientRect = () =>
    ({ top: top + headingOffset - window.scrollY, height: 44 }) as DOMRect;
  el.appendChild(heading);

  document.body.appendChild(el);

  Object.defineProperty(document.documentElement, "scrollHeight", {
    configurable: true,
    value: documentHeight,
  });
  return el;
}

function setViewport(height: number) {
  Object.defineProperty(window, "innerHeight", {
    configurable: true,
    value: height,
  });
}

function setInsets({ top = 0, bottom = 0 }: { top?: number; bottom?: number }) {
  const style = document.documentElement.style;
  if (top) style.setProperty(TOP_INSET_VAR, `${top}px`);
  if (bottom) style.setProperty(BOTTOM_INSET_VAR, `${bottom}px`);
}

describe("reading the published insets", () => {
  afterEach(() => {
    document.documentElement.style.removeProperty(BOTTOM_INSET_VAR);
    document.documentElement.style.removeProperty(TOP_INSET_VAR);
  });

  it("is zero when nothing is fixed over the viewport", () => {
    expect(readBottomInset()).toBe(0);
    expect(readTopInset()).toBe(0);
  });

  it("reads the published banner and navbar heights", () => {
    setInsets({ top: 64, bottom: 126 });

    expect(readTopInset()).toBe(64);
    expect(readBottomInset()).toBe(126);
  });

  it("ignores a malformed or non-positive value", () => {
    document.documentElement.style.setProperty(BOTTOM_INSET_VAR, "auto");
    expect(readBottomInset()).toBe(0);

    document.documentElement.style.setProperty(BOTTOM_INSET_VAR, "0px");
    expect(readBottomInset()).toBe(0);

    document.documentElement.style.setProperty(TOP_INSET_VAR, "-8px");
    expect(readTopInset()).toBe(0);
  });
});

describe("scrollToSection", () => {
  let scrollTo: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    document.body.innerHTML = "";
    window.scrollY = 0;
    setViewport(DESKTOP_VIEWPORT);
    scrollTo = vi.fn();
    Object.defineProperty(window, "scrollTo", {
      configurable: true,
      value: scrollTo,
    });
    document.documentElement.style.removeProperty(BOTTOM_INSET_VAR);
    document.documentElement.style.removeProperty(TOP_INSET_VAR);
  });

  afterEach(() => {
    document.documentElement.style.removeProperty(BOTTOM_INSET_VAR);
    document.documentElement.style.removeProperty(TOP_INSET_VAR);
  });

  /** Where the section's own top edge lands in the viewport after the scroll. */
  function sectionTopOnScreen(sectionTop: number): number {
    return sectionTop - (scrollTo.mock.calls[0][0].top as number);
  }

  it("does nothing when the section is not on the page", () => {
    scrollToSection("nope");
    expect(scrollTo).not.toHaveBeenCalled();
  });

  it("keeps the plain top-alignment when nothing covers the viewport", () => {
    const el = mountSection();

    scrollToSection("pricing");

    expect(el.scrollIntoView).toHaveBeenCalledWith({
      behavior: "smooth",
      block: "start",
    });
    expect(scrollTo).not.toHaveBeenCalled();
  });

  /**
   * The regression. The old rule added the banner's full height to a *top*
   * alignment, so a section arrived that far above the viewport — with the
   * 180-220px banner Russian and Ukrainian wrap to on a phone, `#pricing`
   * landed with its label and title already scrolled off under the navbar.
   */
  it("never scrolls the section's heading off the top of the screen", () => {
    setViewport(PHONE_VIEWPORT);
    setInsets({ top: NAVBAR, bottom: TALL_BANNER });
    mountSection({ top: 2724, height: 903, headingOffset: 124 });

    scrollToSection("pricing");

    const headingOnScreen = 2724 + 124 - (scrollTo.mock.calls[0][0].top as number);
    expect(headingOnScreen).toBeGreaterThanOrEqual(NAVBAR);
    expect(headingOnScreen).toBeLessThan(PHONE_VIEWPORT - TALL_BANNER);
  });

  // The eyebrow label above the heading may slide under the navbar when the
  // section cannot fit — the heading itself may not.
  it("lands the heading exactly on the navbar's lower edge at the limit", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    mountSection({ top: 2724, height: 903, headingOffset: 124 });

    scrollToSection("pricing");

    const headingOnScreen =
      2724 + 124 - (scrollTo.mock.calls[0][0].top as number);
    expect(headingOnScreen).toBe(NAVBAR);
    expect(sectionTopOnScreen(2724)).toBeLessThan(NAVBAR);
  });

  /**
   * A section that fits between the two bars is placed whole inside them: the
   * heading below the navbar, the CTA at its bottom above the banner. No shift
   * beyond the header clearance is needed or wanted.
   */
  it("puts a section that fits entirely inside the usable band", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    const height = DESKTOP_VIEWPORT - NAVBAR - SHORT_BANNER - 40;
    mountSection({ top: 3000, height });

    scrollToSection("pricing");

    const top = sectionTopOnScreen(3000);
    expect(top).toBe(NAVBAR);
    expect(top + height).toBeLessThanOrEqual(DESKTOP_VIEWPORT - SHORT_BANNER);
  });

  // The original complaint: the plan CTAs at the bottom of a too-tall section
  // sat inside the stripe the banner covers, so none of them was clickable.
  it("lifts the CTA of a too-tall section out of the covered stripe", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    // A section just 90px taller than the band: the CTA at its very bottom is
    // reachable only if the scroll goes past a plain top alignment.
    const height = DESKTOP_VIEWPORT - NAVBAR - SHORT_BANNER + 90;
    mountSection({ top: 3000, height, headingOffset: 124 });

    scrollToSection("pricing");

    const ctaOnScreen = 3000 + height - (scrollTo.mock.calls[0][0].top as number);
    expect(ctaOnScreen).toBeLessThanOrEqual(DESKTOP_VIEWPORT - SHORT_BANNER);
  });

  // ...but not at the cost of the heading, which is the anchor's whole point.
  it("stops at the heading when the section cannot fit at all", () => {
    setViewport(PHONE_VIEWPORT);
    setInsets({ top: NAVBAR, bottom: TALL_BANNER });
    mountSection({ top: 3000, height: 1600, headingOffset: 124 });

    scrollToSection("pricing");

    // Exactly the heading limit: 3000 + 124 - 64.
    expect(scrollTo).toHaveBeenCalledWith({ top: 3060, behavior: "smooth" });
  });

  // A section whose heading is its first pixel has no room to give, and must
  // not be pushed at all.
  it("does not move past a section whose heading is at its top edge", () => {
    setInsets({ top: NAVBAR, bottom: TALL_BANNER });
    mountSection({ top: 3000, height: 2000, headingOffset: 0 });

    scrollToSection("pricing");

    expect(scrollTo).toHaveBeenCalledWith({ top: 3000 - NAVBAR, behavior: "smooth" });
  });

  it("tracks the measured banner height rather than a constant", () => {
    setViewport(PHONE_VIEWPORT);
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    // Taller than either band, but with enough heading clearance that the shift
    // is driven by the banner rather than clipped by the heading limit.
    mountSection({ top: 3000, height: 750, headingOffset: 400 });
    scrollToSection("pricing");
    const withShortBanner = scrollTo.mock.calls[0][0].top as number;

    scrollTo.mockClear();
    setInsets({ top: NAVBAR, bottom: TALL_BANNER });
    scrollToSection("pricing");
    const withTallBanner = scrollTo.mock.calls[0][0].top as number;

    expect(withTallBanner).toBeGreaterThan(withShortBanner);
  });

  // Overshooting a short section to clear a tall banner would push it off the
  // top of the screen for no benefit.
  it("never scrolls a short section past the top of the usable band", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    mountSection({ top: 3000, height: 200 });

    scrollToSection("pricing");

    expect(scrollTo).toHaveBeenCalledWith({ top: 3000 - NAVBAR, behavior: "smooth" });
  });

  it("never scrolls past the end of the document", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    mountSection({ top: 7800, height: 900, documentHeight: 8000 });

    scrollToSection("pricing");

    const target = scrollTo.mock.calls[0][0].top as number;
    expect(target).toBeLessThanOrEqual(8000 - DESKTOP_VIEWPORT);
  });

  it("never produces a negative scroll position", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    mountSection({ top: 0, height: 120 });

    scrollToSection("pricing");

    expect(scrollTo.mock.calls[0][0].top).toBe(0);
  });

  // A banner with no navbar (an app route) still has to be respected.
  it("works with only one of the two bars present", () => {
    setInsets({ bottom: SHORT_BANNER });
    mountSection({ top: 3000, height: 200 });

    scrollToSection("pricing");

    expect(scrollTo).toHaveBeenCalledWith({ top: 3000, behavior: "smooth" });
  });

  it("honours an explicit scroll behavior", () => {
    setInsets({ top: NAVBAR, bottom: SHORT_BANNER });
    mountSection({ top: 3000, height: 200 });

    scrollToSection("pricing", { behavior: "auto" });

    expect(scrollTo).toHaveBeenCalledWith({
      top: 3000 - NAVBAR,
      behavior: "auto",
    });
  });
});
