import { useTranslation } from "react-i18next";
export function AIAvailabilityNotice({ unavailable }: { unavailable: boolean }) {
  const { t } = useTranslation();
  return unavailable ? <p role="status" className="rounded-md border bg-muted/50 p-3 text-sm text-muted-foreground">{t("ai.unavailable")}</p> : null;
}
