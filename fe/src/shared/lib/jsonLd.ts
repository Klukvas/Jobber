/**
 * Serialisation for structured data that is about to live inside a
 * `<script type="application/ld+json">`.
 *
 * An HTML parser does not parse the contents of a `<script>` — it scans for the
 * first `</script` and ends the element there. So a blog post titled
 * `Why </script> tags break SEO` produced markup whose script closed in the
 * middle of the JSON, and everything after it — the rest of the object, the
 * second `</script>` — was parsed as page content. The prerenderer bakes these
 * elements into the shipped HTML, so that markup is what crawlers and browsers
 * actually receive.
 *
 * `<` is written as its `<` escape, which is the same character to any
 * JSON reader: the document Google parses is byte-for-byte equivalent, and the
 * sequence the HTML parser looks for never appears. Nothing else needs escaping
 * — JSON has no `<` outside string values, and `>` and `&` are only meaningful
 * to the parser after one.
 */
export function serializeJsonLd(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
