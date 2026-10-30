/**
 * The link to the plan choice, for pages outside the app.
 *
 * The choice is a modal on Settings, so reaching it from a public page takes a
 * parameter: Settings opens the modal when it finds this one, and removes it.
 */
export const CHOOSE_PLAN_PARAM = "choosePlan";

export const CHOOSE_PLAN_PATH = `/app/settings?${CHOOSE_PLAN_PARAM}=1`;
