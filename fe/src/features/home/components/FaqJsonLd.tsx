import { useEffect } from "react";

const SCRIPT_ID = "jobber-faq-jsonld";

interface FaqItem {
  readonly key: string;
  readonly question: string;
  readonly answer: string;
}

interface FaqJsonLdProps {
  readonly items: readonly FaqItem[];
}

export function FaqJsonLd({ items }: FaqJsonLdProps) {
  useEffect(() => {
    if (items.length === 0) return;

    const schema = {
      "@context": "https://schema.org",
      "@type": "FAQPage",
      mainEntity: items.map((item) => ({
        "@type": "Question",
        name: item.question,
        acceptedAnswer: {
          "@type": "Answer",
          text: item.answer,
        },
      })),
    };

    const existing = document.getElementById(SCRIPT_ID);
    if (existing) existing.remove();

    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.type = "application/ld+json";
    script.text = JSON.stringify(schema);
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
  }, [items]);

  return null;
}
