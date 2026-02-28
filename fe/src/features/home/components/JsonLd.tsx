import { useEffect } from 'react';

const SITE_URL = import.meta.env.VITE_SITE_URL ?? 'https://jobber-app';

const jsonLd = {
  '@context': 'https://schema.org',
  '@type': 'WebApplication',
  name: 'Jobber',
  url: SITE_URL,
  description: 'Job application tracking platform',
  applicationCategory: 'BusinessApplication',
  operatingSystem: 'Web',
};

export function JsonLd() {
  useEffect(() => {
    const script = document.createElement('script');
    script.type = 'application/ld+json';
    script.text = JSON.stringify(jsonLd);
    document.head.appendChild(script);
    return () => {
      script.remove();
    };
  }, []);

  return null;
}
