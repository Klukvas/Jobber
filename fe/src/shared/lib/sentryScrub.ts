import type { Breadcrumb } from "@sentry/react";
import { cleanUrl } from "./analyticsUrl";

/**
 * What a report may not carry. A reported error can quote an address or an email
 * anywhere: the page URL, the referrer, a breadcrumb, a span, its own message.
 * Walking the whole report is what keeps "anywhere" true when the SDK adds a
 * field, and it is cheap because reports are rare.
 */
const EMAIL = /[\w.+-]+@[\w-]+\.[\w.-]+/g;
const ABSOLUTE_URL = /https?:\/\/[^\s"'<>)]+/g;
const RELATIVE_URL = /^\/\S*$/;

/** Reports are shallow; this only stops a value that refers to itself. */
const MAX_DEPTH = 16;

/**
 * The attributes the SDK quotes in a click's selector. These hold what is on
 * screen: an aria-label or title here can be a job or company name.
 */
const SELECTOR_TEXT = /\[(aria-label|title|alt|name)="[^"]*"\]/g;

function scrubText(text: string): string {
  const scrubbed = text
    .replace(EMAIL, "[email]")
    .replace(ABSOLUTE_URL, cleanUrl);
  return RELATIVE_URL.test(scrubbed) ? cleanUrl(scrubbed) : scrubbed;
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return (
    value !== null &&
    typeof value === "object" &&
    Object.getPrototypeOf(value) === Object.prototype
  );
}

function scrubValue(value: unknown, depth = 0): unknown {
  if (typeof value === "string") return scrubText(value);
  if (depth >= MAX_DEPTH) return value;
  if (Array.isArray(value)) {
    return value.map((entry) => scrubValue(entry, depth + 1));
  }
  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, entry]) => [
        key,
        scrubValue(entry, depth + 1),
      ]),
    );
  }
  return value;
}

/**
 * For `beforeSend` and `beforeSendTransaction`. `sdkProcessingMetadata` is the
 * SDK's own and never sent: it holds live scopes the client still uses.
 */
export function scrubSentryEvent<T extends object>(event: T): T {
  const { sdkProcessingMetadata, ...sent } = event as T & {
    sdkProcessingMetadata?: unknown;
  };
  return {
    ...(scrubValue(sent) as object),
    ...(sdkProcessingMetadata === undefined ? {} : { sdkProcessingMetadata }),
  } as T;
}

/** For `beforeBreadcrumb`; returning null is how the SDK drops one. */
export function scrubBreadcrumb(breadcrumb: Breadcrumb): Breadcrumb | null {
  if (breadcrumb.category === "console") return null;

  const scrubbed = scrubValue(breadcrumb) as Breadcrumb;
  return breadcrumb.category?.startsWith("ui.")
    ? { ...scrubbed, message: scrubbed.message?.replace(SELECTOR_TEXT, "") }
    : scrubbed;
}
