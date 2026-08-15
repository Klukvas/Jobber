/**
 * Feature flags — centralised on/off switches per environment.
 *
 * Compile-time flags use VITE_FEATURE_* env vars (default: disabled).
 * Hardcoded flags are toggled here directly.
 *
 * GOOGLE_CALENDAR: Google Calendar integration (connect/disconnect in Settings,
 * "Schedule" button on application stages in Timeline).
 * Hidden because the OAuth app hasn't passed Google verification yet.
 * To re-enable: set to true and publish the OAuth app in Google Cloud Console.
 * See: docs/FEATURES.md
 *
 * SENTRY: Error tracking via Sentry. Set VITE_FEATURE_SENTRY=true to enable.
 * Also requires VITE_SENTRY_DSN to be set.
 *
 * EMAIL_NOTIFICATIONS: Controls whether email-dependent UI is shown
 * (e.g. "resend verification" link). Backend has its own flag.
 *
 * PAYMENTS: Paddle checkout & upgrade UI. Set VITE_FEATURE_PAYMENTS=true
 * to enable upgrade banners, pricing modal, and checkout flow.
 * Subscription limits/usage still work regardless of this flag.
 */
export const FEATURES = {
  GOOGLE_CALENDAR: false,
  SENTRY: import.meta.env.VITE_FEATURE_SENTRY === "true",
  EMAIL_NOTIFICATIONS:
    import.meta.env.VITE_FEATURE_EMAIL_NOTIFICATIONS !== "false",
  PAYMENTS: import.meta.env.VITE_FEATURE_PAYMENTS === "true",
} as const;
