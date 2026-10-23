import { useId, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/shared/ui/Button";
import { Input } from "@/shared/ui/Input";
import { Textarea } from "@/shared/ui/Textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/shared/ui/Dialog";
import {
  CONTACT_LIMITS,
  contactErrors,
  EMPTY_CONTACT,
  type ContactFields,
} from "./validation";
import type { Contact } from "@/services/contactsService";

type Props = {
  contact: Contact | null;
  onClose: () => void;
  onSave: (fields: ContactFields) => Promise<void>;
};
export function ContactForm({ contact, onClose, onSave }: Props) {
  const { t } = useTranslation();
  const id = useId();
  const nameRef = useRef<HTMLInputElement>(null);
  const busy = useRef(false);
  const [fields, setFields] = useState<ContactFields>(contact ?? EMPTY_CONTACT);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState(false);
  const [errors, setErrors] = useState<ReturnType<typeof contactErrors>>({});
  useLayoutEffect(() => {
    nameRef.current?.focus();
  }, []);
  const close = () => {
    if (!busy.current) onClose();
  };
  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (busy.current) return;
    const nextErrors = contactErrors(fields);
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) {
      document.getElementById(`${id}-${Object.keys(nextErrors)[0]}`)?.focus();
      return;
    }
    busy.current = true;
    setPending(true);
    setError(false);
    try {
      await onSave(fields);
      onClose();
    } catch {
      setError(true);
    } finally {
      busy.current = false;
      setPending(false);
    }
  };
  return (
    <Dialog open onOpenChange={close}>
      <DialogContent
        onClose={close}
        className="max-h-[90dvh] overflow-y-auto ph-no-capture ph-sensitive sentry-block"
      >
        <DialogHeader>
          <DialogTitle>
            {t(contact ? "contacts.edit" : "contacts.add")}
          </DialogTitle>
          <DialogDescription>{t("contacts.formHint")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} noValidate className="mt-4 space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            {(Object.keys(CONTACT_LIMITS) as (keyof ContactFields)[]).map(
              (key) => (
                <div
                  key={key}
                  className={key === "note" ? "sm:col-span-2" : "min-w-0"}
                >
                  <label
                    htmlFor={`${id}-${key}`}
                    className="mb-1 block text-sm font-medium"
                  >
                    {t(`contacts.fields.${key}`)}
                    {key === "name" ? " *" : ""}
                  </label>
                  {key === "note" ? (
                    <Textarea
                      id={`${id}-${key}`}
                      value={fields[key]}
                      rows={3}
                      disabled={pending}
                      onChange={(e) => {
                        const value = e.target.value;
                        setFields((current) => ({ ...current, [key]: value }));
                      }}
                      aria-invalid={!!errors[key]}
                      aria-describedby={
                        errors[key] ? `${id}-${key}-error` : undefined
                      }
                    />
                  ) : (
                    <Input
                      ref={key === "name" ? nameRef : undefined}
                      id={`${id}-${key}`}
                      value={fields[key]}
                      type={key === "email" ? "email" : "text"}
                      autoCapitalize={
                        key === "name" || key === "role" ? "sentences" : "none"
                      }
                      autoCorrect="off"
                      required={key === "name"}
                      disabled={pending}
                      onChange={(e) => {
                        const value = e.target.value;
                        setFields((current) => ({ ...current, [key]: value }));
                      }}
                      aria-invalid={!!errors[key]}
                      aria-describedby={
                        errors[key] ? `${id}-${key}-error` : undefined
                      }
                      placeholder={
                        key === "telegram"
                          ? "@username"
                          : key === "linkedin"
                            ? "linkedin.com/in/username"
                            : undefined
                      }
                    />
                  )}
                  {errors[key] && (
                    <p
                      id={`${id}-${key}-error`}
                      className="mt-1 text-sm text-destructive"
                    >
                      {t(
                        errors[key] === "required"
                          ? "contacts.required"
                          : `contacts.invalid.${key}`,
                        { max: CONTACT_LIMITS[key] },
                      )}
                    </p>
                  )}
                </div>
              ),
            )}
          </div>
          {error && (
            <p role="alert" className="text-sm text-destructive">
              {t("contacts.saveError")}
            </p>
          )}
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={close}
              disabled={pending}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={pending}>
              {t(pending ? "common.loading" : "common.save")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
