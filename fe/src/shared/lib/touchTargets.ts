/**
 * The 44px floor a thumb needs, on controls the app draws at pointer density:
 * view toggles (32px tall), inline links (28) and icon-only buttons (24x24).
 * Applied below `sm` only — the same pages on a pointer keep every one of
 * these exactly as they were drawn.
 */
export const TOUCH_ROW_TARGET =
  "flex items-center justify-center max-sm:min-h-11";

export const TOUCH_ICON_TARGET =
  "flex h-6 w-6 items-center justify-center max-sm:h-11 max-sm:w-11";
