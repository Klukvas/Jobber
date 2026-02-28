import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

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
    existing.setAttribute('content', content);
    return;
  }
  const meta = document.createElement('meta');
  meta.name = name;
  meta.content = content;
  document.head.appendChild(meta);
}

function setOgTag(property: string, content: string) {
  const existing = document.querySelector(`meta[property="${property}"]`);
  if (existing) {
    existing.setAttribute('content', content);
    return;
  }
  const meta = document.createElement('meta');
  meta.setAttribute('property', property);
  meta.content = content;
  document.head.appendChild(meta);
}

function removeMetaTag(name: string) {
  const existing = document.querySelector(`meta[name="${name}"]`);
  if (existing) {
    existing.remove();
  }
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

  useEffect(() => {
    const title =
      literalTitle ?? (titleKey ? t(titleKey) : t('seo.defaultTitle'));
    const description =
      literalDescription ??
      (descriptionKey ? t(descriptionKey) : t('seo.defaultDescription'));

    document.title = title;
    setMetaTag('description', description);
    setOgTag('og:title', title);
    setOgTag('og:description', description);

    if (noindex) {
      setMetaTag('robots', 'noindex, nofollow');
    } else {
      removeMetaTag('robots');
    }

    return () => {
      if (noindex) {
        removeMetaTag('robots');
      }
    };
  }, [literalTitle, titleKey, literalDescription, descriptionKey, noindex, t]);
}
