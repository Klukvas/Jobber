import { memo } from "react";
import { useDraggable } from "@dnd-kit/core";
import { useNavigate } from "react-router-dom";
import { JobCardBase } from "./JobCardBase";
import type { JobDTO } from "@/shared/types/api";

interface JobKanbanCardProps {
  job: JobDTO;
  onAddComment: (job: JobDTO) => void;
  onAddStage: (job: JobDTO) => void;
  onChangeStatus: (job: JobDTO) => void;
}

export const JobKanbanCard = memo(function JobKanbanCard({
  job,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: JobKanbanCardProps) {
  const navigate = useNavigate();
  const { attributes, listeners, setNodeRef, transform, isDragging } =
    useDraggable({
      id: job.id,
      data: { job },
    });

  const dragStyle = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined;

  return (
    <JobCardBase
      job={job}
      onTitleClick={() => {
        if (!isDragging) navigate(`/app/jobs/${job.id}`);
      }}
      onAddComment={onAddComment}
      onAddStage={onAddStage}
      onChangeStatus={onChangeStatus}
      dragRef={setNodeRef}
      dragStyle={dragStyle}
      dragProps={{ ...listeners, ...attributes }}
      isDragging={isDragging}
    />
  );
});
