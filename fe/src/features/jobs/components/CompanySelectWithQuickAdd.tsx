import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { useSingleFlightMutation } from "@/shared/hooks/useSingleFlightMutation";
import { useCurrentFormSession } from "@/shared/hooks/useCurrentFormSession";
import { FIELD_LIMITS } from "@/shared/lib/fieldLimits";
import { companiesService } from "@/services/companiesService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { Label } from "@/shared/ui/Label";
import { Input } from "@/shared/ui/Input";
import { Button } from "@/shared/ui/Button";
import { Plus, Check, X, Loader2 } from "lucide-react";
import type { CompanyDTO } from "@/shared/types/api";

interface CompanySelectWithQuickAddProps {
  companies: CompanyDTO[];
  value: string;
  onChange: (companyId: string) => void;
  /** Hint text shown below the select (e.g. detected company name from import) */
  hint?: string;
}

export function CompanySelectWithQuickAdd({
  companies,
  value,
  onChange,
  hint,
}: CompanySelectWithQuickAddProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const fieldId = useId();
  const isCurrentSession = useCurrentFormSession();
  const [isAdding, setIsAdding] = useState(false);
  const [newName, setNewName] = useState("");

  const isNameTooLong = Array.from(newName.trim()).length > FIELD_LIMITS.name;
  const createMutation = useSingleFlightMutation({
    mutationFn: (name: string) => companiesService.create({ name }),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
      if (!isCurrentSession()) return;
      showSuccessNotification(t("jobs.quickAddCompanySuccess"));
      onChange(created.id);
      setIsAdding(false);
      setNewName("");
    },
    onError: (error: Error) => {
      if (!isCurrentSession()) return;
      showErrorNotification(
        error.message || t("jobs.quickAddCompanyError"),
      );
    },
  });

  const handleSubmitQuickAdd = () => {
    const trimmed = newName.trim();
    if (!trimmed || isNameTooLong || createMutation.isPending) return;
    createMutation.mutate(trimmed);
  };

  const handleCancel = () => {
    if (createMutation.isPending) return;
    setIsAdding(false);
    setNewName("");
  };

  return (
    <div className="space-y-2">
      <Label htmlFor={fieldId}>{isAdding ? t("companies.name") : t("jobs.company")}</Label>

      {isAdding ? (
        <div className="flex items-center gap-2">
          <Input
            id={fieldId}
            // HTML counts UTF-16 units; the API limit counts Unicode code points.
            maxLength={FIELD_LIMITS.name * 2}
            aria-invalid={isNameTooLong}
            aria-describedby={isNameTooLong ? `${fieldId}-error` : undefined}
            autoFocus
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={t("jobs.quickAddCompanyPlaceholder")}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleSubmitQuickAdd();
              }
              if (e.key === "Escape") {
                handleCancel();
              }
            }}
            disabled={createMutation.isPending}
          />
          <Button
            type="button"
            size="icon"
            onClick={handleSubmitQuickAdd}
            disabled={createMutation.isPending || !newName.trim() || isNameTooLong}
            aria-label={t("companies.create")}
          >
            {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Check className="h-4 w-4" aria-hidden="true" />}
          </Button>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            onClick={handleCancel}
            disabled={createMutation.isPending}
            aria-label={t("common.cancel")}
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
      ) : (
        <div className="flex items-center gap-2">
          <select
            id={fieldId}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="flex h-10 max-sm:h-11 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <option value="">{t("jobs.selectCompany")}</option>
            {companies.map((company) => (
              <option key={company.id} value={company.id}>
                {company.name}
              </option>
            ))}
          </select>
          <Button
            type="button"
            size="icon"
            variant="outline"
            onClick={() => setIsAdding(true)}
            title={t("jobs.quickAddCompany")}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      )}

      {isAdding && isNameTooLong && (
        <p id={`${fieldId}-error`} role="alert" className="text-sm text-destructive">{t("companies.nameTooLong", { max: FIELD_LIMITS.name })}</p>
      )}
      {hint && !isAdding && (
        <p className="text-xs text-muted-foreground">{hint}</p>
      )}
    </div>
  );
}
