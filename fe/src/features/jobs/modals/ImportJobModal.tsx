import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { jobsService } from "@/services/jobsService";
import { companiesService } from "@/services/companiesService";
import {
  showSuccessNotification,
  showErrorNotification,
} from "@/shared/lib/notifications";
import type { ImportParseResponse } from "@/shared/types/api";
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

interface ImportJobModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function ModalContent({ onOpenChange, open }: ImportJobModalProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [step, setStep] = useState<"url" | "preview">("url");
  const [url, setUrl] = useState("");
  const [parsedData, setParsedData] = useState<ImportParseResponse | null>(
    null,
  );

  // Editable preview fields
  const [title, setTitle] = useState("");
  const [companyName, setCompanyName] = useState("");
  const [location, setLocation] = useState("");
  const [description, setDescription] = useState("");
  const [companyId, setCompanyId] = useState("");

  const { data: companiesData } = useQuery({
    queryKey: ["companies"],
    queryFn: () => companiesService.list({ limit: 100, offset: 0 }),
    enabled: open && step === "preview",
  });

  const parseMutation = useMutation({
    mutationFn: (importUrl: string) => jobsService.importParse(importUrl),
    onSuccess: (data) => {
      setParsedData(data);
      setTitle(data.title);
      setCompanyName(data.company_name || "");
      setLocation(data.location || "");
      setDescription(data.description || "");
      setCompanyId("");
      setStep("preview");
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.import.parseError"));
    },
  });

  const createMutation = useMutation({
    mutationFn: jobsService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["jobs"] });
      showSuccessNotification(t("jobs.import.importSuccess"));
      onOpenChange(false);
    },
    onError: (error: Error) => {
      showErrorNotification(error.message || t("jobs.import.importError"));
    },
  });

  const handleParse = (e: React.FormEvent) => {
    e.preventDefault();
    if (url.trim()) {
      parseMutation.mutate(url.trim());
    }
  };

  const handleImport = (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;

    createMutation.mutate({
      title: title.trim(),
      company_id: companyId || undefined,
      url: parsedData?.url || url || undefined,
      source: parsedData?.source || undefined,
      notes: description || undefined,
    });
  };

  const handleBack = () => {
    setStep("url");
    setParsedData(null);
  };

  if (step === "url") {
    return (
      <>
        <DialogHeader>
          <DialogTitle>{t("jobs.import.title")}</DialogTitle>
          <DialogDescription>{t("jobs.import.description")}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleParse}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="import-url">{t("jobs.import.urlLabel")}</Label>
              <Input
                id="import-url"
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder={t("jobs.import.urlPlaceholder")}
                required
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
              disabled={parseMutation.isPending || !url.trim()}
            >
              {parseMutation.isPending
                ? t("jobs.import.parsing")
                : t("jobs.import.parse")}
            </Button>
          </DialogFooter>
        </form>
      </>
    );
  }

  // Preview step
  const companies = companiesData?.items || [];
  // Try to auto-match company by name
  const matchedCompany =
    !companyId && companyName
      ? companies.find(
          (c) => c.name.toLowerCase() === companyName.toLowerCase(),
        )
      : null;
  const effectiveCompanyId = companyId || matchedCompany?.id || "";

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t("jobs.import.reviewTitle")}</DialogTitle>
        <DialogDescription>
          {t("jobs.import.reviewDescription")}
        </DialogDescription>
      </DialogHeader>
      <form onSubmit={handleImport}>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="import-title">{t("jobs.title_field")} *</Label>
            <Input
              id="import-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="import-company">{t("jobs.company")}</Label>
            <select
              id="import-company"
              value={effectiveCompanyId}
              onChange={(e) => setCompanyId(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value="">{t("jobs.selectCompany")}</option>
              {companies.map((company) => (
                <option key={company.id} value={company.id}>
                  {company.name}
                </option>
              ))}
            </select>
            {companyName && !effectiveCompanyId && (
              <p className="text-xs text-muted-foreground">
                {t("jobs.import.detectedCompany")}: {companyName}
              </p>
            )}
          </div>
          {location && (
            <div className="space-y-2">
              <Label>{t("jobs.import.location")}</Label>
              <Input value={location} disabled />
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="import-source">{t("jobs.source")}</Label>
            <Input
              id="import-source"
              value={parsedData?.source || ""}
              disabled
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="import-notes">{t("jobs.notes")}</Label>
            <textarea
              id="import-notes"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder={t("jobs.notesPlaceholder")}
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleBack}>
            {t("common.back")}
          </Button>
          <Button
            type="submit"
            disabled={createMutation.isPending || !title.trim()}
          >
            {createMutation.isPending
              ? t("common.loading")
              : t("jobs.import.importJob")}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

export function ImportJobModal({ open, onOpenChange }: ImportJobModalProps) {
  const [modalKey, setModalKey] = useState(0);

  const handleOpenChange = (isOpen: boolean) => {
    if (isOpen) {
      setModalKey((k) => k + 1);
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent onClose={() => handleOpenChange(false)}>
        <ModalContent
          key={modalKey}
          open={open}
          onOpenChange={handleOpenChange}
        />
      </DialogContent>
    </Dialog>
  );
}
