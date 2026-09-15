import { describe, it, expect } from "vitest";

import en from "../en.json";
import ru from "../ru.json";
import uk from "../uk.json";

type Bundle = Record<string, unknown>;

const LOCALES: Record<string, Bundle> = { en, ru, uk };

/** Suffixes i18next appends per plural category. */
const PLURAL_SUFFIXES = ["_one", "_few", "_many", "_other"] as const;

function flatten(bundle: Bundle, prefix = ""): Map<string, string> {
  const out = new Map<string, string>();
  for (const [key, value] of Object.entries(bundle)) {
    const path = `${prefix}${key}`;
    if (value && typeof value === "object" && !Array.isArray(value)) {
      for (const [k, v] of flatten(value as Bundle, `${path}.`)) out.set(k, v);
    } else {
      out.set(path, String(value));
    }
  }
  return out;
}

/** The key without its plural suffix, so `_few` and `_many` fold into one. */
function baseKey(key: string): string {
  const suffix = PLURAL_SUFFIXES.find((s) => key.endsWith(s));
  return suffix ? key.slice(0, -suffix.length) : key;
}

const FLAT = Object.fromEntries(
  Object.entries(LOCALES).map(([name, bundle]) => [name, flatten(bundle)]),
);

/**
 * Ukrainian moved from the app's own "ua" tag to the BCP 47 "uk", which meant
 * renaming the bundle and touching every message that mentions a plan limit.
 * A key that survived in one file and not another shows up as a raw
 * `settings.subscription.…` string on screen, in one language only — the kind
 * of thing that reaches production because nobody runs the app in Ukrainian.
 */
describe("the three bundles carry the same keys", () => {
  it.each(["ru", "uk"])("%s is missing nothing English has", (locale) => {
    const missing = [...FLAT.en.keys()]
      .map(baseKey)
      .filter((key) => ![...FLAT[locale].keys()].some((k) => baseKey(k) === key));

    expect([...new Set(missing)]).toEqual([]);
  });

  it.each(["ru", "uk"])("%s carries nothing English does not", (locale) => {
    const extra = [...FLAT[locale].keys()]
      .map(baseKey)
      .filter((key) => ![...FLAT.en.keys()].some((k) => baseKey(k) === key));

    expect([...new Set(extra)]).toEqual([]);
  });

  // The Slavic languages need `_few` and `_many` where English needs neither,
  // so a counted key must be complete in each of them.
  it.each(["ru", "uk"])("gives every counted key all four forms in %s", (locale) => {
    const counted = new Set(
      [...FLAT[locale].keys()]
        .filter((key) => PLURAL_SUFFIXES.some((s) => key.endsWith(s)))
        .map(baseKey),
    );

    expect(counted.size).toBeGreaterThan(0);
    for (const key of counted) {
      for (const suffix of PLURAL_SUFFIXES) {
        expect(FLAT[locale].has(key + suffix), `${key}${suffix} in ${locale}`).toBe(
          true,
        );
      }
    }
  });

  it.each(["en", "ru", "uk"])("leaves no message empty in %s", (locale) => {
    for (const [key, value] of FLAT[locale]) {
      expect(value.trim(), `${key} in ${locale}`).not.toBe("");
    }
  });
});
