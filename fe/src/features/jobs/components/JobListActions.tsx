import { useEffect, useId, useRef } from "react";
import { useTranslation } from "react-i18next";
import { GitBranch, MessageSquare, MoreVertical, Trash2 } from "lucide-react";
import type { JobDTO } from "@/shared/types/api";
import { TOUCH_ICON_TARGET } from "@/shared/lib/touchTargets";

type Props = {
  job: JobDTO;
  open: boolean;
  onToggleMenu: (id: string | null) => void;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
};

export function JobListActions({ job, open, onToggleMenu, onAddComment, onAddStage, onDelete }: Props) {
  const { t } = useTranslation();
  const id = useId();
  const trigger = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLDivElement>(null);
  const focusLast = useRef(false);
  useEffect(() => {
    if (!open) return;
    const items = menu.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]');
    items?.[focusLast.current ? items.length - 1 : 0]?.focus();
  }, [open]);
  const actions = [
    { key: "jobs.addComment", Icon: MessageSquare, run: onAddComment },
    { key: "jobs.addStage", Icon: GitBranch, run: onAddStage },
    { key: "jobs.delete", Icon: Trash2, run: onDelete },
  ];
  return (
    <div className={`absolute right-4 top-4 ${open ? "z-20" : "z-10"}`} onBlur={(event) => {
      if (open && !event.currentTarget.contains(event.relatedTarget)) onToggleMenu(null);
    }} onKeyDown={(event) => {
      if (event.key === "Escape" && open) {
        event.preventDefault();
        event.stopPropagation();
        onToggleMenu(null);
        trigger.current?.focus();
      }
    }}>
      <button ref={trigger} type="button" aria-label={t("jobs.actionsMenu")}
        aria-haspopup="menu" aria-expanded={open} aria-controls={open ? id : undefined}
        className={`${TOUCH_ICON_TARGET} rounded-md text-muted-foreground transition-colors hover:bg-accent`}
        onClick={(event) => { event.stopPropagation(); focusLast.current = false; onToggleMenu(open ? null : job.id); }}
        onKeyDown={(event) => {
          if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
          event.preventDefault();
          focusLast.current = event.key === "ArrowUp";
          if (open) {
            const items = menu.current?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]');
            items?.[focusLast.current ? items.length - 1 : 0]?.focus();
          } else onToggleMenu(job.id);
        }}>
        <MoreVertical className="h-4 w-4" />
      </button>
      {open && <div ref={menu} id={id} role="menu" aria-label={t("jobs.actionsMenu")}
        className="absolute right-0 mt-1 w-48 rounded-md border bg-popover shadow-lg"
        onKeyDown={(event) => {
          if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
          event.preventDefault();
          const items = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="menuitem"]'));
          const current = items.indexOf(document.activeElement as HTMLButtonElement);
          const next = event.key === "Home" ? 0 : event.key === "End" ? items.length - 1
            : (current + (event.key === "ArrowDown" ? 1 : -1) + items.length) % items.length;
          items[next]?.focus();
        }}>
        {actions.map(({ key, Icon, run }) => <button key={key} type="button" role="menuitem" tabIndex={-1}
          className={`flex min-h-11 w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-accent ${key === "jobs.delete" ? "text-destructive" : ""}`}
          onClick={(event) => { event.stopPropagation(); onToggleMenu(null); trigger.current?.focus(); run(job); }}>
          <Icon className="h-4 w-4" />{t(key)}
        </button>)}
      </div>}
    </div>
  );
}
