import { useTranslation } from "react-i18next";
import { EditableField } from "../../../inline/EditableField";
import { EditableTextarea } from "../../../inline/EditableTextarea";
import { EditableDateRange } from "../../../inline/EditableDateRange";
import { EntryWrapper } from "../../../inline/EntryWrapper";
import type { SectionContentProps } from "./types";

export function ExperienceContent({
  setup,
  config,
  editable,
  sectionColor,
  inputClassName,
}: SectionContentProps) {
  const { t } = useTranslation();
  const { resume, updateExperience, experienceSection } = setup;
  const effectiveColor = sectionColor ?? setup.color;
  const inputClass = inputClassName ?? "";
  const spacing = config.entrySpacing?.experience ?? "mb-3";
  const isMinimal = config.variant === "minimal";
  const isModern = config.variant === "modern";

  return (
    <>
      {resume.experiences.map((exp) => (
        <EntryWrapper
          key={exp.id}
          entryId={exp.id}
          onRemove={experienceSection.handleRemove}
          editable={editable}
          className={spacing}
        >
          {isMinimal ? (
            <>
              {/* The text side wraps anywhere (the dates are shrink-0): one
                  long unbroken word ran far past the edge of the sheet. */}
              <div className="flex items-baseline justify-between">
                <p className="min-w-0 text-xs font-semibold wrap-anywhere">
                  <EditableField
                    value={exp.position}
                    onChange={(v) =>
                      updateExperience(exp.id, { position: v })
                    }
                    placeholder={t("resumeBuilder.placeholders.position")}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  {(exp.company || editable) && (
                    <span className="font-normal text-gray-500">
                      {exp.position ? " at " : ""}
                      <EditableField
                        value={exp.company}
                        onChange={(v) =>
                          updateExperience(exp.id, { company: v })
                        }
                        placeholder={t("resumeBuilder.placeholders.company")}
                        inputClassName={inputClass}
                        editable={editable}
                      />
                    </span>
                  )}
                </p>
                <EditableDateRange
                  startDate={exp.start_date}
                  endDate={exp.end_date}
                  isCurrent={exp.is_current}
                  onRangeChange={(range) => updateExperience(exp.id, range)}
                  onStartDateChange={(v) =>
                    updateExperience(exp.id, { start_date: v })
                  }
                  onEndDateChange={(v) =>
                    updateExperience(exp.id, { end_date: v })
                  }
                  onIsCurrentChange={(v) =>
                    updateExperience(exp.id, {
                      is_current: v,
                      end_date: v ? "" : exp.end_date,
                    })
                  }
                  className="shrink-0 text-xs text-gray-400"
                  editable={editable}
                />
              </div>
              {(exp.location || editable) && (
                <EditableField
                  value={exp.location}
                  onChange={(v) => updateExperience(exp.id, { location: v })}
                  placeholder={t("resumeBuilder.placeholders.location")}
                  as="p"
                  className="text-xs text-gray-500"
                  inputClassName={inputClass}
                  editable={editable}
                />
              )}
              {(exp.description || editable) && (
                <EditableTextarea
                  value={exp.description}
                  onChange={(v) =>
                    updateExperience(exp.id, { description: v })
                  }
                  placeholder={t("resumeBuilder.placeholders.responsibilities")}
                  className={`mt-1 ${config.textSize} ${config.leadingClass} text-gray-600`}
                  editable={editable}
                />
              )}
            </>
          ) : (
            <>
              <div className="flex items-start justify-between">
                <div className="min-w-0 wrap-anywhere">
                  <EditableField
                    value={exp.position}
                    onChange={(v) =>
                      updateExperience(exp.id, { position: v })
                    }
                    placeholder={t("resumeBuilder.placeholders.position")}
                    as="p"
                    className={`${config.textSize} font-bold`}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                  {isModern ? (
                    <p
                      className={config.textSize}
                      style={{ color: effectiveColor }}
                    >
                      <EditableField
                        value={exp.company}
                        onChange={(v) =>
                          updateExperience(exp.id, { company: v })
                        }
                        placeholder={t("resumeBuilder.placeholders.company")}
                        inputClassName={inputClass}
                        editable={editable}
                      />
                      {(exp.location || editable) && (
                        <>
                          {exp.company && exp.location ? ", " : ""}
                          <EditableField
                            value={exp.location}
                            onChange={(v) =>
                              updateExperience(exp.id, { location: v })
                            }
                            placeholder={t("resumeBuilder.placeholders.location")}
                            inputClassName={inputClass}
                            editable={editable}
                          />
                        </>
                      )}
                    </p>
                  ) : (
                    <div
                      className={`flex items-center gap-1 ${config.textSize} text-gray-600`}
                    >
                      <EditableField
                        value={exp.company}
                        onChange={(v) =>
                          updateExperience(exp.id, { company: v })
                        }
                        placeholder={t("resumeBuilder.placeholders.company")}
                        inputClassName={inputClass}
                        editable={editable}
                      />
                      {(exp.location || editable) && (
                        <>
                          <span>
                            {exp.company && exp.location ? " \u2014 " : ""}
                          </span>
                          <EditableField
                            value={exp.location}
                            onChange={(v) =>
                              updateExperience(exp.id, { location: v })
                            }
                            placeholder={t("resumeBuilder.placeholders.location")}
                            inputClassName={inputClass}
                            editable={editable}
                          />
                        </>
                      )}
                    </div>
                  )}
                </div>
                <EditableDateRange
                  startDate={exp.start_date}
                  endDate={exp.end_date}
                  isCurrent={exp.is_current}
                  onRangeChange={(range) => updateExperience(exp.id, range)}
                  onStartDateChange={(v) =>
                    updateExperience(exp.id, { start_date: v })
                  }
                  onEndDateChange={(v) =>
                    updateExperience(exp.id, { end_date: v })
                  }
                  onIsCurrentChange={(v) =>
                    updateExperience(exp.id, {
                      is_current: v,
                      end_date: v ? "" : exp.end_date,
                    })
                  }
                  className={`shrink-0 ${config.textSize} text-gray-500`}
                  editable={editable}
                />
              </div>
              {(exp.description || editable) && (
                <EditableTextarea
                  value={exp.description}
                  onChange={(v) =>
                    updateExperience(exp.id, { description: v })
                  }
                  placeholder={t("resumeBuilder.placeholders.responsibilities")}
                  className={`mt-1 ${config.textSize} ${config.leadingClass} text-gray-700`}
                  editable={editable}
                />
              )}
            </>
          )}
        </EntryWrapper>
      ))}
    </>
  );
}
