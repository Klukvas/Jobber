import { type ReactNode } from "react";
import { Trash2 } from "lucide-react";
import { cn } from "@/shared/lib/utils";

interface EntryWrapperProps {
  readonly entryId: string;
  readonly onRemove: (id: string) => void;
  readonly editable?: boolean;
  readonly children: ReactNode;
  readonly className?: string;
}

export function EntryWrapper({
  entryId,
  onRemove,
  editable = true,
  children,
  className,
}: EntryWrapperProps) {
  if (!editable) {
    return <div className={className}>{children}</div>;
  }

  return (
    <div className={cn("group relative rounded transition-colors hover:bg-blue-50/30", className)}>
      {children}
      <button
        onClick={() => onRemove(entryId)}
        className="absolute -right-1 -top-1 z-10 flex h-5 w-5 items-center justify-center rounded-full bg-red-100 text-red-600 opacity-0 transition-opacity group-hover:opacity-100 hover:bg-red-200"
        aria-label="Remove entry"
      >
        <Trash2 className="h-3 w-3" />
      </button>
    </div>
  );
}
