// Post-processing for the HTML snapshots the prerenderer writes.
//
// A prerendered page is a photograph of the running app, and the app puts
// state on `<body>` — which is fine until the same file is also the SPA
// fallback. `dist/index.html` is both: the prerendered landing page *and* what
// nginx serves for `/app/*`, for the auth routes and for anything unmatched.
// So a class the landing page had set was shipped to every route in the app,
// where nothing knew to take it off.

/**
 * Classes the running app owns on `<body>`, and must therefore never inherit
 * from the document it was served in.
 *
 * `landing-page` paints `<body>` in the landing palette. Baked into the
 * fallback shell it left the app's light theme drawn over a forced-dark
 * background.
 */
export const RUNTIME_BODY_CLASSES = Object.freeze(["landing-page"]);

/** The opening `<body>` tag and its attributes. */
const BODY_TAG = /<body\b([^>]*)>/i;

/** A `class="…"` (or `class='…'`) attribute, with the space in front of it. */
const CLASS_ATTRIBUTE = /(\s)class\s*=\s*(?:"([^"]*)"|'([^']*)')/gi;

/**
 * The same HTML with every runtime-owned class removed from `<body>`.
 *
 * Only the opening body tag is touched, and only the classes named above:
 * everything the build itself puts on the document is left exactly as it was.
 */
export function stripRuntimeBodyClasses(html) {
  return html.replace(BODY_TAG, (_tag, attributes) => {
    const cleaned = attributes.replace(
      CLASS_ATTRIBUTE,
      (attribute, space, doubleQuoted, singleQuoted) => {
        const value = doubleQuoted ?? singleQuoted ?? "";
        const kept = value
          .split(/\s+/)
          .filter(Boolean)
          .filter((name) => !RUNTIME_BODY_CLASSES.includes(name));

        if (kept.length === 0) return "";
        const quote = doubleQuoted === undefined ? "'" : '"';
        return `${space}class=${quote}${kept.join(" ")}${quote}`;
      },
    );

    return `<body${cleaned}>`;
  });
}
