import { memo, useCallback } from "react";
import { useDraggable } from "@dnd-kit/core";
import { useNavigate } from "react-router-dom";
import { JobCardBase } from "./JobCardBase";
import { useJobsListReturnState } from "@/features/jobs/lib/jobsListReturn";
import type { JobDTO } from "@/shared/types/api";

interface JobKanbanCardProps {
  job: JobDTO;
  flash?: boolean;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onDelete: (job: JobDTO) => void;
}

export const JobKanbanCard = memo(function JobKanbanCard({
  job,
  flash,
  onAddComment,
  onAddStage,
  onDelete,
}: JobKanbanCardProps) {
  const navigate = useNavigate();
  // So the detail page's Back returns to this board, filters and all.
  const returnState = useJobsListReturnState();
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    isDragging,
  } = useDraggable({
    id: job.id,
    data: { job },
  });

  // The card wrapper is both the draggable node and its keyboard activator.
  // dnd-kit's KeyboardSensor only starts a drag when the keydown's target IS
  // the activator node, so Enter/Space on the title button, the "Job actions"
  // button or a menu item (which bubble up to this wrapper) activate that
  // control instead of starting a drag. A focused card still drags with
  // Space/Enter.
  const setRefs = useCallback(
    (element: HTMLElement | null) => {
      setNodeRef(element);
      setActivatorNodeRef(element);
    },
    [setNodeRef, setActivatorNodeRef],
  );

  const dragStyle = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined;

  return (
    <JobCardBase
      job={job}
      onTitleClick={() => {
        if (!isDragging)
          navigate(`/app/jobs/${job.id}`, { state: returnState });
      }}
      onAddComment={onAddComment}
      onAddStage={onAddStage}
      onDelete={onDelete}
      dragRef={setRefs}
      dragStyle={dragStyle}
      dragProps={{ ...listeners, ...attributes }}
      isDragging={isDragging}
      flash={flash}
    />
  );
});
