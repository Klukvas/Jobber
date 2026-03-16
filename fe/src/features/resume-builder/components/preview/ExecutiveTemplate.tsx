import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { executiveConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";

interface ExecutiveTemplateProps {
  readonly editable?: boolean;
}

export function ExecutiveTemplate({
  editable = false,
}: ExecutiveTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, contact, updateContact } = setup;

  return (
    <div>
      {/* Header - dark bar */}
      <div className="mb-5 rounded px-6 py-4" style={{ backgroundColor: color }}>
        <EditableField
          value={contact?.full_name ?? ""}
          onChange={(v) =>
            updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
          }
          placeholder="Your Name"
          as="h1"
          className="text-xl font-bold tracking-wide text-white"
          inputClassName="text-white placeholder:text-white/60"
          editable={editable}
        />
        <div className="mt-2 flex flex-wrap gap-x-4 text-xs text-white/80">
          {(contact?.email || editable) && (
            <EditableField
              value={contact?.email ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), email: v })
              }
              placeholder="email@example.com"
              type="email"
              inputClassName="text-white placeholder:text-white/60"
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
              inputClassName="text-white placeholder:text-white/60"
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
              inputClassName="text-white placeholder:text-white/60"
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
              inputClassName="text-white placeholder:text-white/60"
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
              inputClassName="text-white placeholder:text-white/60"
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
              inputClassName="text-white placeholder:text-white/60"
              editable={editable}
            />
          )}
        </div>
      </div>
      <TemplateLayout setup={setup} config={executiveConfig} editable={editable} />
    </div>
  );
}
