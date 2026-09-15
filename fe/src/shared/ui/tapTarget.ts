/**
 * A 44x44 minimum for a control that renders as inline text.
 *
 * Footer links are 13px type with no padding of their own, so their hit area
 * is the glyphs — "FAQ" measured 24px wide, "Terms" 36. Both WCAG 2.5.5 and
 * the Apple HIG ask for 44 in each direction, and these sit in a wrapped row
 * where the neighbour is a few pixels away. The minimum is dropped from `sm`
 * up so pointer layouts keep the density they were drawn with.
 *
 * Height alone was not enough: the earlier pass gave these links `min-h-11`
 * and left the width at whatever the word happened to be.
 */
export const TAP_TARGET_INLINE =
  "inline-flex min-h-11 min-w-11 items-center justify-center sm:min-h-0 sm:min-w-0";

/**
 * A 44x44 hit area for an icon-only control in the landing navbar.
 *
 * `sm` was the wrong place to hand these back to a pointer. At 768px the
 * navbar already swaps the hamburger for its full row of links, and the icon
 * buttons beside them dropped to 36x36 — on a screen that is still a tablet
 * held in two hands. The compact size now waits for `lg`; in between, the extra
 * 8px in each direction is absorbed by an equal negative margin, so the box a
 * finger hits grows and the row it sits in does not move.
 */
export const TAP_TARGET_ICON = "h-11 w-11 sm:-m-1 lg:m-0 lg:h-9 lg:w-9";
