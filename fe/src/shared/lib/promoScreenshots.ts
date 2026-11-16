// Bump after retaking promo screenshots to refresh CDN and browser caches.
const PROMO_SCREENSHOT_VERSION = "20261116";

export function promoScreenshotSrc(filename: string): string {
  return `/screenshots/${filename}?v=${PROMO_SCREENSHOT_VERSION}`;
}
