/**
 * Anchor scrolling that knows what is covering the viewport.
 *
 * Two things can sit over the landing page: the navbar fixed to the top and the
 * consent banner fixed to the bottom. Each publishes its measured height as a
 * CSS custom property, so the band a visitor can actually see and touch is
 * `[topInset, viewportHeight - bottomInset]`, and this module's only job is to
 * put a section inside it.
 */

/**
 * CSS custom properties carrying the height of the fixed obstructions.
 * Published on `<html>` by whatever owns each bar; absent means nothing is
 * covering that edge of the viewport.
 */
export const BOTTOM_INSET_VAR = "--app-bottom-inset";
export const TOP_INSET_VAR = "--app-top-inset";

function readInset(variable: string): number {
  if (typeof document === "undefined") return 0;
  const raw = getComputedStyle(document.documentElement)
    .getPropertyValue(variable)
    .trim();
  if (!raw) return 0;
  const px = Number.parseFloat(raw);
  return Number.isFinite(px) && px > 0 ? px : 0;
}

/** Reads the obstructed bottom band in pixels. 0 when nothing is covering it. */
export function readBottomInset(): number {
  return readInset(BOTTOM_INSET_VAR);
}

/** Reads the fixed header's height in pixels. 0 when there is no fixed header. */
export function readTopInset(): number {
  return readInset(TOP_INSET_VAR);
}

interface ScrollToSectionOptions {
  readonly behavior?: ScrollBehavior;
}

/**
 * Absolute document position of the first thing in the section that has to stay
 * on screen — its heading. Falls back to the section's own top edge when there
 * is no heading to find, which is the strictest reading and never scrolls past
 * anything.
 */
function firstHeadingTop(section: HTMLElement, sectionTop: number): number {
  const heading = section.querySelector("h1, h2, h3");
  if (!heading) return sectionTop;
  const top = heading.getBoundingClientRect().top + window.scrollY;
  return Number.isFinite(top) ? Math.max(top, sectionTop) : sectionTop;
}

/**
 * Scrolls a section into view, respecting whatever is fixed over the viewport.
 *
 * The rule is geometric, not a fudge factor:
 *
 *  - start at `elementTop - topInset`, which puts the section's own top edge on
 *    the first visible pixel below the navbar. For any section that fits in the
 *    usable band that is the whole answer — heading in view at the top, CTAs
 *    above the banner.
 *  - when the section is taller than the band, scroll on so its bottom edge
 *    reaches the band's bottom and the CTAs down there become usable — but
 *    never past the point where the section's own heading would be pushed under
 *    the navbar.
 *
 * What it must not do is what it used to: add the banner's whole height on top
 * of the alignment. A bar fixed to the *bottom* does not move the *top* of the
 * viewport, so that was compensation pointing the wrong way — with the 180-220px
 * banner Russian and Ukrainian wrap to on a phone, `#pricing` arrived with its
 * label and title already scrolled off underneath the navbar.
 *
 * With neither bar present this is exactly the old
 * `scrollIntoView({ block: "start" })`.
 */
export function scrollToSection(
  id: string,
  { behavior = "smooth" }: ScrollToSectionOptions = {},
): void {
  const element = document.getElementById(id);
  if (!element) return;

  const topInset = readTopInset();
  const bottomInset = readBottomInset();

  if (topInset <= 0 && bottomInset <= 0) {
    element.scrollIntoView({ behavior, block: "start" });
    return;
  }

  const currentTop = window.scrollY;
  const rect = element.getBoundingClientRect();
  const elementTop = rect.top + currentTop;
  const viewportHeight = window.innerHeight;
  const bandBottom = viewportHeight - bottomInset;

  // The section's top edge on the first visible pixel below the navbar.
  const alignTop = elementTop - topInset;
  // The section's bottom edge on the banner's top edge. Only ever greater than
  // alignTop when the section is taller than the band.
  const alignBottom = elementTop + rect.height - bandBottom;
  // The furthest scroll that still leaves the heading below the navbar.
  const headingLimit = firstHeadingTop(element, elementTop) - topInset;

  const maxScroll = Math.max(
    0,
    document.documentElement.scrollHeight - viewportHeight,
  );

  const target = Math.min(
    Math.max(alignTop, Math.min(alignBottom, headingLimit)),
    maxScroll,
  );

  window.scrollTo({ top: Math.max(0, target), behavior });
}
