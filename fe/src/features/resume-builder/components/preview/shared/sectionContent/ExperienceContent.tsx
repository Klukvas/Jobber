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
              <div className="flex items-baseline justify-between">
                <p className="text-xs font-semibold">
                  <EditableField
                    value={exp.position}
                    onChange={(v) =>
                      updateExperience(exp.id, { position: v })
                    }
                    placeholder="Position"
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
                        placeholder="Company"
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
              {(exp.description || editable) && (
                <EditableTextarea
                  value={exp.description}
                  onChange={(v) =>
                    updateExperience(exp.id, { description: v })
                  }
                  placeholder="Describe responsibilities..."
                  className={`mt-1 ${config.textSize} ${config.leadingClass} text-gray-600`}
                  editable={editable}
                />
              )}
            </>
          ) : (
            <>
              <div className="flex items-start justify-between">
                <div>
                  <EditableField
                    value={exp.position}
                    onChange={(v) =>
                      updateExperience(exp.id, { position: v })
                    }
                    placeholder="Position"
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
                        placeholder="Company"
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
                            placeholder="Location"
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
                        placeholder="Company"
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
                            placeholder="Location"
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
                  placeholder="Describe your responsibilities and achievements..."
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
