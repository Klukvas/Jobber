import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { calendarService } from "@/services/calendarService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";

interface ScheduleStageModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  stageId: string;
  stageName: string;
  applicationId: string;
}

function ModalContent({
  onOpenChange,
  stageId,
  stageName,
  applicationId,
}: ScheduleStageModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [title, setTitle] = useState(stageName);
  const [startTime, setStartTime] = useState("");
  const [durationMin, setDurationMin] = useState(60);
  const [description, setDescription] = useState("");

  const createMutation = useMutation({
    mutationFn: calendarService.createEvent,
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["application-stages", applicationId],
      });
      showSuccessNotification(t("applications.schedule.createSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("applications.schedule.createError"),
      );
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim() || !startTime) return;

    // Build ISO string with timezone offset to preserve user's local time
    const startDate = new Date(startTime);
    const tzOffset = -startDate.getTimezoneOffset();
    const sign = tzOffset >= 0 ? "+" : "-";
    const pad = (n: number) => String(Math.floor(Math.abs(n))).padStart(2, "0");
    const isoWithTz = `${startTime}:00${sign}${pad(tzOffset / 60)}:${pad(tzOffset % 60)}`;

    createMutation.mutate({
      stage_id: stageId,
      title: title.trim(),
      start_time: isoWithTz,
      duration_min: durationMin,
      description: description || undefined,
    });
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("applications.schedule.title")}</DialogTitle>
        <DialogDescription>
          {t("applications.schedule.description")}
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleSubmit}>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="event-title">
              {t("applications.schedule.eventTitle")} *
            </Label>
            <Input
              id="event-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="start-time">
              {t("applications.schedule.startTime")} *
            </Label>
            <Input
              id="start-time"
              type="datetime-local"
              value={startTime}
              onChange={(e) => setStartTime(e.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="duration">
              {t("applications.schedule.duration")}
            </Label>
            <select
              id="duration"
              value={durationMin}
              onChange={(e) => setDurationMin(Number(e.target.value))}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value={30}>
                30 {t("applications.schedule.minutes")}
              </option>
              <option value={60}>
                60 {t("applications.schedule.minutes")}
              </option>
              <option value={90}>
                90 {t("applications.schedule.minutes")}
              </option>
              <option value={120}>
                120 {t("applications.schedule.minutes")}
              </option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="event-description">
              {t("applications.schedule.eventDescription")}
            </Label>
            <textarea
              id="event-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              placeholder={t("applications.schedule.descriptionPlaceholder")}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            type="submit"
            disabled={createMutation.isPending || !title.trim() || !startTime}
          >
            {createMutation.isPending
              ? t("common.loading")
              : t("applications.schedule.schedule")}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

export function ScheduleStageModal(props: ScheduleStageModalProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent onClose={() => props.onOpenChange(false)}>
        <ModalContent
          key={`schedule-${props.stageId}-${props.open}`}
          {...props}
        />
      </DialogContent>
    </Dialog>
  );
}
