import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { creativeConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";

interface CreativeTemplateProps {
  readonly editable?: boolean;
}

export function CreativeTemplate({ editable = false }: CreativeTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, textColor, contact, updateContact } = setup;

  return (
    <div>
      {/* Header with initials badge */}
      <div
        className="mb-5 flex items-center gap-4 border-b-2 pb-4"
        style={{ borderColor: color }}
      >
        {/* Initials circle */}
        <div
          className="flex h-14 w-14 shrink-0 items-center justify-center rounded-full text-lg font-bold text-white"
          style={{ backgroundColor: color }}
        >
          {(contact?.full_name ?? "")
            .split(" ")
            .map((w) => w[0])
            .filter(Boolean)
            .slice(0, 2)
            .join("")
            .toUpperCase()}
        </div>
        <div>
          <EditableField
            value={contact?.full_name ?? ""}
            onChange={(v) =>
              updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
            }
            placeholder="Your Name"
            as="h1"
            className="text-xl font-bold"
            style={{ color: textColor }}
            editable={editable}
          />
          <div className="mt-1 flex flex-wrap gap-x-3 text-xs text-gray-500">
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
      </div>
      <TemplateLayout
        setup={setup}
        config={creativeConfig}
        editable={editable}
      />
    </div>
  );
}
