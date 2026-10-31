import { useRouteError } from "react-router-dom";

/**
 * Hands a crash inside a page on to GlobalErrorBoundary.
 *
 * Without an error element React Router catches it first: the visitor gets its
 * bare developer screen with the raw error message, and the error never reaches
 * Sentry, because that boundary reports nothing. Page crashes are the errors
 * worth knowing about, so the router must not be the one to swallow them.
 */
export function RouteError(): never {
  throw useRouteError();
}
