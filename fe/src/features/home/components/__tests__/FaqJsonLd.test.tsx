import { describe, it, expect, afterEach, vi } from "vitest";
import { render } from "@testing-library/react";

import { FaqJsonLd } from "../FaqJsonLd";
import { FeatureFaq } from "../FeatureFaq";

// The FAQ copy is translator-supplied, so an apostrophe-and-angle-bracket
// answer is ordinary editorial content rather than an attack — and it still
// lands inside a <script> the prerenderer bakes into the shipped HTML.
const BREAKOUT = 'Read the </script><img src=x onerror="alert(1)"> guide';

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => `${BREAKOUT} ${key}`,
    i18n: { language: "en" },
  }),
}));

afterEach(() => {
  document.getElementById("jobber-faq-jsonld")?.remove();
  document.getElementById("feature-page-jsonld")?.remove();
});

describe("FaqJsonLd", () => {
  const items = [
    { key: "escape", question: BREAKOUT, answer: BREAKOUT },
  ] as const;

  it("emits structured data that cannot close its own script element", () => {
    render(<FaqJsonLd items={items} />);

    const script = document.getElementById("jobber-faq-jsonld");
    expect(script?.textContent).not.toContain("</script");
    expect(script?.textContent).not.toContain("<");
  });

  it("still describes the questions exactly, escaping and all", () => {
    render(<FaqJsonLd items={items} />);

    const script = document.getElementById("jobber-faq-jsonld");
    const schema = JSON.parse(script?.textContent ?? "{}");

    expect(schema.mainEntity[0].name).toBe(BREAKOUT);
    expect(schema.mainEntity[0].acceptedAnswer.text).toBe(BREAKOUT);
  });
});

describe("FeatureFaq", () => {
  it("emits structured data that cannot close its own script element", () => {
    render(<FeatureFaq ns="featurePages.applications" path="/features/x" />);

    const script = document.getElementById("feature-page-jsonld");
    expect(script?.textContent).not.toContain("</script");
    expect(script?.textContent).not.toContain("<");
  });

  it("still describes the questions and breadcrumb exactly", () => {
    render(<FeatureFaq ns="featurePages.applications" path="/features/x" />);

    const script = document.getElementById("feature-page-jsonld");
    const [faqPage, breadcrumb] = JSON.parse(script?.textContent ?? "[]");

    expect(faqPage.mainEntity[0].name).toContain(BREAKOUT);
    expect(breadcrumb.itemListElement[1].name).toContain(BREAKOUT);
  });
});
