import type { JobStatus } from "@/shared/types/api";

// Single source of truth for the job status enum, ordered the way pickers and
// board columns present it. Importing this everywhere means adding a new status
// (e.g. withdrawn/hired) is a one-line change instead of editing 3+ arrays.
export const JOB_STATUS_VALUES: JobStatus[] = [
  "saved",
  "applied",
  "on_hold",
  "offer",
  "rejected",
  "archived",
];

// i18n key per status (short label).
export const JOB_STATUS_LABEL_KEYS: Record<JobStatus, string> = {
  saved: "jobs.statusSaved",
  applied: "jobs.statusApplied",
  on_hold: "jobs.statusOnHold",
  offer: "jobs.statusOffer",
  rejected: "jobs.statusRejected",
  archived: "jobs.statusArchived",
};

// i18n key per status (longer description, shown in the status <select>).
export const JOB_STATUS_DESC_KEYS: Record<JobStatus, string> = {
  saved: "jobs.statusSavedDesc",
  applied: "jobs.statusAppliedDesc",
  on_hold: "jobs.statusOnHoldDesc",
  offer: "jobs.statusOfferDesc",
  rejected: "jobs.statusRejectedDesc",
  archived: "jobs.statusArchivedDesc",
};
