import { memo } from "react";
import { useDraggable } from "@dnd-kit/core";
import { useNavigate } from "react-router-dom";
import { ApplicationCardBase } from "./ApplicationCardBase";
import type { ApplicationDTO } from "@/shared/types/api";

interface ApplicationKanbanCardProps {
  application: ApplicationDTO;
  onAddComment: (application: ApplicationDTO) => void;
  onAddStage: (application: ApplicationDTO) => void;
  onChangeStatus: (application: ApplicationDTO) => void;
}

export const ApplicationKanbanCard = memo(function ApplicationKanbanCard({
  application,
  onAddComment,
  onAddStage,
  onChangeStatus,
}: ApplicationKanbanCardProps) {
  const navigate = useNavigate();
  const { attributes, listeners, setNodeRef, transform, isDragging } =
    useDraggable({
      id: application.id,
      data: { application },
    });

  const dragStyle = transform
    ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` }
    : undefined;

  return (
    <ApplicationCardBase
      application={application}
      onTitleClick={() => {
        if (!isDragging) navigate(`/app/applications/${application.id}`);
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
