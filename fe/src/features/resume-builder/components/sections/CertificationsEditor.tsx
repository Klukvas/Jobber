import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { useSectionPersistence } from "../../hooks/useSectionPersistence";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import { Button } from "@/shared/ui/Button";
import { Plus, Trash2, ChevronDown, ChevronUp } from "lucide-react";
import type { CertificationDTO } from "@/shared/types/resume-builder";

function createEmptyCertification(sortOrder: number): CertificationDTO {
  return {
    id: crypto.randomUUID(),
    name: "",
    issuer: "",
    issue_date: "",
    expiry_date: "",
    url: "",
    sort_order: sortOrder,
  };
}

function CertificationCard({
  certification,
  onUpdate,
  onRemove,
}: {
  readonly certification: CertificationDTO;
  readonly onUpdate: (id: string, updates: Partial<CertificationDTO>) => void;
  readonly onRemove: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(true);

  const title =
    certification.name || certification.issuer
      ? `${certification.name}${certification.name && certification.issuer ? " - " : ""}${certification.issuer}`
      : t("resumeBuilder.certifications.newEntry");

  return (
    <div className="rounded-lg border bg-card">
      <button
        type="button"
        className="flex w-full items-center justify-between p-4 text-left"
        onClick={() => setIsOpen(!isOpen)}
      >
        <span className="font-medium">{title}</span>
        {isOpen ? (
          <ChevronUp className="h-4 w-4 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
        )}
      </button>

      {isOpen && (
        <div className="space-y-4 border-t px-4 pb-4 pt-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor={`cert-name-${certification.id}`}>
                {t("resumeBuilder.certifications.name")}
              </Label>
              <Input
                id={`cert-name-${certification.id}`}
                value={certification.name}
                onChange={(e) =>
                  onUpdate(certification.id, { name: e.target.value })
                }
                placeholder={t("resumeBuilder.certifications.namePlaceholder")}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`cert-issuer-${certification.id}`}>
                {t("resumeBuilder.certifications.issuer")}
              </Label>
              <Input
                id={`cert-issuer-${certification.id}`}
                value={certification.issuer}
                onChange={(e) =>
                  onUpdate(certification.id, { issuer: e.target.value })
                }
                placeholder={t(
                  "resumeBuilder.certifications.issuerPlaceholder",
                )}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`cert-issue-date-${certification.id}`}>
                {t("resumeBuilder.certifications.issueDate")}
              </Label>
              <Input
                id={`cert-issue-date-${certification.id}`}
                type="date"
                value={certification.issue_date}
                onChange={(e) =>
                  onUpdate(certification.id, {
                    issue_date: e.target.value,
                  })
                }
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={`cert-expiry-date-${certification.id}`}>
                {t("resumeBuilder.certifications.expiryDate")}
              </Label>
              <Input
                id={`cert-expiry-date-${certification.id}`}
                type="date"
                value={certification.expiry_date}
                onChange={(e) =>
                  onUpdate(certification.id, {
                    expiry_date: e.target.value,
                  })
                }
              />
            </div>

            <div className="space-y-1.5 sm:col-span-2">
              <Label htmlFor={`cert-url-${certification.id}`}>
                {t("resumeBuilder.certifications.url")}
              </Label>
              <Input
                id={`cert-url-${certification.id}`}
                type="url"
                value={certification.url}
                onChange={(e) =>
                  onUpdate(certification.id, { url: e.target.value })
                }
                placeholder={t("resumeBuilder.certifications.urlPlaceholder")}
              />
            </div>
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              variant="destructive"
              size="sm"
              onClick={() => onRemove(certification.id)}
            >
              <Trash2 className="mr-1 h-4 w-4" />
              {t("resumeBuilder.actions.remove")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export function CertificationsEditor() {
  const { t } = useTranslation();
  const certifications = useResumeBuilderStore(
    (s) => s.resume?.certifications ?? [],
  );
  const addCertification = useResumeBuilderStore((s) => s.addCertification);
  const updateCertification = useResumeBuilderStore(
    (s) => s.updateCertification,
  );
  const removeCertification = useResumeBuilderStore(
    (s) => s.removeCertification,
  );
  const { add: persistAdd, remove: persistRemove } =
    useSectionPersistence<CertificationDTO>("certifications");

  const handleAdd = () => {
    const item = createEmptyCertification(certifications.length);
    addCertification(item);
    persistAdd(item);
  };

  const handleRemove = (id: string) => {
    removeCertification(id);
    persistRemove(id);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">
          {t("resumeBuilder.sections.certifications")}
        </h2>
        <Button type="button" variant="outline" size="sm" onClick={handleAdd}>
          <Plus className="mr-1 h-4 w-4" />
          {t("resumeBuilder.actions.add")}
        </Button>
      </div>

      {certifications.length === 0 && (
        <p className="text-sm text-muted-foreground">
          {t("resumeBuilder.certifications.empty")}
        </p>
      )}

      <div className="space-y-3">
        {certifications.map((cert) => (
          <CertificationCard
            key={cert.id}
            certification={cert}
            onUpdate={updateCertification}
            onRemove={handleRemove}
          />
        ))}
      </div>
    </div>
  );
}
