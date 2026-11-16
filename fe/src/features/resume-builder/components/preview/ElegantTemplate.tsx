import { useTranslation } from "react-i18next";
import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { elegantConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";

interface ElegantTemplateProps {
  readonly editable?: boolean;
}

export function ElegantTemplate({ editable = false }: ElegantTemplateProps) {
  const { t } = useTranslation();
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, textColor, contact, updateContact } = setup;

  return (
    <div>
      {/* Header */}
      <div className="mb-4">
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
        <div className="mt-1.5 flex flex-wrap items-center gap-x-1 text-xs text-gray-500">
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
            <>
              <span className="text-gray-300">|</span>
              <EditableField
                value={contact?.phone ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), phone: v })
                }
                placeholder={t("resumeBuilder.placeholders.phone")}
                type="tel"
                editable={editable}
              />
            </>
          )}
          {(contact?.location || editable) && (
            <>
              <span className="text-gray-300">|</span>
              <EditableField
                value={contact?.location ?? ""}
                onChange={(v) =>
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    location: v,
                  })
                }
                placeholder={t("resumeBuilder.placeholders.location")}
                editable={editable}
              />
            </>
          )}
          {(contact?.website || editable) && (
            <>
              <span className="text-gray-300">|</span>
              <EditableField
                value={contact?.website ?? ""}
                onChange={(v) =>
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    website: v,
                  })
                }
                placeholder={t("resumeBuilder.placeholders.website")}
                type="url"
                editable={editable}
              />
            </>
          )}
          {(contact?.linkedin || editable) && (
            <>
              <span className="text-gray-300">|</span>
              <EditableField
                value={contact?.linkedin ?? ""}
                onChange={(v) =>
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    linkedin: v,
                  })
                }
                placeholder={t("resumeBuilder.placeholders.linkedin")}
                editable={editable}
              />
            </>
          )}
          {(contact?.github || editable) && (
            <>
              <span className="text-gray-300">|</span>
              <EditableField
                value={contact?.github ?? ""}
                onChange={(v) =>
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    github: v,
                  })
                }
                placeholder={t("resumeBuilder.placeholders.github")}
                editable={editable}
              />
            </>
          )}
        </div>
        <hr className="mt-2.5 border-t-2" style={{ borderColor: color }} />
      </div>
      <TemplateLayout
        setup={setup}
        config={elegantConfig}
        editable={editable}
      />
    </div>
  );
}
