import { useEffect } from "react";
import { useTranslation } from "react-i18next";

const SITE_URL = "https://jobber-app.com";
const SCRIPT_ID = "feature-page-jsonld";
const FAQ_KEYS = ["1", "2", "3", "4", "5"] as const;

interface FeatureFaqProps {
  /** Locale namespace of the page, e.g. "featurePages.applications" */
  readonly ns: string;
  /** Route path, e.g. "/features/applications" — used in BreadcrumbList */
  readonly path: string;
}

/**
 * FAQ section for feature landing pages. Renders the visible Q&A list and
 * injects matching FAQPage + BreadcrumbList JSON-LD, so the structured data
 * always mirrors the on-page content (a Google requirement for rich results).
 */
export function FeatureFaq({ ns, path }: FeatureFaqProps) {
  const { t, i18n } = useTranslation();

  const items = FAQ_KEYS.map((k) => ({
    question: t(`${ns}.faq.q${k}`),
    answer: t(`${ns}.faq.a${k}`),
  }));
  const pageTitle = t(`${ns}.meta.title`);

  useEffect(() => {
    document.getElementById(SCRIPT_ID)?.remove();

    const jsonLd = [
      {
        "@context": "https://schema.org",
        "@type": "FAQPage",
        mainEntity: items.map((item) => ({
          "@type": "Question",
          name: item.question,
          acceptedAnswer: { "@type": "Answer", text: item.answer },
        })),
      },
      {
        "@context": "https://schema.org",
        "@type": "BreadcrumbList",
        itemListElement: [
          {
            "@type": "ListItem",
            position: 1,
            name: "Jobber",
            item: SITE_URL,
          },
          {
            "@type": "ListItem",
            position: 2,
            name: pageTitle,
            item: `${SITE_URL}${path}`,
          },
        ],
      },
    ];

    const script = document.createElement("script");
    script.id = SCRIPT_ID;
    script.type = "application/ld+json";
    script.text = JSON.stringify(jsonLd);
    document.head.appendChild(script);

    return () => {
      script.remove();
    };
    // items derives from t(); language is the only real input that changes it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ns, path, i18n.language]);

  return (
    <section className="border-t bg-muted/30 py-24">
      <div className="mx-auto max-w-3xl px-4">
        <h2 className="mb-10 text-center text-3xl font-bold">
          {t(`${ns}.faq.title`)}
        </h2>
        <dl className="space-y-8">
          {items.map((item) => (
            <div key={item.question}>
              <dt className="mb-2 text-lg font-semibold">{item.question}</dt>
              <dd className="leading-relaxed text-muted-foreground">
                {item.answer}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
