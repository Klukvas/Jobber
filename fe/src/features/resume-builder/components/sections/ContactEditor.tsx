import { useTranslation } from "react-i18next";
import { useResumeBuilderStore } from "@/stores/resumeBuilderStore";
import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import type { ContactDTO } from "@/shared/types/resume-builder";

export function ContactEditor() {
  const { t } = useTranslation();
  const resume = useResumeBuilderStore((s) => s.resume);
  const updateContact = useResumeBuilderStore((s) => s.updateContact);

  const contact = resume?.contact ?? {
    full_name: "",
    email: "",
    phone: "",
    location: "",
    website: "",
    linkedin: "",
    github: "",
  };

  const handleChange = (field: keyof ContactDTO, value: string) => {
    updateContact({ ...contact, [field]: value });
  };

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">
        {t("resumeBuilder.sections.contact")}
      </h2>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="full_name">
            {t("resumeBuilder.contact.fullName")}
          </Label>
          <Input
            id="full_name"
            value={contact.full_name}
            onChange={(e) => handleChange("full_name", e.target.value)}
            placeholder={t("resumeBuilder.contact.fullNamePlaceholder")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="email">{t("resumeBuilder.contact.email")}</Label>
          <Input
            id="email"
            type="email"
            value={contact.email}
            onChange={(e) => handleChange("email", e.target.value)}
            placeholder={t("resumeBuilder.contact.emailPlaceholder")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="phone">{t("resumeBuilder.contact.phone")}</Label>
          <Input
            id="phone"
            value={contact.phone}
            onChange={(e) => handleChange("phone", e.target.value)}
            placeholder={t("resumeBuilder.contact.phonePlaceholder")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="location">
            {t("resumeBuilder.contact.location")}
          </Label>
          <Input
            id="location"
            value={contact.location}
            onChange={(e) => handleChange("location", e.target.value)}
            placeholder={t("resumeBuilder.contact.locationPlaceholder")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="website">{t("resumeBuilder.contact.website")}</Label>
          <Input
            id="website"
            value={contact.website}
            onChange={(e) => handleChange("website", e.target.value)}
            placeholder={t("resumeBuilder.contact.websitePlaceholder")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="linkedin">
            {t("resumeBuilder.contact.linkedin")}
          </Label>
          <Input
            id="linkedin"
            value={contact.linkedin}
            onChange={(e) => handleChange("linkedin", e.target.value)}
            placeholder={t("resumeBuilder.contact.linkedinPlaceholder")}
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="github">{t("resumeBuilder.contact.github")}</Label>
          <Input
            id="github"
            value={contact.github}
            onChange={(e) => handleChange("github", e.target.value)}
            placeholder={t("resumeBuilder.contact.githubPlaceholder")}
          />
        </div>
      </div>
    </div>
  );
}
