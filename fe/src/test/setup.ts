import "@testing-library/jest-dom/vitest";

// jsdom doesn't implement matchMedia. Components using useMediaQuery (e.g. the
// Dialog sheet behavior) call it during render, so provide a no-match stub —
// tests default to the desktop / non-sheet path unless a test overrides it.
if (typeof window !== "undefined" && !window.matchMedia) {
  window.matchMedia = (query: string): MediaQueryList => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => false,
  });
}
