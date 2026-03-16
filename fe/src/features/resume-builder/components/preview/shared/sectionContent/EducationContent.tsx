import { EditableField } from "../../../inline/EditableField";
import { EditableDateRange } from "../../../inline/EditableDateRange";
import { EntryWrapper } from "../../../inline/EntryWrapper";
import type { SectionContentProps } from "./types";

export function EducationContent({
  setup,
  config,
  editable,
  sectionColor,
  inputClassName,
}: SectionContentProps) {
  const { resume, updateEducation, educationSection } = setup;
  const effectiveColor = sectionColor ?? setup.color;
  const inputClass = inputClassName ?? "";
  const spacing = config.entrySpacing?.education ?? "mb-2";
  const isMinimal = config.variant === "minimal";
  const isModern = config.variant === "modern";

  return (
    <>
      {resume.educations.map((edu) => (
        <EntryWrapper
          key={edu.id}
          entryId={edu.id}
          onRemove={educationSection.handleRemove}
          editable={editable}
          className={spacing}
        >
          {isMinimal ? (
            <>
              <p className="text-xs font-semibold">
                <EditableField
                  value={edu.degree}
                  onChange={(v) => updateEducation(edu.id, { degree: v })}
                  placeholder="Degree"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {(edu.field_of_study || editable) && (
                  <>
                    {edu.degree && edu.field_of_study ? ", " : ""}
                    <EditableField
                      value={edu.field_of_study}
                      onChange={(v) =>
                        updateEducation(edu.id, { field_of_study: v })
                      }
                      placeholder="Field of Study"
                      inputClassName={inputClass}
                      editable={editable}
                    />
                  </>
                )}
              </p>
              <p className="text-xs text-gray-500">
                <EditableField
                  value={edu.institution}
                  onChange={(v) =>
                    updateEducation(edu.id, { institution: v })
                  }
                  placeholder="Institution"
                  inputClassName={inputClass}
                  editable={editable}
                />
                {" \u00B7 "}
                <EditableDateRange
                  startDate={edu.start_date}
                  endDate={edu.end_date}
                  isCurrent={edu.is_current}
                  onStartDateChange={(v) =>
                    updateEducation(edu.id, { start_date: v })
                  }
                  onEndDateChange={(v) =>
                    updateEducation(edu.id, { end_date: v })
                  }
                  onIsCurrentChange={(v) =>
                    updateEducation(edu.id, {
                      is_current: v,
                      end_date: v ? "" : edu.end_date,
                    })
                  }
                  className="text-xs text-gray-500"
                  editable={editable}
                />
              </p>
            </>
          ) : (
            <>
              <div className="flex items-start justify-between">
                <div>
                  <div
                    className={`flex items-center gap-1 ${config.textSize} font-bold`}
                  >
                    <EditableField
                      value={edu.degree}
                      onChange={(v) =>
                        updateEducation(edu.id, { degree: v })
                      }
                      placeholder="Degree"
                      inputClassName={inputClass}
                      editable={editable}
                    />
                    {(edu.field_of_study || editable) && (
                      <>
                        <span>
                          {edu.degree && edu.field_of_study ? " in " : ""}
                        </span>
                        <EditableField
                          value={edu.field_of_study}
                          onChange={(v) =>
                            updateEducation(edu.id, { field_of_study: v })
                          }
                          placeholder="Field of Study"
                          inputClassName={inputClass}
                          editable={editable}
                        />
                      </>
                    )}
                  </div>
                  <EditableField
                    value={edu.institution}
                    onChange={(v) =>
                      updateEducation(edu.id, { institution: v })
                    }
                    placeholder="Institution"
                    as="p"
                    className={`${config.textSize} ${isModern ? "" : "text-gray-600"}`}
                    style={isModern ? { color: effectiveColor } : undefined}
                    inputClassName={inputClass}
                    editable={editable}
                  />
                </div>
                <EditableDateRange
                  startDate={edu.start_date}
                  endDate={edu.end_date}
                  isCurrent={edu.is_current}
                  onStartDateChange={(v) =>
                    updateEducation(edu.id, { start_date: v })
                  }
                  onEndDateChange={(v) =>
                    updateEducation(edu.id, { end_date: v })
                  }
                  onIsCurrentChange={(v) =>
                    updateEducation(edu.id, {
                      is_current: v,
                      end_date: v ? "" : edu.end_date,
                    })
                  }
                  className={`shrink-0 ${config.textSize} text-gray-500`}
                  editable={editable}
                />
              </div>
              {(edu.gpa || editable) && (
                <div
                  className={`flex items-center gap-1 ${config.textSize} text-gray-600`}
                >
                  <span>GPA:</span>
                  <EditableField
                    value={edu.gpa}
                    onChange={(v) => updateEducation(edu.id, { gpa: v })}
                    placeholder="3.8"
                    inputClassName={inputClass}
                    editable={editable}
                  />
                </div>
              )}
            </>
          )}
        </EntryWrapper>
      ))}
    </>
  );
}
