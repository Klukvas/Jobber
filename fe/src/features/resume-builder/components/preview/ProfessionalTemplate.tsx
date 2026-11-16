import { useTranslation } from "react-i18next";
import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { professionalConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";

interface ProfessionalTemplateProps {
  readonly editable?: boolean;
}

export function ProfessionalTemplate({
  editable = false,
}: ProfessionalTemplateProps) {
  const { t } = useTranslation();
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, textColor, contact, updateContact } = setup;

  return (
    <div>
      {/* Header */}
      <div
        className="mb-4 text-center border-b-2 pb-3"
        style={{ borderColor: color }}
      >
        <EditableField
          value={contact?.full_name ?? ""}
          onChange={(v) =>
            updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
          }
          placeholder={t("resumeBuilder.placeholders.fullName")}
          as="h1"
          className="text-2xl font-bold"
          style={{ color: textColor }}
          editable={editable}
        />
        <div data-text-contact className="mt-1 flex flex-wrap justify-center gap-x-3 text-xs text-gray-600">
          {(contact?.email || editable) && (
            <EditableField
              value={contact?.email ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), email: v })
              }
              placeholder={t("resumeBuilder.placeholders.email")}
              type="email"
              editable={editable}
            />
          )}
          {(contact?.phone || editable) && (
            <EditableField
              value={contact?.phone ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), phone: v })
              }
              placeholder={t("resumeBuilder.placeholders.phone")}
              type="tel"
              editable={editable}
            />
          )}
          {(contact?.location || editable) && (
            <EditableField
              value={contact?.location ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), location: v })
              }
              placeholder={t("resumeBuilder.placeholders.location")}
              editable={editable}
            />
          )}
          {(contact?.website || editable) && (
            <EditableField
              value={contact?.website ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), website: v })
              }
              placeholder={t("resumeBuilder.placeholders.website")}
              type="url"
              editable={editable}
            />
          )}
          {(contact?.linkedin || editable) && (
            <EditableField
              value={contact?.linkedin ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), linkedin: v })
              }
              placeholder={t("resumeBuilder.placeholders.linkedin")}
              editable={editable}
            />
          )}
          {(contact?.github || editable) && (
            <EditableField
              value={contact?.github ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), github: v })
              }
              placeholder={t("resumeBuilder.placeholders.github")}
              editable={editable}
            />
          )}
        </div>
      </div>
      <TemplateLayout
        setup={setup}
        config={professionalConfig}
        editable={editable}
      />
    </div>
  );
}
