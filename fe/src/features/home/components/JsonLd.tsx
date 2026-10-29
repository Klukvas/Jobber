import { useEffect } from "react";

import { serializeJsonLd } from "@/shared/lib/jsonLd";

const SCRIPT_ID = "jobber-jsonld";
const SITE_URL = "https://cursopus.com";

function buildJsonLd() {
  return serializeJsonLd([
    {
      "@context": "https://schema.org",
      "@type": "WebApplication",
      name: "Cursopus",
      url: SITE_URL,
      description:
        "AI-powered job application tracker: organize your job search on a Kanban board, build and tailor resumes, and generate tailored cover letters.",
      applicationCategory: "BusinessApplication",
      operatingSystem: "Web",
      image: `${SITE_URL}/og-image.png`,
      featureList: [
        "Job application tracking with Kanban board",
        "Custom application stages",
        "Resume builder with AI suggestions and PDF export",
        "AI cover letter generator",
        "Company and contact management",
        "Job search analytics",
      ],
      offers: [
        { "@type": "Offer", name: "Free", price: "0", priceCurrency: "USD", description: "30 AI credits once per account; resource limits apply." },
        { "@type": "Offer", name: "Pro", price: "24", priceCurrency: "USD", description: "Monthly subscription with 500 AI credits per confirmed paid period; no rollover; usage and safety limits apply." },
      ],
    },
    {
      "@context": "https://schema.org",
      "@type": "WebSite",
      name: "Cursopus",
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
      name: "Cursopus",
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
