import { useId, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Pencil, Plus, Trash2, Users } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import { Button } from "@/shared/ui/Button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import { contactsService, type Contact } from "@/services/contactsService";
import { channelLink, emailLink, type ContactFields } from "./validation";
import { ContactForm } from "./ContactForm";

type Editor = { kind: "add" } | { kind: "edit"; contact: Contact } | null;
export function JobContacts({ jobId }: { jobId: string }) {
  const { t } = useTranslation();
  const sectionId = useId();
  const client = useQueryClient();
  const key = ["job-contacts", jobId];
  const query = useQuery({
    queryKey: key,
    queryFn: ({ signal }) => contactsService.list(jobId, { signal }),
  });
  const [editor, setEditor] = useState<Editor>(null);
  const [deleting, setDeleting] = useState<Contact | null>(null);
  const [pending, setPending] = useState(false);
  const [deleteError, setDeleteError] = useState(false);
  const busy = useRef(false);
  const addButton = useRef<HTMLButtonElement>(null);
  const save = async (fields: ContactFields) => {
    const saved = await contactsService.save(
      jobId,
      editor?.kind === "edit" ? editor.contact.id : null,
      fields,
    );
    await client.cancelQueries({ queryKey: key, exact: true });
    client.setQueryData<Contact[]>(key, (rows = []) =>
      rows.some((row) => row.id === saved.id)
        ? rows.map((row) => (row.id === saved.id ? saved : row))
        : [...rows, saved],
    );
  };
  const remove = async () => {
    if (!deleting || busy.current) return;
    busy.current = true;
    setPending(true);
    setDeleteError(false);
    try {
      await contactsService.delete(jobId, deleting.id);
      await client.cancelQueries({ queryKey: key, exact: true });
      client.setQueryData<Contact[]>(key, (rows = []) =>
        rows.filter((row) => row.id !== deleting.id),
      );
      setDeleting(null);
      requestAnimationFrame(() => addButton.current?.focus());
    } catch {
      setDeleteError(true);
    } finally {
      busy.current = false;
      setPending(false);
    }
  };
  const actionClass = "h-11 w-11 sm:h-8 sm:w-8 shrink-0";
  return (
    <Card className="ph-no-capture ph-sensitive sentry-block">
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 space-y-0 pb-4">
        <CardTitle className="flex items-center gap-2 text-lg">
          <Users className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          {t("contacts.title")}
        </CardTitle>
        <Button
          ref={addButton}
          variant="outline"
          size="sm"
          onClick={() => setEditor({ kind: "add" })}
          disabled={query.isPending || query.isError}
        >
          <Plus className="mr-1 h-4 w-4" aria-hidden="true" />
          {t("contacts.add")}
        </Button>
      </CardHeader>
      <CardContent>
        {query.isPending ? (
          <p role="status" className="text-sm text-muted-foreground">
            {t("common.loading")}
          </p>
        ) : query.isError ? (
          <div role="alert" className="space-y-2">
            <p className="text-sm text-destructive">
              {t("contacts.loadError")}
            </p>
            <Button
              variant="outline"
              size="sm"
              onClick={() => void query.refetch()}
            >
              {t("contacts.retry")}
            </Button>
          </div>
        ) : !query.data?.length ? (
          <p className="text-sm text-muted-foreground">{t("contacts.empty")}</p>
        ) : (
          <ul className="divide-y">
            {query.data.map((contact, index) => (
              <li key={contact.id} className="py-3 first:pt-0 last:pb-0">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p
                      id={`${sectionId}-name-${index}`}
                      className="break-words font-medium"
                    >
                      {contact.name}
                    </p>
                    {contact.role && (
                      <p className="break-words text-sm text-muted-foreground">
                        {contact.role}
                      </p>
                    )}
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className={actionClass}
                      aria-label={t("contacts.edit")}
                      aria-describedby={`${sectionId}-name-${index}`}
                      onClick={() => setEditor({ kind: "edit", contact })}
                    >
                      <Pencil className="h-4 w-4" aria-hidden="true" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className={`${actionClass} text-muted-foreground hover:text-destructive`}
                      aria-label={t("contacts.delete")}
                      aria-describedby={`${sectionId}-name-${index}`}
                      onClick={() => {
                        setDeleting(contact);
                        setDeleteError(false);
                      }}
                    >
                      <Trash2 className="h-4 w-4" aria-hidden="true" />
                    </Button>
                  </div>
                </div>
                <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-sm">
                  {(["email", "telegram", "linkedin"] as const).map(
                    (channel) => {
                      const value = contact[channel];
                      const href =
                        channel === "email"
                          ? emailLink(value)
                          : channelLink(channel, value);
                      return value ? (
                        href ? (
                          <a
                            key={channel}
                            href={href}
                            target={channel === "email" ? undefined : "_blank"}
                            rel={
                              channel === "email"
                                ? undefined
                                : "noopener noreferrer"
                            }
                            className="min-w-0 break-all text-primary underline underline-offset-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                          >
                            {channel === "email"
                              ? value
                              : t(`contacts.fields.${channel}`)}
                          </a>
                        ) : (
                          <span
                            key={channel}
                            className="break-all text-muted-foreground"
                          >
                            {value}
                          </span>
                        )
                      ) : null;
                    },
                  )}
                </div>
                {contact.note && (
                  <p className="mt-2 whitespace-pre-wrap break-words text-sm text-muted-foreground">
                    {contact.note}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
      {editor && (
        <ContactForm
          contact={editor.kind === "edit" ? editor.contact : null}
          onClose={() => setEditor(null)}
          onSave={save}
        />
      )}
      <Dialog
        open={!!deleting}
        onOpenChange={(open) => {
          if (!open && !busy.current) setDeleting(null);
        }}
      >
        <DialogContent
          onClose={() => {
            if (!busy.current) setDeleting(null);
          }}
          className="ph-no-capture ph-sensitive sentry-block"
        >
          <DialogHeader>
            <DialogTitle>{t("contacts.delete")}</DialogTitle>
            <DialogDescription>
              {t("contacts.deleteConfirm", { name: deleting?.name })}
            </DialogDescription>
          </DialogHeader>
          {deleteError && (
            <p role="alert" className="mt-3 text-sm text-destructive">
              {t("contacts.deleteError")}
            </p>
          )}
          <DialogFooter className="mt-6">
            <Button
              variant="outline"
              disabled={pending}
              onClick={() => setDeleting(null)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              disabled={pending}
              onClick={() => void remove()}
            >
              {t(pending ? "common.loading" : "common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
