import { describe, it, expect } from "vitest";

import {
  RUNTIME_BODY_CLASSES,
  stripRuntimeBodyClasses,
} from "../shellHtml.mjs";

/**
 * The regression this guards is in the *artifact*, not in the app: the
 * prerendered landing page is written to `dist/index.html`, which is also the
 * SPA fallback nginx serves for `/app/*`. A `landing-page` class captured on
 * `<body>` therefore shipped to every route in the product, and the app
 * rendered its light theme over the landing page's forced-dark background.
 */
describe("stripRuntimeBodyClasses", () => {
  it("removes the landing class the landing snapshot captured", () => {
    const html = stripRuntimeBodyClasses(
      '<!doctype html><html><body class="landing-page"><div id="root"></div></body></html>',
    );

    expect(html).toContain("<body>");
    expect(html).not.toContain("landing-page");
  });

  it("keeps every other class on the body", () => {
    const html = stripRuntimeBodyClasses(
      '<body class="antialiased landing-page print:bg-white">x</body>',
    );

    expect(html).toContain('<body class="antialiased print:bg-white">');
  });

  it("leaves a body that never carried a runtime class alone", () => {
    const html = '<body class="antialiased" data-theme="light">x</body>';

    expect(stripRuntimeBodyClasses(html)).toBe(html);
  });

  it("leaves a body with no attributes at all alone", () => {
    expect(stripRuntimeBodyClasses("<body>x</body>")).toBe("<body>x</body>");
  });

  it("keeps the body's other attributes", () => {
    const html = stripRuntimeBodyClasses(
      '<body data-scroll="0" class="landing-page" lang="en">x</body>',
    );

    expect(html).toContain('<body data-scroll="0" lang="en">');
  });

  it("handles single-quoted class attributes", () => {
    const html = stripRuntimeBodyClasses(
      "<body class='landing-page dark'>x</body>",
    );

    expect(html).toContain("<body class='dark'>");
  });

  it("does not touch classes on any other element", () => {
    const html = stripRuntimeBodyClasses(
      '<body class="landing-page"><div class="landing-page">x</div></body>',
    );

    expect(html).toBe('<body><div class="landing-page">x</div></body>');
  });

  it("names the landing class as runtime-owned", () => {
    expect(RUNTIME_BODY_CLASSES).toContain("landing-page");
  });
});
