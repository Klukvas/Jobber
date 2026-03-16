import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { boldConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";
import { Mail, Phone, MapPin, Globe, Linkedin, Github } from "lucide-react";

interface BoldTemplateProps {
  readonly editable?: boolean;
}

export function BoldTemplate({ editable = false }: BoldTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, contact, updateContact } = setup;

  return (
    <div>
      {/* Header - full-width colored banner */}
      <div className="mb-5 rounded-lg px-6 py-5" style={{ backgroundColor: color }}>
        <EditableField
          value={contact?.full_name ?? ""}
          onChange={(v) =>
            updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
          }
          placeholder="Your Name"
          as="h1"
          className="text-2xl font-extrabold tracking-wide text-white"
          inputClassName="text-white placeholder:text-white/60"
          editable={editable}
        />
        <div className="mt-3 flex flex-wrap gap-3 text-xs text-white/90">
          {(contact?.email || editable) && (
            <span className="flex items-center gap-1">
              <Mail className="h-3 w-3" />
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
            </span>
          )}
          {(contact?.phone || editable) && (
            <span className="flex items-center gap-1">
              <Phone className="h-3 w-3" />
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
            </span>
          )}
          {(contact?.location || editable) && (
            <span className="flex items-center gap-1">
              <MapPin className="h-3 w-3" />
              <EditableField
                value={contact?.location ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), location: v })
                }
                placeholder="City, Country"
                inputClassName="text-white placeholder:text-white/60"
                editable={editable}
              />
            </span>
          )}
          {(contact?.website || editable) && (
            <span className="flex items-center gap-1">
              <Globe className="h-3 w-3" />
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
            </span>
          )}
          {(contact?.linkedin || editable) && (
            <span className="flex items-center gap-1">
              <Linkedin className="h-3 w-3" />
              <EditableField
                value={contact?.linkedin ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), linkedin: v })
                }
                placeholder="linkedin.com/in/you"
                inputClassName="text-white placeholder:text-white/60"
                editable={editable}
              />
            </span>
          )}
          {(contact?.github || editable) && (
            <span className="flex items-center gap-1">
              <Github className="h-3 w-3" />
              <EditableField
                value={contact?.github ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), github: v })
                }
                placeholder="github.com/you"
                inputClassName="text-white placeholder:text-white/60"
                editable={editable}
              />
            </span>
          )}
        </div>
      </div>
      <TemplateLayout setup={setup} config={boldConfig} editable={editable} />
    </div>
  );
}
