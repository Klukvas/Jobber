import { useEffect } from "react";

const SCRIPT_ID = "jobber-jsonld";
const SITE_URL = "https://jobber-app.com";

function buildJsonLd() {
  return JSON.stringify([
    {
      "@context": "https://schema.org",
      "@type": "WebApplication",
      name: "Jobber",
      url: SITE_URL,
      description: "Job application tracking platform",
      applicationCategory: "BusinessApplication",
      operatingSystem: "Web",
    },
    {
      "@context": "https://schema.org",
      "@type": "Organization",
      name: "Jobber",
      url: SITE_URL,
      logo: `${SITE_URL}/favicon.png`,
      sameAs: [],
    },
  ]);
}

export function JsonLd() {
  useEffect(() => {
    if (document.getElementById(SCRIPT_ID)) return;

    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.type = "application/ld+json";
    script.text = buildJsonLd();
    document.head.appendChild(script);
    return () => {
      script.remove();
    };
  }, []);

  return null;
}
