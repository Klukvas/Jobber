import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { companiesService } from "@/services/companiesService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import { Label } from "@/shared/ui/Label";
import { Input } from "@/shared/ui/Input";
import { Button } from "@/shared/ui/Button";
import { Plus, Check, X } from "lucide-react";
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
  const [isAdding, setIsAdding] = useState(false);
  const [newName, setNewName] = useState("");

  const createMutation = useMutation({
    mutationFn: (name: string) => companiesService.create({ name }),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ["companies"] });
      showSuccessNotification(t("jobs.quickAddCompanySuccess"));
      onChange(created.id);
      setIsAdding(false);
      setNewName("");
    },
    onError: (error: Error) => {
      showErrorNotification(
        error.message || t("jobs.quickAddCompanyError"),
      );
    },
  });

  const handleSubmitQuickAdd = () => {
    const trimmed = newName.trim();
    if (!trimmed) return;
    createMutation.mutate(trimmed);
  };

  const handleCancel = () => {
    setIsAdding(false);
    setNewName("");
  };

  return (
    <div className="space-y-2">
      <Label htmlFor="company">{t("jobs.company")}</Label>

      {isAdding ? (
        <div className="flex items-center gap-2">
          <Input
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
            size="sm"
            onClick={handleSubmitQuickAdd}
            disabled={createMutation.isPending || !newName.trim()}
          >
            <Check className="h-4 w-4" />
          </Button>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={handleCancel}
            disabled={createMutation.isPending}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      ) : (
        <div className="flex items-center gap-2">
          <select
            id="company"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
            size="sm"
            variant="outline"
            onClick={() => setIsAdding(true)}
            title={t("jobs.quickAddCompany")}
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      )}

      {hint && !isAdding && (
        <p className="text-xs text-muted-foreground">{hint}</p>
      )}
    </div>
  );
}
