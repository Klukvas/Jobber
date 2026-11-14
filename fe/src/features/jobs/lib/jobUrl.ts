/**
 * Whether a job's link can be saved: empty (the field is optional) or an
 * http(s) URL. The create form enforced this through `type="url"`, but the
 * detail page saves without submitting a form, so `not-a-url` went through.
 */
export function isSaveableJobUrl(url: string): boolean {
  return url.trim() === "" || isHttpUrl(url);
}

export function isHttpUrl(url: string): boolean {
  try {
    const { protocol } = new URL(url);
    return protocol === "http:" || protocol === "https:";
  } catch {
    return false;
  }
}
