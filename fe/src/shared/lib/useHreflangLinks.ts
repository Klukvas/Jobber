import { useEffect } from "react";

const SITE_URL = "https://jobber-app.com";
const DATA_ATTR = "data-hreflang-managed";

export interface HreflangLink {
  readonly hreflang: string;
  readonly href: string;
}

/**
 * Injects <link rel="alternate" hreflang> tags for the current page and
 * removes them on unmount, so alternates never leak onto other routes.
 * Adds x-default pointing at the "en" alternate when one exists.
 *
 * Pass paths (e.g. "/blog/my-post") — they are resolved against the
 * canonical site origin, since hreflang requires absolute URLs.
 */
export function useHreflangLinks(
  links: readonly { hreflang: string; path: string }[],
) {
  // Effect dependency: a stable primitive so re-renders with an equal array
  // don't thrash the DOM.
  const serialized = JSON.stringify(links);

  useEffect(() => {
    document
      .querySelectorAll(`link[${DATA_ATTR}]`)
      .forEach((el) => el.remove());

    const parsed = JSON.parse(serialized) as readonly {
      hreflang: string;
      path: string;
    }[];
    if (parsed.length === 0) return;

    const toInsert: HreflangLink[] = parsed.map((l) => ({
      hreflang: l.hreflang,
      href: `${SITE_URL}${l.path}`,
    }));
    const english = toInsert.find((l) => l.hreflang === "en");
    if (english) {
      toInsert.push({ hreflang: "x-default", href: english.href });
    }

    for (const { hreflang, href } of toInsert) {
      const link = document.createElement("link");
      link.rel = "alternate";
      link.hreflang = hreflang;
      link.href = href;
      link.setAttribute(DATA_ATTR, "true");
      document.head.appendChild(link);
    }

    return () => {
      document
        .querySelectorAll(`link[${DATA_ATTR}]`)
        .forEach((el) => el.remove());
    };
  }, [serialized]);
}
