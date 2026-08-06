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
      description:
        "AI-powered job application tracker: organize your job search on a Kanban board, build ATS-friendly resumes, and generate tailored cover letters.",
      applicationCategory: "BusinessApplication",
      operatingSystem: "Web",
      image: `${SITE_URL}/og-image.png`,
      featureList: [
        "Job application tracking with Kanban board",
        "Custom application stages",
        "AI resume builder with ATS-friendly templates",
        "AI cover letter generator",
        "Company and contact management",
        "Job search analytics",
      ],
      offers: {
        "@type": "Offer",
        price: "0",
        priceCurrency: "USD",
      },
    },
    {
      "@context": "https://schema.org",
      "@type": "WebSite",
      name: "Jobber",
      url: SITE_URL,
      potentialAction: {
        "@type": "SearchAction",
        target: {
          "@type": "EntryPoint",
          urlTemplate: `${SITE_URL}/blog?q={search_term_string}`,
        },
        "query-input": "required name=search_term_string",
      },
    },
    {
      "@context": "https://schema.org",
      "@type": "Organization",
      name: "Jobber",
      url: SITE_URL,
      logo: `${SITE_URL}/favicon.png`,
      // Populate with verified official profile URLs only (Twitter/X, LinkedIn, GitHub).
      // Schema.org Organization.sameAs is a Google trust signal — placeholders hurt.
      sameAs: [] as string[],
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
