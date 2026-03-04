/**
 * Feature flags — set to true to enable, false to hide from UI.
 *
 * GOOGLE_CALENDAR: Google Calendar integration (connect/disconnect in Settings,
 * "Schedule" button on application stages in Timeline).
 * Hidden because the OAuth app hasn't passed Google verification yet.
 * To re-enable: set to true and publish the OAuth app in Google Cloud Console.
 * See: docs/FEATURES.md
 */
export const FEATURES = {
  GOOGLE_CALENDAR: false,
} as const;
