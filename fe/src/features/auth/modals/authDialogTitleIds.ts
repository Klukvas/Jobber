/**
 * The heading ids that give each auth dialog its accessible name.
 *
 * A `role="dialog" aria-modal="true"` with no name is announced as just
 * "dialog": a screen-reader user landing in one is told a modal opened and
 * nothing about which. Each modal points `aria-labelledby` at the heading it
 * already draws, so the name is the visible title and the two cannot drift.
 *
 * They live together so the values are obviously distinct — the three modals
 * are siblings on the landing page, and a duplicate id would name the wrong
 * heading the moment two of them were ever on the page at once. They are
 * literals rather than `useId()` output because the id has to be stable enough
 * to assert on, and there is exactly one instance of each modal.
 */
export const AUTH_DIALOG_TITLE_IDS = {
  login: "login-dialog-title",
  register: "register-dialog-title",
  forgotPassword: "forgot-password-dialog-title",
} as const;
