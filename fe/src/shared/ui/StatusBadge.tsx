import { useTranslation } from "react-i18next";
import type { ApplicationStatus } from "@/shared/types/api";

interface StatusBadgeProps {
  status: ApplicationStatus;
  size?: "sm" | "md" | "lg";
}

const statusClassNames: Record<ApplicationStatus, string> = {
  active:
    "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  on_hold:
    "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  offer: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  archived: "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400",
};

const statusLabelKeys: Record<ApplicationStatus, string> = {
  active: "applications.statusActive",
  on_hold: "applications.statusOnHold",
  rejected: "applications.statusRejected",
  offer: "applications.statusOffer",
  archived: "applications.statusArchived",
};

const sizeClasses = {
  sm: "text-xs px-2 py-0.5",
  md: "text-sm px-2.5 py-1",
  lg: "text-base px-3 py-1.5",
};

export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const { t } = useTranslation();
  const className = statusClassNames[status];
  const labelKey = statusLabelKeys[status];

  if (!className) {
    return (
      <span
        className={`inline-flex items-center rounded-full font-medium ${sizeClasses[size]} bg-gray-100 text-gray-800`}
      >
        {status}
      </span>
    );
  }

  return (
    <span
      className={`inline-flex items-center rounded-full font-medium ${sizeClasses[size]} ${className}`}
    >
      {t(labelKey)}
    </span>
  );
}
