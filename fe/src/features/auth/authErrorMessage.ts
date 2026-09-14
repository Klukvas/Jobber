import type { ApiError } from "@/services/api";

/**
 * The localisation key for an authentication failure.
 *
 * The API answers with a machine-readable `error_code` and an English
 * sentence. The sentence is for logs and for API clients — it is not copy, it
 * is never translated, and it went straight onto the login form: a visitor
 * reading the app in Russian who mistyped a password was told "Invalid email
 * or password" in English. `auth.invalidCredentials` had existed in all three
 * locales the whole time, unused.
 *
 * So the code is what is mapped, and only codes this app has something to say
 * about. Everything else — an unrecognised code, a 500, a request that never
 * left the browser — is one generic localised line: a backend sentence is not
 * copy either way, and a raw code on screen is worse than useless to the
 * person reading it.
 */
const MESSAGE_KEY_BY_CODE: Readonly<Record<string, string>> = {
  INVALID_CREDENTIALS: "auth.invalidCredentials",
};

/** What the visitor is told when nothing more specific is known. */
export const GENERIC_AUTH_ERROR_KEY = "errors.somethingWentWrong";

export function authErrorMessageKey(error: unknown): string {
  const code = (error as Partial<ApiError> | null)?.code;
  if (typeof code !== "string") return GENERIC_AUTH_ERROR_KEY;
  return MESSAGE_KEY_BY_CODE[code] ?? GENERIC_AUTH_ERROR_KEY;
}
