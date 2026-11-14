/**
 * Character limits the API enforces (most text columns are VARCHAR(255)).
 *
 * Mirrored on the inputs so a field stops accepting text at the limit, instead
 * of the whole form being rejected with a generic "Invalid request payload".
 */
export const FIELD_LIMITS = {
  title: 255,
  name: 255,
  location: 255,
  source: 100,
  url: 2048,
} as const;
