import { StatusBadge } from "@/shared/ui/StatusBadge";
import type { JobStatus } from "@/shared/types/api";

const JOB_STATUS_VALUES: JobStatus[] = [
  "saved",
  "applied",
  "on_hold",
  "offer",
  "rejected",
  "archived",
];

interface StatusQuickPickProps {
  currentStatus: JobStatus;
  onSelect: (status: JobStatus) => void;
}

// One-click status list for context menus — offers every status except the
// current one (transition semantics are enforced by the backend).
export function StatusQuickPick({
  currentStatus,
  onSelect,
}: StatusQuickPickProps) {
  return (
    <div role="group">
      {JOB_STATUS_VALUES.filter((status) => status !== currentStatus).map(
        (status) => (
          <button
            key={status}
            role="menuitem"
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
              onSelect(status);
            }}
            onPointerDown={(e) => e.stopPropagation()}
            className="flex items-center gap-2 w-full px-3 py-2 text-sm hover:bg-accent text-left"
          >
            <StatusBadge status={status} size="sm" />
          </button>
        ),
      )}
    </div>
  );
}
