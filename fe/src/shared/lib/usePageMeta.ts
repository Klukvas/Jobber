import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";

const SITE_URL = "https://jobber-app.com";

interface PageMetaOptions {
  readonly title?: string;
  readonly titleKey?: string;
  readonly description?: string;
  readonly descriptionKey?: string;
  readonly noindex?: boolean;
}

function setMetaTag(name: string, content: string) {
  const existing = document.querySelector(`meta[name="${name}"]`);
  if (existing) {
    existing.setAttribute("content", content);
    return;
  }
  const meta = document.createElement("meta");
  meta.name = name;
  meta.content = content;
  document.head.appendChild(meta);
}

function setOgTag(property: string, content: string) {
  const existing = document.querySelector(`meta[property="${property}"]`);
  if (existing) {
    existing.setAttribute("content", content);
    return;
  }
  const meta = document.createElement("meta");
  meta.setAttribute("property", property);
  meta.content = content;
  document.head.appendChild(meta);
}

function removeMetaTag(name: string) {
  document.querySelector(`meta[name="${name}"]`)?.remove();
}

function setCanonical(href: string) {
  let link = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (link) {
    link.href = href;
    return;
  }
  link = document.createElement("link");
  link.rel = "canonical";
  link.href = href;
  document.head.appendChild(link);
}

export function usePageMeta(options: PageMetaOptions = {}) {
  const {
    title: literalTitle,
    titleKey,
    description: literalDescription,
    descriptionKey,
    noindex = false,
  } = options;
  const { t } = useTranslation();
  const { pathname } = useLocation();

  useEffect(() => {
    const title =
      literalTitle ?? (titleKey ? t(titleKey) : t("seo.defaultTitle"));
    const description =
      literalDescription ??
      (descriptionKey ? t(descriptionKey) : t("seo.defaultDescription"));

    const pageUrl = `${SITE_URL}${pathname === "/" ? "" : pathname}`;

    document.title = title;
    setMetaTag("description", description);
    setOgTag("og:title", title);
    setOgTag("og:description", description);
    setOgTag("og:url", pageUrl);
    setCanonical(pageUrl);

    if (noindex) {
      setMetaTag("robots", "noindex, nofollow");
    } else {
      removeMetaTag("robots");
    }

    return () => {
      if (noindex) {
        removeMetaTag("robots");
      }
    };
  }, [
    literalTitle,
    titleKey,
    literalDescription,
    descriptionKey,
    noindex,
    pathname,
    t,
  ]);
}
