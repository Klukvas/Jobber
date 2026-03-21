import { useEffect } from "react";

const SCRIPT_ID = "jobber-jsonld";

const SITE_URL = import.meta.env.VITE_SITE_URL ?? "https://jobber-app.com";

function buildJsonLd() {
  const safeUrl = String(SITE_URL).replace(/<\/script>/gi, "");
  return JSON.stringify({
    "@context": "https://schema.org",
    "@type": "WebApplication",
    name: "Jobber",
    url: safeUrl,
    description: "Job application tracking platform",
    applicationCategory: "BusinessApplication",
    operatingSystem: "Web",
  });
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
