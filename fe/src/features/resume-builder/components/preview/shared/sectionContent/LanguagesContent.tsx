import { EditableField } from "../../../inline/EditableField";
import { EditableSelect } from "../../../inline/EditableSelect";
import { EntryWrapper } from "../../../inline/EntryWrapper";
import { PROFICIENCY_OPTIONS } from "../../../inline/constants";
import type { SectionContentProps } from "./types";

export function LanguagesContent({
  setup,
  config,
  editable,
  inputClassName,
}: SectionContentProps) {
  const { resume, updateLanguage, languagesSection } = setup;
  const inputClass = inputClassName ?? "";

  // Minimal: special read-only mode that joins with dots
  if (config.variant === "minimal") {
    if (editable) {
      return (
        <div className={config.languages.containerClass}>
          {resume.languages.map((l) => (
            <EntryWrapper
              key={l.id}
              entryId={l.id}
              onRemove={languagesSection.handleRemove}
              editable={editable}
            >
              <span className="text-xs text-gray-600">
                <EditableField
                  value={l.name}
                  onChange={(v) => updateLanguage(l.id, { name: v })}
                  placeholder="Language"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(l.proficiency || editable) && (
                  <>
                    {" ("}
                    <EditableSelect
                      value={l.proficiency}
                      onChange={(v) => updateLanguage(l.id, { proficiency: v })}
                      options={PROFICIENCY_OPTIONS}
                      placeholder="Proficiency"
                      className="text-xs"
                      editable={editable}
                    />
                    {")"}
                  </>
                )}
                {" \u00B7 "}
              </span>
            </EntryWrapper>
          ))}
        </div>
      );
    }
    return resume.languages.length > 0 ? (
      <p className="text-xs text-gray-600">
        {resume.languages
          .map((language) => {
            const proficiency = PROFICIENCY_OPTIONS.find(
              (option) => option.value.toLowerCase() === language.proficiency.toLowerCase(),
            )?.label ?? language.proficiency;
            return `${language.name}${proficiency ? ` (${proficiency})` : ""}`;
          })
          .join(" \u00B7 ")}
      </p>
    ) : null;
  }

  // Modern: vertical stack with opacity
  if (config.variant === "modern") {
    return (
      <div className={config.languages.containerClass}>
        {resume.languages.map((lang) => (
          <EntryWrapper
            key={lang.id}
            entryId={lang.id}
            onRemove={languagesSection.handleRemove}
            editable={editable}
          >
            <p className="text-xs">
              <EditableField
                value={lang.name}
                onChange={(v) => updateLanguage(lang.id, { name: v })}
                placeholder="Language"
                inputClassName={inputClass}
                editable={editable}
              />
              {(lang.proficiency || editable) && (
                <span className="opacity-70">
                  {" \u2014 "}
                  <EditableSelect
                    value={lang.proficiency}
                    onChange={(v) =>
                      updateLanguage(lang.id, { proficiency: v })
                    }
                    options={PROFICIENCY_OPTIONS}
                    placeholder="Proficiency"
                    className="text-xs"
                    editable={editable}
                  />
                </span>
              )}
            </p>
          </EntryWrapper>
        ))}
      </div>
    );
  }

  // Default: flex or grid with proficiency select
  return (
    <div className={config.languages.containerClass}>
      {resume.languages.map((lang) => (
        <EntryWrapper
          key={lang.id}
          entryId={lang.id}
          onRemove={languagesSection.handleRemove}
          editable={editable}
        >
          <span className={`${config.textSize} text-gray-700`}>
            <EditableField
              value={lang.name}
              onChange={(v) => updateLanguage(lang.id, { name: v })}
              placeholder="Language"
              inputClassName={inputClass}
              editable={editable}
            />
            {(lang.proficiency || editable) && (
              <>
                {" \u2014 "}
                <EditableSelect
                  value={lang.proficiency}
                  onChange={(v) =>
                    updateLanguage(lang.id, { proficiency: v })
                  }
                  options={PROFICIENCY_OPTIONS}
                  placeholder="Proficiency"
                  className={config.textSize}
                  editable={editable}
                />
              </>
            )}
          </span>
        </EntryWrapper>
      ))}
    </div>
  );
}
