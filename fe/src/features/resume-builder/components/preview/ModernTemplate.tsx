import { EditableField } from "../inline/EditableField";
import { SectionDivider } from "../inline/SectionDivider";
import { EMPTY_CONTACT } from "../inline/constants";
import { useTemplateSetup } from "./shared/useTemplateSetup";
import { modernConfig } from "./shared/configs";
import { SectionRenderer } from "./shared/TemplateSections";
import { TwoColumnLayout } from "./TwoColumnLayout";

interface ModernTemplateProps {
  readonly editable?: boolean;
}

// Inline-editable inputs on colored bg need white text
const WHITE_INPUT_CLASS = "text-white placeholder:text-white/60";

export function ModernTemplate({ editable = false }: ModernTemplateProps) {
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
    visibleSections,
    mainSections,
    sidebarSections,
  } = setup;

  const renderContactBlock = (onColoredBg: boolean) => {
    const inputClass = onColoredBg ? WHITE_INPUT_CLASS : "";
    return (
      <div className={onColoredBg ? "mb-6" : "mb-4"}>
        <h2
          className={`mb-2 text-xs font-bold uppercase tracking-wider ${onColoredBg ? "opacity-80" : ""}`}
          style={onColoredBg ? undefined : { color: textColor }}
        >
          Contact
        </h2>
        <div className="space-y-1 text-xs opacity-90">
          {(contact?.email || editable) && (
            <EditableField
              value={contact?.email ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), email: v })
              }
              placeholder="email@example.com"
              as="p"
              type="email"
              inputClassName={inputClass}
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
              as="p"
              type="tel"
              inputClassName={inputClass}
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
              as="p"
              inputClassName={inputClass}
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
              as="p"
              type="url"
              inputClassName={inputClass}
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
              as="p"
              inputClassName={inputClass}
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
              inputClassName={inputClass}
              editable={editable}
            />
          )}
        </div>
      </div>
    );
  };

  // Single-column: name header, then iterate sections
  // (SectionRenderer returns null for "contact", so we handle it ourselves)
  if (!isTwoColumn) {
    return (
      <div>
        <div className="mb-4 border-b-2 pb-3" style={{ borderColor: color }}>
          <EditableField
            value={contact?.full_name ?? ""}
            onChange={(v) =>
              updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
            }
            placeholder="Your Name"
            as="h1"
            className="text-xl font-bold leading-tight"
            style={{ color: textColor }}
            editable={editable}
          />
        </div>
        {visibleSections.map((s) => (
          <div key={s.section_key} data-avoid-break>
            {s.section_key === "contact" ? (
              renderContactBlock(false)
            ) : (
              <SectionRenderer
                sectionKey={s.section_key}
                setup={setup}
                config={modernConfig}
                editable={editable}
              />
            )}
            <SectionDivider
              insertAtOrder={s.sort_order + 1}
              editable={editable}
              color={color}
            />
          </div>
        ))}
      </div>
    );
  }

  // Two-column: colored sidebar with name + sidebar sections
  const effectiveMode =
    layoutMode === "double-right" ? "double-right" : "double-left";

  return (
    <TwoColumnLayout
      sidebarWidth={sidebarWidth}
      layoutMode={effectiveMode}
      sidebarStyle={{
        backgroundColor: color,
        color: "white",
        padding: "1.25rem",
      }}
      sidebarContent={
        <>
          <div className="mb-6">
            <EditableField
              value={contact?.full_name ?? ""}
              onChange={(v) =>
                updateContact({ ...(contact ?? EMPTY_CONTACT), full_name: v })
              }
              placeholder="Your Name"
              as="h1"
              className="text-xl font-bold leading-tight"
              inputClassName={WHITE_INPUT_CLASS}
              editable={editable}
            />
          </div>
          {sidebarSections.map((s) => (
            <div key={s.section_key} data-avoid-break>
              {s.section_key === "contact" ? (
                renderContactBlock(true)
              ) : (
                <SectionRenderer
                  sectionKey={s.section_key}
                  setup={setup}
                  config={modernConfig}
                  editable={editable}
                  sectionColor="rgba(255,255,255,0.8)"
                  inputClassName={WHITE_INPUT_CLASS}
                />
              )}
              <SectionDivider
                insertAtOrder={s.sort_order + 1}
                editable={editable}
                color="white"
                column="sidebar"
              />
            </div>
          ))}
        </>
      }
      mainContent={
        <div className="p-5">
          {mainSections.map((s) => (
            <div key={s.section_key} data-avoid-break>
              {s.section_key === "contact" ? (
                renderContactBlock(false)
              ) : (
                <SectionRenderer
                  sectionKey={s.section_key}
                  setup={setup}
                  config={modernConfig}
                  editable={editable}
                />
              )}
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
    />
  );
}
