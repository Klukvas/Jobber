import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import { MessageCircleQuestion, Loader2 } from "lucide-react";
import { supportService } from "@/services/supportService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import { Label } from "@/shared/ui/Label";

export function SupportButton() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        aria-label={t("support.title")}
        className="fixed bottom-6 right-6 z-40 flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-transform hover:scale-105 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
      >
        <MessageCircleQuestion className="h-5 w-5" />
      </button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent onClose={() => setOpen(false)}>
          <SupportForm onClose={() => setOpen(false)} key={String(open)} />
        </DialogContent>
      </Dialog>
    </>
  );
}

function SupportForm({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const location = useLocation();
  const [subject, setSubject] = useState("");
  const [message, setMessage] = useState("");

  const mutation = useMutation({
    mutationFn: supportService.submit,
    onSuccess: () => {
      showSuccessNotification(t("support.success"));
      onClose();
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("support.error"));
    },
  });

  const MIN_SUBJECT = 3;
  const MIN_MESSAGE = 10;
  const subjectLen = subject.trim().length;
  const messageLen = message.trim().length;

  const canSubmit =
    subjectLen >= MIN_SUBJECT &&
    messageLen >= MIN_MESSAGE &&
    !mutation.isPending;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    mutation.mutate({
      subject: subject.trim(),
      message: message.trim(),
      page: location.pathname,
    });
  };

  return (
    <form onSubmit={handleSubmit}>
      <DialogHeader>
        <DialogTitle>{t("support.title")}</DialogTitle>
        <DialogDescription>{t("support.description")}</DialogDescription>
      </DialogHeader>

      <div className="mt-4 space-y-4">
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="support-subject">{t("support.subject")}</Label>
            {subjectLen > 0 && subjectLen < MIN_SUBJECT && (
              <span className="text-xs text-muted-foreground">
                {t("support.minChars", { count: MIN_SUBJECT - subjectLen })}
              </span>
            )}
          </div>
          <Input
            id="support-subject"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder={t("support.subjectPlaceholder")}
            maxLength={200}
            autoFocus
          />
        </div>

        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label htmlFor="support-message">{t("support.message")}</Label>
            {messageLen > 0 && messageLen < MIN_MESSAGE && (
              <span className="text-xs text-muted-foreground">
                {t("support.minChars", { count: MIN_MESSAGE - messageLen })}
              </span>
            )}
          </div>
          <Textarea
            id="support-message"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder={t("support.messagePlaceholder")}
            maxLength={2000}
            rows={5}
          />
        </div>
      </div>

      <DialogFooter className="mt-6">
        <Button
          type="button"
          variant="outline"
          onClick={onClose}
          disabled={mutation.isPending}
        >
          {t("common.cancel")}
        </Button>
        <Button type="submit" disabled={!canSubmit}>
          {mutation.isPending && (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          )}
          {t("support.send")}
        </Button>
      </DialogFooter>
    </form>
  );
}
