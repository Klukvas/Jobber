import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { compactConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";

interface CompactTemplateProps {
  readonly editable?: boolean;
}

export function CompactTemplate({ editable = false }: CompactTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, textColor, contact, updateContact } = setup;

  return (
    <div>
      {/* Compact header — name left, contact right */}
      <div
        className="mb-3 flex items-baseline justify-between border-b-2 pb-2"
        style={{ borderColor: color }}
      >
        <EditableField
          value={contact?.full_name ?? ""}
          onChange={(v) =>
            updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
          }
          placeholder="Your Name"
          as="h1"
          className="text-lg font-bold"
          style={{ color: textColor }}
          editable={editable}
        />
        <div className="flex flex-wrap justify-end gap-x-2 text-[10px] text-gray-500">
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
      <TemplateLayout
        setup={setup}
        config={compactConfig}
        editable={editable}
      />
    </div>
  );
}
