import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { format } from "date-fns";
import { Bell, BellPlus, Trash2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import { Checkbox } from "@/shared/ui/Checkbox";
import { remindersService } from "@/services/remindersService";
import { useDateLocale } from "@/shared/lib/dateFnsLocale";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import type { ReminderDTO } from "@/shared/types/api";

interface JobRemindersProps {
  readonly jobId: string;
}

export function JobReminders({ jobId }: JobRemindersProps) {
  const { t } = useTranslation();
  const dateLocale = useDateLocale();
  const queryClient = useQueryClient();

  const [message, setMessage] = useState("");
  const [remindAt, setRemindAt] = useState("");

  const remindersKey = ["job-reminders", jobId];

  const { data: remindersData, isLoading } = useQuery({
    queryKey: remindersKey,
    queryFn: () => remindersService.listByJob(jobId),
    enabled: !!jobId,
  });
  const reminders = remindersData ?? [];

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: remindersKey });

  const addMutation = useMutation({
    mutationFn: () =>
      remindersService.create({
        job_id: jobId,
        message: message.trim(),
        // datetime-local is local time; convert to a UTC ISO string for the API.
        remind_at: new Date(remindAt).toISOString(),
      }),
    onSuccess: () => {
      invalidate();
      setMessage("");
      setRemindAt("");
      showSuccessNotification(t("jobs.reminderAddedSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.reminderAddedError"));
    },
  });

  const toggleMutation = useMutation({
    mutationFn: (reminder: ReminderDTO) =>
      remindersService.update(reminder.id, { is_done: !reminder.is_done }),
    onSuccess: () => invalidate(),
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.reminderUpdateError"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => remindersService.delete(id),
    onSuccess: () => {
      invalidate();
      showSuccessNotification(t("jobs.reminderDeletedSuccess"));
    },
    onError: (err: Error) => {
      showErrorNotification(err.message || t("jobs.reminderDeleteError"));
    },
  });

  const canSubmit =
    message.trim().length > 0 && remindAt !== "" && !addMutation.isPending;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    addMutation.mutate();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-lg">
          <Bell className="h-4 w-4" />
          {t("jobs.reminders")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {reminders.length > 0 && (
          <ul className="mb-4 space-y-2">
            {reminders.map((reminder) => (
              <li
                key={reminder.id}
                className="flex items-start gap-3 rounded-lg border bg-muted/50 p-3"
              >
                <Checkbox
                  checked={reminder.is_done}
                  onCheckedChange={() => toggleMutation.mutate(reminder)}
                  disabled={toggleMutation.isPending}
                  className="mt-0.5"
                />
                <div className="min-w-0 flex-1">
                  <p
                    className={`text-sm whitespace-pre-wrap ${
                      reminder.is_done
                        ? "text-muted-foreground line-through"
                        : ""
                    }`}
                  >
                    {reminder.message}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {format(new Date(reminder.remind_at), "PPp", {
                      locale: dateLocale,
                    })}
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() => deleteMutation.mutate(reminder.id)}
                  disabled={deleteMutation.isPending}
                  aria-label={t("common.delete")}
                  className="text-muted-foreground transition-colors hover:text-destructive disabled:opacity-50"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </li>
            ))}
          </ul>
        )}

        {!isLoading && reminders.length === 0 && (
          <p className="mb-4 text-sm text-muted-foreground">
            {t("jobs.noReminders")}
          </p>
        )}

        <form onSubmit={handleSubmit} className="space-y-2">
          <Textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder={t("jobs.reminderPlaceholder")}
            rows={2}
          />
          <Input
            type="datetime-local"
            value={remindAt}
            onChange={(e) => setRemindAt(e.target.value)}
            aria-label={t("jobs.reminderTime")}
          />
          <Button type="submit" size="sm" disabled={!canSubmit}>
            <BellPlus className="mr-2 h-4 w-4" />
            {t("jobs.addReminder")}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
