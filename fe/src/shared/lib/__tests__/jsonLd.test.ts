import { describe, it, expect } from "vitest";

import { serializeJsonLd } from "../jsonLd";

/**
 * The prerenderer bakes these `<script type="application/ld+json">` elements
 * into the HTML it ships, so whatever they contain is what a crawler parses.
 * An HTML parser ends a `<script>` at the first `</script`, wherever it falls —
 * including the middle of a JSON string.
 */
describe("serializeJsonLd", () => {
  it("never emits a sequence that closes the script element", () => {
    const serialized = serializeJsonLd({
      headline: "Why </script> tags break SEO",
    });

    expect(serialized).not.toContain("</script");
    expect(serialized).not.toContain("<");
  });

  it("keeps the data byte-for-byte the same to a JSON reader", () => {
    const post = {
      headline: 'Why </script><img src=x onerror="alert(1)"> breaks SEO',
      description: "Angle brackets < and > in prose",
      tags: ["<b>", "seo"],
    };

    expect(JSON.parse(serializeJsonLd(post))).toEqual(post);
  });

  it("leaves structured data without any markup untouched", () => {
    const plain = { "@type": "BlogPosting", headline: "Ten resume tips" };

    expect(serializeJsonLd(plain)).toBe(JSON.stringify(plain));
  });
});
