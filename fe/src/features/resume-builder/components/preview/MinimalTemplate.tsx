import { EditableField } from "../inline/EditableField";
import { SectionDivider } from "../inline/SectionDivider";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { minimalConfig } from "./shared/configs";
import { SectionRenderer, TemplateLayout } from "./shared/TemplateSections";
import { TwoColumnLayout } from "./TwoColumnLayout";

interface MinimalTemplateProps {
  readonly editable?: boolean;
}

export function MinimalTemplate({ editable = false }: MinimalTemplateProps) {
  const setup = useTemplateSetup();
  if (!setup) return null;

  const {
    color,
    textColor,
    contact,
    updateContact,
    isTwoColumn,
    layoutMode,
    sidebarWidth,
    mainSections,
    sidebarSections,
  } = setup;

  const headerBlock = (
    <div className="mb-6">
      <EditableField
        value={contact?.full_name ?? ""}
        onChange={(v) =>
          updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
        }
        placeholder="Your Name"
        as="h1"
        className="text-2xl font-light tracking-wide"
        style={{ color: textColor }}
        editable={editable}
      />
      {!isTwoColumn && (
        <>
          <div className="mt-2 flex flex-wrap gap-x-4 text-xs text-gray-500">
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
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    location: v,
                  })
                }
                placeholder="City, Country"
                editable={editable}
              />
            )}
            {(contact?.website || editable) && (
              <EditableField
                value={contact?.website ?? ""}
                onChange={(v) =>
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    website: v,
                  })
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
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    linkedin: v,
                  })
                }
                placeholder="linkedin.com/in/you"
                editable={editable}
              />
            )}
            {(contact?.github || editable) && (
              <EditableField
                value={contact?.github ?? ""}
                onChange={(v) =>
                  updateContact({
                    ...(contact ?? EMPTY_CONTACT),
                    github: v,
                  })
                }
                placeholder="github.com/you"
                editable={editable}
              />
            )}
          </div>
          <hr className="mt-3 border-t-2" style={{ borderColor: color }} />
        </>
      )}
    </div>
  );

  // Single-column: header with inline contact, then TemplateLayout
  if (!isTwoColumn) {
    return (
      <div>
        {headerBlock}
        <TemplateLayout
          setup={setup}
          config={minimalConfig}
          editable={editable}
        />
      </div>
    );
  }

  // Two-column: header (name only), then manual TwoColumnLayout
  // because SectionRenderer returns null for "contact" key and
  // Minimal needs to render a contact block in the sidebar.
  const effectiveMode =
    layoutMode === "double-right" ? "double-right" : "double-left";

  const renderContactBlock = () => (
    <div className="mb-4">
      <div className="space-y-1 text-xs text-gray-500">
        {(contact?.email || editable) && (
          <EditableField
            value={contact?.email ?? ""}
            onChange={(v) =>
              updateContact({ ...(contact ?? EMPTY_CONTACT), email: v })
            }
            placeholder="email@example.com"
            type="email"
            as="p"
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
            as="p"
            editable={editable}
          />
        )}
        {(contact?.location || editable) && (
          <EditableField
            value={contact?.location ?? ""}
            onChange={(v) =>
              updateContact({
                ...(contact ?? EMPTY_CONTACT),
                location: v,
              })
            }
            placeholder="City, Country"
            as="p"
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
            as="p"
            editable={editable}
          />
        )}
        {(contact?.linkedin || editable) && (
          <EditableField
            value={contact?.linkedin ?? ""}
            onChange={(v) =>
              updateContact({
                ...(contact ?? EMPTY_CONTACT),
                linkedin: v,
              })
            }
            placeholder="linkedin.com/in/you"
            as="p"
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
            as="p"
            editable={editable}
          />
        )}
      </div>
    </div>
  );

  return (
    <div>
      {headerBlock}
      <TwoColumnLayout
        sidebarWidth={sidebarWidth}
        layoutMode={effectiveMode}
        mainContent={
          <div className="pr-4">
            {mainSections.map((s) => (
              <div key={s.section_key} data-avoid-break>
                <SectionRenderer
                  sectionKey={s.section_key}
                  setup={setup}
                  config={minimalConfig}
                  editable={editable}
                />
                <SectionDivider
                  insertAtOrder={s.sort_order + 1}
                  editable={editable}
                  color={color}
                  column="main"
                />
              </div>
            ))}
          </div>
        }
        sidebarContent={
          <div className="border-l border-gray-200 pl-4">
            {sidebarSections.map((s) => (
              <div key={s.section_key} data-avoid-break>
                {s.section_key === "contact" ? (
                  renderContactBlock()
                ) : (
                  <SectionRenderer
                    sectionKey={s.section_key}
                    setup={setup}
                    config={minimalConfig}
                    editable={editable}
                  />
                )}
                <SectionDivider
                  insertAtOrder={s.sort_order + 1}
                  editable={editable}
                  color={color}
                  column="sidebar"
                />
              </div>
            ))}
          </div>
        }
      />
    </div>
  );
}
