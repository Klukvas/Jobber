import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { accentConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";

interface AccentTemplateProps {
  readonly editable?: boolean;
}

export function AccentTemplate({ editable = false }: AccentTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, textColor, contact, updateContact } = setup;

  return (
    <div>
      {/* Header - name with thick left accent bar */}
      <div className="mb-5">
        <div className="border-l-[6px] pl-4" style={{ borderColor: color }}>
          <EditableField
            value={contact?.full_name ?? ""}
            onChange={(v) =>
              updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
            }
            placeholder="Your Name"
            as="h1"
            className="text-2xl font-bold"
            style={{ color: textColor }}
            editable={editable}
          />
        </div>
        <div className="mt-2 flex flex-wrap gap-x-4 border-t border-gray-200 pt-2 text-xs text-gray-500">
          {(contact?.email || editable) && (
            <EditableField
              value={contact?.email ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), email: v })
              }
              placeholder="email@example.com"
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
              placeholder="+1 (555) 123-4567"
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
              placeholder="City, Country"
              editable={editable}
            />
          )}
          {(contact?.website || editable) && (
            <EditableField
              value={contact?.website ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), website: v })
              }
              placeholder="website.com"
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
              placeholder="linkedin.com/in/you"
              editable={editable}
            />
          )}
          {(contact?.github || editable) && (
            <EditableField
              value={contact?.github ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), github: v })
              }
              placeholder="github.com/you"
              editable={editable}
            />
          )}
        </div>
      </div>
      <TemplateLayout setup={setup} config={accentConfig} editable={editable} />
    </div>
  );
}
