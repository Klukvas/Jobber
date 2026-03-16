import { EditableField } from "../inline/EditableField";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { vividConfig } from "./shared/configs";
import { TemplateLayout } from "./shared/TemplateSections";
import { Mail, Phone, MapPin, Globe, Linkedin, Github } from "lucide-react";

interface VividTemplateProps {
  readonly editable?: boolean;
}

export function VividTemplate({ editable = false }: VividTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const { color, contact, updateContact } = setup;

  return (
    <div>
      {/* Header - two-tone: colored top + white contact row */}
      <div className="mb-5 overflow-hidden rounded-lg">
        <div className="px-6 py-4" style={{ backgroundColor: color }}>
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
        </div>
        <div className="flex flex-wrap gap-3 border border-t-0 border-gray-200 bg-white px-6 py-3 text-xs text-gray-600">
          {(contact?.email || editable) && (
            <span className="flex items-center gap-1">
              <span
                className="inline-flex h-4 w-4 items-center justify-center rounded-full text-white"
                style={{ backgroundColor: color }}
              >
                <Mail className="h-2.5 w-2.5" />
              </span>
              <EditableField
                value={contact?.email ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), email: v })
                }
                placeholder="email@example.com"
                type="email"
                editable={editable}
              />
            </span>
          )}
          {(contact?.phone || editable) && (
            <span className="flex items-center gap-1">
              <span
                className="inline-flex h-4 w-4 items-center justify-center rounded-full text-white"
                style={{ backgroundColor: color }}
              >
                <Phone className="h-2.5 w-2.5" />
              </span>
              <EditableField
                value={contact?.phone ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), phone: v })
                }
                placeholder="+1 (555) 123-4567"
                type="tel"
                editable={editable}
              />
            </span>
          )}
          {(contact?.location || editable) && (
            <span className="flex items-center gap-1">
              <span
                className="inline-flex h-4 w-4 items-center justify-center rounded-full text-white"
                style={{ backgroundColor: color }}
              >
                <MapPin className="h-2.5 w-2.5" />
              </span>
              <EditableField
                value={contact?.location ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), location: v })
                }
                placeholder="City, Country"
                editable={editable}
              />
            </span>
          )}
          {(contact?.website || editable) && (
            <span className="flex items-center gap-1">
              <span
                className="inline-flex h-4 w-4 items-center justify-center rounded-full text-white"
                style={{ backgroundColor: color }}
              >
                <Globe className="h-2.5 w-2.5" />
              </span>
              <EditableField
                value={contact?.website ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), website: v })
                }
                placeholder="website.com"
                type="url"
                editable={editable}
              />
            </span>
          )}
          {(contact?.linkedin || editable) && (
            <span className="flex items-center gap-1">
              <span
                className="inline-flex h-4 w-4 items-center justify-center rounded-full text-white"
                style={{ backgroundColor: color }}
              >
                <Linkedin className="h-2.5 w-2.5" />
              </span>
              <EditableField
                value={contact?.linkedin ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), linkedin: v })
                }
                placeholder="linkedin.com/in/you"
                editable={editable}
              />
            </span>
          )}
          {(contact?.github || editable) && (
            <span className="flex items-center gap-1">
              <span
                className="inline-flex h-4 w-4 items-center justify-center rounded-full text-white"
                style={{ backgroundColor: color }}
              >
                <Github className="h-2.5 w-2.5" />
              </span>
              <EditableField
                value={contact?.github ?? ""}
                onChange={(v) =>
                  updateContact({ ...(contact ?? EMPTY_CONTACT), github: v })
                }
                placeholder="github.com/you"
                editable={editable}
              />
            </span>
          )}
        </div>
      </div>
      <TemplateLayout setup={setup} config={vividConfig} editable={editable} />
    </div>
  );
}
